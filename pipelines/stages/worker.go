// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package stages

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mdhender/tnrpt/adapters"
	"github.com/mdhender/tnrpt/model"
	"github.com/mdhender/tnrpt/pipelines/parsers/bistre"
	"github.com/mdhender/tnrpt/pipelines/parsers/docx"
	"github.com/spf13/afero"
)

// WorkerService claims and executes pipeline jobs.
type WorkerService struct {
	store    WorkerStore
	dataDir  string
	workerID string
	fs       afero.Fs
}

// WorkerStore defines the store operations needed by WorkerService.
type WorkerStore interface {
	ClaimWork(ctx context.Context, stage, workerID string) (*model.Work, error)
	FinishWork(ctx context.Context, id int64, status, errorCode, errorMsg string) error
	InsertWork(ctx context.Context, work *model.Work) (int64, error)
	GetReportFileByID(ctx context.Context, id int64) (*model.ReportFile, error)

	// For parsing stage - persist extracted data
	InsertReportExtract(ctx context.Context, rx *model.ReportX) (int64, error)
	InsertUnitExtract(ctx context.Context, ux *model.UnitX) (int64, error)
	InsertAct(ctx context.Context, act *model.Act) (int64, error)
	InsertStep(ctx context.Context, step *model.Step) (int64, error)
}

// NewWorkerService creates a new WorkerService.
func NewWorkerService(store WorkerStore, dataDir, workerID string) *WorkerService {
	if workerID == "" {
		hostname, _ := os.Hostname()
		workerID = fmt.Sprintf("%s:%d", hostname, os.Getpid())
	}
	return &WorkerService{
		store:    store,
		dataDir:  dataDir,
		workerID: workerID,
		fs:       afero.NewOsFs(),
	}
}

// SetFS sets the filesystem for testing.
func (w *WorkerService) SetFS(fs afero.Fs) {
	w.fs = fs
}

// WorkResult represents the outcome of executing a job.
type WorkResult struct {
	Success      bool
	ErrorCode    string
	ErrorMessage string
}

// ClaimJob atomically claims a queued job for the given stage.
// Returns nil if no work is available.
func (w *WorkerService) ClaimJob(ctx context.Context, stage string) (*model.Work, error) {
	return w.store.ClaimWork(ctx, stage, w.workerID)
}

// ExecuteExtract reads a DOCX file, extracts text, and writes it to a .report.txt file.
// If the file is already a .txt file, this is a no-op (skip extraction).
// On success, creates a 'parse' work row for the next stage.
func (w *WorkerService) ExecuteExtract(ctx context.Context, job *model.Work, rf *model.ReportFile) error {
	fullPath := filepath.Join(w.dataDir, rf.FsPath)
	ext := strings.ToLower(filepath.Ext(rf.FsPath))

	if ext == ".txt" {
		return w.queueParseStage(ctx, job.ReportFileID)
	}

	data, err := afero.ReadFile(w.fs, fullPath)
	if err != nil {
		return &ErrWriteFile{Op: "read", Path: fullPath, Err: err}
	}

	parsed, err := docx.ParseReader(bytes.NewReader(data), true, true, true, false, false)
	if err != nil {
		return &ErrDocxCorrupt{Path: fullPath, Err: err}
	}

	txtPath := strings.TrimSuffix(fullPath, ext) + ".report.txt"
	if err := afero.WriteFile(w.fs, txtPath, parsed.Text, 0644); err != nil {
		return &ErrWriteFile{Op: "write", Path: txtPath, Err: err}
	}

	return w.queueParseStage(ctx, job.ReportFileID)
}

// ExecuteParse reads extracted text and parses it using the bistre parser.
// The parsed data is stored in the model tables.
func (w *WorkerService) ExecuteParse(ctx context.Context, job *model.Work, rf *model.ReportFile) error {
	txtPath := w.findTextFile(rf)
	if txtPath == "" {
		return &ErrWriteFile{Op: "find", Path: rf.FsPath, Err: fmt.Errorf("no text file found")}
	}

	data, err := afero.ReadFile(w.fs, txtPath)
	if err != nil {
		return &ErrWriteFile{Op: "read", Path: txtPath, Err: err}
	}

	fid := rf.Name
	tid := formatTurnID(rf.TurnNo)

	turn, err := bistre.ParseInput(
		fid, tid, data,
		true,  // acceptLoneDash
		false, // debugParser
		false, // debugSections
		false, // debugSteps
		false, // debugNodes
		false, // debugFleetMovement
		false, // experimentalUnitSplit
		false, // experimentalScoutStill
		bistre.ParseConfig{},
	)
	if err != nil {
		return &ErrParseSyntax{Line: 0, Msg: err.Error()}
	}

	_, err = adapters.BistreTurnToStoreWithReportFile(ctx, w.store, rf, turn)
	if err != nil {
		return &ErrDatabase{Op: "persist parse result", Err: err}
	}

	return nil
}

// FinishJob marks a job as completed (ok or failed) based on the result.
func (w *WorkerService) FinishJob(ctx context.Context, job *model.Work, result WorkResult) error {
	status := model.WorkStatusOk
	errorCode := ""
	errorMsg := ""

	if !result.Success {
		status = model.WorkStatusFailed
		errorCode = result.ErrorCode
		errorMsg = result.ErrorMessage
	}

	return w.store.FinishWork(ctx, job.ID, status, errorCode, errorMsg)
}

// GetReportFile retrieves the report file associated with a job.
func (w *WorkerService) GetReportFile(ctx context.Context, job *model.Work) (*model.ReportFile, error) {
	return w.store.GetReportFileByID(ctx, job.ReportFileID)
}

// ProcessJob claims, executes, and finishes a single job for the given stage.
// Returns (jobProcessed, error). jobProcessed is true if a job was claimed.
func (w *WorkerService) ProcessJob(ctx context.Context, stage string) (bool, error) {
	job, err := w.ClaimJob(ctx, stage)
	if err != nil {
		return false, fmt.Errorf("claim job: %w", err)
	}
	if job == nil {
		return false, nil
	}

	rf, err := w.GetReportFile(ctx, job)
	if err != nil {
		w.FinishJob(ctx, job, WorkResult{
			Success:      false,
			ErrorCode:    ErrCodeDatabase,
			ErrorMessage: fmt.Sprintf("get report file: %v", err),
		})
		return true, fmt.Errorf("get report file: %w", err)
	}
	if rf == nil {
		w.FinishJob(ctx, job, WorkResult{
			Success:      false,
			ErrorCode:    ErrCodeDatabase,
			ErrorMessage: "report file not found",
		})
		return true, fmt.Errorf("report file %d not found", job.ReportFileID)
	}

	var execErr error
	switch stage {
	case model.WorkStageExtract:
		execErr = w.ExecuteExtract(ctx, job, rf)
	case model.WorkStageParse:
		execErr = w.ExecuteParse(ctx, job, rf)
	default:
		execErr = fmt.Errorf("unknown stage: %s", stage)
	}

	if execErr != nil {
		w.FinishJob(ctx, job, WorkResult{
			Success:      false,
			ErrorCode:    ErrorCode(execErr),
			ErrorMessage: execErr.Error(),
		})
		return true, execErr
	}

	if err := w.FinishJob(ctx, job, WorkResult{Success: true}); err != nil {
		return true, fmt.Errorf("finish job: %w", err)
	}

	return true, nil
}

// queueParseStage creates a work row for the parse stage.
func (w *WorkerService) queueParseStage(ctx context.Context, reportFileID int64) error {
	work := &model.Work{
		ReportFileID: reportFileID,
		Stage:        model.WorkStageParse,
		Status:       model.WorkStatusQueued,
		Attempt:      0,
		AvailableAt:  time.Now().UTC(),
	}
	_, err := w.store.InsertWork(ctx, work)
	if err != nil {
		return &ErrDatabase{Op: "insert parse work", Err: err}
	}
	return nil
}

// findTextFile locates the text file for parsing.
// For DOCX files, looks for the .report.txt file; for .txt files, uses the original.
func (w *WorkerService) findTextFile(rf *model.ReportFile) string {
	fullPath := filepath.Join(w.dataDir, rf.FsPath)
	ext := strings.ToLower(filepath.Ext(rf.FsPath))

	if ext == ".txt" {
		return fullPath
	}

	txtPath := strings.TrimSuffix(fullPath, ext) + ".report.txt"
	exists, _ := afero.Exists(w.fs, txtPath)
	if exists {
		return txtPath
	}

	return ""
}

// formatTurnID converts a turn number (e.g., 89912) to turn ID format (e.g., "0899-12").
func formatTurnID(turnNo int) string {
	year := turnNo / 100
	month := turnNo % 100
	return fmt.Sprintf("%04d-%02d", year, month)
}

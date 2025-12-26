// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package stages

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/mdhender/tnrpt/model"
	"github.com/spf13/afero"
)

// IngestService handles file ingestion into the pipeline.
type IngestService struct {
	store   IngestStore
	dataDir string
	fs      afero.Fs
}

// IngestStore defines the store operations needed by IngestService.
type IngestStore interface {
	InsertUploadBatch(ctx context.Context, batch *model.UploadBatch) (int64, error)
	GetUploadBatch(ctx context.Context, id int64) (*model.UploadBatch, error)
	GetReportFileBySHA256(ctx context.Context, sha256 string) (*model.ReportFile, error)
	InsertReportFileWithBatch(ctx context.Context, rf *model.ReportFile) (int64, error)
	InsertWork(ctx context.Context, work *model.Work) (int64, error)
}

// NewIngestService creates a new IngestService.
func NewIngestService(store IngestStore, dataDir string) *IngestService {
	return &IngestService{
		store:   store,
		dataDir: dataDir,
		fs:      afero.NewOsFs(),
	}
}

// SetFS sets the filesystem for testing.
func (s *IngestService) SetFS(fs afero.Fs) {
	s.fs = fs
}

// IngestRequest contains the parameters for ingesting a file.
type IngestRequest struct {
	Game     string // e.g., "0301"
	ClanNo   string // e.g., "0512"
	TurnNo   int    // e.g., 89912 (year 899, month 12)
	Filename string // original filename
	Data     []byte // file content
}

// IngestResult contains the result of an ingest operation.
type IngestResult struct {
	ReportFileID int64
	WorkID       int64
	Duplicate    bool // true if file was already ingested (idempotent no-op)
}

// IngestFile ingests a single file into the pipeline.
// Returns IngestResult with Duplicate=true if the file already exists (idempotent no-op).
func (s *IngestService) IngestFile(ctx context.Context, batchID int64, req IngestRequest) (*IngestResult, error) {
	hash := sha256.Sum256(req.Data)
	hashStr := hex.EncodeToString(hash[:])

	existing, err := s.store.GetReportFileBySHA256(ctx, hashStr)
	if err != nil {
		return nil, fmt.Errorf("check duplicate: %w", err)
	}
	if existing != nil {
		return &IngestResult{
			ReportFileID: existing.ID,
			Duplicate:    true,
		}, nil
	}

	ext := strings.ToLower(filepath.Ext(req.Filename))
	mime := detectMime(ext)
	stdName := formatStandardFilename(req.Game, req.TurnNo, req.ClanNo, ext)

	fsPath := filepath.Join("batches", fmt.Sprintf("%d", batchID), stdName)
	fullPath := filepath.Join(s.dataDir, fsPath)

	if err := s.fs.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return nil, &ErrWriteFile{Op: "mkdir", Path: filepath.Dir(fullPath), Err: err}
	}
	if err := afero.WriteFile(s.fs, fullPath, req.Data, 0644); err != nil {
		return nil, &ErrWriteFile{Op: "write", Path: fullPath, Err: err}
	}

	rf := &model.ReportFile{
		Game:      req.Game,
		ClanNo:    req.ClanNo,
		TurnNo:    req.TurnNo,
		Name:      stdName,
		SHA256:    hashStr,
		Mime:      mime,
		CreatedAt: time.Now().UTC(),
		FsPath:    fsPath,
		BatchID:   &batchID,
	}
	rfID, err := s.store.InsertReportFileWithBatch(ctx, rf)
	if err != nil {
		return nil, &ErrDatabase{Op: "insert report_file", Err: err}
	}

	stage := determineStage(ext)
	work := &model.Work{
		ReportFileID: rfID,
		Stage:        stage,
		Status:       model.WorkStatusQueued,
		Attempt:      0,
		AvailableAt:  time.Now().UTC(),
	}
	workID, err := s.store.InsertWork(ctx, work)
	if err != nil {
		return nil, &ErrDatabase{Op: "insert work", Err: err}
	}

	return &IngestResult{
		ReportFileID: rfID,
		WorkID:       workID,
		Duplicate:    false,
	}, nil
}

// IngestBatch creates a batch and ingests multiple files.
func (s *IngestService) IngestBatch(ctx context.Context, game, clanNo string, turnNo int, createdBy string, files []IngestRequest) (int64, []IngestResult, error) {
	batch := &model.UploadBatch{
		Game:      game,
		ClanNo:    clanNo,
		TurnNo:    turnNo,
		CreatedBy: createdBy,
		CreatedAt: time.Now().UTC(),
	}
	batchID, err := s.store.InsertUploadBatch(ctx, batch)
	if err != nil {
		return 0, nil, &ErrDatabase{Op: "insert batch", Err: err}
	}

	var results []IngestResult
	for _, file := range files {
		file.Game = game
		file.ClanNo = clanNo
		file.TurnNo = turnNo
		result, err := s.IngestFile(ctx, batchID, file)
		if err != nil {
			return batchID, results, err
		}
		results = append(results, *result)
	}

	return batchID, results, nil
}

// formatStandardFilename generates the standard filename: GGGG.YYYY-MM.CCCC.{ext}
// Example: 0301.899-12.0512.docx
func formatStandardFilename(game string, turnNo int, clanNo string, ext string) string {
	year := turnNo / 100
	month := turnNo % 100
	return fmt.Sprintf("%s.%03d-%02d.%s%s", game, year, month, clanNo, ext)
}

// determineStage returns the initial pipeline stage based on file extension.
func determineStage(ext string) string {
	switch ext {
	case ".txt":
		return model.WorkStageParse
	default:
		return model.WorkStageExtract
	}
}

// detectMime returns the MIME type based on file extension.
func detectMime(ext string) string {
	switch ext {
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".txt":
		return "text/plain"
	case ".pdf":
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}

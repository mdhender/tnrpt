// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package model

import "context"

// Store is an interface for loading data.
type Store interface {
	AddReportFile(rf *ReportFile) error
	AddReport(rx *ReportX) error

	InsertReportExtract(ctx context.Context, rx *ReportX) (int64, error)
	InsertReportFile(ctx context.Context, rf *ReportFile) (int64, error)

	InsertUnitExtract(ctx context.Context, ux *UnitX) (int64, error)

	InsertAct(ctx context.Context, act *Act) (int64, error)
	InsertStep(ctx context.Context, step *Step) (int64, error)

	// stages

	InsertUploadBatch(ctx context.Context, batch *UploadBatch) (int64, error)
	GetUploadBatch(ctx context.Context, id int64) (*UploadBatch, error)
	InsertWork(ctx context.Context, work *Work) (int64, error)
	ClaimWork(ctx context.Context, stage, workerID string) (*Work, error)
	FinishWork(ctx context.Context, id int64, status, errorCode, errorMsg string) error
	ResetFailedWork(ctx context.Context, stage string) (int, error)
	GetFailedWork(ctx context.Context, stage string) ([]Work, error)
	GetWorkSummaryByBatch(ctx context.Context, batchID int64) (map[string]map[string]int, error)

	// ingest

	GetReportFileBySHA256(ctx context.Context, sha256 string) (*ReportFile, error)
	GetReportFileByID(ctx context.Context, id int64) (*ReportFile, error)
	InsertReportFileWithBatch(ctx context.Context, rf *ReportFile) (int64, error)
}

// Stats holds store statistics.
type Stats struct {
	Reports int
	Units   int
	Acts    int
	Steps   int
}

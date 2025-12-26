// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/mdhender/tnrpt/model"
)

// InsertUploadBatch inserts an UploadBatch and returns its assigned ID.
func (s *SQLiteStore) InsertUploadBatch(ctx context.Context, batch *model.UploadBatch) (int64, error) {
	const query = `
		INSERT INTO upload_batches (game, clan_no, turn_no, created_by, created_at)
		VALUES (?, ?, ?, ?, ?)
	`
	result, err := s.db.ExecContext(ctx, query,
		batch.Game,
		batch.ClanNo,
		batch.TurnNo,
		nullString(batch.CreatedBy),
		batch.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return 0, fmt.Errorf("insert upload_batch: %w", err)
	}
	return result.LastInsertId()
}

// GetUploadBatch retrieves an UploadBatch by ID.
func (s *SQLiteStore) GetUploadBatch(ctx context.Context, id int64) (*model.UploadBatch, error) {
	const query = `
		SELECT id, game, clan_no, turn_no, created_by, created_at
		FROM upload_batches
		WHERE id = ?
	`
	row := s.db.QueryRowContext(ctx, query, id)
	var batch model.UploadBatch
	var createdBy sql.NullString
	var createdAt string
	if err := row.Scan(
		&batch.ID,
		&batch.Game,
		&batch.ClanNo,
		&batch.TurnNo,
		&createdBy,
		&createdAt,
	); err != nil {
		return nil, fmt.Errorf("get upload_batch: %w", err)
	}
	if createdBy.Valid {
		batch.CreatedBy = createdBy.String
	}
	if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
		batch.CreatedAt = t
	}
	return &batch, nil
}

// InsertWork inserts a Work job and returns its assigned ID.
func (s *SQLiteStore) InsertWork(ctx context.Context, work *model.Work) (int64, error) {
	const query = `
		INSERT INTO work (report_file_id, stage, status, attempt, available_at)
		VALUES (?, ?, ?, ?, ?)
	`
	result, err := s.db.ExecContext(ctx, query,
		work.ReportFileID,
		work.Stage,
		work.Status,
		work.Attempt,
		work.AvailableAt.Format(time.RFC3339),
	)
	if err != nil {
		return 0, fmt.Errorf("insert work: %w", err)
	}
	return result.LastInsertId()
}

// ClaimWork atomically claims a queued job for a stage, returning nil if none available.
func (s *SQLiteStore) ClaimWork(ctx context.Context, stage, workerID string) (*model.Work, error) {
	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)

	const query = `
		UPDATE work
		SET status = 'running',
		    locked_by = ?,
		    locked_at = ?,
		    started_at = COALESCE(started_at, ?),
		    attempt = attempt + 1
		WHERE id = (
			SELECT id FROM work
			WHERE stage = ?
			  AND status = 'queued'
			  AND available_at <= ?
			ORDER BY available_at
			LIMIT 1
		)
		RETURNING id, report_file_id, stage, status, attempt, available_at,
		          locked_by, locked_at, started_at, finished_at, error_code, error_message
	`

	row := s.db.QueryRowContext(ctx, query, workerID, nowStr, nowStr, stage, nowStr)
	work, err := scanWork(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("claim work: %w", err)
	}
	return work, nil
}

// FinishWork updates a job's status to ok or failed with optional error info.
func (s *SQLiteStore) FinishWork(ctx context.Context, id int64, status, errorCode, errorMsg string) error {
	const query = `
		UPDATE work
		SET status = ?,
		    finished_at = ?,
		    error_code = ?,
		    error_message = ?,
		    locked_by = NULL,
		    locked_at = NULL
		WHERE id = ?
	`
	_, err := s.db.ExecContext(ctx, query,
		status,
		time.Now().UTC().Format(time.RFC3339),
		nullString(errorCode),
		nullString(errorMsg),
		id,
	)
	if err != nil {
		return fmt.Errorf("finish work: %w", err)
	}
	return nil
}

// ResetFailedWork resets failed jobs for a stage back to queued, returning count reset.
func (s *SQLiteStore) ResetFailedWork(ctx context.Context, stage string) (int, error) {
	const query = `
		UPDATE work
		SET status = 'queued',
		    available_at = ?,
		    locked_by = NULL,
		    locked_at = NULL,
		    finished_at = NULL,
		    error_code = NULL,
		    error_message = NULL
		WHERE stage = ?
		  AND status = 'failed'
	`
	result, err := s.db.ExecContext(ctx, query, time.Now().UTC().Format(time.RFC3339), stage)
	if err != nil {
		return 0, fmt.Errorf("reset failed work: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected: %w", err)
	}
	return int(n), nil
}

// GetFailedWork returns all failed jobs for a stage.
func (s *SQLiteStore) GetFailedWork(ctx context.Context, stage string) ([]model.Work, error) {
	const query = `
		SELECT id, report_file_id, stage, status, attempt, available_at,
		       locked_by, locked_at, started_at, finished_at, error_code, error_message
		FROM work
		WHERE stage = ?
		  AND status = 'failed'
		ORDER BY id
	`
	rows, err := s.db.QueryContext(ctx, query, stage)
	if err != nil {
		return nil, fmt.Errorf("get failed work: %w", err)
	}
	defer rows.Close()

	var works []model.Work
	for rows.Next() {
		work, err := scanWorkRows(rows)
		if err != nil {
			return nil, err
		}
		works = append(works, *work)
	}
	return works, rows.Err()
}

// GetWorkSummaryByBatch returns work counts grouped by stage and status for a batch.
// Returns map[stage]map[status]count.
func (s *SQLiteStore) GetWorkSummaryByBatch(ctx context.Context, batchID int64) (map[string]map[string]int, error) {
	const query = `
		SELECT w.stage, w.status, COUNT(*) as cnt
		FROM work w
		JOIN report_files rf ON w.report_file_id = rf.id
		WHERE rf.batch_id = ?
		GROUP BY w.stage, w.status
	`
	rows, err := s.db.QueryContext(ctx, query, batchID)
	if err != nil {
		return nil, fmt.Errorf("get work summary: %w", err)
	}
	defer rows.Close()

	result := make(map[string]map[string]int)
	for rows.Next() {
		var stage, status string
		var cnt int
		if err := rows.Scan(&stage, &status, &cnt); err != nil {
			return nil, fmt.Errorf("scan work summary: %w", err)
		}
		if result[stage] == nil {
			result[stage] = make(map[string]int)
		}
		result[stage][status] = cnt
	}
	return result, rows.Err()
}

// scanWork scans a Work from a sql.Row.
func scanWork(row *sql.Row) (*model.Work, error) {
	var w model.Work
	var availableAt, lockedBy, lockedAt, startedAt, finishedAt, errorCode, errorMessage sql.NullString
	if err := row.Scan(
		&w.ID, &w.ReportFileID, &w.Stage, &w.Status, &w.Attempt, &availableAt,
		&lockedBy, &lockedAt, &startedAt, &finishedAt, &errorCode, &errorMessage,
	); err != nil {
		return nil, err
	}
	w.AvailableAt = parseTime(availableAt.String)
	w.LockedBy = nullStringPtr(lockedBy)
	w.LockedAt = parseTimePtr(lockedAt)
	w.StartedAt = parseTimePtr(startedAt)
	w.FinishedAt = parseTimePtr(finishedAt)
	w.ErrorCode = nullStringPtr(errorCode)
	w.ErrorMessage = nullStringPtr(errorMessage)
	return &w, nil
}

// scanWorkRows scans a Work from sql.Rows.
func scanWorkRows(rows *sql.Rows) (*model.Work, error) {
	var w model.Work
	var availableAt, lockedBy, lockedAt, startedAt, finishedAt, errorCode, errorMessage sql.NullString
	if err := rows.Scan(
		&w.ID, &w.ReportFileID, &w.Stage, &w.Status, &w.Attempt, &availableAt,
		&lockedBy, &lockedAt, &startedAt, &finishedAt, &errorCode, &errorMessage,
	); err != nil {
		return nil, err
	}
	w.AvailableAt = parseTime(availableAt.String)
	w.LockedBy = nullStringPtr(lockedBy)
	w.LockedAt = parseTimePtr(lockedAt)
	w.StartedAt = parseTimePtr(startedAt)
	w.FinishedAt = parseTimePtr(finishedAt)
	w.ErrorCode = nullStringPtr(errorCode)
	w.ErrorMessage = nullStringPtr(errorMessage)
	return &w, nil
}

// Helper functions

func parseTime(s string) time.Time {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	return time.Time{}
}

func parseTimePtr(ns sql.NullString) *time.Time {
	if !ns.Valid || ns.String == "" {
		return nil
	}
	if t, err := time.Parse(time.RFC3339, ns.String); err == nil {
		return &t
	}
	return nil
}

func nullStringPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

// GetReportFileByID returns a report file by ID, or nil if not found.
func (s *SQLiteStore) GetReportFileByID(ctx context.Context, id int64) (*model.ReportFile, error) {
	const query = `
		SELECT id, game, clan_no, turn_no, name, sha256, mime, created_at, fs_path, batch_id
		FROM report_files
		WHERE id = ?
	`
	row := s.db.QueryRowContext(ctx, query, id)
	var rf model.ReportFile
	var createdAt string
	var fsPath sql.NullString
	var batchID sql.NullInt64
	if err := row.Scan(
		&rf.ID,
		&rf.Game,
		&rf.ClanNo,
		&rf.TurnNo,
		&rf.Name,
		&rf.SHA256,
		&rf.Mime,
		&createdAt,
		&fsPath,
		&batchID,
	); err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("get report_file by id: %w", err)
	}
	if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
		rf.CreatedAt = t
	}
	rf.FsPath = fsPath.String
	if batchID.Valid {
		rf.BatchID = &batchID.Int64
	}
	return &rf, nil
}

// GetReportFileBySHA256 returns a report file by SHA256 hash, or nil if not found.
func (s *SQLiteStore) GetReportFileBySHA256(ctx context.Context, sha256 string) (*model.ReportFile, error) {
	const query = `
		SELECT id, game, clan_no, turn_no, name, sha256, mime, created_at, fs_path, batch_id
		FROM report_files
		WHERE sha256 = ?
		LIMIT 1
	`
	row := s.db.QueryRowContext(ctx, query, sha256)
	var rf model.ReportFile
	var createdAt string
	var fsPath sql.NullString
	var batchID sql.NullInt64
	if err := row.Scan(
		&rf.ID,
		&rf.Game,
		&rf.ClanNo,
		&rf.TurnNo,
		&rf.Name,
		&rf.SHA256,
		&rf.Mime,
		&createdAt,
		&fsPath,
		&batchID,
	); err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("get report_file by sha256: %w", err)
	}
	if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
		rf.CreatedAt = t
	}
	rf.FsPath = fsPath.String
	if batchID.Valid {
		rf.BatchID = &batchID.Int64
	}
	return &rf, nil
}

// InsertReportFileWithBatch inserts a report_files row including fs_path and batch_id.
func (s *SQLiteStore) InsertReportFileWithBatch(ctx context.Context, rf *model.ReportFile) (int64, error) {
	const query = `
		INSERT INTO report_files (game, clan_no, turn_no, name, sha256, mime, created_at, fs_path, batch_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	var batchID any
	if rf.BatchID != nil {
		batchID = *rf.BatchID
	}
	result, err := s.db.ExecContext(ctx, query,
		rf.Game,
		rf.ClanNo,
		rf.TurnNo,
		rf.Name,
		rf.SHA256,
		rf.Mime,
		rf.CreatedAt.Format(time.RFC3339),
		nullString(rf.FsPath),
		batchID,
	)
	if err != nil {
		return 0, fmt.Errorf("insert report_file: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get report_file id: %w", err)
	}
	rf.ID = id
	return id, nil
}

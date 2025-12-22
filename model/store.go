// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package model

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// Store wraps a SQLite database connection for persisting turn report data.
type Store struct {
	db *sql.DB
}

// NewStore creates a new Store with the given data source name.
// Use ":memory:" for an in-memory database.
func NewStore(ctx context.Context, dsn string) (*Store, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Run the embedded schema
	if _, err := db.ExecContext(ctx, schemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("exec schema: %w", err)
	}

	return &Store{db: db}, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// InsertReportFile inserts a ReportFile and returns its assigned ID.
func (s *Store) InsertReportFile(ctx context.Context, rf *ReportFile) (int64, error) {
	const query = `
		INSERT INTO report_files (game, clan_no, turn_no, name, sha256, mime, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	result, err := s.db.ExecContext(ctx, query,
		rf.Game,
		rf.ClanNo,
		rf.TurnNo,
		rf.Name,
		rf.SHA256,
		rf.Mime,
		rf.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return 0, fmt.Errorf("insert report_file: %w", err)
	}
	return result.LastInsertId()
}

// InsertReportExtract inserts a ReportX and returns its assigned ID.
func (s *Store) InsertReportExtract(ctx context.Context, rx *ReportX) (int64, error) {
	const query = `
		INSERT INTO report_extracts (report_file_id, game, clan_no, turn_no, created_at)
		VALUES (?, ?, ?, ?, ?)
	`
	result, err := s.db.ExecContext(ctx, query,
		rx.ReportFileID,
		rx.Game,
		rx.ClanNo,
		rx.TurnNo,
		rx.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return 0, fmt.Errorf("insert report_extract: %w", err)
	}
	return result.LastInsertId()
}

// InsertUnitExtract inserts a UnitX and returns its assigned ID.
func (s *Store) InsertUnitExtract(ctx context.Context, ux *UnitX) (int64, error) {
	// Parse TNCoord to grid/col/row
	startGrid, startCol, startRow := parseTNCoord(ux.StartTN)
	endGrid, endCol, endRow := parseTNCoord(ux.EndTN)

	const query = `
		INSERT INTO unit_extracts (
			report_x_id, unit_id, turn_no,
			start_grid, start_col, start_row,
			end_grid, end_col, end_row,
			src_doc_id, src_note
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var srcDocID sql.NullInt64
	var srcNote sql.NullString
	if ux.Src != nil {
		srcDocID = sql.NullInt64{Int64: ux.Src.DocID, Valid: ux.Src.DocID != 0}
		srcNote = sql.NullString{String: ux.Src.Note, Valid: ux.Src.Note != ""}
	}

	result, err := s.db.ExecContext(ctx, query,
		ux.ReportXID,
		ux.UnitID,
		ux.TurnNo,
		startGrid,
		startCol,
		startRow,
		endGrid,
		endCol,
		endRow,
		srcDocID,
		srcNote,
	)
	if err != nil {
		return 0, fmt.Errorf("insert unit_extract: %w", err)
	}
	return result.LastInsertId()
}

// InsertAct inserts an Act and returns its assigned ID.
func (s *Store) InsertAct(ctx context.Context, act *Act) (int64, error) {
	// Parse dest TNCoord for goto acts
	destGrid, destCol, destRow := parseTNCoord(act.DestTN)

	const query = `
		INSERT INTO acts (
			unit_x_id, seq, kind, ok, note,
			target_unit_id, dest_grid, dest_col, dest_row,
			src_doc_id, src_turn_no, src_unit_id, src_act_seq, src_note
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var ok sql.NullInt64
	ok = sql.NullInt64{Int64: boolToInt(act.Ok), Valid: true}

	var targetUnitID sql.NullString
	if act.TargetUnitID != "" {
		targetUnitID = sql.NullString{String: act.TargetUnitID, Valid: true}
	}

	var srcDocID, srcTurnNo, srcActSeq sql.NullInt64
	var srcUnitID, srcNote sql.NullString
	if act.Src != nil {
		srcDocID = sql.NullInt64{Int64: act.Src.DocID, Valid: act.Src.DocID != 0}
		srcTurnNo = sql.NullInt64{Int64: int64(act.Src.TurnNo), Valid: act.Src.TurnNo != 0}
		srcUnitID = sql.NullString{String: act.Src.UnitID, Valid: act.Src.UnitID != ""}
		srcActSeq = sql.NullInt64{Int64: int64(act.Src.ActSeq), Valid: act.Src.ActSeq != 0}
		srcNote = sql.NullString{String: act.Src.Note, Valid: act.Src.Note != ""}
	}

	result, err := s.db.ExecContext(ctx, query,
		act.UnitXID,
		act.Seq,
		string(act.Kind),
		ok,
		nullString(act.Note),
		targetUnitID,
		nullString(destGrid),
		nullInt(destCol),
		nullInt(destRow),
		srcDocID,
		srcTurnNo,
		srcUnitID,
		srcActSeq,
		srcNote,
	)
	if err != nil {
		return 0, fmt.Errorf("insert act: %w", err)
	}
	return result.LastInsertId()
}

// InsertStep inserts a Step and its child records, returning the step's assigned ID.
func (s *Store) InsertStep(ctx context.Context, step *Step) (int64, error) {
	const query = `
		INSERT INTO steps (
			act_id, seq, kind, ok, note,
			dir, fail_why, terr, special, label,
			src_doc_id, src_turn_no, src_unit_id, src_act_seq, src_step_seq, src_note
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var ok sql.NullInt64
	ok = sql.NullInt64{Int64: boolToInt(step.Ok), Valid: true}

	var srcDocID, srcTurnNo, srcActSeq, srcStepSeq sql.NullInt64
	var srcUnitID, srcNote sql.NullString
	if step.Src != nil {
		srcDocID = sql.NullInt64{Int64: step.Src.DocID, Valid: step.Src.DocID != 0}
		srcTurnNo = sql.NullInt64{Int64: int64(step.Src.TurnNo), Valid: step.Src.TurnNo != 0}
		srcUnitID = sql.NullString{String: step.Src.UnitID, Valid: step.Src.UnitID != ""}
		srcActSeq = sql.NullInt64{Int64: int64(step.Src.ActSeq), Valid: step.Src.ActSeq != 0}
		srcStepSeq = sql.NullInt64{Int64: int64(step.Src.StepSeq), Valid: step.Src.StepSeq != 0}
		srcNote = sql.NullString{String: step.Src.Note, Valid: step.Src.Note != ""}
	}

	result, err := s.db.ExecContext(ctx, query,
		step.ActID,
		step.Seq,
		string(step.Kind),
		ok,
		nullString(step.Note),
		nullString(step.Dir),
		nullString(step.FailWhy),
		nullString(step.Terr),
		boolToInt(step.Special),
		nullString(step.Label),
		srcDocID,
		srcTurnNo,
		srcUnitID,
		srcActSeq,
		srcStepSeq,
		srcNote,
	)
	if err != nil {
		return 0, fmt.Errorf("insert step: %w", err)
	}

	stepID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get step id: %w", err)
	}

	// Insert child records for encounters
	if step.Enc != nil {
		if err := s.insertStepEncounters(ctx, stepID, step.Enc); err != nil {
			return 0, err
		}
	}

	// Insert borders
	for _, border := range step.Borders {
		if err := s.insertStepBorder(ctx, stepID, border); err != nil {
			return 0, err
		}
	}

	return stepID, nil
}

func (s *Store) insertStepEncounters(ctx context.Context, stepID int64, enc *Enc) error {
	for _, u := range enc.Units {
		const query = `INSERT INTO step_enc_units (step_id, unit_id, name, clan_no) VALUES (?, ?, ?, ?)`
		if _, err := s.db.ExecContext(ctx, query, stepID, u.UnitID, nullString(u.Name), nullString(u.ClanNo)); err != nil {
			return fmt.Errorf("insert step_enc_unit: %w", err)
		}
	}

	for _, st := range enc.Sets {
		const query = `INSERT INTO step_enc_sets (step_id, name, kind, clan_no) VALUES (?, ?, ?, ?)`
		if _, err := s.db.ExecContext(ctx, query, stepID, st.Name, nullString(st.Kind), nullString(st.ClanNo)); err != nil {
			return fmt.Errorf("insert step_enc_set: %w", err)
		}
	}

	for _, r := range enc.Rsrc {
		const query = `INSERT INTO step_enc_rsrc (step_id, kind, qty) VALUES (?, ?, ?)`
		if _, err := s.db.ExecContext(ctx, query, stepID, r.Kind, nullInt(r.Qty)); err != nil {
			return fmt.Errorf("insert step_enc_rsrc: %w", err)
		}
	}

	return nil
}

func (s *Store) insertStepBorder(ctx context.Context, stepID int64, border *BorderObs) error {
	const query = `INSERT INTO step_borders (step_id, dir, kind) VALUES (?, ?, ?)`
	if _, err := s.db.ExecContext(ctx, query, stepID, border.Dir, border.Kind); err != nil {
		return fmt.Errorf("insert step_border: %w", err)
	}
	return nil
}

// TableStats returns row counts for all tables.
func (s *Store) TableStats(ctx context.Context) (map[string]int64, error) {
	tables := []string{
		"report_files",
		"report_extracts",
		"unit_extracts",
		"acts",
		"steps",
		"step_enc_units",
		"step_enc_sets",
		"step_enc_rsrc",
		"step_borders",
		"tiles",
		"tile_units",
		"tile_sets",
		"tile_rsrc",
		"tile_borders",
		"tile_src",
		"render_jobs",
		"render_job_units",
		"render_job_turns",
	}

	stats := make(map[string]int64, len(tables))
	for _, table := range tables {
		var count int64
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
		if err := s.db.QueryRowContext(ctx, query).Scan(&count); err != nil {
			return nil, fmt.Errorf("count %s: %w", table, err)
		}
		stats[table] = count
	}

	return stats, nil
}

// Helper functions

func parseTNCoord(tn TNCoord) (grid string, col, row int) {
	s := string(tn)
	if s == "" || s == "N/A" {
		return "", 0, 0
	}
	// Expected format: "XX CCCC" where XX is grid, CCCC is col+row (e.g., "QQ 0205")
	if len(s) >= 7 && s[2] == ' ' {
		grid = s[:2]
		// Parse the 4-digit number: first 2 digits = col, last 2 = row
		if len(s) >= 7 {
			fmt.Sscanf(s[3:], "%02d%02d", &col, &row)
		}
	}
	return grid, col, row
}

func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullInt(n int) sql.NullInt64 {
	if n == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(n), Valid: true}
}

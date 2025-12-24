// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/mdhender/tnrpt/model"
)

// InsertReportFile inserts a ReportFile and returns its assigned ID.
func (s *SQLiteStore) InsertReportFile(ctx context.Context, rf *model.ReportFile) (int64, error) {
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
func (s *SQLiteStore) InsertReportExtract(ctx context.Context, rx *model.ReportX) (int64, error) {
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
func (s *SQLiteStore) InsertUnitExtract(ctx context.Context, ux *model.UnitX) (int64, error) {
	// Parse TNCoord to grid/col/row
	startGrid, startCol, startRow := parseTNCoord(ux.StartTN)
	endGrid, endCol, endRow := parseTNCoord(ux.EndTN)

	const query = `
		INSERT INTO unit_extracts (
			report_x_id, unit_id, clan_id, turn_no,
			start_grid, start_col, start_row,
			end_grid, end_col, end_row,
			src_doc_id, src_note
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
		ux.ClanID,
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
func (s *SQLiteStore) InsertAct(ctx context.Context, act *model.Act) (int64, error) {
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
func (s *SQLiteStore) InsertStep(ctx context.Context, step *model.Step) (int64, error) {
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

// TableStats returns row counts for all tables.
func (s *SQLiteStore) TableStats(ctx context.Context) (map[string]int64, error) {
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
		query := `SELECT COUNT(*) ` + `FROM ` + table
		if err := s.db.QueryRowContext(ctx, query).Scan(&count); err != nil {
			return nil, fmt.Errorf("count %s: %w", table, err)
		}
		stats[table] = count
	}

	return stats, nil
}

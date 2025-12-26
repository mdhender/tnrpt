// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package store

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"time"

	"github.com/mdhender/tnrpt/model"
	"github.com/mdhender/tnrpt/web/auth"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// SQLiteStore is a SQLite-backed store for turn report data.
// It wraps an in-memory SQLite database with foreign key support.
type SQLiteStore struct {
	db *sql.DB
}

// StoreConfig holds configuration for creating a SQLiteStore.
type StoreConfig struct {
	// Path is the file path for file-based SQLite.
	// If empty, an in-memory database is used.
	Path string

	// InitSchema controls whether to run schema initialization.
	// For file-based mode, this should typically be false since the server
	// expects the database to already exist with schema applied.
	InitSchema bool
}

// NewSQLiteStore creates a new in-memory SQLite store with schema loaded.
func NewSQLiteStore() (*SQLiteStore, error) {
	return NewSQLiteStoreWithConfig(StoreConfig{InitSchema: true})
}

// NewSQLiteStoreWithConfig creates a SQLite store based on the provided configuration.
// For file-based mode (Path is set), the database file MUST already exist.
// Use InitDatabase to create and initialize a new database file.
func NewSQLiteStoreWithConfig(cfg StoreConfig) (*SQLiteStore, error) {
	var dsn string

	if cfg.Path == "" {
		// In-memory mode
		dsn = "file::memory:?cache=shared&_pragma=foreign_keys(1)"
	} else {
		// File-based mode: verify the database file exists before opening
		// (SQLite will create it automatically otherwise, which we don't want)
		if _, err := os.Stat(cfg.Path); os.IsNotExist(err) {
			return nil, fmt.Errorf("database file does not exist: %s (run init-db command to create it)", cfg.Path)
		}

		// Apply PRAGMA's per-connection via DSN so the pool always has them.
		// modernc.org/sqlite supports repeated _pragma=... parameters.
		dsn = fmt.Sprintf(
			"file:%s?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)",
			cfg.Path,
		)
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Initialize schema if requested (always true for in-memory, configurable for file-based)
	if cfg.InitSchema || cfg.Path == "" {
		if _, err := db.Exec(schemaSQL); err != nil {
			db.Close()
			return nil, fmt.Errorf("exec schema: %w", err)
		}
	}

	return &SQLiteStore{db: db}, nil
}

// InitDatabase creates a new SQLite database file and initializes the schema.
// This should be called by an init-db command before starting the server in file-based mode.
// Returns an error if the file already exists.
func InitDatabase(path string) error {
	// Check if file already exists
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("database file already exists: %s", path)
	}

	// Apply PRAGMA's per-connection via DSN so the pool always has them.
	dsn := fmt.Sprintf(
		"file:%s?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)",
		path,
	)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	// Run the embedded schema to create tables
	if _, err := db.Exec(schemaSQL); err != nil {
		return fmt.Errorf("exec schema: %w", err)
	}

	return nil
}

// CompactDatabase compacts a SQLite database file by running VACUUM and checkpointing WAL.
// This creates a single compact database file suitable for backup or export.
func CompactDatabase(path string) error {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("database file does not exist: %s", path)
	}

	dsn := fmt.Sprintf(
		"file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)",
		path,
	)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	// Checkpoint WAL to merge all changes into the main database file
	if _, err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		return fmt.Errorf("checkpoint WAL: %w", err)
	}

	// VACUUM rebuilds the database file, repacking it into minimal disk space
	if _, err := db.Exec("VACUUM"); err != nil {
		return fmt.Errorf("vacuum: %w", err)
	}

	return nil
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Game represents a game in the system.
type Game struct {
	ID          string
	Description string
	Turns       []GameTurn
}

// GameTurn represents a turn in a game.
type GameTurn struct {
	TurnNo   int
	IsActive bool
}

// GetAllGames returns all games with their turns.
func (s *SQLiteStore) GetAllGames(ctx context.Context) ([]Game, error) {
	const gameQuery = `SELECT id, COALESCE(description, id) FROM games ORDER BY id`
	rows, err := s.db.QueryContext(ctx, gameQuery)
	if err != nil {
		return nil, fmt.Errorf("query games: %w", err)
	}
	defer rows.Close()

	var games []Game
	for rows.Next() {
		var g Game
		if err := rows.Scan(&g.ID, &g.Description); err != nil {
			return nil, err
		}
		g.Turns = []GameTurn{}
		games = append(games, g)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Load turns for each game
	const turnQuery = `SELECT turn_id, is_active FROM game_turns WHERE game_id = ? ORDER BY turn_id`
	for i, game := range games {
		turnRows, err := s.db.QueryContext(ctx, turnQuery, game.ID)
		if err != nil {
			return nil, fmt.Errorf("query turns for game %s: %w", game.ID, err)
		}
		defer turnRows.Close()

		for turnRows.Next() {
			var turnNo int
			var isActive int
			if err := turnRows.Scan(&turnNo, &isActive); err != nil {
				return nil, err
			}
			games[i].Turns = append(games[i].Turns, GameTurn{
				TurnNo:   turnNo,
				IsActive: isActive == 1,
			})
		}
		if err := turnRows.Err(); err != nil {
			return nil, err
		}
	}

	return games, nil
}

// AddReportFile inserts a report_files row and sets rf.ID.
func (s *SQLiteStore) AddReportFile(rf *model.ReportFile) error {

	const query = `
		INSERT INTO report_files (game, clan_no, turn_no, name, sha256, mime, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	result, err := s.db.Exec(query,
		rf.Game,
		rf.ClanNo,
		rf.TurnNo,
		rf.Name,
		rf.SHA256,
		rf.Mime,
		rf.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("insert report_file: %w", err)
	}
	rf.ID, err = result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get report_file id: %w", err)
	}
	return nil
}

// AddReport adds a parsed report to the store.
func (s *SQLiteStore) AddReport(rx *model.ReportX) error {

	ctx := context.Background()

	// Insert report extract
	reportID, err := s.insertReportExtract(ctx, rx)
	if err != nil {
		return err
	}
	rx.ID = reportID

	// Insert units
	for _, ux := range rx.Units {
		ux.ReportXID = reportID
		unitID, err := s.insertUnitExtract(ctx, ux)
		if err != nil {
			return err
		}
		ux.ID = unitID

		// Insert acts
		for _, act := range ux.Acts {
			act.UnitXID = unitID
			actID, err := s.insertAct(ctx, act)
			if err != nil {
				return err
			}
			act.ID = actID

			// Insert steps
			for _, step := range act.Steps {
				step.ActID = actID
				stepID, err := s.insertStep(ctx, step)
				if err != nil {
					return err
				}
				step.ID = stepID
			}
		}
	}

	return nil
}

func (s *SQLiteStore) insertReportExtract(ctx context.Context, rx *model.ReportX) (int64, error) {
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

func (s *SQLiteStore) insertUnitExtract(ctx context.Context, ux *model.UnitX) (int64, error) {
	startGrid, startCol, startRow := parseTNCoord(ux.StartTN)
	endGrid, endCol, endRow := parseTNCoord(ux.EndTN)
	clanID := ux.ClanID
	if clanID == "" {
		clanID = extractClanID(ux.UnitID)
	}

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
		clanID,
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

func extractClanID(unitID string) string {
	if len(unitID) >= 4 {
		return unitID[1:4]
	}
	return ""
}

func (s *SQLiteStore) insertAct(ctx context.Context, act *model.Act) (int64, error) {
	destGrid, destCol, destRow := parseTNCoord(act.DestTN)

	const query = `
		INSERT INTO acts (
			unit_x_id, seq, kind, ok, note,
			target_unit_id, dest_grid, dest_col, dest_row,
			src_doc_id, src_turn_no, src_unit_id, src_act_seq, src_note
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	ok := sql.NullInt64{Int64: boolToInt(act.Ok), Valid: true}

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

func (s *SQLiteStore) insertStep(ctx context.Context, step *model.Step) (int64, error) {
	const query = `
		INSERT INTO steps (
			act_id, seq, kind, ok, note,
			dir, fail_why, terr, special, label,
			src_doc_id, src_turn_no, src_unit_id, src_act_seq, src_step_seq, src_note
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	ok := sql.NullInt64{Int64: boolToInt(step.Ok), Valid: true}

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

func (s *SQLiteStore) insertStepEncounters(ctx context.Context, stepID int64, enc *model.Enc) error {
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

func (s *SQLiteStore) insertStepBorder(ctx context.Context, stepID int64, border *model.BorderObs) error {
	const query = `INSERT INTO step_borders (step_id, dir, kind) VALUES (?, ?, ?)`
	if _, err := s.db.ExecContext(ctx, query, stepID, border.Dir, border.Kind); err != nil {
		return fmt.Errorf("insert step_border: %w", err)
	}
	return nil
}

// Query methods

// Units returns all units, optionally sorted.
func (s *SQLiteStore) Units(orderBy string) ([]*model.UnitX, error) {

	order := "unit_id, turn_no"
	switch orderBy {
	case "turn":
		order = "turn_no, unit_id"
	case "unit":
		order = "unit_id, turn_no"
	}

	query := fmt.Sprintf(`
		SELECT id, report_x_id, unit_id, turn_no,
		       start_grid, start_col, start_row,
		       end_grid, end_col, end_row
		FROM unit_extracts
		ORDER BY %s
	`, order)

	return s.queryUnits(query)
}

// UnitsByTurn returns units filtered by turn number.
func (s *SQLiteStore) UnitsByTurn(turnNo int) ([]*model.UnitX, error) {

	const query = `
		SELECT id, report_x_id, unit_id, turn_no,
		       start_grid, start_col, start_row,
		       end_grid, end_col, end_row
		FROM unit_extracts
		WHERE turn_no = ?
		ORDER BY unit_id
	`

	return s.queryUnitsWithArgs(query, turnNo)
}

// UnitsByClan returns units filtered by clan ID.
func (s *SQLiteStore) UnitsByClan(clanID string, turnNo int) ([]*model.UnitX, error) {

	if len(clanID) < 3 {
		return nil, nil
	}
	clanSuffix := clanID[len(clanID)-3:]

	if turnNo > 0 {
		const query = `
			SELECT id, report_x_id, unit_id, turn_no,
			       start_grid, start_col, start_row,
			       end_grid, end_col, end_row
			FROM unit_extracts
			WHERE clan_id = ? AND turn_no = ?
			ORDER BY unit_id, turn_no
		`
		return s.queryUnitsWithArgs(query, clanSuffix, turnNo)
	}

	const query = `
		SELECT id, report_x_id, unit_id, turn_no,
		       start_grid, start_col, start_row,
		       end_grid, end_col, end_row
		FROM unit_extracts
		WHERE clan_id = ?
		ORDER BY unit_id, turn_no
	`

	return s.queryUnitsWithArgs(query, clanSuffix)
}

// UnitsByGameClan returns units filtered by game and clan number.
func (s *SQLiteStore) UnitsByGameClan(gameID string, clanNo int, turnNo int) ([]*model.UnitX, error) {
	clanStr := formatClanNo(clanNo)

	if turnNo > 0 {
		const query = `
			SELECT u.id, u.report_x_id, u.unit_id, u.turn_no,
			       u.start_grid, u.start_col, u.start_row,
			       u.end_grid, u.end_col, u.end_row
			FROM unit_extracts u
			JOIN report_extracts r ON u.report_x_id = r.id
			WHERE r.game = ? AND u.clan_id = ? AND u.turn_no = ?
			ORDER BY u.unit_id, u.turn_no
		`
		return s.queryUnitsWithArgs(query, gameID, clanStr, turnNo)
	}

	const query = `
		SELECT u.id, u.report_x_id, u.unit_id, u.turn_no,
		       u.start_grid, u.start_col, u.start_row,
		       u.end_grid, u.end_col, u.end_row
		FROM unit_extracts u
		JOIN report_extracts r ON u.report_x_id = r.id
		WHERE r.game = ? AND u.clan_id = ?
		ORDER BY u.unit_id, u.turn_no
	`

	return s.queryUnitsWithArgs(query, gameID, clanStr)
}

// UnitByID returns a single unit by database ID.
func (s *SQLiteStore) UnitByID(id int64) (*model.UnitX, error) {
	const query = `
		SELECT id, report_x_id, unit_id, turn_no,
		       start_grid, start_col, start_row,
		       end_grid, end_col, end_row
		FROM unit_extracts
		WHERE id = ?
	`

	units, err := s.queryUnitsWithArgs(query, id)
	if err != nil {
		return nil, err
	}
	if len(units) == 0 {
		return nil, nil
	}
	return units[0], nil
}

// UnitByIDAndClan returns a single unit by database ID, verifying clan ownership.
func (s *SQLiteStore) UnitByIDAndClan(id int64, clanID string) (*model.UnitX, error) {
	if len(clanID) < 3 {
		return nil, nil
	}
	clanSuffix := clanID[len(clanID)-3:]

	const query = `
		SELECT id, report_x_id, unit_id, turn_no,
		       start_grid, start_col, start_row,
		       end_grid, end_col, end_row
		FROM unit_extracts
		WHERE id = ? AND clan_id = ?
	`

	units, err := s.queryUnitsWithArgs(query, id, clanSuffix)
	if err != nil {
		return nil, err
	}
	if len(units) == 0 {
		return nil, nil
	}
	return units[0], nil
}

// UnitByIDAndGameClan returns a single unit by database ID, verifying game and clan ownership.
func (s *SQLiteStore) UnitByIDAndGameClan(id int64, gameID string, clanNo int) (*model.UnitX, error) {
	clanStr := formatClanNo(clanNo)

	const query = `
		SELECT u.id, u.report_x_id, u.unit_id, u.turn_no,
		       u.start_grid, u.start_col, u.start_row,
		       u.end_grid, u.end_col, u.end_row
		FROM unit_extracts u
		JOIN report_extracts r ON u.report_x_id = r.id
		WHERE u.id = ? AND r.game = ? AND u.clan_id = ?
	`

	units, err := s.queryUnitsWithArgs(query, id, gameID, clanStr)
	if err != nil {
		return nil, err
	}
	if len(units) == 0 {
		return nil, nil
	}
	return units[0], nil
}

func (s *SQLiteStore) queryUnits(query string) ([]*model.UnitX, error) {
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query units: %w", err)
	}
	defer rows.Close()

	return s.scanUnits(rows)
}

func (s *SQLiteStore) queryUnitsWithArgs(query string, args ...any) ([]*model.UnitX, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query units: %w", err)
	}
	defer rows.Close()

	return s.scanUnits(rows)
}

func (s *SQLiteStore) scanUnits(rows *sql.Rows) ([]*model.UnitX, error) {
	var units []*model.UnitX
	for rows.Next() {
		var u model.UnitX
		var startGrid, endGrid string
		var startCol, startRow, endCol, endRow int

		if err := rows.Scan(
			&u.ID, &u.ReportXID, &u.UnitID, &u.TurnNo,
			&startGrid, &startCol, &startRow,
			&endGrid, &endCol, &endRow,
		); err != nil {
			return nil, fmt.Errorf("scan unit: %w", err)
		}

		u.StartTN = formatTNCoord(startGrid, startCol, startRow)
		u.EndTN = formatTNCoord(endGrid, endCol, endRow)

		units = append(units, &u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Load acts for each unit after closing the rows cursor
	// (SQLite doesn't allow nested queries on the same connection)
	for _, u := range units {
		acts, err := s.loadActsForUnit(u.ID)
		if err != nil {
			return nil, err
		}
		u.Acts = acts
	}

	return units, nil
}

func (s *SQLiteStore) loadActsForUnit(unitID int64) ([]*model.Act, error) {
	const query = `
		SELECT id, unit_x_id, seq, kind, ok, note, target_unit_id, dest_grid, dest_col, dest_row
		FROM acts
		WHERE unit_x_id = ?
		ORDER BY seq
	`

	rows, err := s.db.Query(query, unitID)
	if err != nil {
		return nil, fmt.Errorf("query acts: %w", err)
	}
	defer rows.Close()

	var acts []*model.Act
	for rows.Next() {
		var a model.Act
		var ok sql.NullInt64
		var note, targetUnitID, destGrid sql.NullString
		var destCol, destRow sql.NullInt64

		if err := rows.Scan(
			&a.ID, &a.UnitXID, &a.Seq, &a.Kind, &ok, &note,
			&targetUnitID, &destGrid, &destCol, &destRow,
		); err != nil {
			return nil, fmt.Errorf("scan act: %w", err)
		}

		a.Ok = ok.Valid && ok.Int64 == 1
		a.Note = note.String
		a.TargetUnitID = targetUnitID.String
		if destGrid.Valid {
			a.DestTN = formatTNCoord(destGrid.String, int(destCol.Int64), int(destRow.Int64))
		}

		acts = append(acts, &a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Load steps for each act after closing the rows cursor
	for _, a := range acts {
		steps, err := s.loadStepsForAct(a.ID)
		if err != nil {
			return nil, err
		}
		a.Steps = steps
	}

	return acts, nil
}

func (s *SQLiteStore) loadStepsForAct(actID int64) ([]*model.Step, error) {
	const query = `
		SELECT id, act_id, seq, kind, ok, note, dir, fail_why, terr, special, label
		FROM steps
		WHERE act_id = ?
		ORDER BY seq
	`

	rows, err := s.db.Query(query, actID)
	if err != nil {
		return nil, fmt.Errorf("query steps: %w", err)
	}
	defer rows.Close()

	var steps []*model.Step
	for rows.Next() {
		var st model.Step
		var ok sql.NullInt64
		var note, dir, failWhy, terr, label sql.NullString
		var special int

		if err := rows.Scan(
			&st.ID, &st.ActID, &st.Seq, &st.Kind, &ok, &note,
			&dir, &failWhy, &terr, &special, &label,
		); err != nil {
			return nil, fmt.Errorf("scan step: %w", err)
		}

		st.Ok = ok.Valid && ok.Int64 == 1
		st.Note = note.String
		st.Dir = dir.String
		st.FailWhy = failWhy.String
		st.Terr = terr.String
		st.Special = special == 1
		st.Label = label.String

		steps = append(steps, &st)
	}
	return steps, rows.Err()
}

// Movements returns all movement steps (adv steps with direction).
type Movement struct {
	UnitID  string
	TurnNo  int
	ActSeq  int
	StepSeq int
	Dir     string
	Ok      bool
	FailWhy string
	Terr    string
}

func (s *SQLiteStore) Movements() ([]Movement, error) {

	const query = `
		SELECT u.unit_id, u.turn_no, a.seq, st.seq, st.dir, st.ok, st.fail_why, st.terr
		FROM steps st
		JOIN acts a ON st.act_id = a.id
		JOIN unit_extracts u ON a.unit_x_id = u.id
		WHERE st.kind = 'adv' AND st.dir IS NOT NULL AND st.dir != ''
		ORDER BY u.turn_no, u.unit_id, a.seq, st.seq
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query movements: %w", err)
	}
	defer rows.Close()

	var movements []Movement
	for rows.Next() {
		var m Movement
		var ok sql.NullInt64
		var failWhy, terr sql.NullString

		if err := rows.Scan(&m.UnitID, &m.TurnNo, &m.ActSeq, &m.StepSeq, &m.Dir, &ok, &failWhy, &terr); err != nil {
			return nil, fmt.Errorf("scan movement: %w", err)
		}

		m.Ok = ok.Valid && ok.Int64 == 1
		m.FailWhy = failWhy.String
		m.Terr = terr.String
		movements = append(movements, m)
	}
	return movements, rows.Err()
}

// MovementsByClan returns movement steps filtered by clan ID.
func (s *SQLiteStore) MovementsByClan(clanID string, turnNo int) ([]Movement, error) {
	if len(clanID) < 3 {
		return nil, nil
	}
	clanSuffix := clanID[len(clanID)-3:]

	var rows *sql.Rows
	var err error

	if turnNo > 0 {
		const query = `
			SELECT u.unit_id, u.turn_no, a.seq, st.seq, st.dir, st.ok, st.fail_why, st.terr
			FROM steps st
			JOIN acts a ON st.act_id = a.id
			JOIN unit_extracts u ON a.unit_x_id = u.id
			WHERE st.kind = 'adv' AND st.dir IS NOT NULL AND st.dir != ''
			  AND u.clan_id = ? AND u.turn_no = ?
			ORDER BY u.turn_no, u.unit_id, a.seq, st.seq
		`
		rows, err = s.db.Query(query, clanSuffix, turnNo)
	} else {
		const query = `
			SELECT u.unit_id, u.turn_no, a.seq, st.seq, st.dir, st.ok, st.fail_why, st.terr
			FROM steps st
			JOIN acts a ON st.act_id = a.id
			JOIN unit_extracts u ON a.unit_x_id = u.id
			WHERE st.kind = 'adv' AND st.dir IS NOT NULL AND st.dir != ''
			  AND u.clan_id = ?
			ORDER BY u.turn_no, u.unit_id, a.seq, st.seq
		`
		rows, err = s.db.Query(query, clanSuffix)
	}
	if err != nil {
		return nil, fmt.Errorf("query movements: %w", err)
	}
	defer rows.Close()

	var movements []Movement
	for rows.Next() {
		var m Movement
		var ok sql.NullInt64
		var failWhy, terr sql.NullString

		if err := rows.Scan(&m.UnitID, &m.TurnNo, &m.ActSeq, &m.StepSeq, &m.Dir, &ok, &failWhy, &terr); err != nil {
			return nil, fmt.Errorf("scan movement: %w", err)
		}

		m.Ok = ok.Valid && ok.Int64 == 1
		m.FailWhy = failWhy.String
		m.Terr = terr.String
		movements = append(movements, m)
	}
	return movements, rows.Err()
}

// MovementsByGameClan returns movement steps filtered by game and clan number.
func (s *SQLiteStore) MovementsByGameClan(gameID string, clanNo int, turnNo int) ([]Movement, error) {
	clanStr := formatClanNo(clanNo)

	var rows *sql.Rows
	var err error

	if turnNo > 0 {
		const query = `
			SELECT u.unit_id, u.turn_no, a.seq, st.seq, st.dir, st.ok, st.fail_why, st.terr
			FROM steps st
			JOIN acts a ON st.act_id = a.id
			JOIN unit_extracts u ON a.unit_x_id = u.id
			JOIN report_extracts r ON u.report_x_id = r.id
			WHERE st.kind = 'adv' AND st.dir IS NOT NULL AND st.dir != ''
			  AND r.game = ? AND u.clan_id = ? AND u.turn_no = ?
			ORDER BY u.turn_no, u.unit_id, a.seq, st.seq
		`
		rows, err = s.db.Query(query, gameID, clanStr, turnNo)
	} else {
		const query = `
			SELECT u.unit_id, u.turn_no, a.seq, st.seq, st.dir, st.ok, st.fail_why, st.terr
			FROM steps st
			JOIN acts a ON st.act_id = a.id
			JOIN unit_extracts u ON a.unit_x_id = u.id
			JOIN report_extracts r ON u.report_x_id = r.id
			WHERE st.kind = 'adv' AND st.dir IS NOT NULL AND st.dir != ''
			  AND r.game = ? AND u.clan_id = ?
			ORDER BY u.turn_no, u.unit_id, a.seq, st.seq
		`
		rows, err = s.db.Query(query, gameID, clanStr)
	}
	if err != nil {
		return nil, fmt.Errorf("query movements: %w", err)
	}
	defer rows.Close()

	var movements []Movement
	for rows.Next() {
		var m Movement
		var ok sql.NullInt64
		var failWhy, terr sql.NullString

		if err := rows.Scan(&m.UnitID, &m.TurnNo, &m.ActSeq, &m.StepSeq, &m.Dir, &ok, &failWhy, &terr); err != nil {
			return nil, fmt.Errorf("scan movement: %w", err)
		}

		m.Ok = ok.Valid && ok.Int64 == 1
		m.FailWhy = failWhy.String
		m.Terr = terr.String
		movements = append(movements, m)
	}
	return movements, rows.Err()
}

// Resource represents a resource sighting.
type Resource struct {
	UnitID  string
	TurnNo  int
	Kind    string
	Qty     int
	Terrain string
}

func (s *SQLiteStore) Resources() ([]Resource, error) {

	const query = `
		SELECT u.unit_id, u.turn_no, r.kind, r.qty, st.terr
		FROM step_enc_rsrc r
		JOIN steps st ON r.step_id = st.id
		JOIN acts a ON st.act_id = a.id
		JOIN unit_extracts u ON a.unit_x_id = u.id
		ORDER BY r.kind, u.turn_no, u.unit_id
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query resources: %w", err)
	}
	defer rows.Close()

	var resources []Resource
	for rows.Next() {
		var r Resource
		var qty sql.NullInt64
		var terr sql.NullString

		if err := rows.Scan(&r.UnitID, &r.TurnNo, &r.Kind, &qty, &terr); err != nil {
			return nil, fmt.Errorf("scan resource: %w", err)
		}

		r.Qty = int(qty.Int64)
		r.Terrain = terr.String
		resources = append(resources, r)
	}
	return resources, rows.Err()
}

// ResourcesByClan returns resources filtered by clan ID.
func (s *SQLiteStore) ResourcesByClan(clanID string, turnNo int) ([]Resource, error) {
	if len(clanID) < 3 {
		return nil, nil
	}
	clanSuffix := clanID[len(clanID)-3:]

	var rows *sql.Rows
	var err error

	if turnNo > 0 {
		const query = `
			SELECT u.unit_id, u.turn_no, r.kind, r.qty, st.terr
			FROM step_enc_rsrc r
			JOIN steps st ON r.step_id = st.id
			JOIN acts a ON st.act_id = a.id
			JOIN unit_extracts u ON a.unit_x_id = u.id
			WHERE u.clan_id = ? AND u.turn_no = ?
			ORDER BY r.kind, u.turn_no, u.unit_id
		`
		rows, err = s.db.Query(query, clanSuffix, turnNo)
	} else {
		const query = `
			SELECT u.unit_id, u.turn_no, r.kind, r.qty, st.terr
			FROM step_enc_rsrc r
			JOIN steps st ON r.step_id = st.id
			JOIN acts a ON st.act_id = a.id
			JOIN unit_extracts u ON a.unit_x_id = u.id
			WHERE u.clan_id = ?
			ORDER BY r.kind, u.turn_no, u.unit_id
		`
		rows, err = s.db.Query(query, clanSuffix)
	}
	if err != nil {
		return nil, fmt.Errorf("query resources: %w", err)
	}
	defer rows.Close()

	var resources []Resource
	for rows.Next() {
		var r Resource
		var qty sql.NullInt64
		var terr sql.NullString

		if err := rows.Scan(&r.UnitID, &r.TurnNo, &r.Kind, &qty, &terr); err != nil {
			return nil, fmt.Errorf("scan resource: %w", err)
		}

		r.Qty = int(qty.Int64)
		r.Terrain = terr.String
		resources = append(resources, r)
	}
	return resources, rows.Err()
}

// ResourcesByGameClan returns resources filtered by game and clan number.
func (s *SQLiteStore) ResourcesByGameClan(gameID string, clanNo int, turnNo int) ([]Resource, error) {
	clanStr := formatClanNo(clanNo)

	var rows *sql.Rows
	var err error

	if turnNo > 0 {
		const query = `
			SELECT u.unit_id, u.turn_no, r.kind, r.qty, st.terr
			FROM step_enc_rsrc r
			JOIN steps st ON r.step_id = st.id
			JOIN acts a ON st.act_id = a.id
			JOIN unit_extracts u ON a.unit_x_id = u.id
			JOIN report_extracts re ON u.report_x_id = re.id
			WHERE re.game = ? AND u.clan_id = ? AND u.turn_no = ?
			ORDER BY r.kind, u.turn_no, u.unit_id
		`
		rows, err = s.db.Query(query, gameID, clanStr, turnNo)
	} else {
		const query = `
			SELECT u.unit_id, u.turn_no, r.kind, r.qty, st.terr
			FROM step_enc_rsrc r
			JOIN steps st ON r.step_id = st.id
			JOIN acts a ON st.act_id = a.id
			JOIN unit_extracts u ON a.unit_x_id = u.id
			JOIN report_extracts re ON u.report_x_id = re.id
			WHERE re.game = ? AND u.clan_id = ?
			ORDER BY r.kind, u.turn_no, u.unit_id
		`
		rows, err = s.db.Query(query, gameID, clanStr)
	}
	if err != nil {
		return nil, fmt.Errorf("query resources: %w", err)
	}
	defer rows.Close()

	var resources []Resource
	for rows.Next() {
		var r Resource
		var qty sql.NullInt64
		var terr sql.NullString

		if err := rows.Scan(&r.UnitID, &r.TurnNo, &r.Kind, &qty, &terr); err != nil {
			return nil, fmt.Errorf("scan resource: %w", err)
		}

		r.Qty = int(qty.Int64)
		r.Terrain = terr.String
		resources = append(resources, r)
	}
	return resources, rows.Err()
}

// TerrainObs represents an observed terrain.
type TerrainObs struct {
	UnitID  string
	TurnNo  int
	Terrain string
	Special bool
	Label   string
}

func (s *SQLiteStore) TerrainObservations() ([]TerrainObs, error) {

	const query = `
		SELECT u.unit_id, u.turn_no, st.terr, st.special, st.label
		FROM steps st
		JOIN acts a ON st.act_id = a.id
		JOIN unit_extracts u ON a.unit_x_id = u.id
		WHERE st.terr IS NOT NULL AND st.terr != ''
		ORDER BY st.terr, u.turn_no, u.unit_id
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query terrain: %w", err)
	}
	defer rows.Close()

	var obs []TerrainObs
	for rows.Next() {
		var t TerrainObs
		var special int
		var label sql.NullString

		if err := rows.Scan(&t.UnitID, &t.TurnNo, &t.Terrain, &special, &label); err != nil {
			return nil, fmt.Errorf("scan terrain: %w", err)
		}

		t.Special = special == 1
		t.Label = label.String
		obs = append(obs, t)
	}
	return obs, rows.Err()
}

// TerrainObservationsByClan returns terrain observations filtered by clan ID.
func (s *SQLiteStore) TerrainObservationsByClan(clanID string, turnNo int) ([]TerrainObs, error) {
	if len(clanID) < 3 {
		return nil, nil
	}
	clanSuffix := clanID[len(clanID)-3:]

	var rows *sql.Rows
	var err error

	if turnNo > 0 {
		const query = `
			SELECT u.unit_id, u.turn_no, st.terr, st.special, st.label
			FROM steps st
			JOIN acts a ON st.act_id = a.id
			JOIN unit_extracts u ON a.unit_x_id = u.id
			WHERE st.terr IS NOT NULL AND st.terr != ''
			  AND u.clan_id = ? AND u.turn_no = ?
			ORDER BY st.terr, u.turn_no, u.unit_id
		`
		rows, err = s.db.Query(query, clanSuffix, turnNo)
	} else {
		const query = `
			SELECT u.unit_id, u.turn_no, st.terr, st.special, st.label
			FROM steps st
			JOIN acts a ON st.act_id = a.id
			JOIN unit_extracts u ON a.unit_x_id = u.id
			WHERE st.terr IS NOT NULL AND st.terr != ''
			  AND u.clan_id = ?
			ORDER BY st.terr, u.turn_no, u.unit_id
		`
		rows, err = s.db.Query(query, clanSuffix)
	}
	if err != nil {
		return nil, fmt.Errorf("query terrain: %w", err)
	}
	defer rows.Close()

	var obs []TerrainObs
	for rows.Next() {
		var t TerrainObs
		var special int
		var label sql.NullString

		if err := rows.Scan(&t.UnitID, &t.TurnNo, &t.Terrain, &special, &label); err != nil {
			return nil, fmt.Errorf("scan terrain: %w", err)
		}

		t.Special = special == 1
		t.Label = label.String
		obs = append(obs, t)
	}
	return obs, rows.Err()
}

// TerrainObservationsByGameClan returns terrain observations filtered by game and clan number.
func (s *SQLiteStore) TerrainObservationsByGameClan(gameID string, clanNo int, turnNo int) ([]TerrainObs, error) {
	clanStr := formatClanNo(clanNo)

	var rows *sql.Rows
	var err error

	if turnNo > 0 {
		const query = `
			SELECT u.unit_id, u.turn_no, st.terr, st.special, st.label
			FROM steps st
			JOIN acts a ON st.act_id = a.id
			JOIN unit_extracts u ON a.unit_x_id = u.id
			JOIN report_extracts r ON u.report_x_id = r.id
			WHERE st.terr IS NOT NULL AND st.terr != ''
			  AND r.game = ? AND u.clan_id = ? AND u.turn_no = ?
			ORDER BY st.terr, u.turn_no, u.unit_id
		`
		rows, err = s.db.Query(query, gameID, clanStr, turnNo)
	} else {
		const query = `
			SELECT u.unit_id, u.turn_no, st.terr, st.special, st.label
			FROM steps st
			JOIN acts a ON st.act_id = a.id
			JOIN unit_extracts u ON a.unit_x_id = u.id
			JOIN report_extracts r ON u.report_x_id = r.id
			WHERE st.terr IS NOT NULL AND st.terr != ''
			  AND r.game = ? AND u.clan_id = ?
			ORDER BY st.terr, u.turn_no, u.unit_id
		`
		rows, err = s.db.Query(query, gameID, clanStr)
	}
	if err != nil {
		return nil, fmt.Errorf("query terrain: %w", err)
	}
	defer rows.Close()

	var obs []TerrainObs
	for rows.Next() {
		var t TerrainObs
		var special int
		var label sql.NullString

		if err := rows.Scan(&t.UnitID, &t.TurnNo, &t.Terrain, &special, &label); err != nil {
			return nil, fmt.Errorf("scan terrain: %w", err)
		}

		t.Special = special == 1
		t.Label = label.String
		obs = append(obs, t)
	}
	return obs, rows.Err()
}

// TileDetail represents detailed tile information for a specific location.
type TileDetail struct {
	Grid      string
	Col       int
	Row       int
	Coord     string
	Sightings []TileSighting
}

// TileSighting is a single observation of a tile.
type TileSighting struct {
	UnitID  string
	TurnNo  int
	Terrain string
	Special bool
	Label   string
}

// TileDetailByCoord returns detailed tile information for a grid location.
func (s *SQLiteStore) TileDetailByCoord(grid string, col, row int, clanID string) (*TileDetail, error) {
	if len(clanID) < 3 {
		return nil, nil
	}
	clanSuffix := clanID[len(clanID)-3:]

	const query = `
		SELECT u.unit_id, u.turn_no, st.terr, st.special, st.label
		FROM steps st
		JOIN acts a ON st.act_id = a.id
		JOIN unit_extracts u ON a.unit_x_id = u.id
		WHERE st.terr IS NOT NULL AND st.terr != ''
		  AND u.clan_id = ?
		  AND (
		      (u.end_grid = ? AND u.end_col = ? AND u.end_row = ?)
		      OR (u.start_grid = ? AND u.start_col = ? AND u.start_row = ?)
		  )
		ORDER BY u.turn_no, u.unit_id
	`

	rows, err := s.db.Query(query, clanSuffix, grid, col, row, grid, col, row)
	if err != nil {
		return nil, fmt.Errorf("query tile detail: %w", err)
	}
	defer rows.Close()

	detail := &TileDetail{
		Grid:  grid,
		Col:   col,
		Row:   row,
		Coord: fmt.Sprintf("%s %02d%02d", grid, col, row),
	}

	for rows.Next() {
		var s TileSighting
		var special int
		var label sql.NullString

		if err := rows.Scan(&s.UnitID, &s.TurnNo, &s.Terrain, &special, &label); err != nil {
			return nil, fmt.Errorf("scan tile sighting: %w", err)
		}

		s.Special = special == 1
		s.Label = label.String
		detail.Sightings = append(detail.Sightings, s)
	}
	return detail, rows.Err()
}

// TileDetailByGameClanCoord returns detailed tile information for a grid location, filtered by game and clan.
func (s *SQLiteStore) TileDetailByGameClanCoord(grid string, col, row int, gameID string, clanNo int) (*TileDetail, error) {
	clanStr := formatClanNo(clanNo)

	const query = `
		SELECT u.unit_id, u.turn_no, st.terr, st.special, st.label
		FROM steps st
		JOIN acts a ON st.act_id = a.id
		JOIN unit_extracts u ON a.unit_x_id = u.id
		JOIN report_extracts r ON u.report_x_id = r.id
		WHERE st.terr IS NOT NULL AND st.terr != ''
		  AND r.game = ? AND u.clan_id = ?
		  AND (
		      (u.end_grid = ? AND u.end_col = ? AND u.end_row = ?)
		      OR (u.start_grid = ? AND u.start_col = ? AND u.start_row = ?)
		  )
		ORDER BY u.turn_no, u.unit_id
	`

	rows, err := s.db.Query(query, gameID, clanStr, grid, col, row, grid, col, row)
	if err != nil {
		return nil, fmt.Errorf("query tile detail: %w", err)
	}
	defer rows.Close()

	detail := &TileDetail{
		Grid:  grid,
		Col:   col,
		Row:   row,
		Coord: fmt.Sprintf("%s %02d%02d", grid, col, row),
	}

	for rows.Next() {
		var sg TileSighting
		var special int
		var label sql.NullString

		if err := rows.Scan(&sg.UnitID, &sg.TurnNo, &sg.Terrain, &special, &label); err != nil {
			return nil, fmt.Errorf("scan tile sighting: %w", err)
		}

		sg.Special = special == 1
		sg.Label = label.String
		detail.Sightings = append(detail.Sightings, sg)
	}
	return detail, rows.Err()
}

// Stats returns basic statistics about the store.
func (s *SQLiteStore) Stats() model.Stats {

	var stats model.Stats

	s.db.QueryRow("SELECT COUNT(*) FROM report_extracts").Scan(&stats.Reports)
	s.db.QueryRow("SELECT COUNT(*) FROM unit_extracts").Scan(&stats.Units)
	s.db.QueryRow("SELECT COUNT(*) FROM acts").Scan(&stats.Acts)
	s.db.QueryRow("SELECT COUNT(*) FROM steps").Scan(&stats.Steps)

	return stats
}

// Turns returns distinct turn numbers in the store.
func (s *SQLiteStore) Turns() ([]int, error) {

	const query = `SELECT DISTINCT turn_no FROM unit_extracts ORDER BY turn_no`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query turns: %w", err)
	}
	defer rows.Close()

	var turns []int
	for rows.Next() {
		var t int
		if err := rows.Scan(&t); err != nil {
			return nil, fmt.Errorf("scan turn: %w", err)
		}
		turns = append(turns, t)
	}
	return turns, rows.Err()
}

// TurnsByClan returns distinct turn numbers filtered by clan ID.
func (s *SQLiteStore) TurnsByClan(clanID string) ([]int, error) {
	if len(clanID) < 3 {
		return nil, nil
	}
	clanSuffix := clanID[len(clanID)-3:]

	const query = `SELECT DISTINCT turn_no FROM unit_extracts WHERE clan_id = ? ORDER BY turn_no`

	rows, err := s.db.Query(query, clanSuffix)
	if err != nil {
		return nil, fmt.Errorf("query turns: %w", err)
	}
	defer rows.Close()

	var turns []int
	for rows.Next() {
		var t int
		if err := rows.Scan(&t); err != nil {
			return nil, fmt.Errorf("scan turn: %w", err)
		}
		turns = append(turns, t)
	}
	return turns, rows.Err()
}

// Helper functions

func formatClanNo(clanNo int) string {
	if clanNo < 100 {
		return fmt.Sprintf("%03d", clanNo)
	}
	return fmt.Sprintf("%d", clanNo)
}

func parseTNCoord(tn model.TNCoord) (grid string, col, row int) {
	str := string(tn)
	if str == "" || str == "N/A" {
		return "", 0, 0
	}
	if len(str) >= 7 && str[2] == ' ' {
		grid = str[:2]
		if len(str) >= 7 {
			fmt.Sscanf(str[3:], "%02d%02d", &col, &row)
		}
	}
	return grid, col, row
}

func formatTNCoord(grid string, col, row int) model.TNCoord {
	if grid == "" {
		return ""
	}
	return model.TNCoord(fmt.Sprintf("%s %02d%02d", grid, col, row))
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

// User authentication methods

// ValidateCredentials checks username/password and returns an auth.User if valid.
func (s *SQLiteStore) ValidateCredentials(ctx context.Context, handle, password, gameID string) (*auth.User, error) {
	const userQuery = `SELECT handle, user_name, password_hash FROM users WHERE handle = ?`

	var dbHandle, userName, passwordHash string
	err := s.db.QueryRowContext(ctx, userQuery, handle).Scan(&dbHandle, &userName, &passwordHash)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}

	// Check if user is active
	active, err := s.isUserActive(ctx, handle)
	if err != nil {
		return nil, err
	}
	if !active {
		return nil, nil
	}

	// Verify password
	if !auth.CheckPassword(password, passwordHash) {
		return nil, nil
	}

	// Get clan for this game
	clanNo, err := s.getClanForUser(ctx, gameID, handle)
	if err != nil {
		return nil, err
	}

	return &auth.User{
		Handle:   dbHandle,
		UserName: userName,
		GameID:   gameID,
		ClanNo:   clanNo,
	}, nil
}

func (s *SQLiteStore) isUserActive(ctx context.Context, handle string) (bool, error) {
	const query = `SELECT role FROM user_roles WHERE user_handle = ?`
	rows, err := s.db.QueryContext(ctx, query, handle)
	if err != nil {
		return false, fmt.Errorf("query roles: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return false, err
		}
		if role == "active" {
			return true, nil
		}
	}
	return false, rows.Err()
}

// IsUserGM checks if a user has the "gm" role.
func (s *SQLiteStore) IsUserGM(ctx context.Context, handle string) (bool, error) {
	const query = `SELECT 1 FROM user_roles WHERE user_handle = ? AND role = 'gm' LIMIT 1`
	var exists int
	err := s.db.QueryRowContext(ctx, query, handle).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check gm role: %w", err)
	}
	return true, nil
}

func (s *SQLiteStore) getClanForUser(ctx context.Context, gameID, handle string) (int, error) {
	const query = `SELECT clan_no FROM game_clans WHERE game_id = ? AND user_handle = ?`
	var clanNo int
	err := s.db.QueryRowContext(ctx, query, gameID, handle).Scan(&clanNo)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("query clan: %w", err)
	}
	return clanNo, nil
}

// GetClanForUser returns the clan number for a user in a specific game (exported version).
func (s *SQLiteStore) GetClanForUser(ctx context.Context, gameID, handle string) (int, error) {
	return s.getClanForUser(ctx, gameID, handle)
}

// GetHandleForClan returns the user handle for a clan in a specific game.
func (s *SQLiteStore) GetHandleForClan(ctx context.Context, gameID string, clanNo int) (string, error) {
	const query = `SELECT user_handle FROM game_clans WHERE game_id = ? AND clan_no = ?`
	var handle string
	err := s.db.QueryRowContext(ctx, query, gameID, clanNo).Scan(&handle)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("query handle: %w", err)
	}
	return handle, nil
}

// TurnsByGameClan returns distinct turn numbers filtered by game and clan.
func (s *SQLiteStore) TurnsByGameClan(gameID string, clanNo int) ([]int, error) {
	clanStr := fmt.Sprintf("%d", clanNo)
	if clanNo < 100 {
		clanStr = fmt.Sprintf("%03d", clanNo)
	}

	const query = `
		SELECT DISTINCT u.turn_no 
		FROM unit_extracts u
		JOIN report_extracts r ON u.report_x_id = r.id
		WHERE r.game = ? AND u.clan_id = ?
		ORDER BY u.turn_no
	`

	rows, err := s.db.Query(query, gameID, clanStr)
	if err != nil {
		return nil, fmt.Errorf("query turns: %w", err)
	}
	defer rows.Close()

	var turns []int
	for rows.Next() {
		var t int
		if err := rows.Scan(&t); err != nil {
			return nil, fmt.Errorf("scan turn: %w", err)
		}
		turns = append(turns, t)
	}
	return turns, rows.Err()
}

// LoadUsersFromJSON loads users from a JSON file.
func (s *SQLiteStore) LoadUsersFromJSON(ctx context.Context, path string) error {
	return loadUsersFromJSON(ctx, s.db, path)
}

// LoadGamesFromJSON loads games from a JSON file.
func (s *SQLiteStore) LoadGamesFromJSON(ctx context.Context, path string) error {
	return loadGamesFromJSON(ctx, s.db, path)
}

// UserGame represents a game the user belongs to with their clan number.
type UserGame struct {
	GameID      string
	Description string
	ClanNo      int
}

// QueryResult holds the result of a raw SQL query.
type QueryResult struct {
	Columns []string
	Rows    [][]string
	Error   string
}

// ExecRawQuery executes a raw SQL query and returns results as strings.
// This is intended for admin/debugging use only.
func (s *SQLiteStore) ExecRawQuery(ctx context.Context, query string) *QueryResult {
	result := &QueryResult{}

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.Columns = cols

	for rows.Next() {
		values := make([]any, len(cols))
		valuePtrs := make([]any, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			result.Error = err.Error()
			return result
		}

		row := make([]string, len(cols))
		for i, v := range values {
			if v == nil {
				row[i] = "NULL"
			} else {
				row[i] = fmt.Sprintf("%v", v)
			}
		}
		result.Rows = append(result.Rows, row)
	}

	if err := rows.Err(); err != nil {
		result.Error = err.Error()
	}

	return result
}

// GetGamesForUser returns all games a user belongs to, sorted by game ID.
func (s *SQLiteStore) GetGamesForUser(ctx context.Context, handle string) ([]UserGame, error) {
	const query = `
		SELECT g.id, g.description, gc.clan_no
		FROM games g
		JOIN game_clans gc ON g.id = gc.game_id
		WHERE gc.user_handle = ?
		ORDER BY g.id
	`

	rows, err := s.db.QueryContext(ctx, query, handle)
	if err != nil {
		return nil, fmt.Errorf("query games: %w", err)
	}
	defer rows.Close()

	var games []UserGame
	for rows.Next() {
		var g UserGame
		var desc sql.NullString
		if err := rows.Scan(&g.GameID, &desc, &g.ClanNo); err != nil {
			return nil, fmt.Errorf("scan game: %w", err)
		}
		g.Description = desc.String
		games = append(games, g)
	}
	return games, rows.Err()
}

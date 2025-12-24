// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package store

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mdhender/tnrpt/adapters"
	"github.com/mdhender/tnrpt/model"
	"github.com/mdhender/tnrpt/pipelines/parsers/bistre"
	"github.com/mdhender/tnrpt/pipelines/parsers/docx"
	"github.com/mdhender/tnrpt/pipelines/parsers/report"
	"github.com/mdhender/tnrpt/web/auth"
)

// Store is an interface for loading data.
type Store interface {
	AddReportFile(rf *model.ReportFile) error
	AddReport(rx *model.ReportX) error
}

var (
	reDocxReportFileName = regexp.MustCompile(`^\d{4}.\d{4}-\d{2}.0\d{3}.docx$`)
	reTextReportFileName = regexp.MustCompile(`^\d{4}.\d{4}-\d{2}.0\d{3}.report.txt$`)
)

// LoadDocxFromDir loads all .docx files from a directory into the store.
// File names are expected to follow the pattern: GGGG.YYYY-MM.CCCC.docx
// where GGGG is game, YYYY-MM is turn, CCCC is clan.
func LoadDocxFromDir(s Store, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read dir: %w", err)
	}

	var loaded, failed int
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !reDocxReportFileName.MatchString(strings.ToLower(name)) {
			continue
		} else if !strings.HasSuffix(strings.ToLower(name), ".docx") {
			continue
		}

		path := filepath.Join(dir, name)
		if err := LoadDocxFile(s, path); err != nil {
			log.Printf("store: load %s: %v", name, err)
			failed++
			continue
		}
		loaded++
	}

	log.Printf("store: loaded %d docx files (%d failed) from %s", loaded, failed, dir)
	return nil
}

// LoadDocxFile loads a single .docx file into the store.
func LoadDocxFile(s Store, path string) error {
	name := filepath.Base(path)
	if !reDocxReportFileName.MatchString(strings.ToLower(name)) {
		return fmt.Errorf("invalid report file name")
	}
	game, clanNo := parseFilename(name)

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}
	hash := sha256.Sum256(data)

	doc, err := docx.ParsePath(path, true, true, true, false, false)
	if err != nil {
		return fmt.Errorf("parse docx: %w", err)
	}

	rpt, err := report.ParseReportText(doc, true, true, true, false, false)
	if err != nil {
		return fmt.Errorf("parse report: %w", err)
	}

	var text []byte
	for _, section := range rpt.Sections {
		text = append(text, bytes.Join(section.Lines, []byte{'\n'})...)
		text = append(text, '\n')
	}

	turn, err := bistre.ParseInput(rpt.Name, rpt.TurnNo, text, false, false, false, false, false, false, false, false, bistre.ParseConfig{})
	if err != nil {
		return fmt.Errorf("parse input: %w", err)
	}
	if turn == nil {
		return fmt.Errorf("parser returned nil")
	}

	turnNo := 100*turn.Year + turn.Month
	rf := &model.ReportFile{
		Game:      game,
		ClanNo:    clanNo,
		TurnNo:    turnNo,
		Name:      name,
		SHA256:    hex.EncodeToString(hash[:]),
		Mime:      "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		CreatedAt: time.Now().UTC(),
	}
	if err := s.AddReportFile(rf); err != nil {
		return fmt.Errorf("add report file: %w", err)
	}

	rx, err := adapters.BistreTurnToModelReportX(name, turn, game, clanNo)
	if err != nil {
		return fmt.Errorf("adapt to model: %w", err)
	}
	rx.ReportFileID = rf.ID

	err = s.AddReport(rx)
	if err != nil {
		return err
	}
	return nil
}

var filenameRe = regexp.MustCompile(`^(\d{4})\.(\d{3,4}-\d{2})\.(\d{4})\.docx$`)

func parseFilename(name string) (game, clan string) {
	matches := filenameRe.FindStringSubmatch(name)
	if len(matches) == 4 {
		return matches[1], matches[3]
	}
	return "", ""
}

// JSON types for user/game loading

type jsonUser struct {
	Handle   string   `json:"handle"`
	UserName string   `json:"user-name"`
	Email    string   `json:"email"`
	Timezone string   `json:"tz"`
	Password string   `json:"password"`
	Roles    []string `json:"roles"`
}

type jsonGame struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Clans       []struct {
		Handle string `json:"handle"`
		Clan   int    `json:"clan"`
	} `json:"clans"`
	Turns []struct {
		ID      int    `json:"id"`               // e.g., 89912 for year 899 month 12
		Year    int    `json:"year"`             // e.g., 899
		Month   int    `json:"month"`            // e.g., 12
		DueDate string `json:"orders-due-date"` // "2025/11/15 18:00:00 Australia/Sydney"
		Active  bool   `json:"active"`
	} `json:"turns"`
}

func loadUsersFromJSON(ctx context.Context, db *sql.DB, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read users file: %w", err)
	}

	var users []jsonUser
	if err := json.Unmarshal(data, &users); err != nil {
		return fmt.Errorf("parse users json: %w", err)
	}

	now := time.Now().Format(time.RFC3339)
	// Invalid bcrypt hash that will never match any password
	const invalidHash = "$2a$10$INVALID.HASH.THAT.WILL.NEVER.MATCH.ANY.PASSWORD.EVER"

	for _, ju := range users {
		isActive := hasRole(ju.Roles, "active")

		var hash string
		if isActive && ju.Password != "" {
			var err error
			hash, err = auth.HashPassword(ju.Password)
			if err != nil {
				return fmt.Errorf("hash password for %s: %w", ju.Handle, err)
			}
		} else {
			// Inactive users get an invalid hash so they can never log in
			hash = invalidHash
		}

		userName := ju.UserName
		if userName == "" {
			userName = ju.Handle // Use handle as fallback for inactive users
		}

		_, err = db.ExecContext(ctx, `
			INSERT INTO users (handle, user_name, email, timezone, password_hash, created_at)
			VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT(handle) DO UPDATE SET
				user_name = excluded.user_name,
				email = excluded.email,
				timezone = excluded.timezone,
				password_hash = excluded.password_hash
		`, ju.Handle, userName, ju.Email, ju.Timezone, hash, now)
		if err != nil {
			return fmt.Errorf("insert user %s: %w", ju.Handle, err)
		}

		// Delete existing roles and insert new ones
		_, _ = db.ExecContext(ctx, `DELETE FROM user_roles WHERE user_handle = ?`, ju.Handle)
		for _, role := range ju.Roles {
			_, err = db.ExecContext(ctx, `
				INSERT INTO user_roles (user_handle, role) VALUES (?, ?)
				ON CONFLICT DO NOTHING
			`, ju.Handle, role)
			if err != nil {
				return fmt.Errorf("insert role for %s: %w", ju.Handle, err)
			}
		}
	}

	return nil
}

func loadGamesFromJSON(ctx context.Context, db *sql.DB, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read games file: %w", err)
	}

	var games []jsonGame
	if err := json.Unmarshal(data, &games); err != nil {
		return fmt.Errorf("parse games json: %w", err)
	}

	for _, jg := range games {
		_, err = db.ExecContext(ctx, `
			INSERT INTO games (id, description) VALUES (?, ?)
			ON CONFLICT(id) DO UPDATE SET description = excluded.description
		`, jg.ID, jg.Description)
		if err != nil {
			return fmt.Errorf("insert game %s: %w", jg.ID, err)
		}

		for _, c := range jg.Clans {
			_, err = db.ExecContext(ctx, `
				INSERT INTO game_clans (game_id, user_handle, clan_no) VALUES (?, ?, ?)
				ON CONFLICT(game_id, user_handle) DO UPDATE SET clan_no = excluded.clan_no
			`, jg.ID, c.Handle, c.Clan)
			if err != nil {
				return fmt.Errorf("insert game clan %s/%s: %w", jg.ID, c.Handle, err)
			}
		}

		// Load game turns with timezone conversion to UTC
		for _, t := range jg.Turns {
			var dueDateUTC sql.NullString
			if t.DueDate != "" {
				// Parse format: "2025/11/15 18:00:00 Australia/Sydney"
				utc, err := parseDueDateWithTimezone(t.DueDate)
				if err != nil {
					return fmt.Errorf("invalid due_date %q for game %s turn %d: %w", t.DueDate, jg.ID, t.ID, err)
				}
				dueDateUTC = sql.NullString{String: utc.Format(time.RFC3339), Valid: true}
			}

			isActive := 0
			if t.Active {
				isActive = 1
			}

			_, err = db.ExecContext(ctx, `
				INSERT INTO game_turns (game_id, turn_id, year, month, is_active, due_date) VALUES (?, ?, ?, ?, ?, ?)
				ON CONFLICT(game_id, turn_id) DO UPDATE SET year = excluded.year, month = excluded.month, is_active = excluded.is_active, due_date = excluded.due_date
			`, jg.ID, t.ID, t.Year, t.Month, isActive, dueDateUTC)
			if err != nil {
				return fmt.Errorf("insert game turn %s/%d: %w", jg.ID, t.ID, err)
			}
		}
	}

	return nil
}

func hasRole(roles []string, target string) bool {
	for _, r := range roles {
		if r == target {
			return true
		}
	}
	return false
}

// parseDueDateWithTimezone parses a due date string in format "2025/11/15 18:00:00 Australia/Sydney"
// and returns the time in UTC.
func parseDueDateWithTimezone(s string) (time.Time, error) {
	// Find the last space to separate datetime from timezone
	lastSpace := strings.LastIndex(s, " ")
	if lastSpace == -1 {
		return time.Time{}, fmt.Errorf("invalid format: expected 'YYYY/MM/DD HH:MM:SS Timezone'")
	}

	datetimePart := s[:lastSpace]
	tzPart := s[lastSpace+1:]

	loc, err := time.LoadLocation(tzPart)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timezone %q: %w", tzPart, err)
	}

	t, err := time.ParseInLocation("2006/01/02 15:04:05", datetimePart, loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid datetime %q: %w", datetimePart, err)
	}

	return t.UTC(), nil
}

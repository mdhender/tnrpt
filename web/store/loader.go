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

// LoadFromDir loads all .docx files from a directory into the store.
// File names are expected to follow the pattern: GGGG.YYYY-MM.CCCC.docx
// where GGGG is game, YYYY-MM is turn, CCCC is clan.
func LoadFromDir(s Store, dir string) error {
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
		if !strings.HasSuffix(strings.ToLower(name), ".docx") {
			continue
		}

		path := filepath.Join(dir, name)
		if err := LoadFile(s, path); err != nil {
			log.Printf("store: load %s: %v", name, err)
			failed++
			continue
		}
		loaded++
	}

	log.Printf("store: loaded %d files (%d failed) from %s", loaded, failed, dir)
	return nil
}

// LoadFile loads a single .docx file into the store.
func LoadFile(s Store, path string) error {
	name := filepath.Base(path)
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

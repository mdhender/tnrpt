// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mdhender/tnrpt/web/auth"
)

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
		ID      int    `json:"id"`              // e.g., 89912 for year 899 month 12
		Year    int    `json:"year"`            // e.g., 899
		Month   int    `json:"month"`           // e.g., 12
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

// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package model

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/mdhender/tnrpt/web/auth"
)

type User struct {
	Handle       string
	UserName     string
	Email        string
	Timezone     string
	PasswordHash string
	CreatedAt    time.Time
	Roles        []string
}

type Game struct {
	ID          string
	Description string
}

type GameClan struct {
	GameID     string
	UserHandle string
	ClanNo     int
}

func (s *Store) InsertUser(ctx context.Context, u *User) error {
	const query = `
		INSERT INTO users (handle, user_name, email, timezone, password_hash, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(handle) DO UPDATE SET
			user_name = excluded.user_name,
			email = excluded.email,
			timezone = excluded.timezone,
			password_hash = excluded.password_hash
	`
	_, err := s.db.ExecContext(ctx, query,
		u.Handle,
		u.UserName,
		u.Email,
		u.Timezone,
		u.PasswordHash,
		u.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}

	if err := s.deleteUserRoles(ctx, u.Handle); err != nil {
		return err
	}
	for _, role := range u.Roles {
		if err := s.insertUserRole(ctx, u.Handle, role); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) deleteUserRoles(ctx context.Context, handle string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM user_roles WHERE user_handle = ?`, handle)
	if err != nil {
		return fmt.Errorf("delete user roles: %w", err)
	}
	return nil
}

func (s *Store) insertUserRole(ctx context.Context, handle, role string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO user_roles (user_handle, role) VALUES (?, ?) ON CONFLICT DO NOTHING`,
		handle, role)
	if err != nil {
		return fmt.Errorf("insert user role: %w", err)
	}
	return nil
}

func (s *Store) GetUserByHandle(ctx context.Context, handle string) (*User, error) {
	const query = `
		SELECT handle, user_name, email, timezone, password_hash, created_at
		FROM users WHERE handle = ?
	`
	var u User
	var email, tz sql.NullString
	var createdAt string

	err := s.db.QueryRowContext(ctx, query, handle).Scan(
		&u.Handle, &u.UserName, &email, &tz, &u.PasswordHash, &createdAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	u.Email = email.String
	u.Timezone = tz.String
	u.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

	roles, err := s.getUserRoles(ctx, handle)
	if err != nil {
		return nil, err
	}
	u.Roles = roles

	return &u, nil
}

func (s *Store) getUserRoles(ctx context.Context, handle string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT role FROM user_roles WHERE user_handle = ?`, handle)
	if err != nil {
		return nil, fmt.Errorf("get user roles: %w", err)
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

func (s *Store) IsUserActive(ctx context.Context, handle string) (bool, error) {
	roles, err := s.getUserRoles(ctx, handle)
	if err != nil {
		return false, err
	}
	for _, r := range roles {
		if r == "active" {
			return true, nil
		}
	}
	return false, nil
}

func (s *Store) InsertGame(ctx context.Context, g *Game) error {
	const query = `
		INSERT INTO games (id, description) VALUES (?, ?)
		ON CONFLICT(id) DO UPDATE SET description = excluded.description
	`
	_, err := s.db.ExecContext(ctx, query, g.ID, g.Description)
	if err != nil {
		return fmt.Errorf("insert game: %w", err)
	}
	return nil
}

func (s *Store) InsertGameClan(ctx context.Context, gc *GameClan) error {
	const query = `
		INSERT INTO game_clans (game_id, user_handle, clan_no) VALUES (?, ?, ?)
		ON CONFLICT(game_id, user_handle) DO UPDATE SET clan_no = excluded.clan_no
	`
	_, err := s.db.ExecContext(ctx, query, gc.GameID, gc.UserHandle, gc.ClanNo)
	if err != nil {
		return fmt.Errorf("insert game clan: %w", err)
	}
	return nil
}

func (s *Store) GetClanForUser(ctx context.Context, gameID, handle string) (int, error) {
	var clanNo int
	err := s.db.QueryRowContext(ctx,
		`SELECT clan_no FROM game_clans WHERE game_id = ? AND user_handle = ?`,
		gameID, handle).Scan(&clanNo)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get clan for user: %w", err)
	}
	return clanNo, nil
}

func (s *Store) ValidateCredentials(ctx context.Context, handle, password string) (*User, error) {
	u, err := s.GetUserByHandle(ctx, handle)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, nil
	}

	active, err := s.IsUserActive(ctx, handle)
	if err != nil {
		return nil, err
	}
	if !active {
		return nil, nil
	}

	if !auth.CheckPassword(password, u.PasswordHash) {
		return nil, nil
	}

	return u, nil
}

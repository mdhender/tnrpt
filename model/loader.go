// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package model

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/mdhender/tnrpt/web/auth"
)

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

func (s *Store) LoadUsersFromJSON(ctx context.Context, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read users file: %w", err)
	}

	var users []jsonUser
	if err := json.Unmarshal(data, &users); err != nil {
		return fmt.Errorf("parse users json: %w", err)
	}

	now := time.Now()
	for _, ju := range users {
		if !hasRole(ju.Roles, "active") {
			continue
		}

		hash, err := auth.HashPassword(ju.Password)
		if err != nil {
			return fmt.Errorf("hash password for %s: %w", ju.Handle, err)
		}

		u := &User{
			Handle:       ju.Handle,
			UserName:     ju.UserName,
			Email:        ju.Email,
			Timezone:     ju.Timezone,
			PasswordHash: hash,
			CreatedAt:    now,
			Roles:        ju.Roles,
		}

		if err := s.InsertUser(ctx, u); err != nil {
			return fmt.Errorf("insert user %s: %w", ju.Handle, err)
		}
	}

	return nil
}

func (s *Store) LoadGamesFromJSON(ctx context.Context, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read games file: %w", err)
	}

	var games []jsonGame
	if err := json.Unmarshal(data, &games); err != nil {
		return fmt.Errorf("parse games json: %w", err)
	}

	for _, jg := range games {
		g := &Game{
			ID:          jg.ID,
			Description: jg.Description,
		}
		if err := s.InsertGame(ctx, g); err != nil {
			return fmt.Errorf("insert game %s: %w", jg.ID, err)
		}

		for _, c := range jg.Clans {
			gc := &GameClan{
				GameID:     jg.ID,
				UserHandle: c.Handle,
				ClanNo:     c.Clan,
			}
			if err := s.InsertGameClan(ctx, gc); err != nil {
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

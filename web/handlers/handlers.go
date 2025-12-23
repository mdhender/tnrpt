// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/mdhender/tnrpt"
	"github.com/mdhender/tnrpt/web/auth"
	"github.com/mdhender/tnrpt/web/store"
	"github.com/mdhender/tnrpt/web/templates"
)

// Handlers holds dependencies for HTTP handlers.
type Handlers struct {
	store        *store.SQLiteStore
	sessions     *auth.SessionStore
	autoAuthUser *auth.User
}

// New creates a new Handlers with the given store and session store.
func New(s *store.SQLiteStore, sessions *auth.SessionStore) *Handlers {
	return &Handlers{store: s, sessions: sessions}
}

// getLayoutData returns layout data with turns for the authenticated user.
func (h *Handlers) getLayoutData(r *http.Request, session *auth.Session) templates.LayoutData {
	var data templates.LayoutData
	data.CurrentPath = r.URL.Path
	data.Version = tnrpt.Version().String()

	if session == nil {
		return data
	}

	turns, err := h.store.TurnsByClan(session.User.ClanID)
	if err != nil {
		log.Printf("warning: failed to get turns: %v", err)
		return data
	}
	data.Turns = turns

	if turnStr := r.URL.Query().Get("turn"); turnStr != "" {
		if t, err := strconv.Atoi(turnStr); err == nil {
			data.SelectedTurn = t
		}
	}

	return data
}

// Store returns the underlying SQLite store.
func (h *Handlers) Store() *store.SQLiteStore {
	return h.store
}

// Sessions returns the session store.
func (h *Handlers) Sessions() *auth.SessionStore {
	return h.sessions
}

// SetAutoAuth configures automatic authentication for testing.
// The username should be like "clan0500" which extracts ClanID "0500".
func (h *Handlers) SetAutoAuth(username string) {
	if len(username) >= 8 && username[:4] == "clan" {
		h.autoAuthUser = &auth.User{
			Username: username,
			ClanID:   username[4:],
		}
	}
}

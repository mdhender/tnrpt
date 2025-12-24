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
// It reads ?game= from the query string to determine which game context to use.
func (h *Handlers) getLayoutData(r *http.Request, session *auth.Session) templates.LayoutData {
	var data templates.LayoutData
	data.CurrentPath = r.URL.Path
	data.Version = tnrpt.Version().String()

	if session == nil {
		return data
	}

	data.UserHandle = session.User.Handle

	// Get all games for this user
	games, err := h.store.GetGamesForUser(r.Context(), session.User.Handle)
	if err != nil {
		log.Printf("warning: failed to get games for user: %v", err)
		return data
	}
	data.Games = games

	// Determine current game from ?game= param, defaulting to first game
	gameID := r.URL.Query().Get("game")
	if gameID == "" && len(games) > 0 {
		gameID = games[0].GameID
	}
	data.CurrentGameID = gameID

	// Find clan number for current game
	for _, g := range games {
		if g.GameID == gameID {
			data.CurrentClanNo = g.ClanNo
			break
		}
	}

	turns, err := h.store.TurnsByGameClan(gameID, data.CurrentClanNo)
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

	isGM, _ := h.store.IsUserGM(r.Context(), session.User.Handle)
	data.IsGM = isGM

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
func (h *Handlers) SetAutoAuth(gameID, handle string, clanNo int) {
	h.autoAuthUser = &auth.User{
		Handle:   handle,
		UserName: handle,
		GameID:   gameID,
		ClanNo:   clanNo,
	}
}

// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package handlers

import (
	"github.com/mdhender/tnrpt/web/auth"
	"github.com/mdhender/tnrpt/web/store"
)

// Handlers holds dependencies for HTTP handlers.
type Handlers struct {
	store    *store.MemoryStore
	sessions *auth.SessionStore
}

// New creates a new Handlers with the given store and session store.
func New(s *store.MemoryStore, sessions *auth.SessionStore) *Handlers {
	return &Handlers{store: s, sessions: sessions}
}

// Store returns the underlying memory store.
func (h *Handlers) Store() *store.MemoryStore {
	return h.store
}

// Sessions returns the session store.
func (h *Handlers) Sessions() *auth.SessionStore {
	return h.sessions
}

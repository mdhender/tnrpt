// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package handlers

import (
	"net/http"

	"github.com/mdhender/tnrpt/web/auth"
)

func (h *Handlers) Index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session := auth.GetSessionFromRequest(r, h.sessions)
	if session == nil && h.autoAuthUser != nil {
		session = h.sessions.Create(*h.autoAuthUser)
		auth.SetSessionCookie(w, session)
	}

	if session != nil {
		http.Redirect(w, r, "/units", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

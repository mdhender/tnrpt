// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package handlers

import (
	"net/http"

	"github.com/mdhender/tnrpt/web/auth"
	"github.com/mdhender/tnrpt/web/templates"
)

// SQLConsolePage renders the SQL console page (GET).
func (h *Handlers) SQLConsolePage(w http.ResponseWriter, r *http.Request) {
	session := auth.GetSessionFromRequest(r, h.sessions)
	data := h.getLayoutData(r, session)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.SQLConsole("", nil, data).Render(r.Context(), w); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// SQLConsoleExec executes a SQL query and renders results (POST).
func (h *Handlers) SQLConsoleExec(w http.ResponseWriter, r *http.Request) {
	session := auth.GetSessionFromRequest(r, h.sessions)
	data := h.getLayoutData(r, session)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	query := r.FormValue("query")
	result := h.store.ExecRawQuery(r.Context(), query)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.SQLConsole(query, result, data).Render(r.Context(), w); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package handlers

import (
	"net/http"

	"github.com/mdhender/tnrpt/web/auth"
	"github.com/mdhender/tnrpt/web/templates"
)

func (h *Handlers) Units(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session := auth.GetSessionFromRequest(r, h.sessions)
	if session == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	units := h.store.UnitsByClan(session.User.ClanID)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if r.Header.Get("HX-Request") == "true" {
		if err := templates.UnitsTable(units).Render(r.Context(), w); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	if err := templates.UnitsPage(units).Render(r.Context(), w); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (h *Handlers) UnitsNoAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	units := h.store.Units()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if r.Header.Get("HX-Request") == "true" {
		if err := templates.UnitsTable(units).Render(r.Context(), w); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	if err := templates.UnitsPage(units).Render(r.Context(), w); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

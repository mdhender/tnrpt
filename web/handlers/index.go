// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package handlers

import (
	"net/http"

	"github.com/mdhender/tnrpt/web/templates"
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

	stats := h.store.Stats()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.Index(stats).Render(r.Context(), w); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

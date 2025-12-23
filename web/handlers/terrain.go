// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package handlers

import (
	"net/http"
	"strconv"

	"github.com/mdhender/tnrpt/web/auth"
	"github.com/mdhender/tnrpt/web/templates"
)

func (h *Handlers) Terrain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session := auth.GetSessionFromRequest(r, h.sessions)
	if session == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	layoutData := h.getLayoutData(r, session)

	observations, err := h.store.TerrainObservationsByGameClan(layoutData.CurrentGameID, layoutData.CurrentClanNo, layoutData.SelectedTurn)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if r.Header.Get("HX-Request") == "true" {
		if err := templates.TerrainTable(observations).Render(r.Context(), w); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	if err := templates.TerrainPageWithData(observations, layoutData).Render(r.Context(), w); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (h *Handlers) TileDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session := auth.GetSessionFromRequest(r, h.sessions)
	if session == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	grid := r.PathValue("grid")
	colStr := r.PathValue("col")
	rowStr := r.PathValue("row")

	col, err := strconv.Atoi(colStr)
	if err != nil {
		http.Error(w, "Invalid column", http.StatusBadRequest)
		return
	}
	row, err := strconv.Atoi(rowStr)
	if err != nil {
		http.Error(w, "Invalid row", http.StatusBadRequest)
		return
	}

	layoutData := h.getLayoutData(r, session)
	layoutData.HideTurnSelect = true

	tile, err := h.store.TileDetailByGameClanCoord(grid, col, row, layoutData.CurrentGameID, layoutData.CurrentClanNo)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if tile == nil {
		http.Error(w, "Tile not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := templates.TileDetailPage(tile, layoutData).Render(r.Context(), w); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

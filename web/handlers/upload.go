// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/mdhender/tnrpt/adapters"
	"github.com/mdhender/tnrpt/model"
	"github.com/mdhender/tnrpt/pipelines/parsers/bistre"
	"github.com/mdhender/tnrpt/pipelines/parsers/docx"
	"github.com/mdhender/tnrpt/pipelines/parsers/report"
	"github.com/mdhender/tnrpt/web/auth"
	"github.com/mdhender/tnrpt/web/templates"
)

var (
	// Clan numbers are 0001-0999 (must start with 0)
	docxPattern             = regexp.MustCompile(`^(0\d{3})\.docx$`)
	gameTurnClanDocxPattern = regexp.MustCompile(`^(\d{4})\.(\d{4}-\d{2})\.(0\d{3})\.docx$`)
	gameTurnClanTextPattern = regexp.MustCompile(`^(\d{4})\.(\d{4}-\d{2})\.(0\d{3})\.report\.txt$`)
)

const (
	docxContentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
)

type uploadResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	Clan    string `json:"clan,omitempty"`
	Game    string `json:"game,omitempty"`
	Turn    string `json:"turn,omitempty"`
	Units   int    `json:"units,omitempty"`
	Acts    int    `json:"acts,omitempty"`
	Steps   int    `json:"steps,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, resp uploadResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

// UploadPage renders the upload page for GMs.
func (h *Handlers) UploadPage(w http.ResponseWriter, r *http.Request) {
	session := auth.GetSessionFromRequest(r, h.sessions)
	data := h.getLayoutData(r, session)

	games, err := h.store.GetAllGames(r.Context())
	if err != nil {
		http.Error(w, "Failed to load games", http.StatusInternalServerError)
		return
	}

	gameOptions := make([]templates.GameOption, len(games))
	for i, g := range games {
		turns := make([]templates.GameTurnOption, len(g.Turns))
		for j, t := range g.Turns {
			year, month := t.TurnNo/100, t.TurnNo%100
			turns[j] = templates.GameTurnOption{ID: fmt.Sprintf("%04d-%02d", year, month), IsActive: t.IsActive}
		}
		gameOptions[i] = templates.GameOption{ID: g.ID, Description: g.Description, Turns: turns}
	}

	selectedGame := r.URL.Query().Get("game")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.UploadPage(data, gameOptions, selectedGame, nil, 0).Render(r.Context(), w)
}

// UploadHandler handles POST requests to upload files.
// Protected route: requires GM role.
// Accepts files named CCCC.docx or GGGG.YYYY-MM.CCCC.report.txt
func (h *Handlers) UploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, uploadResponse{Error: "method not allowed"})
		return
	}

	if err := r.ParseMultipartForm(100 << 10); err != nil { // 100KB max
		writeJSON(w, http.StatusBadRequest, uploadResponse{Error: "failed to parse form: " + err.Error()})
		return
	}

	game := r.FormValue("game")
	turn := r.FormValue("turn")

	if game == "" {
		writeJSON(w, http.StatusBadRequest, uploadResponse{Error: "game is required"})
		return
	}
	if turn == "" {
		writeJSON(w, http.StatusBadRequest, uploadResponse{Error: "turn is required"})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, uploadResponse{Error: "no file uploaded"})
		return
	}
	defer file.Close()

	filename := header.Filename
	contentType := header.Header.Get("Content-Type")

	clan, fileGame, fileTurn, validationErr := validateFilename(filename)
	if validationErr != "" {
		writeJSON(w, http.StatusBadRequest, uploadResponse{Error: validationErr})
		return
	}

	if strings.HasSuffix(strings.ToLower(filename), ".docx") {
		if contentType != "" && contentType != docxContentType {
			writeJSON(w, http.StatusBadRequest, uploadResponse{
				Error: "invalid content type for .docx file: expected Word document",
			})
			return
		}
	}

	if fileGame != "" && fileGame != game {
		writeJSON(w, http.StatusBadRequest, uploadResponse{
			Error: "game in filename (" + fileGame + ") does not match selected game (" + game + ")",
		})
		return
	}
	if fileTurn != "" && fileTurn != turn {
		writeJSON(w, http.StatusBadRequest, uploadResponse{
			Error: "turn in filename (" + fileTurn + ") does not match selected turn (" + turn + ")",
		})
		return
	}

	// Read the file contents
	data, err := io.ReadAll(file)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, uploadResponse{Error: "failed to read file: " + err.Error()})
		return
	}
	hash := sha256.Sum256(data)

	// Parse the file based on type
	var text []byte
	var mime string

	if strings.HasSuffix(strings.ToLower(filename), ".docx") {
		// Parse DOCX file
		doc, err := docx.ParseReader(bytes.NewReader(data), true, true, true, false, false)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, uploadResponse{Error: "failed to parse docx: " + err.Error()})
			return
		}

		// Parse report to extract sections
		rpt, err := report.ParseReportText(doc, true, true, true, false, false)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, uploadResponse{Error: "failed to parse report: " + err.Error()})
			return
		}

		// Combine sections into text for bistre parser
		for _, section := range rpt.Sections {
			text = append(text, bytes.Join(section.Lines, []byte{'\n'})...)
			text = append(text, '\n')
		}
		mime = docxContentType
	} else {
		// Plain text report
		text = data
		mime = "text/plain"
	}

	// Run bistre parser
	parsedTurn, err := bistre.ParseInput(filename, turn, text, false, false, false, false, false, false, false, false, bistre.ParseConfig{})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, uploadResponse{Error: "failed to parse turn report: " + err.Error()})
		return
	}
	if parsedTurn == nil {
		writeJSON(w, http.StatusBadRequest, uploadResponse{Error: "parser returned no data"})
		return
	}

	// Convert to model and store
	turnNo := 100*parsedTurn.Year + parsedTurn.Month
	now := time.Now().UTC()

	// Create report file record
	rf := &model.ReportFile{
		Game:      game,
		ClanNo:    clan,
		TurnNo:    turnNo,
		Name:      filename,
		SHA256:    hex.EncodeToString(hash[:]),
		Mime:      mime,
		CreatedAt: now,
	}
	if err := h.store.AddReportFile(rf); err != nil {
		writeJSON(w, http.StatusInternalServerError, uploadResponse{Error: "failed to store report file: " + err.Error()})
		return
	}

	// Convert parsed turn to model
	rx, err := adapters.BistreTurnToModelReportX(filename, parsedTurn, game, clan)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, uploadResponse{Error: "failed to convert report: " + err.Error()})
		return
	}
	rx.ReportFileID = rf.ID

	// Store the report
	if err := h.store.AddReport(rx); err != nil {
		writeJSON(w, http.StatusInternalServerError, uploadResponse{Error: "failed to store report: " + err.Error()})
		return
	}

	// Count results for response
	units := len(rx.Units)
	acts := 0
	steps := 0
	for _, u := range rx.Units {
		acts += len(u.Acts)
		for _, a := range u.Acts {
			steps += len(a.Steps)
		}
	}

	writeJSON(w, http.StatusOK, uploadResponse{
		Success: true,
		Clan:    clan,
		Game:    game,
		Turn:    turn,
		Units:   units,
		Acts:    acts,
		Steps:   steps,
	})
}

// validateFilename checks if the filename matches expected patterns.
// Returns clan, game (if in filename), turn (if in filename), and error message.
// Patterns:
//   - CCCC.docx -> clan only
//   - GGGG.YYYY-MM.CCCC.docx -> game, turn, clan
//   - GGGG.YYYY-MM.CCCC.report.txt -> game, turn, clan
func validateFilename(filename string) (clan, game, turn, errMsg string) {
	if matches := docxPattern.FindStringSubmatch(filename); matches != nil {
		return matches[1], "", "", ""
	}
	if matches := gameTurnClanDocxPattern.FindStringSubmatch(filename); matches != nil {
		return matches[3], matches[1], matches[2], ""
	}
	if matches := gameTurnClanTextPattern.FindStringSubmatch(filename); matches != nil {
		return matches[3], matches[1], matches[2], ""
	}

	return "", "", "", "invalid filename: must be CCCC.docx or GGGG.YYYY-MM.CCCC.report.txt (where CCCC is clan 0001-0999, GGGG is 4-digit game, YYYY-MM is turn)"
}

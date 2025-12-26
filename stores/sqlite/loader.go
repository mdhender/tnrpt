// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package store

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mdhender/tnrpt/adapters"
	"github.com/mdhender/tnrpt/model"
	"github.com/mdhender/tnrpt/pipelines/parsers/bistre"
	"github.com/mdhender/tnrpt/pipelines/parsers/docx"
	"github.com/mdhender/tnrpt/pipelines/parsers/report"
)

var (
	reDocxReportFileName = regexp.MustCompile(`^\d{4}.\d{4}-\d{2}.0\d{3}.docx$`)
	reTextReportFileName = regexp.MustCompile(`^\d{4}.\d{4}-\d{2}.0\d{3}.report.txt$`)
)

// LoadDocxFromDir loads all .docx files from a directory into the store.
// File names are expected to follow the pattern: GGGG.YYYY-MM.CCCC.docx
// where GGGG is game, YYYY-MM is turn, CCCC is clan.
func LoadDocxFromDir(s model.Store, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read dir: %w", err)
	}

	var loaded, failed int
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !reDocxReportFileName.MatchString(strings.ToLower(name)) {
			continue
		} else if !strings.HasSuffix(strings.ToLower(name), ".docx") {
			continue
		}

		path := filepath.Join(dir, name)
		if err := LoadDocxFile(s, path); err != nil {
			log.Printf("store: load %s: %v", name, err)
			failed++
			continue
		}
		loaded++
	}

	log.Printf("store: loaded %d docx files (%d failed) from %s", loaded, failed, dir)
	return nil
}

// LoadDocxFile loads a single .docx file into the store.
func LoadDocxFile(s model.Store, path string) error {
	name := filepath.Base(path)
	if !reDocxReportFileName.MatchString(strings.ToLower(name)) {
		return fmt.Errorf("invalid report file name")
	}
	game, clanNo := parseFilename(name)

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}
	hash := sha256.Sum256(data)

	doc, err := docx.ParsePath(path, true, true, true, false, false)
	if err != nil {
		return fmt.Errorf("parse docx: %w", err)
	}

	rpt, err := report.ParseReportText(doc, true, true, true, false, false)
	if err != nil {
		return fmt.Errorf("parse report: %w", err)
	}

	var text []byte
	for _, section := range rpt.Sections {
		text = append(text, bytes.Join(section.Lines, []byte{'\n'})...)
		text = append(text, '\n')
	}

	turn, err := bistre.ParseInput(rpt.Name, rpt.TurnNo, text, false, false, false, false, false, false, false, false, bistre.ParseConfig{})
	if err != nil {
		return fmt.Errorf("parse input: %w", err)
	}
	if turn == nil {
		return fmt.Errorf("parser returned nil")
	}

	turnNo := 100*turn.Year + turn.Month
	rf := &model.ReportFile{
		Game:      game,
		ClanNo:    clanNo,
		TurnNo:    turnNo,
		Name:      name,
		SHA256:    hex.EncodeToString(hash[:]),
		Mime:      "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		CreatedAt: time.Now().UTC(),
	}
	if err := s.AddReportFile(rf); err != nil {
		return fmt.Errorf("add report file: %w", err)
	}

	rx, err := adapters.BistreTurnToModelReportX(name, turn, game, clanNo)
	if err != nil {
		return fmt.Errorf("adapt to model: %w", err)
	}
	rx.ReportFileID = rf.ID

	err = s.AddReport(rx)
	if err != nil {
		return err
	}
	return nil
}

var filenameRe = regexp.MustCompile(`^(\d{4})\.(\d{3,4}-\d{2})\.(\d{4})\.docx$`)

func parseFilename(name string) (game, clan string) {
	matches := filenameRe.FindStringSubmatch(name)
	if len(matches) == 4 {
		return matches[1], matches[3]
	}
	return "", ""
}

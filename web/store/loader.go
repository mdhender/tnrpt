// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package store

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mdhender/tnrpt/adapters"
	"github.com/mdhender/tnrpt/pipelines/parsers/bistre"
	"github.com/mdhender/tnrpt/pipelines/parsers/docx"
	"github.com/mdhender/tnrpt/pipelines/parsers/report"
)

// LoadFromDir loads all .docx files from a directory into the store.
// File names are expected to follow the pattern: GGGG.YYYY-MM.CCCC.docx
// where GGGG is game, YYYY-MM is turn, CCCC is clan.
func LoadFromDir(s *MemoryStore, dir string) error {
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
		if !strings.HasSuffix(strings.ToLower(name), ".docx") {
			continue
		}

		path := filepath.Join(dir, name)
		if err := LoadFile(s, path); err != nil {
			log.Printf("store: load %s: %v", name, err)
			failed++
			continue
		}
		loaded++
	}

	log.Printf("store: loaded %d files (%d failed) from %s", loaded, failed, dir)
	return nil
}

// LoadFile loads a single .docx file into the store.
func LoadFile(s *MemoryStore, path string) error {
	name := filepath.Base(path)
	game, clanNo := parseFilename(name)

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

	rx, err := adapters.BistreTurnToModelReportX(name, turn, game, clanNo)
	if err != nil {
		return fmt.Errorf("adapt to model: %w", err)
	}

	s.AddReport(rx)
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

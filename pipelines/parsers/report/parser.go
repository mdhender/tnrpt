// Copyright (c) 2025 Michael D Henderson. All rights reserved.

// Package report implements a parser that splits a turn report into
// sections. Sections contain only the lines needed to create maps.
package report

import (
	"bytes"
	"fmt"
	"log"
	"path/filepath"
	"regexp"

	"github.com/mdhender/tnrpt/pipelines/parsers/docx"
)

func ParseReportText(d *docx.Docx, normalizeCRLF, normalizeCR, quiet, verbose, debug bool) (Report, error) {
	r := Report{
		Path: filepath.Dir(d.Source),
		Name: filepath.Base(d.Source),
	}

	text := d.Text
	if normalizeCRLF { // CR + LF is Windows end-of-line marker
		if debug {
			log.Printf("report: replacing CR+LF with LF")
		}
		text = bytes.ReplaceAll(text, []byte{CR, LF}, []byte{LF})
	}
	if normalizeCR { // CR is old Mac end-of-line marker?
		if debug {
			log.Printf("report: replacing CR with LF")
		}
		text = bytes.ReplaceAll(text, []byte{CR}, []byte{LF})
	}

	var section *Section
	for _, line := range bytes.Split(text, []byte{LF}) {
		if idx := reClanSection.FindSubmatchIndex(line); idx != nil {
			if section != nil {
				r.Sections = append(r.Sections, section)
			}
			section = &Section{
				UnitId: string(line[idx[2]:idx[3]]),
				Kind:   "clan",
				Lines:  [][]byte{line},
			}
			continue
		} else if idx = reCourierSection.FindSubmatchIndex(line); idx != nil {
			if section != nil {
				r.Sections = append(r.Sections, section)
			}
			section = &Section{
				UnitId: string(line[idx[2]:idx[3]]),
				Kind:   "courier",
				Lines:  [][]byte{line},
			}
			continue
		} else if idx = reElementSection.FindSubmatchIndex(line); idx != nil {
			if section != nil {
				r.Sections = append(r.Sections, section)
			}
			section = &Section{
				UnitId: string(line[idx[2]:idx[3]]),
				Kind:   "element",
				Lines:  [][]byte{line},
			}
			continue
		} else if idx = reFleetSection.FindSubmatchIndex(line); idx != nil {
			if section != nil {
				r.Sections = append(r.Sections, section)
			}
			section = &Section{
				UnitId: string(line[idx[2]:idx[3]]),
				Kind:   "fleet",
				Lines:  [][]byte{line},
			}
			continue
		} else if idx = reGarrisonSection.FindSubmatchIndex(line); idx != nil {
			if section != nil {
				r.Sections = append(r.Sections, section)
			}
			section = &Section{
				UnitId: string(line[idx[2]:idx[3]]),
				Kind:   "garrison",
				Lines:  [][]byte{line},
			}
			continue
		} else if idx = reTribeSection.FindSubmatchIndex(line); idx != nil {
			if section != nil {
				r.Sections = append(r.Sections, section)
			}
			section = &Section{
				UnitId: string(line[idx[2]:idx[3]]),
				Kind:   "tribe",
				Lines:  [][]byte{line},
			}
		}

		if section == nil {
			continue
		}

		if idx := reCurrentTurn.FindSubmatchIndex(line); idx != nil {
			section.TurnNo = string(line[idx[2]:idx[3]])
			section.Lines = append(section.Lines, line)
			continue
		}

		if reClanScry.Match(line) {
			section.Lines = append(section.Lines, line)
		} else if reCourierScry.Match(line) {
			section.Lines = append(section.Lines, line)
		} else if reElementScry.Match(line) {
			section.Lines = append(section.Lines, line)
		} else if reFleetScry.Match(line) {
			section.Lines = append(section.Lines, line)
		} else if reGarrisonScry.Match(line) {
			section.Lines = append(section.Lines, line)
		} else if reTribeScry.Match(line) {
			section.Lines = append(section.Lines, line)
		}

		if reFleetMovement.Match(line) {
			section.Lines = append(section.Lines, line)
			continue
		} else if reTribeFollows.Match(line) {
			section.Lines = append(section.Lines, line)
			continue
		} else if reTribeGoesTo.Match(line) {
			section.Lines = append(section.Lines, line)
			continue
		} else if reTribeMovement.Match(line) {
			section.Lines = append(section.Lines, line)
			continue
		}

		if reScout.Match(line) {
			section.Lines = append(section.Lines, line)
			continue
		}

		if reClanStatus.Match(line) {
			section.Lines = append(section.Lines, line)
			r.Sections = append(r.Sections, section)
			section = nil
		} else if reCourierStatus.Match(line) {
			section.Lines = append(section.Lines, line)
			r.Sections = append(r.Sections, section)
			section = nil
		} else if reElementStatus.Match(line) {
			section.Lines = append(section.Lines, line)
			r.Sections = append(r.Sections, section)
			section = nil
		} else if reFleetStatus.Match(line) {
			section.Lines = append(section.Lines, line)
			r.Sections = append(r.Sections, section)
			section = nil
		} else if reGarrisonStatus.Match(line) {
			section.Lines = append(section.Lines, line)
			r.Sections = append(r.Sections, section)
			section = nil
		} else if reTribeStatus.Match(line) {
			section.Lines = append(section.Lines, line)
			r.Sections = append(r.Sections, section)
			section = nil
		}
	}
	if section != nil {
		r.Sections = append(r.Sections, section)
	}

	if len(r.Sections) == 0 {
		return Report{}, fmt.Errorf("invalid report: no sections")
	}

	for _, section := range r.Sections {
		if r.TurnNo == "" {
			r.TurnNo = section.TurnNo
			continue
		}
		if r.TurnNo != section.TurnNo {
			return Report{}, fmt.Errorf("invalid report: multiple turns")
		}
	}

	if r.TurnNo == "" {
		return Report{}, fmt.Errorf("invalid report: no turn info")
	}

	return r, nil
}

const (
	CR = 0x0d // carriage return
	LF = 0x0a // line feed
)

var (
	// Tribe 0987, , Current Hex = QQ 0909, (Previous Hex = QQ 1010)
	reClanSection     = regexp.MustCompile(`^Tribe\s(0\d{3}),`)
	reCourierSection  = regexp.MustCompile(`^Courier\s(\d{4}c[1-9]),`)
	reElementSection  = regexp.MustCompile(`^Element\s(\d{4}e[1-9]),`)
	reFleetSection    = regexp.MustCompile(`^Fleet\s(\d{4}f[1-9]),`)
	reGarrisonSection = regexp.MustCompile(`^Garrison\s(\d{4}g[1-9]),`)
	reTribeSection    = regexp.MustCompile(`^Tribe\s(\d{4}),`)

	// Current Turn 899-12 (#0), Winter, FINE Next Turn 900-01 (#1), 28/11/2025
	reCurrentTurn = regexp.MustCompile(`^Current\sTurn\s(\d{3,4}-\d{2})\s\(#\d+\),`)

	// 0987 Scry: QQ 1010:
	reClanScry     = regexp.MustCompile(`^0\d{3}\sScry:`)
	reCourierScry  = regexp.MustCompile(`^\d{4}c[1-9]\sScry:`)
	reElementScry  = regexp.MustCompile(`^\d{4}e[1-9]\sScry:`)
	reFleetScry    = regexp.MustCompile(`^\d{4}f[1-9]\sScry:`)
	reGarrisonScry = regexp.MustCompile(`^\d{4}g[1-9]\sScry:`)
	reTribeScry    = regexp.MustCompile(`^\d{4}\sScry:`)

	// CALM NE Fleet Movement:
	reFleetMovement = regexp.MustCompile(`^(CALM|MILD|STRONG|GALE)\s[NS][EW]?\sFleet\sMovement:`)

	// Tribe Follows 0987e1
	reTribeFollows = regexp.MustCompile(`^Tribe Follows\s`)

	// Tribe Goes to QQ 1010
	reTribeGoesTo = regexp.MustCompile(`^Tribe Goes to\s`)

	// Tribe Movement: Move
	reTribeMovement = regexp.MustCompile(`^Tribe Movement:`)

	// Scout 1:Scout
	reScout = regexp.MustCompile(`^Scout\s\d:Scout`)

	// 0987 Status:
	reClanStatus     = regexp.MustCompile(`^0\d{3}\sStatus:`)
	reCourierStatus  = regexp.MustCompile(`^\d{4}c[1-9]\sStatus:`)
	reElementStatus  = regexp.MustCompile(`^\d{4}e[1-9]\sStatus:`)
	reFleetStatus    = regexp.MustCompile(`^\d{4}f[1-9]\sStatus:`)
	reGarrisonStatus = regexp.MustCompile(`^\d{4}g[1-9]\sStatus:`)
	reTribeStatus    = regexp.MustCompile(`^\d{4}\sStatus:`)
)

type Report struct {
	Path     string
	Name     string
	TurnNo   string
	Sections []*Section
}

type Section struct {
	UnitId string
	Kind   string
	TurnNo string // yyyy-mm
	Lines  [][]byte
}

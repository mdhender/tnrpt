// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package renderer

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/mdhender/tnrpt"
	"github.com/mdhender/tnrpt/parsers/azul"
	"github.com/mdhender/tnrpt/turns"
)

type Renderer struct {
	source       string
	input        []byte
	debug        bool
	quiet        bool
	verbose      bool
	excludeUnits map[string]bool
	includeUnits map[string]bool
	autoEOL      bool
	parser       azul.ParseConfig
	stripCR      bool
}

func New(path string, quiet, verbose, debug bool, options ...Option) (*Renderer, error) {
	p := &Renderer{
		source:       path,
		debug:        debug,
		quiet:        quiet,
		verbose:      verbose,
		excludeUnits: make(map[string]bool),
		includeUnits: make(map[string]bool),
	}
	for _, option := range options {
		err := option(p)
		if err != nil {
			return nil, err
		}
	}
	return p, nil
}

func (r *Renderer) Run() (*azul.Turn_t, error) {
	started := time.Now()

	path, fileName := filepath.Dir(r.source), filepath.Base(r.source)
	inputs, err := turns.CollectInputs(path, fileName, r.quiet, r.verbose, r.debug)
	if err != nil {
		log.Fatalf("error: inputs: %v\n", err)
	}
	log.Printf("inputs: found %d turn reports\n", len(inputs))

	// allTurns holds the turn and move data and allows multiple clans to be loaded.
	allTurns := map[string][]*azul.Turn_t{}
	totalUnitMoves := 0
	var turnId, maxTurnId string // will be set to the last/maximum turnId we process

	var turn *azul.Turn_t // the last turn parsed

	for _, i := range inputs {
		data, err := os.ReadFile(i.Path)
		if err != nil {
			if r.debug {
				log.Printf("render: %q: %v\n", i.Path, err)
			}
			return nil, err
		}
		if r.autoEOL {
			if r.debug {
				log.Printf("render: auto-eol: replacing CR+LF and CR with LF")
			}
			data = bytes.ReplaceAll(data, []byte{'\r', '\n'}, []byte{'\n'})
			data = bytes.ReplaceAll(data, []byte{'\r'}, []byte{'\n'})
		} else if r.stripCR {
			if r.debug {
				log.Printf("render: strip-cr: replacing CR+LF with LF")
			}
			data = bytes.ReplaceAll(data, []byte{'\r', '\n'}, []byte{'\n'})
		}

		turnId = fmt.Sprintf("%04d-%02d", i.Turn.Year, i.Turn.Month)
		if turnId > maxTurnId {
			maxTurnId = turnId
		}

		var acceptLoneDash, parserDebugFlag, sectionsDebugFlag, stepsDebugFlag, nodesDebugFlag, fleetMovementDebugFlag, splitTrailingUnits, cleanupScoutStill bool

		turn, err = azul.ParseInput(i.Id, turnId, data, acceptLoneDash, parserDebugFlag, sectionsDebugFlag, stepsDebugFlag, nodesDebugFlag, fleetMovementDebugFlag, splitTrailingUnits, cleanupScoutStill, r.parser)
		if err != nil {
			log.Fatal(err)
		} else if turnId != fmt.Sprintf("%04d-%02d", turn.Year, turn.Month) {
			if turn.Year == 0 && turn.Month == 0 {
				log.Printf("error: unable to locate turn information in file\n")
				log.Printf("error: this is usually caused by unexpected line endings in the file\n")
				log.Printf("error: try running with --auto-eol\n")
			}
			log.Fatalf("error: expected turn %q: got turn %q\n", turnId, fmt.Sprintf("%04d-%02d", turn.Year, turn.Month))
		}
		//log.Printf("len(turn.SpecialNames) = %d\n", len(turn.SpecialNames))

		allTurns[turnId] = append(allTurns[turnId], turn)
		totalUnitMoves += len(turn.UnitMoves)
		log.Printf("%q: parsed %6d units in %v\n", i.Id, len(turn.UnitMoves), time.Since(started))
	}
	log.Printf("parsed %d inputs in to %d turns and %d units in %v\n", len(inputs), len(allTurns), totalUnitMoves, time.Since(started))

	return turn, nil
}

func (r *Renderer) ParseInput() (Node, error) {
	return &RootNode{
		BaseNode: BaseNode{Type: "root"},
		Source:   r.source,
		Version:  tnrpt.Version().String(),
	}, nil
}

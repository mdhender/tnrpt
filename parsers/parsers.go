// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package parsers

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/mdhender/tnrpt/parsers/azul"
	"github.com/mdhender/tnrpt/turns"
)

func ParseTurnReport(source string, autoEOL, stripCR, quiet, verbose, debug bool) (*azul.Turn_t, error) {
	started := time.Now()

	path, fileName := filepath.Dir(source), filepath.Base(source)
	inputs, err := turns.CollectInputs(path, fileName, quiet, verbose, debug)
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
		started := time.Now()

		data, err := os.ReadFile(i.Path)
		if err != nil {
			if debug {
				log.Printf("render: %q: %v\n", i.Path, err)
			}
			return nil, err
		}
		if autoEOL {
			if debug {
				log.Printf("render: auto-eol: replacing CR+LF and CR with LF")
			}
			data = bytes.ReplaceAll(data, []byte{'\r', '\n'}, []byte{'\n'})
			data = bytes.ReplaceAll(data, []byte{'\r'}, []byte{'\n'})
		} else if stripCR {
			if debug {
				log.Printf("render: strip-cr: replacing CR+LF with LF")
			}
			data = bytes.ReplaceAll(data, []byte{'\r', '\n'}, []byte{'\n'})
		}

		turnId = fmt.Sprintf("%04d-%02d", i.Turn.Year, i.Turn.Month)
		if turnId > maxTurnId {
			maxTurnId = turnId
		}

		var acceptLoneDash, parserDebugFlag, sectionsDebugFlag, stepsDebugFlag, nodesDebugFlag, fleetMovementDebugFlag, splitTrailingUnits, cleanupScoutStill bool

		turn, err = azul.ParseInput(i.Id, turnId, data, acceptLoneDash, parserDebugFlag, sectionsDebugFlag, stepsDebugFlag, nodesDebugFlag, fleetMovementDebugFlag, splitTrailingUnits, cleanupScoutStill, azul.ParseConfig{})
		if err != nil {
			return nil, err
		}
		if turnId != fmt.Sprintf("%04d-%02d", turn.Year, turn.Month) {
			if turn.Year == 0 && turn.Month == 0 {
				log.Printf("error: unable to locate turn information in file\n")
				log.Printf("error: this is usually caused by unexpected line endings in the file\n")
				log.Printf("error: try running with --auto-eol\n")
				return nil, fmt.Errorf("unable to find current turn in source")
			}
			log.Printf("error: expected turn %q: got turn %q\n", turnId, fmt.Sprintf("%04d-%02d", turn.Year, turn.Month))
			return nil, fmt.Errorf("unexpected current turn in source")
		}

		allTurns[turnId] = append(allTurns[turnId], turn)
		totalUnitMoves += len(turn.UnitMoves)
		if verbose {
			log.Printf("%q: parsed %6d units in %v\n", i.Id, len(turn.UnitMoves), time.Since(started))
		}
	}
	if verbose {
		log.Printf("parsed %d inputs in to %d turns and %d units in %v\n", len(inputs), len(allTurns), totalUnitMoves, time.Since(started))
	}

	return turn, nil
}

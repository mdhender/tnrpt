// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package adapters_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mdhender/tnrpt"
	"github.com/mdhender/tnrpt/adapters"
	"github.com/mdhender/tnrpt/parsers/azul"
)

const testdataPath = "../testdata"

func TestAzulParserTurnToModel(t *testing.T) {
	inputPath := filepath.Join(testdataPath, "0899-12.0987.report.txt")
	input, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("read input: %v", err)
	}

	pt, err := azul.ParseInput(
		inputPath,          // fid
		"",                 // tid (will be parsed from input)
		input,              // input
		false,              // acceptLoneDash
		false,              // debugParser
		false,              // debugSections
		false,              // debugSteps
		false,              // debugNodes
		false,              // debugFleetMovement
		false,              // experimentalUnitSplit
		false,              // experimentalScoutStill
		azul.ParseConfig{}, // cfg
	)
	if err != nil {
		t.Fatalf("parse: %v", err)
	} else if pt == nil {
		t.Fatalf("parse: parsed turn is nil")
	}

	at, err := adapters.AzulParserTurnToModel("<input>", pt)
	if err != nil {
		t.Fatalf("adapt: parser -> model %v", err)
	} else if at == nil {
		t.Fatalf("adapt: adapted turn is nil")
	}

	if at.Id != pt.Id {
		t.Errorf("Turn.Id: want %q, got %q", pt.Id, at.Id)
	}
	if at.Year != pt.Year {
		t.Errorf("Turn.Year: want %d, got %d", pt.Year, at.Year)
	}
	if at.Month != pt.Month {
		t.Errorf("Turn.Month: want %d, got %d", pt.Month, at.Month)
	}

	if len(at.UnitMoves) != len(pt.UnitMoves) {
		t.Errorf("Turn.UnitMoves length: want %d, got %d", len(pt.UnitMoves), len(at.UnitMoves))
	}

	for pUnitId, pMoves := range pt.UnitMoves {
		mUnitId := tnrpt.UnitId_t(pUnitId)
		mMoves, ok := at.UnitMoves[mUnitId]
		if !ok {
			t.Errorf("UnitMoves[%s]: missing in adapted model", pUnitId)
			continue
		}

		compareMoves(t, string(pUnitId), pMoves, mMoves)
	}

	data, err := json.MarshalIndent(at, "", "  ")
	if err != nil {
		t.Fatalf("json: marshal %v", err)
	}
	if len(data) == 0 {
		t.Errorf("json: length: want > 0, got %d", len(data))
	}
}

func compareMoves(t *testing.T, unitId string, pm *azul.Moves_t, mm *tnrpt.Moves_t) {
	t.Helper()

	if string(mm.Follows) != string(pm.Follows) {
		t.Errorf("Moves[%s].Follows: want %q, got %q", unitId, pm.Follows, mm.Follows)
	}
	if mm.GoesTo != pm.GoesTo {
		t.Errorf("Moves[%s].GoesTo: want %q, got %q", unitId, pm.GoesTo, mm.GoesTo)
	}
	if mm.PreviousHex != pm.PreviousHex {
		t.Errorf("Moves[%s].PreviousHex: want %q, got %q", unitId, pm.PreviousHex, mm.PreviousHex)
	}
	if mm.CurrentHex != pm.CurrentHex {
		t.Errorf("Moves[%s].CurrentHex: want %q, got %q", unitId, pm.CurrentHex, mm.CurrentHex)
	}

	if len(mm.Moves) != len(pm.Moves) {
		t.Errorf("Moves[%s].Moves length: want %d, got %d", unitId, len(pm.Moves), len(mm.Moves))
	}
	if len(mm.Scries) != len(pm.Scries) {
		t.Errorf("Moves[%s].Scries length: want %d, got %d", unitId, len(pm.Scries), len(mm.Scries))
	}
	if len(mm.Scouts) != len(pm.Scouts) {
		t.Errorf("Moves[%s].Scouts length: want %d, got %d", unitId, len(pm.Scouts), len(mm.Scouts))
	}
}

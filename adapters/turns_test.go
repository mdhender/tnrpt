// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package adapters_test

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/mdhender/tnrpt"
	"github.com/mdhender/tnrpt/adapters"
	"github.com/mdhender/tnrpt/parsers/azul"
	"github.com/mdhender/tnrpt/renderer"
)

const testdataPath = "../testdata"
const quiet, verbose, debug = false, true, true

func TestAdaptParserTurnToModel(t *testing.T) {
	r, err := renderer.New(filepath.Join(testdataPath, "0899-12.0987.report.txt"), quiet, verbose, debug)
	if err != nil {
		t.Fatalf("adpt: render: new %v\n", err)
	}

	pt, err := r.Run()
	if err != nil {
		t.Fatalf("adpt: render: run %v\n", err)
	} else if pt == nil {
		t.Fatalf("adpt: render: parsed turns is nil\n")
	}

	at, err := adapters.AdaptParserTurnToModel("<input>", pt)
	if err != nil {
		t.Fatalf("adpt: adapt: parser -> model %v\n", err)
	} else if at == nil {
		t.Fatalf("adpt: render: adapted turns is nil\n")
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
		t.Fatalf("adpt: json: marshal %v\n", err)
	}
	if len(data) == 0 {
		t.Errorf("adpt: json: length: want > 0, got %d\n", len(data))
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

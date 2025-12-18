// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package tnrpt_test

import (
	"testing"

	"github.com/mdhender/tnrpt"
)

func TestBuildUnitLocationLine_HappyPath(t *testing.T) {
	input := []byte("Tribe 0987, , Current Hex = QQ 0203, (Previous Hex = QQ 0101)\n")

	toks := scanAll("", input) // however you get []*Token
	p := tnrpt.ParseCST(toks)

	cstLine := p.ParseUnitLocationLine() // *cst.UnitLocationLineNode

	astLine := tnrpt.BuildUnitLocationLine(cstLine, input)

	if got, want := astLine.UnitType, "Tribe"; got != want {
		t.Fatalf("UnitType = %q, want %q", got, want)
	}
	if got, want := astLine.UnitId.Text, "0987"; got != want {
		t.Fatalf("UnitID.Text = %q, want %q", got, want)
	}
	if astLine.Note != "" {
		t.Fatalf("Note = %q, want empty", astLine.Note)
	}

	if astLine.Current.IsNA {
		t.Fatalf("Current.IsNA = true, want false")
	}
	if got, want := astLine.Current.Grid, "QQ"; got != want {
		t.Fatalf("Current.Grid = %q, want %q", got, want)
	}
	if got, want := astLine.Current.RowCol, "0203"; got != want {
		t.Fatalf("Current.RowCol = %q, want %q", got, want)
	}

	// …and so on for Previous.
}

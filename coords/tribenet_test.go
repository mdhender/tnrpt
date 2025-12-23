// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package coords_test

//import (
//	"github.com/maloquacious/hexg"
//	"testing"
//)
//
//func TestTribeNet_Layout(t *testing.T) {
//	l := hexg.NewTribeNetLayout()
//
//	if l.IsHorizontal() {
//		t.Fatalf("tn3: isHorizontal: got %v, want %v\n", !l.IsHorizontal(), false)
//	} else if !l.IsVertical() {
//		t.Fatalf("tn3: isVertical: got %v, want %v\n", l.IsVertical(), true)
//	} else if l.OffsetType() != hexg.OddQ {
//		t.Fatalf("tn3: offsetType: got %q, want %q\n", l.OffsetType(), hexg.OddQ)
//	}
//}
//
//func TestTribeNet_Neighbor(t *testing.T) {
//	l := hexg.NewTribeNetLayout()
//
//	from := hexg.NewHex(0, 0, 0)
//	for _, move := range []struct {
//		id        int
//		direction int
//		expect    string
//	}{
//		// move one hex and then back
//		{id: 1, direction: hexg.TNNorth, expect: "+0-1+1"},
//		{id: 2, direction: hexg.TNSouth, expect: "+0+0+0"},
//		{id: 3, direction: hexg.TNNorthEast, expect: "+1-1+0"},
//		{id: 4, direction: hexg.TNSouthWest, expect: "+0+0+0"},
//		{id: 5, direction: hexg.TNSouthEast, expect: "+1+0-1"},
//		{id: 6, direction: hexg.TNNorthWest, expect: "+0+0+0"},
//		{id: 7, direction: hexg.TNSouth, expect: "+0+1-1"},
//		{id: 8, direction: hexg.TNNorth, expect: "+0+0+0"},
//		{id: 9, direction: hexg.TNSouthWest, expect: "-1+1+0"},
//		{id: 10, direction: hexg.TNNorthEast, expect: "+0+0+0"},
//		{id: 11, direction: hexg.TNNorthWest, expect: "-1+0+1"},
//		{id: 12, direction: hexg.TNSouthEast, expect: "+0+0+0"},
//	} {
//		to := from.Neighbor(move.direction)
//		if to.ConciseString() != move.expect {
//			t.Fatalf("move: %3d: from %s: move %q: got %q, want %q\n", move.id, from.ConciseString(), l.DirectionToBearing(move.direction), to.ConciseString(), move.expect)
//		}
//		from = to
//	}
//}
//
//func TestTribeNet_ToHex(t *testing.T) {
//	l := hexg.NewTribeNetLayout()
//
//	for _, tc := range []struct {
//		input  string
//		expect string
//	}{
//		{"AA 0101", "+0+0+0"},
//		{"AA 0201", "+1+0-1"},
//		{"AA 0301", "+2-1-1"},
//	} {
//		h, err := l.TribeNetCoordToHex(tc.input)
//		if err != nil {
//			t.Errorf("tn %q: hex: error %v\n", tc.input, err)
//			continue
//		} else if h.ConciseString() != tc.expect {
//			t.Errorf("tn %q: hex: got %q, wanted %q\n", tc.input, h.ConciseString(), tc.expect)
//			continue
//		}
//		tn, err := l.HexToTribeNetCoord(h)
//		if err != nil {
//			t.Errorf("tn %q: hex: error %v\n", tc.input, err)
//			continue
//		}
//		if tn != tc.input {
//			t.Errorf("tn %q: hex: %q: tn got %q, wanted %q\n", tc.input, h.ConciseString(), tn, tc.input)
//			continue
//		}
//	}
//}
//
//func TestTribeNet_RoundTrip(t *testing.T) {
//	//tests := []struct {
//	//	name     string
//	//	input    string
//	//	expected string // expected output after round-trip
//	//	wantErr  bool
//	//}{
//	//	// Valid round-trip test cases
//	//	{"Top-left", "AA 0101", "AA 0101", false},
//	//	{"AA 0202", "AA 0201", "AA 0201", false},
//	//	{"Mid-grid", "BC 0812", "BC 0812", false},
//	//	{"Lower-right", "ZZ 3021", "ZZ 3021", false},
//	//	{"Random valid", "JK 0609", "JK 0609", false},
//	//
//	//	// Valid bounds of grids
//	//	{"Grid AA upper-left", "AA 0101", "AA 0101", false},
//	//	{"Grid AA lower-right", "AA 3021", "AA 3021", false},
//	//	{"Grid AZ upper-left", "AZ 0101", "AZ 0101", false},
//	//	{"Grid AZ lower-right", "AZ 3021", "AZ 3021", false},
//	//	{"Grid ZA upper-left", "ZA 0101", "ZA 0101", false},
//	//	{"Grid ZA lower-right", "ZA 3021", "ZA 3021", false},
//	//	{"Grid ZZ upper-left", "ZZ 0101", "ZZ 0101", false},
//	//	{"Grid ZZ lower-right", "ZZ 3021", "ZZ 3021", false},
//	//
//	//	// invalid row or column
//	//	{"BC 0021", "BC 0021", "", true},
//	//	{"BC 0800", "BC 0800", "", true},
//	//	{"BC 0824", "BC 0824", "", true},
//	//	{"BC 3112", "BC 3112", "", true},
//	//
//	//	// Edge cases (invalid formats)
//	//	{"Too short", "A 0102", "", true},
//	//	{"No space", "AA0102", "", true},
//	//	{"Bad grid row", "1A 0102", "", true},
//	//	{"Bad grid col", "A1 0102", "", true},
//	//	{"Bad subcol", "AA 0001", "", true},
//	//	{"Bad subrow", "AA 0100", "", true},
//	//	{"Subcol too big", "AA 3101", "", true},
//	//	{"Subrow too big", "AA 0122", "", true},
//	//
//	//	// Out of grid bounds (ZZ + 1)
//	//	{"Grid row overflow", "Z[ 0101", "", true},
//	//	{"Grid col overflow", "[Z 0101", "", true},
//	//}
//	//
//	////l := hexg.NewLayoutTribeNet()
//	//for _, tt := range tests {
//	//	t.Run(tt.name, func(t *testing.T) {
//	//		//h, err := l.HexFromTribeNetCoord(tt.input)
//	//		//if err != nil {
//	//		//	if !tt.wantErr {
//	//		//		t.Errorf("%s: HexFromTribeNetCoord(%q) error = %v, wantErr %v", tt.name, tt.input, err, tt.wantErr)
//	//		//	}
//	//		//	return
//	//		//}
//	//		//
//	//		//// Round-trip
//	//		//tn, err := l.HexToTribeNetCoord()
//	//		//if err != nil {
//	//		//	t.Errorf("%s: HexToTribeNetCoord() error = %v", tt.name, err)
//	//		//	return
//	//		//}
//	//		//if tn != tt.expected {
//	//		//	t.Errorf("%s: Round-trip mismatch: got = %q, want = %q", tt.name, tn, tt.expected)
//	//		//}
//	//	})
//	//}
//}

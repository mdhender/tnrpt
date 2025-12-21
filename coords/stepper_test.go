// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package coords_test

import (
	"testing"

	"github.com/maloquacious/hexg"
	"github.com/mdhender/tnrpt/coords"
	"github.com/mdhender/tnrpt/model"
	"github.com/mdhender/tnrpt/steppers"
)

func TestNewStepper(t *testing.T) {
	var nav steppers.Stepper
	nav = coords.NewTribeNetLayout()
	for _, tc := range []struct {
		coord     string
		want      hexg.Hex
		wantCoord string
	}{
		{coord: "", want: hexg.Hex{}, wantCoord: "N/A"},
		{coord: "N/A", want: hexg.Hex{}, wantCoord: "N/A"},
		{coord: "## 0101", want: hexg.NewHex(1, 0), wantCoord: "## 0101"},
		{coord: "## 0201", want: hexg.NewHex(2, 0), wantCoord: "## 0201"},
		{coord: "## 0301", want: hexg.NewHex(3, -1), wantCoord: "## 0301"},
		{coord: "## 0401", want: hexg.NewHex(4, -1), wantCoord: "## 0401"},
		{coord: "## 0501", want: hexg.NewHex(5, -2), wantCoord: "## 0501"},
		{coord: "## 0601", want: hexg.NewHex(6, -2), wantCoord: "## 0601"},
		{coord: "## 0701", want: hexg.NewHex(7, -3), wantCoord: "## 0701"},
		{coord: "AA 0101", want: hexg.NewHex(31, 6), wantCoord: "AA 0101"},
	} {
		got, _ := nav.CoordToHex(model.TNCoord(tc.coord))
		if tc.want.ConciseString() != got.ConciseString() {
			t.Errorf("toHex: %-8s: want %-8s, got %s\n", tc.coord, tc.want.ConciseString(), got.ConciseString())
		} else if coord, _ := nav.HexToCoord(got); string(coord) != tc.wantCoord {
			t.Errorf("toHex: %-8s: want %-8s, got %s\n", tc.coord, tc.wantCoord, coord)
			t.Errorf("toHex: %-8s: hexes %q\n", tc.coord, got.String())
		}
	}

	for _, tc := range []struct {
		dir  string
		from hexg.Hex
		want hexg.Hex
	}{
		{dir: "N", from: hexg.Hex{}, want: hexg.NewHex(0, -1)},
		{dir: "NE", from: hexg.Hex{}, want: hexg.NewHex(+1, -1)},
		{dir: "SE", from: hexg.Hex{}, want: hexg.NewHex(+1, 0)},
		{dir: "S", from: hexg.Hex{}, want: hexg.NewHex(0, +1)},
		{dir: "SW", from: hexg.Hex{}, want: hexg.NewHex(-1, +1)},
		{dir: "NW", from: hexg.Hex{}, want: hexg.NewHex(-1, 0)},
	} {
		from := tc.from
		got, _ := nav.StepForwardHex(from, tc.dir)
		if tc.want.ConciseString() != got.ConciseString() {
			t.Errorf("step: forward: from %-8s: %-2s: want %-8s, got %s\n", from.ConciseString(), tc.dir, tc.want.ConciseString(), got.ConciseString())
		}
	}

	for _, tc := range []struct {
		dir  string
		from hexg.Hex
		want hexg.Hex
	}{
		{dir: "N", from: hexg.NewHex(0, -1), want: hexg.Hex{}},
		{dir: "NE", from: hexg.NewHex(+1, -1), want: hexg.Hex{}},
		{dir: "SE", from: hexg.NewHex(+1, 0), want: hexg.Hex{}},
		{dir: "S", from: hexg.NewHex(0, +1), want: hexg.Hex{}},
		{dir: "SW", from: hexg.NewHex(-1, +1), want: hexg.Hex{}},
		{dir: "NW", from: hexg.NewHex(-1, 0), want: hexg.Hex{}},
	} {
		from := tc.from
		got, _ := nav.StepBackwardHex(from, tc.dir)
		if tc.want.ConciseString() != got.ConciseString() {
			t.Errorf("step: back: from %-8s: %-2s: want %-8s, got %s\n", from.ConciseString(), tc.dir, tc.want.ConciseString(), got.ConciseString())
		}
	}

	for _, tc := range []struct {
		from string
		dir  string
		want string
	}{
		{from: "AA 1712", dir: "N", want: "AA 1711"},
		{from: "AA 1712", dir: "NE", want: "AA 1811"},
		{from: "AA 1712", dir: "SE", want: "AA 1812"},
		{from: "AA 1712", dir: "S", want: "AA 1713"},
		{from: "AA 1712", dir: "SW", want: "AA 1612"},
		{from: "AA 1712", dir: "NW", want: "AA 1611"},
		{from: "CS 3021", dir: "N", want: "CS 3020"},
		{from: "CS 3021", dir: "NE", want: "CT 0121"},
		{from: "CS 3021", dir: "SE", want: "DT 0101"},
		{from: "CS 3021", dir: "S", want: "DS 3001"},
		{from: "CS 3021", dir: "SW", want: "DS 2901"},
		{from: "CS 3021", dir: "NW", want: "CS 2921"},
		{from: "CT 0121", dir: "N", want: "CT 0120"},
		{from: "CT 0121", dir: "NE", want: "CT 0220"},
		{from: "CT 0121", dir: "SE", want: "CT 0221"},
		{from: "CT 0121", dir: "S", want: "DT 0101"},
		{from: "CT 0121", dir: "SW", want: "CS 3021"},
		{from: "CT 0121", dir: "NW", want: "CS 3020"},
		{from: "DS 3001", dir: "N", want: "CS 3021"},
		{from: "DS 3001", dir: "NE", want: "DT 0101"},
		{from: "DS 3001", dir: "SE", want: "DT 0102"},
		{from: "DS 3001", dir: "S", want: "DS 3002"},
		{from: "DS 3001", dir: "SW", want: "DS 2902"},
		{from: "DS 3001", dir: "NW", want: "DS 2901"},
		{from: "DT 0101", dir: "N", want: "CT 0121"},
		{from: "DT 0101", dir: "NE", want: "CT 0221"},
		{from: "DT 0101", dir: "SE", want: "DT 0201"},
		{from: "DT 0101", dir: "S", want: "DT 0102"},
		{from: "DT 0101", dir: "SW", want: "DS 3001"},
		{from: "DT 0101", dir: "NW", want: "CS 3021"},
	} {
		from := model.TNCoord(tc.from)
		fromHex, err := nav.CoordToHex(from)
		if err != nil {
			t.Fatalf("step: forward: from %-8s: %-2s: %v\n", from, tc.dir, err)
		}
		gotHex, ok := nav.StepForwardHex(fromHex, tc.dir)
		if !ok {
			t.Fatalf("step: forward: from %-8s: %-2s: !ok\n", from, tc.dir)
		}
		got, err := nav.HexToCoord(gotHex)
		if err != nil {
			t.Fatalf("step: forward: from %-8s: %-2s: %v\n", from, tc.dir, err)
		}
		if tc.want != string(got) {
			t.Errorf("step: forward: from %-8s: %-2s: want %-8s, got %s\n", from, tc.dir, tc.want, got)
		}
	}

	for _, tc := range []struct {
		from string
		dir  string
		want string
	}{
		{from: "AA 1711", dir: "N", want: "AA 1712"},
		{from: "AA 1811", dir: "NE", want: "AA 1712"},
		{from: "AA 1812", dir: "SE", want: "AA 1712"},
		{from: "AA 1713", dir: "S", want: "AA 1712"},
		{from: "AA 1612", dir: "SW", want: "AA 1712"},
		{from: "AA 1611", dir: "NW", want: "AA 1712"},
		{from: "CS 3020", dir: "N", want: "CS 3021"},
		{from: "CT 0121", dir: "NE", want: "CS 3021"},
		{from: "DT 0101", dir: "SE", want: "CS 3021"},
		{from: "DS 3001", dir: "S", want: "CS 3021"},
		{from: "DS 2901", dir: "SW", want: "CS 3021"},
		{from: "CS 2921", dir: "NW", want: "CS 3021"},
		{from: "CT 0120", dir: "N", want: "CT 0121"},
		{from: "CT 0220", dir: "NE", want: "CT 0121"},
		{from: "CT 0221", dir: "SE", want: "CT 0121"},
		{from: "DT 0101", dir: "S", want: "CT 0121"},
		{from: "CS 3021", dir: "SW", want: "CT 0121"},
		{from: "CS 3020", dir: "NW", want: "CT 0121"},
		{from: "CS 3021", dir: "N", want: "DS 3001"},
		{from: "DT 0101", dir: "NE", want: "DS 3001"},
		{from: "DT 0102", dir: "SE", want: "DS 3001"},
		{from: "DS 3002", dir: "S", want: "DS 3001"},
		{from: "DS 2902", dir: "SW", want: "DS 3001"},
		{from: "DS 2901", dir: "NW", want: "DS 3001"},
		{from: "CT 0121", dir: "N", want: "DT 0101"},
		{from: "CT 0221", dir: "NE", want: "DT 0101"},
		{from: "DT 0201", dir: "SE", want: "DT 0101"},
		{from: "DT 0102", dir: "S", want: "DT 0101"},
		{from: "DS 3001", dir: "SW", want: "DT 0101"},
		{from: "CS 3021", dir: "NW", want: "DT 0101"},
	} {
		from := model.TNCoord(tc.from)
		fromHex, err := nav.CoordToHex(from)
		if err != nil {
			t.Fatalf("step: backward: from %-8s: %-2s: %v\n", from, tc.dir, err)
		}
		gotHex, ok := nav.StepBackwardHex(fromHex, tc.dir)
		if !ok {
			t.Fatalf("step: backward: from %-8s: %-2s: !ok\n", from, tc.dir)
		}
		got, err := nav.HexToCoord(gotHex)
		if err != nil {
			t.Fatalf("step: backward: from %-8s: %-2s: %v\n", from, tc.dir, err)
		}
		if tc.want != string(got) {
			t.Errorf("step: backward: from %-8s: %-2s: want %-8s, got %s\n", from, tc.dir, tc.want, got)
		}
	}
}

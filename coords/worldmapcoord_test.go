// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package coords_test

import (
	"encoding/json"
	"testing"

	"github.com/mdhender/tnrpt/coords"
	"github.com/mdhender/tnrpt/direction"
)

func TestNewWorldMapCoord(t *testing.T) {
	tests := []struct {
		id         int
		input      string
		wantString string
		wantErr    bool
	}{
		{1001, "AA 0101", "AA 0101", false},
		{1002, "AZ 3001", "AZ 3001", false},
		{1003, "ZA 0121", "ZA 0121", false},
		{1004, "ZZ 3021", "ZZ 3021", false},
		{1005, "JK 1508", "JK 1508", false},
		{1006, "aa 0101", "AA 0101", false}, // lowercase converted to uppercase
		{1007, "jk 1508", "JK 1508", false},
		// N/A special case
		{2001, "N/A", "AA 0101", false}, // N/A has zero cube coords, which converts to AA 0101
		// ## obscured coordinates (mapped to QQ)
		{2101, "## 0101", "QQ 0101", false},
		{2102, "## 3021", "QQ 3021", false},
		{2103, "## 1508", "QQ 1508", false},
		// Invalid inputs
		{3001, "AA0101", "", true},   // missing space
		{3002, "AA-0101", "", true},  // wrong separator
		{3003, "A 0101", "", true},   // too short
		{3004, "AAA 0101", "", true}, // wrong length
		{3005, "AA 0001", "", true},  // column 0 invalid
		{3006, "AA 3101", "", true},  // column 31 invalid
		{3007, "AA 0100", "", true},  // row 0 invalid
		{3008, "AA 0122", "", true},  // row 22 invalid
	}

	for _, tc := range tests {
		wmc, err := coords.NewWorldMapCoord(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Errorf("%d: %q: expected error, got nil", tc.id, tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("%d: %q: unexpected error: %v", tc.id, tc.input, err)
			continue
		}
		if got := wmc.String(); got != tc.wantString {
			t.Errorf("%d: %q: String() = %q, want %q", tc.id, tc.input, got, tc.wantString)
		}
	}
}

func TestWorldMapCoord_IsNA(t *testing.T) {
	tests := []struct {
		input  string
		wantNA bool
	}{
		{"N/A", true},
		{"AA 0101", false},
		{"## 1508", false},
	}

	for _, tc := range tests {
		wmc, err := coords.NewWorldMapCoord(tc.input)
		if err != nil {
			t.Fatalf("%q: unexpected error: %v", tc.input, err)
		}
		if got := wmc.IsNA(); got != tc.wantNA {
			t.Errorf("%q: IsNA() = %v, want %v", tc.input, got, tc.wantNA)
		}
	}
}

func TestWorldMapCoord_Equals(t *testing.T) {
	tests := []struct {
		a, b      string
		wantEqual bool
	}{
		{"AA 0101", "AA 0101", true},
		{"aa 0101", "AA 0101", true}, // case insensitive
		{"AA 0101", "AA 0102", false},
		{"N/A", "N/A", true},
		{"## 1508", "## 1508", true},
	}

	for _, tc := range tests {
		a, _ := coords.NewWorldMapCoord(tc.a)
		b, _ := coords.NewWorldMapCoord(tc.b)
		if got := a.Equals(b); got != tc.wantEqual {
			t.Errorf("Equals(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.wantEqual)
		}
	}
}

func TestWorldMapCoord_Move(t *testing.T) {
	tests := []struct {
		id         int
		start      string
		directions []direction.Direction_e
		wantString string
	}{
		// Single moves from JK 1508 (column 15 is odd in 0-based, so odd-q rules apply)
		{1001, "JK 1508", []direction.Direction_e{direction.North}, "JK 1507"},
		{1002, "JK 1508", []direction.Direction_e{direction.South}, "JK 1509"},
		{1003, "JK 1508", []direction.Direction_e{direction.NorthEast}, "JK 1607"},
		{1004, "JK 1508", []direction.Direction_e{direction.SouthEast}, "JK 1608"},
		{1005, "JK 1508", []direction.Direction_e{direction.NorthWest}, "JK 1407"},
		{1006, "JK 1508", []direction.Direction_e{direction.SouthWest}, "JK 1408"},
		// Multiple moves
		{2001, "JK 1508", []direction.Direction_e{direction.North, direction.North}, "JK 1506"},
		{2002, "JK 1508", []direction.Direction_e{direction.NorthEast, direction.SouthEast}, "JK 1708"},
		// Grid boundary crossing (out of bounds shown with < and >)
		{3001, "AA 0101", []direction.Direction_e{direction.NorthWest}, "AA <<<<"},
		{3002, "AA 3001", []direction.Direction_e{direction.NorthEast}, "AB 0101"},
	}

	for _, tc := range tests {
		start, err := coords.NewWorldMapCoord(tc.start)
		if err != nil {
			t.Fatalf("%d: %q: unexpected error: %v", tc.id, tc.start, err)
		}
		got := start.Move(tc.directions...)
		if got.String() != tc.wantString {
			t.Errorf("%d: Move from %q: got %q, want %q", tc.id, tc.start, got.String(), tc.wantString)
		}
	}
}

func TestWorldMapCoord_ID(t *testing.T) {
	tests := []struct {
		input  string
		wantID string
	}{
		{"AA 0101", "AA 0101"},
		{"JK 1508", "JK 1508"},
		{"ZZ 3021", "ZZ 3021"},
		{"N/A", "N/A"},         // N/A should return N/A, not AA 0101
		{"## 1508", "## 1508"}, // obscured should return original ##, not QQ
	}

	for _, tc := range tests {
		wmc, err := coords.NewWorldMapCoord(tc.input)
		if err != nil {
			t.Fatalf("%q: unexpected error: %v", tc.input, err)
		}
		if got := wmc.ID(); got != tc.wantID {
			t.Errorf("%q: ID() = %q, want %q", tc.input, got, tc.wantID)
		}
	}

	// Test zero-value WorldMapCoord
	var zero coords.WorldMapCoord
	if got := zero.ID(); got != "N/A" {
		t.Errorf("zero-value: ID() = %q, want %q", got, "N/A")
	}
}

func TestWorldMapCoord_JSON(t *testing.T) {
	tests := []struct {
		input    string
		wantJSON string
	}{
		{"AA 0101", `"AA 0101"`},
		{"JK 1508", `"JK 1508"`},
		{"ZZ 3021", `"ZZ 3021"`},
		{"## 1508", `"## 1508"`}, // obscured should marshal with ##
	}

	for _, tc := range tests {
		wmc, err := coords.NewWorldMapCoord(tc.input)
		if err != nil {
			t.Fatalf("%q: unexpected error: %v", tc.input, err)
		}

		// Test Marshal
		data, err := json.Marshal(wmc)
		if err != nil {
			t.Errorf("%q: Marshal error: %v", tc.input, err)
			continue
		}
		if string(data) != tc.wantJSON {
			t.Errorf("%q: Marshal = %s, want %s", tc.input, data, tc.wantJSON)
		}

		// Test Unmarshal
		var unmarshaled coords.WorldMapCoord
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Errorf("%q: Unmarshal error: %v", tc.input, err)
			continue
		}
		if !unmarshaled.Equals(wmc) {
			t.Errorf("%q: Unmarshal roundtrip failed: got %q", tc.input, unmarshaled.String())
		}
	}

	// Test zero-value WorldMapCoord marshals as null (IsZero handles omitzero)
	var zero coords.WorldMapCoord
	data, err := json.Marshal(zero)
	if err != nil {
		t.Fatalf("zero-value: Marshal error: %v", err)
	}
	if string(data) != "null" {
		t.Errorf("zero-value: Marshal = %s, want null", data)
	}

	// Test N/A WorldMapCoord marshals as "N/A"
	na, _ := coords.NewWorldMapCoord("N/A")
	data, err = json.Marshal(na)
	if err != nil {
		t.Fatalf("N/A: Marshal error: %v", err)
	}
	if string(data) != `"N/A"` {
		t.Errorf("N/A: Marshal = %s, want %q", data, "N/A")
	}
}

func TestWorldMapCoord_JSON_Omitzero(t *testing.T) {
	type wrapper struct {
		Location coords.WorldMapCoord `json:"location,omitzero"`
	}

	// Valid coordinate should appear in JSON
	w := wrapper{}
	w.Location, _ = coords.NewWorldMapCoord("JK 1508")
	data, _ := json.Marshal(w)
	if string(data) != `{"location":"JK 1508"}` {
		t.Errorf("valid: got %s, want %s", data, `{"location":"JK 1508"}`)
	}

	// Zero-value should be omitted (IsZero returns true)
	w = wrapper{}
	data, _ = json.Marshal(w)
	if string(data) != `{}` {
		t.Errorf("zero-value: got %s, want {}", data)
	}
	if !w.Location.IsZero() {
		t.Errorf("zero-value: IsZero() = false, want true")
	}

	// N/A should appear in JSON (IsZero returns false)
	w = wrapper{}
	w.Location, _ = coords.NewWorldMapCoord("N/A")
	data, _ = json.Marshal(w)
	if string(data) != `{"location":"N/A"}` {
		t.Errorf("N/A: got %s, want %s", data, `{"location":"N/A"}`)
	}
	if w.Location.IsZero() {
		t.Errorf("N/A: IsZero() = true, want false")
	}

	// Obscured should appear in JSON
	w = wrapper{}
	w.Location, _ = coords.NewWorldMapCoord("## 1508")
	data, _ = json.Marshal(w)
	if string(data) != `{"location":"## 1508"}` {
		t.Errorf("obscured: got %s, want %s", data, `{"location":"## 1508"}`)
	}
}

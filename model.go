// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package tnrpt

import (
	"github.com/maloquacious/hexg"
	"github.com/mdhender/tnrpt/compass"
	"github.com/mdhender/tnrpt/direction"
	"github.com/mdhender/tnrpt/edges"
	"github.com/mdhender/tnrpt/items"
	"github.com/mdhender/tnrpt/resources"
	"github.com/mdhender/tnrpt/results"
	"github.com/mdhender/tnrpt/terrain"
	"github.com/mdhender/tnrpt/unit_movement"
)

// Turns_t uses the turn's id as the key.
type Turns_t map[string]*Turn_t

// Turn_t represents the data from a single file, which must contain
// a single turn. The turn is identified by year and month.
type Turn_t struct {
	// Source is the name of the input file
	Source string `json:"source,omitempty"`

	Id     string `json:"turn-id,omitempty"`
	Year   int    `json:"year,omitempty"`
	Month  int    `json:"month,omitempty"`
	ClanNo int    `json:"clanNo,omitempty"`

	// UnitMoves holds the units that moved in this turn
	UnitMoves map[UnitId_t]*Moves_t `json:"unit-moves,omitempty"`

	// SpecialNames holds the names of the hexes that are special.
	// It's a hack to get around the fact that the parser doesn't know about the hexes.
	// They are added to the map when parsing and are forced to lower case.
	SpecialNames map[string]*Special_t `json:"special-names,omitempty"`
}

type UnitId_t string

// Moves_t represents the results for a unit that moves and reports in a turn.
// There will be one instance of this struct for each turn the unit moves in.
type Moves_t struct {
	UnitId UnitId_t `json:"unit-id,omitempty"`

	// PreviousHex is the hex the unit starts the move in.
	// This could be "N/A" if the unit was created this turn.
	// In that case, we will populate it when we know where the unit started.
	PreviousHex string `json:"previous-hex,omitempty"`

	// CurrentHex is the hex is unit ends the movement in.
	// This should always be set from the turn report.
	// It might be the same as the PreviousHex if the unit stays in place or fails to move.
	CurrentHex string `json:"current-hex,omitempty"`

	// all the moves made this turn
	Moves   []*Move_t `json:"moves,omitempty"`
	Follows UnitId_t  `json:"follows,omitempty"`
	GoesTo  string    `json:"goes-to,omitempty"`

	// Scries are optional; these are the results
	Scries []*Scry_t `json:"scries,omitempty"`

	// Scouts are optional and move at the end of the turn
	Scouts []*Scout_t `json:"scouts,omitempty"`
}

// Move_t represents a single move by a unit.
// The move can be follows, goes to, stay in place, or attempt to advance a direction.
// The move will fail, succeed, or the unit can simply vanish without a trace.
type Move_t struct {
	LineNo int    `json:"line-no,omitempty"`
	StepNo int    `json:"step-no,omitempty"`
	Line   string `json:"line,omitempty"`

	// the types of movement that a unit can make.
	Advance   direction.Direction_e `json:"advance,omitempty"`
	Follows   UnitId_t              `json:"follows,omitempty"`
	GoesToHex string                `json:"goes-to-hex,omitempty"`
	Still     bool                  `json:"still,omitempty"`

	// Result should be failed, succeeded, or vanished
	Result results.Result_e `json:"result,omitempty"`

	Report *Report_t `json:"report,omitempty"`
}

type Scry_t struct {
	Text string `json:"text,omitempty"`

	// OriginHex is the location that was scried.
	OriginHex string `json:"origin-hex,omitempty"`

	Type unit_movement.Type_e `json:"type,omitempty"`

	Moves  []*Move_t `json:"moves,omitempty"`
	Scouts *Scout_t  `json:"scouts,omitempty"`
}

// Scout_t represents a scout sent out by a unit.
type Scout_t struct {
	LineNo int    `json:"line-no,omitempty"`
	Line   string `json:"line,omitempty"`

	No int `json:"no,omitempty"`

	Moves []*Move_t `json:"moves,omitempty"`
}

// Report_t represents the observations made by a unit.
// All reports are relative to the hex that the unit is reporting from.
type Report_t struct {
	// permanent items in this hex
	Terrain terrain.Terrain_e `json:"terrain,omitempty"`
	Borders []*Border_t       `json:"borders,omitempty"`

	// transient items in this hex
	Encounters  []*Encounter_t         `json:"encounters,omitempty"`
	Items       []*FoundItem_t         `json:"items,omitempty"`
	Resources   []resources.Resource_e `json:"resources,omitempty"`
	Settlements []*Settlement_t        `json:"settlements,omitempty"`
	FarHorizons []*FarHorizon_t        `json:"far-horizons,omitempty"`
}

// Border_t represents details about the hex border.
type Border_t struct {
	Direction direction.Direction_e `json:"direction,omitempty"`
	// Edge is set if there is an edge feature like a river or pass
	Edge edges.Edge_e `json:"edge,omitempty"`
	// Terrain is set if the neighbor is observable from this hex
	Terrain terrain.Terrain_e `json:"terrain,omitempty"`
}

type Encounter_t struct {
	UnitId   UnitId_t `json:"unit-id,omitempty"`
	Friendly bool     `json:"friendly,omitempty"`
}

// FoundItem_t represents items discovered by Scouts as they pass through a hex.
type FoundItem_t struct {
	Quantity int          `json:"quantity,omitempty"`
	Item     items.Item_e `json:"item,omitempty"`
}

// Settlement_t is a settlement that the unit sees in the current hex.
type Settlement_t struct {
	Name string `json:"name,omitempty"`
}

type FarHorizon_t struct {
	Point   compass.Point_e   `json:"point,omitempty"`
	Terrain terrain.Terrain_e `json:"terrain,omitempty"`
}

type Special_t struct {
	Id   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// Hex_t uses cube coordinates.
// 0,0,0 is not a valid TribeNet hex.
type Hex_t struct {
	coords hexg.Hex
	id     string
}

func (h Hex_t) Hash() string {
	if h.id == "" {
		return "N/A"
	}
	return "## 0101"
}

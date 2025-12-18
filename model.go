// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package tnrpt

import (
	"github.com/mdhender/tnrpt/compass"
	"github.com/mdhender/tnrpt/coords"
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

// Turn represents a single turn identified by year and month.
type Turn_t struct {
	Id    string
	Year  int
	Month int

	// UnitMoves holds the units that moved in this turn
	UnitMoves            map[UnitId_t]*Moves_t
	SortedMoves          []*Moves_t
	MovesSortedByElement []*Moves_t

	// SpecialNames holds the names of the hexes that are special.
	// It's a hack to get around the fact that the parser doesn't know about the hexes.
	// They are added to the map when parsing and are forced to lower case.
	SpecialNames map[string]*Special_t
}

type UnitId_t string

// Moves_t represents the results for a unit that moves and reports in a turn.
// There will be one instance of this struct for each turn the unit moves in.
type Moves_t struct {
	TurnId string
	UnitId UnitId_t // unit that is moving

	// all the moves made this turn
	Moves   []*Move_t
	Follows UnitId_t
	GoesTo  string

	// all the scry results for this turn
	Scries []*Scry_t

	// Scouts are optional and move at the end of the turn
	Scouts []*Scout_t

	// FromHex is the hex the unit starts the move in.
	// This could be "N/A" if the unit was created this turn.
	// In that case, we will populate it when we know where the unit started.
	FromHex string

	// ToHex is the hex is unit ends the movement in.
	// This should always be set from the turn report.
	// It might be the same as the FromHex if the unit stays in place or fails to move.
	ToHex string

	Coordinates coords.WorldMapCoord // coordinates of the tile the unit ends the move in
	Location    coords.Map           // Location is the tile the unit ends the move in
}

// Move_t represents a single move by a unit.
// The move can be follows, goes to, stay in place, or attempt to advance a direction.
// The move will fail, succeed, or the unit can simply vanish without a trace.
type Move_t struct {
	UnitId UnitId_t // unit that is moving

	// the types of movement that a unit can make.
	Advance direction.Direction_e // set only if the unit is advancing
	Follows UnitId_t              // id of the unit being followed
	GoesTo  string                // hex teleporting to
	Still   bool                  // true if the unit is not moving (garrison) or a status entry

	// Result should be failed, succeeded, or vanished
	Result results.Result_e

	Report *Report_t // all observations made by the unit at the end of this move

	LineNo int
	StepNo int
	Line   []byte

	TurnId     string
	CurrentHex string

	// warning: we're changing from "location" to "coordinates" for tiles.
	// this is a breaking change so we're introducing new fields, FromCoordinates and ToCoordinates, to help.
	FromCoordinates coords.WorldMapCoord // the tile the unit starts the move in
	ToCoordinates   coords.WorldMapCoord // the tile the unit ends the move in

	// Location is the tile the unit ends the move in
	Location coords.Map // soon to be replaced with FromCoordinates and ToCoordinates
}

type Scry_t struct {
	UnitId      UnitId_t // the unit scrying
	Type        unit_movement.Type_e
	Origin      string // the hex the scry originates in
	Coordinates coords.WorldMapCoord
	Location    coords.Map
	Text        []byte // the results of scrying in that hex
	Moves       []*Move_t
	Scouts      *Scout_t
}

// Scout_t represents a scout sent out by a unit.
type Scout_t struct {
	No     int // usually from 1..8
	TurnId string
	Moves  []*Move_t

	LineNo int
	Line   []byte
}

// Report_t represents the observations made by a unit.
// All reports are relative to the hex that the unit is reporting from.
type Report_t struct {
	UnitId UnitId_t // id of the unit that made the report

	Location      coords.Map
	TurnId        string // turn the report was received
	ScoutedTurnId string // turn the report was received from a scouting party

	// permanent items in this hex
	Terrain terrain.Terrain_e
	Borders []*Border_t

	// transient items in this hex
	Encounters  []*Encounter_t // other units in the hex
	Items       []*FoundItem_t
	Resources   []resources.Resource_e
	Settlements []*Settlement_t
	FarHorizons []*FarHorizon_t

	WasVisited bool // set to true if the location was visited by any unit
	WasScouted bool // set to true if the location was visited by a scouting party or a unit ended the turn here
}

// Border_t represents details about the hex border.
type Border_t struct {
	Direction direction.Direction_e
	// Edge is set if there is an edge feature like a river or pass
	Edge edges.Edge_e
	// Terrain is set if the neighbor is observable from this hex
	Terrain terrain.Terrain_e
}

type Encounter_t struct {
	TurnId   string // turn the encounter happened
	UnitId   UnitId_t
	Friendly bool // true if the encounter was friendly
}

// FoundItem_t represents items discovered by Scouts as they pass through a hex.
type FoundItem_t struct {
	Quantity int
	Item     items.Item_e
}

// Settlement_t is a settlement that the unit sees in the current hex.
type Settlement_t struct {
	TurnId string // turn the settlement was observed
	Name   string
}

type FarHorizon_t struct {
	Point   compass.Point_e
	Terrain terrain.Terrain_e
}

type Special_t struct {
	TurnId string // turn the special hex was observed
	Id     string // id of the special hex, full name converted to lower case
	Name   string // short name of the special hex (id if name is empty)
}

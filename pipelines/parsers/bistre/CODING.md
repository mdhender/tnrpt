# Bistre Parser Coding Guide

> **Note**: This guide was written at commit `037827b` (2025-12-23). The bistre parser code itself is actively maintained, but this documentation is **not aggressively updated**. If you notice discrepancies between this guide and the actual code, please check the code as the source of truth.

The bistre parser is TNRpt's core turn report parser. It reads TribeNet turn report text and produces a structured `Turn_t` object representing all units, movements, and observations from that turn.

## Overview

**Purpose**: Parse raw turn report text into structured Go types  
**Input**: Turn report text (typically extracted from DOCX files)  
**Output**: `Turn_t` containing units, moves, scouts, scries, and observations  
**Status**: Core parser (production-ready); replaces legacy azul parser  

## Architecture

The parser operates in three stages:

```
Raw Text → Line-based Parsing → Movement Analysis → Node Parsing → Structured Output
```

### File Organization

- **parser.go**: Main entry point (`ParseInput`), line-by-line section parsing, movement result aggregation
- **types.go**: Domain types returned by the parser (Turn_t, Moves_t, Move_t, Report_t, etc.)
- **nodes.go**: Node tree operations for parsing hex report components
- **grammar.peg**: PEG grammar for parsing individual field values (used by pigeon)
- **grammar.go**: Auto-generated parser from grammar.peg (do NOT edit)
- **generate.go**: Build directive for pigeon code generation

## Core Types

### Turn_t
The root result object representing a single game turn.

```go
type Turn_t struct {
	Id             string                 // YYYY-MM format
	Year, Month    int
	UnitMoves      map[UnitId_t]*Moves_t  // All units that moved
	SortedMoves    []*Moves_t              // Topologically sorted (follows/goto deps)
	SpecialNames   map[string]*Special_t   // Named hexes from ">>" markers
}
```

**Important methods**:
- `TopoSortMoves()`: Sort moves respecting "Follows" and "Goes To" dependencies
- `SortMovesByElement()`: Sort moves by unit ID

### Moves_t
Represents all the movement results for a single unit in a turn.

```go
type Moves_t struct {
	TurnId      string
	UnitId      UnitId_t
	Moves       []*Move_t       // Movement attempts
	Follows     UnitId_t        // If following another unit
	GoesTo      string          // If teleporting to a hex
	Scries      []*Scry_t       // Scrying results
	Scouts      []*Scout_t      // Scout movements (1-8 scouts)
	PreviousHex string          // Starting hex
	CurrentHex  string          // Ending hex
}
```

### Move_t
A single movement step (may be succeeded, failed, or the unit vanished).

```go
type Move_t struct {
	UnitId      UnitId_t
	Advance     direction.Direction_e   // Attempted direction
	Follows     UnitId_t                 // If following
	GoesTo      string                   // If teleporting
	Still       bool                     // True if not moving
	Result      results.Result_e         // Failed/Succeeded/Vanished
	Report      *Report_t                // Observations from this move
	LineNo      int
	StepNo      int                      // Position within movement line
	Line        []byte                   // Raw text
}
```

### Report_t
All observations made by a unit during a move.

```go
type Report_t struct {
	UnitId        UnitId_t
	TurnId        string
	ScoutedTurnId string  // Set only for scouted data
	Terrain       terrain.Terrain_e
	Borders       []*Border_t        // Neighboring hexes
	Encounters    []*Encounter_t     // Other units found
	Items         []*FoundItem_t     // Items discovered
	Resources     []resources.Resource_e
	Settlements   []*Settlement_t
	FarHorizons   []*FarHorizon_t    // Crow's nest observations
	WasVisited    bool
	WasScouted    bool
}
```

**Key merge methods** (all are idempotent):
- `MergeBorders(b *Border_t) bool`
- `MergeEncounters(e *Encounter_t) bool`
- `MergeResources(rs resources.Resource_e) bool`
- `MergeSettlements(s *Settlement_t) bool`
- `mergeFarHorizons(fh FarHorizon_t) bool`

## Entry Point: ParseInput

```go
func ParseInput(
    fid, tid string,                    // File ID, Turn ID (e.g., "0903-04")
    input []byte,
    acceptLoneDash bool,                // Allow orphaned dashes (rare)
    debugParser, debugSections,
    debugSteps, debugNodes,
    debugFleetMovement bool,
    experimentalUnitSplit,              // Try splitting units from end of strings
    experimentalScoutStill bool,        // Treat "Scout Still?" as "Scout Still,,"
    cfg ParseConfig,
) (*Turn_t, error)
```

### Typical usage:

```go
turn, err := bistre.ParseInput(
    "report.docx",
    "0903-04",
    reportText,
    false,     // acceptLoneDash
    false,     // debugParser
    false,     // debugSections
    false,     // debugSteps
    false,     // debugNodes
    false,     // debugFleetMovement
    false,     // experimentalUnitSplit
    false,     // experimentalScoutStill
    bistre.ParseConfig{},
)
```

## Parsing Pipeline

### Stage 1: Line-by-Line Parsing (parser.go, ~Line 52-310)

The parser scans input line-by-line and recognizes section headers:

- **Courier, Element, Fleet, Garrison, Tribe**: Unit headers with location data
- **Current Turn**: Turn number and date info
- **Tribe Movement**: Main movement line
- **Fleet Movement**: Nautical unit movement
- **Scout**: Individual scout movements (8 max)
- **Scry**: Remote location observations
- **Status**: Unit status observations
- **Tribe Follows/Goes To**: Special movement commands

**Key regex patterns**:
```go
rxCourierSection  = regexp.MustCompile(`^Courier \d{4}c\d, `)
rxElementSection  = regexp.MustCompile(`^Element \d{4}e\d, `)
rxFleetSection    = regexp.MustCompile(`^Fleet \d{4}f\d, `)
rxGarrisonSection = regexp.MustCompile(`^Garrison \d{4}g\d, `)
rxTribeSection    = regexp.MustCompile(`^Tribe \d{4}, `)
```

**Important state machine**:
- When a section is found, extract location and create new `Moves_t`
- Subsequent lines belong to that unit until the next section
- Lines outside sections before a location are logged as errors

### Stage 2: Movement Line Parsing

#### parseMovementLine (lines 632-858)

This is the heavy lifter. It:

1. **Splits into steps** using `\` as separator
2. **Splits steps into rings**: current hex, inner ring (6 adjacent), outer ring (12 one-hex-away)
3. **Parses each ring**:
   - Current hex: Movement direction+terrain and observations
   - Inner ring: Neighboring terrain codes (e.g., "ALPS N, FOREST NE")
   - Outer ring: Crow's nest sightings (e.g., "Sight Water - NE/NE")

#### parseMove (lines 871-1084)

Parses a single step within a movement line. It:

1. Converts the raw text to a node tree via `hexReportToNodes`
2. Splits nodes into individual parse-able elements
3. Processes each element as one of these types:
   - `DirectionTerrain_t`: Successful move (e.g., "N-ARID")
   - `BlockedByEdge_t`: Failed due to edge (e.g., "RIVER")
   - `Exhausted_t`: Failed due to MP exhaustion
   - `ProhibitedFrom_t`: Can't enter that terrain
   - `Edge_t`, `Neighbor_t`: Observations
   - `FoundUnit_t`, `Encounter_t`: Unit encounters
   - `FoundItem_t`: Items found (scout only)
   - `Settlement_t`: Named settlements

### Stage 3: Node Parsing (nodes.go)

#### hexReportToNodes (lines 16-172)

Converts comma-separated components into a linked list of nodes:

```
"N-PRAIRIE, ARID W, Sight Water - S/S" 
    ↓
node("N-PRAIRIE") → node("ARID W") → node("Sight Water - S/S")
```

**Node type helpers**:
- `isDirection()`: Checks for N, NE, SE, S, SW, NW (case-insensitive)
- `isAlpsEdge()`, `isRiverEdge()`, etc.: Edge feature detection
- `isFindQuantityItem()`: Item parsing
- `isUnitId()`: Unit ID matching

**Node repair**: After splitting, some components are spliced back together if they contain multi-directional edges:
```go
if tmp.isRiverEdge() {
    for tmp.next.isDirection() {
        tmp.addText(tmp.next)  // Merge "RIVER N" and "SE" → "RIVER N SE"
        tmp.next = tmp.next.next
    }
}
```

#### nodesToSteps (lines 174-187)

Converts node list to string slices for further processing.

## Grammar (grammar.peg)

The PEG grammar handles parsing of specific field values:

### Key Rules

- **Step**: Matches movement results like "N-PRAIRIE", "Cannot Move Wagons...", "Found 5 Wheat", etc.
- **FleetMovement**: Parses "CALM NE Fleet Movement: Move ..."
- **Location**: Parses unit location lines
- **ScoutMovement**: Parses "Scout 1:Scout ..."
- **ScryLine**: Parses "0903 Scry: 1234, ..."
- **StatusLine**: Parses "0903 Status: ..."
- **TurnInfo**: Parses "Current Turn 0903-04, ..."

### Terminals

- **UNIT_ID**: 4 digits optionally followed by [cefg][1-9]
- **DIRECTION**: N, NE, SE, S, SW, NW (case-insensitive)
- **TERRAIN_CODE**: Short codes (e.g., "AR" for arid flat)
- **COORDS**: Grid coordinates (e.g., "AA1234") or "N/A"
- **RESOURCE**: Coal, Gold, Iron Ore, etc.
- **ITEM**: 140+ item types (adze, arrows, boats, etc.)

## Common Workflows

### Add a New Movement Result Type

If the parser encounters an unknown phrase like "Cannot Do X", add it to the Step rule in grammar.peg:

```peg
Step <- 
    ... existing alternatives ...
    / "Cannot Do X to" SP d:DIRECTION SP "of HEX" EOF {
        return &FailureType_t{
            Direction: d.(direction.Direction_e),
        }, nil
    }
```

Then regenerate: `go generate ./...`

### Debug a Parse Failure

Enable debug flags:

```go
bistre.ParseInput(
    fid, tid, input,
    false,              // acceptLoneDash
    true,               // debugParser
    true,               // debugSections
    true,               // debugSteps
    true,               // debugNodes
    true,               // debugFleetMovement
    false, false,
    config,
)
```

This prints to log:
- `parser: ...` messages from grammar
- `parser: root: before split` / `after split` / `after consolidating` from nodes.go
- `parser: step %d: dirt|deck|crow` per step parse
- `parser: ... error: ...` for failures

### Handle Edge Cases

#### Lone Dashes
Some reports have orphaned dashes that should be ignored:
```go
if acceptLoneDash {
    continue  // Skip lone dashes
}
```

#### Scout Still vs Scout Still?
Some scouts report "Scout Still?" (ambiguous). Set `experimentalScoutStill=true` to rewrite as "Scout Still,,".

#### Unit Splitting
Some items incorrectly include unit IDs at the end:
```
"Settlement Name 0903"  // Should be split?
```
Set `experimentalUnitSplit=true` to attempt splitting via regex.

#### Last Turn Obscured Locations
Before turn "0902-02", current hex location can be "##XXXX" (obscured). The parser rejects these unless you're parsing an older turn.

## Testing

No dedicated test file exists. Tests are in:
- **adapters/golden_test.go**: Golden file comparison
- **cmd/tnrpt/main.go**: Command-line invocation tests

To test locally:
```bash
go run ./cmd/tnrpt parse-turn <file.txt> -o <output.json>
```

## Performance Notes

- Parser is single-pass for line scanning, O(n) per line for regex matching
- Node consolidation is O(n²) worst-case but typically O(n)
- Grammar compilation via pigeon is not memoized by default
- Most turn reports parse in <100ms

## Migration from Azul Parser

The old azul parser is deprecated. To migrate existing code:

1. Replace `azul.ParseInput()` with `bistre.ParseInput()`
2. Use `adapters.BistreParserTurnToModel()` to convert to old types (if needed)
3. Use `adapters.BistreTurnToModelReportX()` to convert to new model types
4. Update debug flags as needed (different names and behaviors)

## Debugging Tips

1. **Enable debugNodes**: Shows how components are split and consolidated
2. **Enable debugSteps**: Shows individual step parsing
3. **Check regex patterns**: Modify `rxCourierSection` etc. if section headers change
4. **Look at PEG errors**: Grammar failures show what didn't match
5. **Manual node inspection**: Insert `log.Printf(printNodes(root))` to see node tree

## Dependencies

- `github.com/maloquacious/semver`: Version parsing
- `direction`, `terrain`, `edges`, `resources`, `results`: Domain enums
- `coords`: Coordinate system
- `items`: Item type enum
- `unit_movement`: Movement type enum
- `winds`: Wind strength/direction

## Copyright & Style

All files have the copyright header:
```go
// Copyright (c) 2025 Michael D Henderson. All rights reserved.
```

Follow AGENTS.md style guide:
- Error handling: Return errors, no panics (except in parseMove for parser bugs)
- Types: Use `_t` suffix only in this legacy parser
- JSON: Use kebab-case with omitempty
- Imports: stdlib, then external, then internal (goimports order)

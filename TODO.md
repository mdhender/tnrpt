# Transition Plan: Standardize on WorldMapCoord

## Phase 1: Add WorldMapCoord tests
- [x] Create `coords/worldmapcoord_test.go` with tests for `NewWorldMapCoord()`, `Move()`, `String()`, JSON marshaling
- [x] Cover edge cases: "N/A", "##" obscured, boundary grids

## Phase 2: Update parsers/azul
- [x] Removed `Location coords.Map` from `Moves_t`, `Move_t`, `Report_t` in types.go
- [x] Removed `Location coords.Map` from `Scry_t` in parser.go
- [x] Updated adapters/turns.go to remove Location field mappings
- [x] Updated adapters/turns_test.go to remove Location assertions

## Phase 3: Update model.go
- [x] Removed `Location coords.Map` and `Coordinates` from `Moves_t`
- [x] Removed `Location coords.Map` from `Move_t`
- [x] Removed `Location coords.Map` from `Scry_t`
- [x] Removed `Location coords.Map` from `Report_t`
- [x] Updated adapters/turns.go and adapters/turns_test.go
- [x] Updated golden test files

## Phase 4: Update adapters
- [x] Ensure all conversions from parser types to model types use `WorldMapCoord`

## Phase 5: Delete obsolete files
- [x] `coords/grid.go`
- [x] `coords/xlat.go`
- [x] `coords/xlat_test.go`
- [x] `coords/map.go`
- [x] `coords/vectors.go`
- [x] `coords/helpers.go`

## Phase 6: Final verification
- [x] `go build ./...`
- [x] `go test ./...`

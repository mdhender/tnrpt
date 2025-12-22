# AGENTS.md

**TNRpt** implements tools for importing, parsing, and rendering TribeNet turn reports.

## Build & Test Commands
- Run tnrpt: `go run ./cmd/tnrpt options...`
- Build: `go build ./...`
- Test all: `go test ./...`
- Single test: `go test -run TestName ./path/to/package`
- Update golden files: `go test -update-golden ./adapters/...`

## Architecture
- **Module**: `github.com/mdhender/tnrpt` — Turn report parser for TribeNet
- **cmd/tnrpt**: CLI entry point using cobra for testing pipeline stages
- **parsers/azul**: Core parser for turn reports
- **adapters**: Converts parser types to model types (includes `to_model_store.go` for DB persistence)
- **model/**: New schema-aligned types (ReportFile, ReportX, UnitX, Act, Step, Tile) with SQLite Store
  - `store.go`: Store type with embedded schema.sql and repository methods
  - `types.go`: Domain types with db struct tags
  - `schema.sql`: SQLite DDL for all tables
- **model.go**: Legacy domain types (Turn_t, Move_t, etc.) — **deprecated**, use model/ package instead
- **Domain packages**: coords, terrain, direction, edges, compass, items, resources, results, winds

## Code Style
- Types use `_t` suffix (e.g., `Turn_t`, `UnitId_t`) (we want to migrate away from this)
- Constant errors in `cerrs` package using `type Error string` pattern
- JSON tags use kebab-case with `omitempty`
- Copyright header: `// Copyright (c) 2025 Michael D Henderson. All rights reserved.`
- Imports: stdlib first, then external, then internal (goimports style)
- Error handling: return errors up the call stack, no panics

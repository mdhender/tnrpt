# AGENTS.md

## Build & Test Commands
- Build: `go build ./...`
- Test all: `go test ./...`
- Single test: `go test -run TestName ./path/to/package`
- Update golden files: `go test -update-golden ./adapters/...`

## Architecture
- **Module**: `github.com/mdhender/tnrpt` — Turn report parser for TribeNet
- **cmd/azul**: CLI entry point using cobra
- **parsers/azul**: Core parser for turn reports
- **adapters**: Converts parser types to model types
- **model.go**: Core domain types (Turn_t, Move_t, Report_t, etc.)
- **Domain packages**: coords, terrain, direction, edges, compass, items, resources, results, winds

## Code Style
- Types use `_t` suffix (e.g., `Turn_t`, `UnitId_t`)
- Constant errors in `cerrs` package using `type Error string` pattern
- JSON tags use kebab-case with `omitempty`
- Copyright header: `// Copyright (c) 2025 Michael D Henderson. All rights reserved.`
- Imports: stdlib first, then external, then internal (goimports style)
- Error handling: return errors up the call stack, no panics

# OttoMap agent-optimized model (Go + SQLite)

This archive contains:

- `model/types.go` — compact Go structs with JSON kind discriminators and optional provenance (for merge conflict resolution).
- `schema.sql` — SQLite3 schema normalized around extracts (unit sections/actions/steps) and walker tiles (observations).
- `json_shape.md` — JSON shapes + examples that round-trip cleanly to the normalized tables.

Design goals

1. **Compact names**: `ReportFile`, `ReportX`, `UnitX`, `Act`, `Step`, `TNCoord`, `HexCoord`, `Enc`, `Tile`, `RenderJob`.
2. **Round-trip without polymorphic pain**:
   - `acts` and `steps` are single tables with `kind` plus **kind-specific columns** (nullable) to avoid per-kind tables.
   - Encounters and observations are normalized into child tables keyed by `step_id` (and for tiles, `tile_id`).
3. **Provenance for merge conflicts**:
   - Every extracted object can carry stable `src` pointers (`doc_id`, `unit_id`, `turn_no`, sequence numbers).
   - Tiles carry `src` entries that identify which extraction records contributed to the current tile.

Notes

- Coordinates:
  - `TNCoord` is the TribeNet coordinate system - it's a label containing the grid, column and row.
  - `Hex` is the `hexg.Hex` coordinate.
- Enums are strings in JSON and `TEXT` in SQLite for stability and legibility.

# Implementation

Stages
* [ ] Stage to import turn report (`.docx`) file to database
* [ ] Stage to import turn report (`.report.txt`) file to database
* [ ] Stage to split turn report into sections
* [ ] Stage to parse turn report section
* [ ] Stage to walk parsed section into unit/tile observations
* [ ] Stage to render unit/tile observations into Worldographer maps

## Sprint 11

The goal of Sprint 11 is to add an in memory modernc.org/sqlite database to the `cmd/tnrpt` pipeline.

### Plan
* [x] Review proposed schema in `model/types.go`
* [x] Create `model/store.go` with:
  - `//go:embed schema.sql` for the DDL
  - `Store` type wrapping `*sql.DB`
  - `NewStore(ctx, ":memory:")` constructor that runs embedded schema
* [x] Add repository methods to `Store` for core writes:
  - `InsertReportFile`, `InsertReportExtract`, `InsertUnitExtract`, `InsertAct`, `InsertStep`
* [x] Create new adapter: `adapters/to_model_store.go`
  - Converts `bistre.Turn` (or current model) → `model.ReportX` + inserts via `Store`
* [x] Integrate into `cmdPipeline()`:
  - Open in-memory DB at start
  - After parsing, use new adapter to persist to `Store`
* [x] Add `--show-db-stats` flag to dump row counts from each table
* [x] Defer cleanup: Mark root `model.go` types as deprecated (but don't remove yet)

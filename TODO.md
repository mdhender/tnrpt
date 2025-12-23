# Implementation

Stages
* [ ] Stage to import turn report (`.docx`) file to database
* [ ] Stage to import turn report (`.report.txt`) file to database
* [ ] Stage to split turn report into sections
* [ ] Stage to parse turn report section
* [ ] Stage to walk parsed section into unit/tile observations
* [ ] Stage to render unit/tile observations into Worldographer maps

## Sprint 13 — Spike: Validate Templ + HTMX + In-Memory Store

**Goal**: Prove out the stack with a minimal working app that displays parsed turn report data. App starts at end of pipeline for data exploration.

**Decisions made**:
- Framework: HTMX + Templ (typed components solve maintainability concerns)
- Data store: SQLite (existing `model/store.go`), in-memory for spike
- Auth: Session-based with bcrypt (future sprint)
- Pocketbase: Not needed at this scale

### Sprint 13 Tasks

#### 13.1 — Templ Setup
* [x] Add Templ to project (`go install github.com/a-h/templ/cmd/templ@latest`)
* [x] Create `web/` directory structure: `web/templates/`, `web/static/`, `web/handlers/`
* [x] Create base layout component (`web/templates/layout.templ`)
* [x] Verify Templ generates and compiles

#### 13.2 — Minimal HTTP Server
* [x] Create `cmd/server/` with basic Chi or stdlib router
* [x] Serve static files (HTMX, minimal CSS)
* [x] Create index handler that renders layout
* [x] Verify server starts and serves HTML

#### 13.3 — In-Memory Store Integration
* [x] Create in-memory store adapter (wraps existing model types)
* [x] Load parsed turn report data into store at startup
* [x] Expose store to handlers via dependency injection

#### 13.4 — First Data Table
* [x] Create Templ component for units table (`web/templates/units_table.templ`)
* [x] Create handler that queries store and renders component
* [x] Add HTMX to load table without full page reload
* [x] Verify data displays correctly

#### 13.5 — Spike Auth (Hardcoded)
* [x] Simple login page component
* [x] Hardcoded users: username `clan0500`, password `clan-0500` pattern
* [x] Session cookie (no bcrypt for spike, just string match)
* [x] Filter data by logged-in clan

#### 13.6 — Pipeline Integration
* [x] Add `--serve` and `--serve-no-auth` flags to pipeline CLI
* [x] After parsing, start HTTP server with loaded data
* [x] Document spike findings and decide go/no-go

---

## Sprint 14 — Core Data Tables (Read-Only)

**Goal**: Full read-only views of all major data types.

### Sprint 14 Tasks

#### 14.1 — Table Components
* [ ] Units table with sorting
* [ ] Tiles/terrain table
* [ ] Movement history table
* [ ] Resources/items table

#### 14.2 — Detail Views
* [ ] Unit detail page (click row → see full unit data)
* [ ] Tile detail page

#### 14.3 — Navigation
* [ ] Sidebar navigation component
* [ ] Turn selector (switch between loaded turns)

---

## Sprint 15 — Authentication + Persistence

**Goal**: Users log in and see only their data. Data persists to SQLite.

### Sprint 15 Tasks

#### 15.1 — Auth Foundation
* [ ] Add users table to schema (id, username, password_hash, clan_id)
* [ ] bcrypt password hashing utilities
* [ ] Session middleware (cookie-based)

#### 15.2 — Login Flow
* [ ] Login page component
* [ ] Login handler (validate credentials, create session)
* [ ] Logout handler
* [ ] Protect routes middleware

#### 15.3 — Data Isolation
* [ ] Filter all queries by authenticated user's clan_id
* [ ] Verify user A cannot see user B's data

#### 15.4 — SQLite Persistence
* [ ] Switch from in-memory to file-based SQLite
* [ ] Add migration tooling if needed

---

## Sprint 16 — Turn Report Upload

**Goal**: GM can upload turn reports via drag-drop; CLI fallback.

### Sprint 16 Tasks

#### 16.1 — Upload UI
* [ ] Drag-drop upload component (HTMX + JS)
* [ ] Progress indicator
* [ ] Success/error feedback

#### 16.2 — Upload Handler
* [ ] Accept `.docx` or `.report.txt` files
* [ ] Run existing parser pipeline
* [ ] Store results in database
* [ ] Associate with correct clan

#### 16.3 — CLI Upload (Fallback)
* [ ] `tnrpt upload --file report.docx --clan 1234`
* [ ] Same parsing pipeline as web upload

---

## Sprint 17 — Map Rendering + Download

**Goal**: Users click button, server renders map, user downloads file.

### Sprint 17 Tasks

#### 17.1 — Render Configuration UI
* [ ] Select turns/units to include on map
* [ ] Basic options (zoom, center, etc.)

#### 17.2 — Server-Side Rendering
* [ ] Integrate existing Worldographer renderer
* [ ] Generate map file on request
* [ ] Store generated files temporarily

#### 17.3 — Download Flow
* [ ] Download button triggers render
* [ ] Progress indicator for long renders
* [ ] Serve file download when complete

---

## Sprint 18 — Polish + Deploy

**Goal**: Production-ready deployment to DigitalOcean.

### Sprint 18 Tasks

#### 18.1 — Hardening
* [ ] Rate limiting
* [ ] CSRF protection
* [ ] Security headers
* [ ] Error pages (404, 500)

#### 18.2 — Deployment
* [ ] Dockerfile
* [ ] DigitalOcean droplet setup docs
* [ ] Systemd service file
* [ ] Backup strategy for SQLite

#### 18.3 — Documentation
* [ ] User guide (how to log in, view data, download maps)
* [ ] Admin guide (how to add users, upload reports)

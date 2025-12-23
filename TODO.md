# Implementation

Stages
* [ ] Stage to import turn report (`.docx`) file to database
* [ ] Stage to import turn report (`.report.txt`) file to database
* [ ] Stage to split turn report into sections
* [ ] Stage to parse turn report section
* [ ] Stage to walk parsed section into unit/tile observations
* [ ] Stage to render unit/tile observations into Worldographer maps

---

## Sprint 15 — Authentication + Persistence

**Goal**: Users log in and see only their data. Data persists to SQLite.

### Sprint 15 Tasks

#### 15.1 — Auth Foundation
* [x] Add users table to schema (id, username, password_hash, clan_id)
* [ ] bcrypt password hashing utilities
* [x] Session middleware (cookie-based)

#### 15.2 — Login Flow
* [x] Login page component
* [x] Login handler (validate credentials, create session)
* [x] Logout handler
* [x] Protect routes middleware

#### 15.3 — Data Isolation
* [x] Filter all queries by authenticated user's clan_id
* [x] Add clan_id column to unit_extracts for indexed filtering
* [ ] Verify user A cannot see user B's data

#### 15.4 — SQLite Persistence
* [ ] Add flag to select in-memory or file-based SQLite
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

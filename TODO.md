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
* [x] bcrypt password hashing utilities (web/auth/password.go)
* [x] Session middleware (cookie-based)
* [x] Add users, user_roles, game_clans tables to schema.sql
* [x] Add User/GameClan types and store methods (model/users.go)
* [x] Add JSON loader for users.json and games.json

#### 15.2 — Login Flow
* [x] Login page component
* [x] Logout handler
* [x] Protect routes middleware
* [x] Update auth.User to include Handle, GameID, ClanNo
* [x] Update ValidateCredentials to query DB with bcrypt
* [x] Update --auth-as to accept game:handle format

#### 15.3 — Data Isolation
* [x] Update all queries to filter by (game, clan_no) tuple
* [x] Verify user A cannot see user B's data

**Notes:**
- User handle is the primary key (e.g., "xtc69", "clan0500")
- Clan ID is per-game: xtc69 has clan 373 in game 0300, clan 669 in game 0301
- Session stores (Handle, GameID, ClanNo) for the active game context
- Data filtering requires both game AND clan_no, not just clan_no

#### 15.4 — SQLite Persistence
* [x] Add flag to select in-memory or file-based SQLite (--db flag)
* [x] Add init-db command to create and initialize database
* [x] Add compact-db command for backup/export
* [x] WAL mode enabled for file-based SQLite (concurrent access support)
* [x] Server refuses to start if file-based DB doesn't exist

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

# Implementation

Stages
* [ ] Stage to import turn report (`.docx`) file to database
* [ ] Stage to import turn report (`.report.txt`) file to database
* [ ] Stage to split turn report into sections
* [ ] Stage to parse turn report section
* [ ] Stage to walk parsed section into unit/tile observations
* [ ] Stage to render unit/tile observations into Worldographer maps

---

## Sprint 16 — Turn Report Upload

**Goal**: GM can upload turn reports via drag-drop; CLI fallback.

### Sprint 16 Tasks

#### 16.1 — Upload UI
* [x] Protected page (only for GMs)
* [x] Must select game and turn prior to uploading (all uploaded reports are associated with the selected game and turn)
* [x] Drag-drop upload component (HTMX + JS)
* [x] Progress indicator
* [x] Success/error feedback

#### 16.2 — Upload Handler
* [x] Protected route (only for GMs)
* [x] Accept files named `CCCC.docx` or `GGGG.YYYY-MM.CCCC.report.txt` (where GGGG is game, YYYY-MM is turn, CCCC is clan 0001-0999)

#### 16.3 — Upload Pipeline
* [x] Run existing parser pipeline
* [x] Store results in database
* [x] Associate with correct game, turn, and clan

#### 16.4 — CLI Upload (Fallback)
* [x] `tnrpt upload --file report.docx --clan 1234`
* [x] Same parsing pipeline as web upload

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

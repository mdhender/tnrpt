# Implementation

Stages
* [ ] Stage to import turn report (`.docx`) file to database
* [ ] Stage to import turn report (`.report.txt`) file to database
* [ ] Stage to split turn report into sections
* [ ] Stage to parse turn report section
* [ ] Stage to walk parsed section into unit/tile observations
* [ ] Stage to render unit/tile observations into Worldographer maps

---

## Sprint 18 — Map Rendering + Download

**Goal**: Users click button, server renders map, user downloads file.

### Sprint 18 Tasks

#### 18.1 — Render Configuration UI
* [ ] Select turns/units to include on map
* [ ] Basic options (zoom, center, etc.)

#### 18.2 — Server-Side Rendering
* [ ] Integrate existing Worldographer renderer
* [ ] Generate map file on request
* [ ] Store generated files temporarily

#### 18.3 — Download Flow
* [ ] Download button triggers render
* [ ] Progress indicator for long renders
* [ ] Serve file download when complete

---

## Sprint 19 — Polish + Deploy

**Goal**: Production-ready deployment to DigitalOcean.

### Sprint 19 Tasks

#### 19.1 — Hardening
* [ ] Rate limiting
* [ ] CSRF protection
* [ ] Security headers
* [ ] Error pages (404, 500)

#### 19.2 — Deployment
* [ ] Dockerfile
* [ ] DigitalOcean droplet setup docs
* [ ] Systemd service file
* [ ] Backup strategy for SQLite

#### 19.3 — Documentation
* [ ] User guide (how to log in, view data, download maps)
* [ ] Admin guide (how to add users, upload reports)

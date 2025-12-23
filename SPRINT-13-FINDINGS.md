# Sprint 13 Spike Findings: HTMX + Templ + In-Memory Store

**Date**: 2025-12-23  
**Goal**: Validate stack with minimal working app displaying parsed turn report data

## Summary

**Verdict: GO** — Stack is suitable for production use.

## What Worked Well

### Templ
- Type-safe components eliminate runtime template errors
- `templ generate` is fast and integrates cleanly with Go build
- Hot reload with `templ generate --watch` speeds development
- Components compose naturally (Layout wraps pages, pages include partials)

### HTMX
- Partial page updates work seamlessly with Templ components
- `HX-Request` header detection allows handler reuse for full/partial responses
- Minimal JavaScript required — CDN include is sufficient
- Progressive enhancement: works without JS, enhanced with JS

### In-Memory Store
- Simple `sync.RWMutex` + slices pattern is sufficient for spike scale
- Easy to swap for SQLite persistence layer later
- Filter methods (UnitsByClan) work well with session-based auth

### Authentication (Hardcoded Spike)
- Cookie-based sessions work as expected
- Session middleware pattern (`RequireAuth`) is clean
- Clan-based data filtering proves the model works

### Pipeline Integration
- `--serve` and `--serve-no-auth` flags enable rapid testing
- Parse → Load → Serve workflow validates full data path
- Memory store receives parsed data correctly

## Issues Encountered

### Minor Issues (Resolved)
1. **Package name collision**: `model.Store` variable shadowed `web/store` import — solved with import alias
2. **Templ generation**: Must run `templ generate` before build — add to build scripts

### Deferred Issues
1. **No CSRF protection** — Needed for Sprint 18
2. **No session expiry cleanup** — Acceptable for spike, needs goroutine for production
3. **Hardcoded credentials** — Replaced with bcrypt in Sprint 15

## Performance Observations

- Cold start (parse + serve): ~200ms for single docx
- Memory usage: Minimal for test data (~40 users, few turns)
- Response time: Sub-millisecond for in-memory queries

## Architecture Decisions Confirmed

| Decision | Rationale |
|----------|-----------|
| HTMX over React/Vue | Simpler, less JS, sufficient for data tables |
| Templ over html/template | Type safety, better IDE support |
| stdlib router | Sufficient for ~10 routes, no framework needed |
| SQLite (future) | Single binary deployment, backup is file copy |

## Next Steps (Sprint 14+)

1. Add more data tables (tiles, movements, resources)
2. Implement detail views with HTMX navigation
3. Add bcrypt authentication (Sprint 15)
4. Switch to file-based SQLite (Sprint 15)

## How to Run the Spike

```bash
# Full pipeline with auth
go run ./cmd/tnrpt pipeline --docx testdata/0301.0899-12.0987.docx --game 0301 --clan 0987 --serve

# Without auth (show all data)
go run ./cmd/tnrpt pipeline --docx testdata/0301.0899-12.0987.docx --game 0301 --clan 0987 --serve-no-auth

# Standalone server (loads from directory)
go run ./cmd/server --data testdata/sprint-13
```

Login with: `clan0987` / `clan-0987` (pattern: clanXXXX / clan-XXXX)

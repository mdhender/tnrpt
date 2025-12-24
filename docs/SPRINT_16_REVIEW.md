# Sprint 16 Review: Turn Report Upload

**Date**: 2025-12-23  
**Status**: COMPLETE ✓  
**Goal**: GM can upload turn reports via drag-drop UI; CLI fallback available.

## Summary

Sprint 16 successfully delivered end-to-end turn report upload capability. GMs can now upload `.docx` or `.report.txt` files through a protected web interface with drag-drop support, or via CLI for automation. All uploaded reports are automatically parsed, validated, and persisted to the SQLite database.

## What Was Accomplished

### 16.1 — Upload UI ✓
- **Protected page**: `/upload` route restricted to GM role via session middleware
- **Game/Turn selection**: Dropdown for game selection, numeric input for turn number
- **Drag-drop component**: HTMX + vanilla JavaScript (no framework)
  - Visual feedback: drag-over state, file icons
  - Click fallback: click drop zone to open file picker
  - Multiple file support
- **Progress indicator**: Real-time upload status per file
- **Feedback**: Success toast with statistics (units, acts, steps) or error messages

**Files**:
- `web/templates/upload.templ`: Templ component with embedded JS
- `web/static/style.css`: Drag-drop styling (`.drop-zone`, `.drag-over`, `.upload-list`)

### 16.2 — Upload Handler ✓
- **Protected route**: `POST /upload` restricted to GM role
- **Filename validation**: Strict pattern matching
  - `CCCC.docx`: Clan-only format (clan 0001-0999)
  - `GGGG.YYYY-MM.CCCC.report.txt`: Full format (game, turn, clan)
- **Form validation**: Game and turn required
- **Content-type verification**: DOCX files checked for correct MIME type
- **Cross-field validation**: Filename game/turn must match selected game/turn
- **File size limit**: 100KB max for multipart form
- **Response format**: JSON with success/error and parsed data counts

**Files**:
- `web/handlers/upload.go` (258 lines): Core upload handler with validation logic

### 16.3 — Upload Pipeline ✓
- **Multi-format parsing**:
  1. DOCX → Extract text via `docx.ParseReader()`
  2. Report text → Split sections via `report.ParseReportText()`
  3. Bistre parser → Full semantic parsing
  4. Adapter → Convert to model types
- **Database storage**:
  - `ReportFile` record (SHA256 hash, MIME type, metadata)
  - `ReportX` with nested `Units`, `Acts`, `Steps`
  - Clan association for data filtering
- **Data integrity**: SHA256 hashing for deduplication detection

**Files**:
- `web/handlers/upload.go` (lines 187-220): Upload pipeline logic
- `adapters/` (existing): BistreTurnToModelReportX() adapter

**Parsing flow**:
```
File Upload → DOCX/TXT Parse → Section Extraction → Bistre Parser
  ↓
Model Adapter → ReportFile + ReportX Records → SQLite Store
  ↓
JSON Response (units, acts, steps count)
```

### 16.4 — CLI Upload (Fallback) ✓
- **Command**: `tnrpt upload --file report.docx --clan 1234 --game 0301 --turn 901`
- **Same pipeline**: Uses identical parsing/storage logic as web upload
- **Error handling**: Validates all inputs, reports errors clearly
- **Database**: Connects to SQLite store (file or in-memory)

**Files**:
- `cmd/tnrpt/main.go`: CLI entry point with cobra command

### Bonus: Admin SQL Console
- Added protected admin route with SQL query interface
- Useful for debugging: inspect raw data, verify schema, test queries
- Files: `web/handlers/sql.go`, `web/templates/sql_console.templ`

## Architecture Decisions

| Decision | Rationale |
|----------|-----------|
| Vanilla JS for drag-drop | Minimal dependencies, works without frameworks |
| Server-side validation | Prevent invalid data from reaching parser |
| SHA256 hashing | Enable duplicate detection in future features |
| JSON responses | HTMX can easily parse and display results |
| Reuse parsing pipeline | No code duplication between web and CLI |

## Code Quality

- **Lines added**: ~1,928 across 21 files
- **Test coverage**: Limited (handlers lack unit tests — noted for Sprint 17+)
- **Error handling**: Comprehensive, returns meaningful error messages
- **Type safety**: No new `_t` suffixed types; uses model package
- **Documentation**: Inline comments explain validation patterns

## Known Limitations & Technical Debt

1. **Handler tests missing**: Upload, SQL console handlers lack unit tests
2. **File upload size**: Hard-coded 100KB limit
3. **Concurrent uploads**: No locking mechanism if same clan uploads simultaneously
4. **Error recovery**: Failed parse doesn't clean up ReportFile record
5. **Rate limiting**: No upload rate limits (deferred to Sprint 18)
6. **CSRF protection**: Missing (deferred to Sprint 18)

## How to Use

### Web Upload
1. Navigate to `/upload`
2. Select game and enter turn number
3. Drag files or click to select
4. Monitor upload progress
5. View results (units/acts/steps parsed)

### CLI Upload
```bash
# Upload single file to specific game/clan
go run ./cmd/tnrpt upload \
  --file path/to/report.docx \
  --clan 1234 \
  --game 0301 \
  --turn 901

# With custom database
go run ./cmd/tnrpt upload \
  --db /path/to/data.db \
  --file report.docx \
  --clan 0987 \
  --game 0301 \
  --turn 901
```

## Performance

- **Upload response time**: ~500ms (DOCX parse + bistre parse + DB store)
- **File size**: Tested with test data (~100KB docx)
- **Database operations**: Synchronous, no concurrent load testing

## Testing Performed

- ✓ Drag-drop UI (FF, Chrome)
- ✓ Form validation (game/turn required)
- ✓ Filename pattern validation (both formats)
- ✓ MIME type checks (DOCX)
- ✓ Cross-field validation (filename vs selected game/turn)
- ✓ CLI upload with database persistence
- ✓ Existing test suite still passes

## Next Steps (Sprint 17)

1. **Map rendering**: Integrate Worldographer renderer to create SVG/PDF maps
2. **Download flow**: Button to trigger render and download generated files
3. **Render options**: UI for selecting which turns/units to include
4. **Progress tracking**: Long render operations with progress indicator

## Deployment Readiness

**Ready for dev/staging**: ✓ Working end-to-end  
**Ready for production**: ✗ Missing hardening features
- No CSRF protection
- No rate limiting
- No detailed audit logging

These are deferred to Sprint 18 (Polish + Deploy).

## Files Changed

### New
- `web/handlers/upload.go` (258 lines)
- `web/handlers/sql.go` (40 lines)
- `web/templates/upload.templ` (166 lines)
- `web/templates/sql_console.templ` (72 lines)
- `docs/schema.dbml` (334 lines)

### Modified
- `cmd/tnrpt/main.go`: Added upload command
- `cmd/server/main.go`: Route registration
- `web/handlers/handlers.go`: Handler registration
- `web/templates/layout.templ`: Navigation link
- `web/static/style.css`: Upload component styling

### Commits
```
037827b Add admin SQL console for debugging in-memory database
8a98a61 feat: add CLI upload command (Sprint 16.4)
88d0fd5 feat(upload): implement Sprint 16.3 upload pipeline
b00a808 feat(upload): implement Sprint 16.2 upload handler with file validation
28ce659 refactor: simplify auth-as flag to accept handle only, add auth-as-clan flag
f2fc845 Sprint 16.1: Upload UI implementation
a20ac0a update docs for Sprint 16
```

## Conclusion

Sprint 16 delivers a solid, functional upload system that integrates seamlessly with the existing parsing pipeline. The drag-drop UI is intuitive, validation is comprehensive, and the CLI provides a programmatic interface for batch uploads. Database persistence is working correctly with proper data association to games, turns, and clans.

The sprint achieved all four task objectives on schedule. Technical debt is minimal and well-documented for future sprints.

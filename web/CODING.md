// Copyright (c) 2025 Michael D Henderson. All rights reserved.

# Web Application Coding Guide

> **Note**: This guide was written at commit `[TBD]` (2025-12-23). The web code is actively maintained, but this documentation is **not aggressively updated**. If you notice discrepancies between this guide and the actual code, please check the code as the source of truth.

The web application provides an HTMX + Alpine.js frontend for exploring turn reports and OttoMap game state. It uses Templ for type-safe HTML generation and SQLite for persistent storage.

## Overview

**Purpose**: HTTP server serving turn report analysis and game data  
**Frontend Stack**: HTMX + Alpine.js + Templ components  
**Backend**: Go net/http handlers  
**Data Layer**: SQLite with schema in web/store/schema.sql  
**Authentication**: Session-based (cookie, in-memory or persistent)  
**Status**: Actively developed; transitioning from HTMX prototype to full SPA  

## Architecture

```
HTTP Request → Auth Middleware → Handler → Store Query → Templ Template → HTMX Response
```

### Directory Structure

- **auth/**: Session and authentication management
  - `auth.go`: Session store, user types, cookie helpers
  - `password.go`: Password hashing utilities (bcrypt)
- **handlers/**: HTTP handler functions for all routes
  - `handlers.go`: Handler struct, session injection, layout data
  - `auth.go`: Login/logout endpoints
  - `index.go`: Dashboard and home page
  - `upload.go`: Turn report file uploads
  - `units.go`: Unit listing and search
  - `movements.go`: Movement analysis
  - `terrain.go`: Terrain distribution views
  - `resources.go`: Resource distribution views
  - `sql.go`: SQL console (GM-only)
- **store/**: Data access layer
  - `schema.sql`: SQLite DDL (users, games, clans, tiles, acts, steps)
  - `sqlite.go`: SQLiteStore type with repo methods
  - `loader.go`: Bulk load from parsed turn data
  - `memory.go`: In-memory store (testing)
- **static/**: CSS, JavaScript, images
  - `style.css`: Base styles (minimal; mostly using HTML structure)
- **templates/**: Templ components (pre-compiled to _templ.go files)
  - `layout.templ`: Master layout with header, sidebar, footer
  - `login.templ`: Login form
  - `upload.templ`: File upload interface
  - `units_table.templ`: Unit listing
  - `unit_detail.templ`: Single unit view
  - `movements.templ`: Movement timeline
  - `terrain.templ`: Terrain heatmap or list
  - `resources.templ`: Resource availability grid
  - `sql_console.templ`: SQL query interface (GM-only)
  - `tile_detail.templ`: Individual hex details

## Core Types

### Session & Authentication (auth/)

```go
type User struct {
    Handle   string  // Unique user identifier (e.g., "xtc69")
    UserName string  // Display name
    GameID   string  // Currently active game context
    ClanNo   int     // Clan number in active game
}

type Session struct {
    ID        string    // Secure session token
    User      User
    ExpiresAt time.Time // 24-hour default
}

type SessionStore struct {
    mu       sync.RWMutex
    sessions map[string]*Session
}
```

**Key methods**:
- `NewSessionStore()`: Create empty store
- `Create(user User) *Session`: Issue new session
- `Get(id string) *Session`: Retrieve and validate session
- `Delete(id string)`: Revoke session
- `SetSessionCookie(w http.ResponseWriter, session *Session)`: Write secure cookie
- `GetSessionFromRequest(r *http.Request, store *SessionStore) *Session`: Extract from request

### Handlers (handlers/)

```go
type Handlers struct {
    store        *store.SQLiteStore
    sessions     *auth.SessionStore
    autoAuthUser *auth.User  // For testing (bypasses auth)
}
```

**Lifecycle**:
1. `New(s *store.SQLiteStore, sessions *auth.SessionStore) *Handlers`
2. Register handler methods (e.g., `h.HandleIndex`, `h.HandleUnits`)
3. Optional: `h.SetAutoAuth(gameID, handle, clanNo)` for tests

### Layout Data (templates/layout.templ)

```go
type LayoutData struct {
    CurrentPath    string        // Request path
    Turns          []int         // Available turns
    SelectedTurn   int           // Turn query param
    Version        string        // App version
    HideTurnSelect bool
    Games          []store.UserGame
    CurrentGameID  string
    CurrentClanNo  int
    UserHandle     string
    IsGM           bool
}
```

**Key methods**:
- `LinkWithTurn(path string) string`: Build URL preserving game/turn context
- `GameSwitchURL(gameID string) string`: URL for game selection

## Entry Points

### Server (cmd/server/main.go)

```go
func main() {
    // Load database
    s, err := store.NewSQLiteStore("data.db")
    
    // Create session store
    sessions := auth.NewSessionStore()
    
    // Create handlers
    h := handlers.New(s, sessions)
    
    // Register routes
    mux := http.NewServeMux()
    mux.HandleFunc("/", h.HandleIndex)
    mux.HandleFunc("/login", h.HandleLogin)
    // ... more routes
    
    // Start server
    http.ListenAndServe(":8080", mux)
}
```

### Route Registration Pattern

```go
// Handlers typically follow this pattern:
func (h *Handlers) HandleSomePage(w http.ResponseWriter, r *http.Request) {
    // 1. Extract session (optional, some routes public)
    session := auth.GetSessionFromRequest(r, h.sessions)
    
    // 2. Check auth if needed
    if session == nil {
        http.Redirect(w, r, "/login", http.StatusSeeOther)
        return
    }
    
    // 3. Fetch layout data
    layoutData := h.getLayoutData(r, session)
    
    // 4. Query data from store
    data, err := h.store.GetSomeData(r.Context(), ...)
    if err != nil {
        // Handle error
    }
    
    // 5. Render template
    templates.SomeTemplate(layoutData, data).Render(r.Context(), w)
}
```

## Templ Components

### What is Templ?

Templ is a type-safe Go templating language:
- **Type-safe**: Component parameters are Go types, checked at compile time
- **Compiled**: Generates Go code (in _templ.go files); no runtime parsing
- **Composable**: Components nest like React components
- **SSR-ready**: Perfect for HTMX + Alpine patterns

### Layout Component (layout.templ)

```templ
templ Layout(title string) {
    @LayoutWithData(title, LayoutData{})
}

templ LayoutWithData(title string, data LayoutData) {
    <!DOCTYPE html>
    <html>
        <head>
            <title>{ title } - TribeNet Reports</title>
            <script src="https://unpkg.com/htmx.org@2.0.4"></script>
            <!-- Alpine.js for interactivity -->
        </head>
        <body>
            <header><!-- Navigation, user info, game switcher --></header>
            <aside><!-- Sidebar with turn selector --></aside>
            <main>{ children... }</main>
            <footer><!-- Version, copyright --></footer>
        </body>
    </html>
}
```

**Key features**:
- Master layout used by all pages
- Dynamic navigation based on authentication
- Game/turn context preserved via query parameters
- Turn selector dropdown with `redirectWithTurn` script

### Script Blocks in Templ

```templ
script redirectWithTurn(path string) {
    var turn = document.getElementById('turn-select').value;
    if (turn) {
        window.location.href = path + '?turn=' + turn;
    } else {
        window.location.href = path;
    }
}
```

Compiled to JavaScript; called from event handlers like `onchange={ redirectWithTurn(path) }`.

### Component Nesting

```templ
templ Index(stats store.Stats, data LayoutData) {
    @LayoutWithData("Home", data) {
        <h1>Welcome</h1>
        <!-- Content inside children... -->
    }
}
```

## Data Layer (store/)

### SQLiteStore Type

```go
type SQLiteStore struct {
    db *sql.DB
    // Repository methods: GetUnits, GetTiles, GetActsByUnit, etc.
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
    db, err := sql.Open("sqlite", dbPath)
    // ... init schema
    return &SQLiteStore{db: db}, nil
}
```

### Schema Overview (schema.sql)

**Core tables** (user-facing):
- `tiles`: Game hexes with terrain
- `tile_units`: Units at each tile
- `tile_rsrc`: Resources found at tile
- `tile_borders`: Edge types (RIVER, ALPS, etc.)

**Extract/Import tables** (turn parsing):
- `report_files`: Source documents uploaded
- `report_extracts`: Root container per file
- `unit_extracts`: One per unit in extract
- `acts`: Movement commands (follow, goto, move, scout, status)
- `steps`: Individual movement steps (adv, still, patrol, obs)

**Normalized details** (acts/steps):
- `step_enc_units`: Encountered units
- `step_enc_sets`: Encountered settlements
- `step_enc_rsrc`: Encountered resources
- `step_borders`: Observations of edges

**Authentication**:
- `users`: User accounts (handle PK, password_hash)
- `user_roles`: Permissions (GM, player, etc.)
- `games`: Game definitions
- `game_clans`: User-to-clan membership per game

### Common Queries

```go
// Get all units for current game/clan
units, err := h.store.GetUnitsForClan(ctx, gameID, clanNo)

// Get tile details (terrain, encounters, resources)
tile, err := h.store.GetTile(ctx, hexCoord)

// Get acts (commands) for a unit
acts, err := h.store.GetActsForUnit(ctx, unitID)

// Get games for authenticated user
games, err := h.store.GetGamesForUser(ctx, userHandle)

// Get turns available for clan
turns, err := h.store.TurnsByGameClan(gameID, clanNo)
```

## Authentication Flow

### Login

1. User POSTs credentials to `/login`
2. `HandleLogin` validates against `users` table
3. On success:
   - Create `Session` via `h.sessions.Create(user)`
   - Set session cookie via `auth.SetSessionCookie(w, session)`
   - Redirect to `/`
4. On failure: Re-render login form with error

### Per-Request Auth

```go
session := auth.GetSessionFromRequest(r, h.sessions)
if session == nil {
    // Not logged in; redirect or return 403
}
// Use session.User for access control
```

### Logout

1. User clicks logout
2. `HandleLogout`:
   - Delete session from store: `h.sessions.Delete(sessionID)`
   - Clear cookie: `auth.ClearSessionCookie(w)`
   - Redirect to login

## HTMX Integration

HTMX allows partial page updates without full reloads. Common patterns:

```html
<!-- Load and swap HTML -->
<button hx-get="/units" hx-target="#data-view" hx-swap="innerHTML">Load Units</button>

<!-- Submit form and update page -->
<form hx-post="/api/move" hx-target="#result">
    <!-- inputs -->
</form>

<!-- Polling -->
<div hx-get="/status" hx-trigger="every 5s">Status...</div>

<!-- Sorting/filtering via query params -->
<tr hx-get="/units?sort=name&filter=active">
```

**Typical handler response**:

```go
func (h *Handlers) HandleUnits(w http.ResponseWriter, r *http.Request) {
    // ... validate session, fetch data ...
    
    // Render only the content (not full page)
    templates.UnitsTable(units, layoutData).Render(r.Context(), w)
}
```

Note: HTMX requests return fragments; full-page handlers use layout.

## Common Workflows

### Add a New Report View

1. **Create template** in `web/templates/`:
   ```templ
   templ MyView(data MyData, layout LayoutData) {
       @LayoutWithData("My View", layout) {
           <h1>My View</h1>
           <!-- Content -->
       }
   }
   ```

2. **Create handler** in `web/handlers/`:
   ```go
   func (h *Handlers) HandleMyView(w http.ResponseWriter, r *http.Request) {
       session := auth.GetSessionFromRequest(r, h.sessions)
       if session == nil {
           http.Redirect(w, r, "/login", http.StatusSeeOther)
           return
       }
       layout := h.getLayoutData(r, session)
       data, _ := h.store.FetchMyData(r.Context(), session.User.GameID, ...)
       templates.MyView(data, layout).Render(r.Context(), w)
   }
   ```

3. **Register route** in `cmd/server/main.go`:
   ```go
   mux.HandleFunc("/myview", h.HandleMyView)
   ```

4. **Add navigation link** in `layout.templ`:
   ```templ
   <li><a href={ templ.SafeURL(data.LinkWithTurn("/myview")) }>My View</a></li>
   ```

5. **Compile Templ**: `templ generate ./...`

### Add a Query to the Store

1. **Add method** to `SQLiteStore` in `web/store/sqlite.go`:
   ```go
   func (s *SQLiteStore) GetMyData(ctx context.Context, gameID string, clanNo int) ([]MyType, error) {
       query := `SELECT ... FROM ... WHERE game = ? AND clan_no = ?`
       rows, err := s.db.QueryContext(ctx, query, gameID, clanNo)
       // ... scan rows into slice ...
       return data, nil
   }
   ```

2. **Use in handler**:
   ```go
   data, err := h.store.GetMyData(r.Context(), session.User.GameID, session.User.ClanNo)
   ```

### Add Authentication Check

```go
// For route requiring specific role:
isGM, _ := h.store.IsUserGM(r.Context(), session.User.Handle)
if !isGM {
    http.Error(w, "Forbidden", http.StatusForbidden)
    return
}
```

### Debug SQL Queries

Use the SQL console (GM-only at `/sql`) to test queries against live database. Console handler is in `handlers/sql.go`.

## Performance Notes

- **Templ compilation**: One-time at `templ generate ./...`; no runtime overhead
- **Session store**: In-memory map; scans for expired sessions on each `Get()`
- **SQLite**: Single connection; uses connection pool for concurrent requests
- **HTMX**: Incremental updates reduce payload size vs. full-page reloads

## Testing

**Session testing**:
```go
h.SetAutoAuth("game123", "testuser", 500)
// All handlers auto-authenticate as testuser
```

**Template testing**:
```go
buf := new(strings.Builder)
templates.MyTemplate(data).Render(context.Background(), buf)
html := buf.String()
// Assert on output
```

## Dependencies

- `github.com/a-h/templ`: HTML templating
- `net/http`: Standard HTTP server
- `database/sql` + `github.com/mattn/go-sqlite3`: SQLite driver
- `golang.org/x/crypto/bcrypt`: Password hashing
- `github.com/mdhender/tnrpt`: Core parser and types

## Code Style

Follow AGENTS.md:
- Error handling: Return errors, no panics
- No `_t` suffix in new types (legacy parser only)
- JSON tags: kebab-case with `omitempty`
- Imports: stdlib, external, internal (goimports order)
- Copyright header: `// Copyright (c) 2025 Michael D Henderson. All rights reserved.`

## Context Management

All data access operations use `context.Context` from the request:

```go
func (h *Handlers) HandleSomething(w http.ResponseWriter, r *http.Request) {
    // Pass r.Context() to all store methods
    data, err := h.store.GetData(r.Context(), ...)
    
    // Pass r.Context() to Templ render
    templates.SomeTemplate(data).Render(r.Context(), w)
}
```

This enables cancellation, tracing, and request-scoped values.

## Extending for Multi-Game Scenarios

Current design assumes user-per-game. To support multi-game accounts:

1. **LayoutData.Games**: Already supports multiple games (UserGame slice)
2. **Game switcher**: Dropdown in header when `len(Games) > 1`
3. **Preserve context**: `LinkWithTurn()` builds URLs with `?game=X&turn=Y`
4. **Store queries**: Requires explicit `gameID` parameter

All existing code already handles this; no changes needed to support it.

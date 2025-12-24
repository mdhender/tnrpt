// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/mdhender/tnrpt"
	"github.com/mdhender/tnrpt/web/auth"
	"github.com/mdhender/tnrpt/web/handlers"
	"github.com/mdhender/tnrpt/web/store"
)

func main() {
	addr := flag.String("addr", ":8787", "HTTP listen address")
	authAs := flag.String("auth-as", "", "auto-authenticate as handle (e.g., xtc69) for testing")
	authAsClan := flag.String("auth-as-clan", "", "auto-authenticate as game.clan (e.g., 0301.500) for testing")
	dataPath := flag.String("data", "", "directory containing .docx turn reports")
	dbPath := flag.String("db", "", "SQLite database file path (empty = in-memory)")
	gameDataPath := flag.String("game-data", "testdata/sprint-13", "path to games initialization file")
	logWithDefaultFlags := flag.Bool("log-with-default-flags", false, "log with default flags")
	logWithShortFileName := flag.Bool("log-with-shortfile", true, "log with short file name")
	logWithTimestamp := flag.Bool("log-with-timestamp", false, "log with timestamp")
	showVersion := flag.Bool("version", false, "show version and exit")
	staticDir := flag.String("static", "web/static", "static files directory")
	timeout := flag.Duration("timeout", 0, "auto-shutdown after duration (e.g., 5s, 1m)")
	userDataPath := flag.String("user-data", "testdata/sprint-13", "path to users initialization file")
	flag.Parse()

	if *showVersion {
		fmt.Println(tnrpt.Version().Core())
		os.Exit(0)
	}

	logFlags := 0
	if *logWithShortFileName {
		logFlags |= log.Lshortfile
	}
	if *logWithTimestamp {
		logFlags |= log.Ltime
	}
	if *logWithDefaultFlags || logFlags == 0 {
		logFlags = log.LstdFlags
	}
	log.SetFlags(logFlags)

	err := run(*dbPath, *dataPath, *gameDataPath, *userDataPath, *staticDir, *authAs, *authAsClan, *addr, *timeout)
	if err != nil {
		log.Printf("error: %v\n", err)
	}
}

func run(dbPath, dataPath, gameDataPath, userDataPath, staticDir, authAs, authAsClan, addr string, timeout time.Duration) error {
	var sqliteStore *store.SQLiteStore
	var err error

	if dbPath != "" {
		// File-based mode: database must already exist (created by init-db command)
		log.Printf("store: using file-based SQLite: %s", dbPath)
		sqliteStore, err = store.NewSQLiteStoreWithConfig(store.StoreConfig{
			Path:       dbPath,
			InitSchema: false, // schema already applied by init-db
		})
	} else {
		// In-memory mode (default)
		log.Printf("store: using in-memory SQLite")
		sqliteStore, err = store.NewSQLiteStore()
	}
	if err != nil {
		return fmt.Errorf("failed to create SQLite store: %v", err)
	}
	defer sqliteStore.Close()

	ctx := context.Background()

	// Load users and games first (needed for auth)
	if userDataPath != "" {
		usersPath := filepath.Join(userDataPath, "users.json")
		if err := sqliteStore.LoadUsersFromJSON(ctx, usersPath); err != nil {
			return fmt.Errorf("failed to load users: %w", err)
		}
	}
	if gameDataPath != "" {
		gamesPath := filepath.Join(gameDataPath, "games.json")
		if err := sqliteStore.LoadGamesFromJSON(ctx, gamesPath); err != nil {
			return fmt.Errorf("failed to load games: %w", err)
		}
	}

	// load any new data files
	if dataPath != "" {
		if err := store.LoadDocxFromDir(sqliteStore, dataPath); err != nil {
			return fmt.Errorf("failed to load data: %w", err)
		}
	}

	stats := sqliteStore.Stats()
	log.Printf("store: %d reports, %d units, %d acts, %d steps",
		stats.Reports, stats.Units, stats.Acts, stats.Steps)

	sessions := auth.NewSessionStore()
	h := handlers.New(sqliteStore, sessions)

	if authAs != "" && authAsClan != "" {
		return fmt.Errorf("auth: cannot use both --auth-as and --auth-as-clan")
	}

	if authAsClan != "" {
		parts := strings.SplitN(authAsClan, ".", 2)
		if len(parts) != 2 {
			return fmt.Errorf("auth: invalid format %q (expected game.clan)", authAsClan)
		}
		gameID := parts[0]
		clanNo, err := strconv.Atoi(parts[1])
		if err != nil {
			return fmt.Errorf("auth: invalid clan number %q: %v", parts[1], err)
		}
		handle, err := sqliteStore.GetHandleForClan(ctx, gameID, clanNo)
		if err != nil {
			return fmt.Errorf("auth: failed to get handle for %s: %v", authAsClan, err)
		}
		if handle == "" {
			return fmt.Errorf("auth: clan %d not found in game %s", clanNo, gameID)
		}
		h.SetAutoAuth(gameID, handle, clanNo)
		log.Printf("auth: auto-authenticating as %s (game %s, clan %d)", handle, gameID, clanNo)
	}

	if authAs != "" {
		games, err := sqliteStore.GetGamesForUser(ctx, authAs)
		if err != nil {
			return fmt.Errorf("auth: failed to get games for %s: %v", authAs, err)
		}
		if len(games) == 0 {
			return fmt.Errorf("auth: user %s not found in any game", authAs)
		}
		game := games[0]
		h.SetAutoAuth(game.GameID, authAs, game.ClanNo)
		log.Printf("auth: auto-authenticating as %s (game %s, clan %d)", authAs, game.GameID, game.ClanNo)
	}

	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir(staticDir))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	mux.HandleFunc("/", h.Index)
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.Login(w, r)
		} else {
			h.LoginPage(w, r)
		}
	})
	mux.HandleFunc("/logout", h.Logout)
	mux.HandleFunc("/units", h.RequireAuth(h.Units))
	mux.HandleFunc("/units/{id}", h.RequireAuth(h.UnitDetail))
	mux.HandleFunc("/movements", h.RequireAuth(h.Movements))
	mux.HandleFunc("/terrain", h.RequireAuth(h.Terrain))
	mux.HandleFunc("/tiles/{grid}/{col}/{row}", h.RequireAuth(h.TileDetail))
	mux.HandleFunc("/resources", h.RequireAuth(h.Resources))
	mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.RequireGM(h.UploadHandler)(w, r)
		} else {
			h.RequireGM(h.UploadPage)(w, r)
		}
	})
	mux.HandleFunc("/admin/sql", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.RequireGM(h.SQLConsoleExec)(w, r)
		} else {
			h.RequireGM(h.SQLConsolePage)(w, r)
		}
	})

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	if timeout > 0 {
		go func() {
			log.Printf("server: will auto-shutdown in %v", timeout)
			time.Sleep(timeout)
			log.Printf("server: timeout reached, initiating shutdown")
			shutdown <- os.Interrupt
		}()
	}

	go func() {
		log.Printf("server: listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("server: %v", err)
		}
	}()

	<-shutdown
	log.Printf("server: shutting down gracefully")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server: shutdown error: %w", err)
	}

	log.Printf("server: stopped")
	return nil
}

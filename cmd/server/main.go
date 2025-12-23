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
	"syscall"
	"time"

	"github.com/mdhender/tnrpt"
	"github.com/mdhender/tnrpt/web/auth"
	"github.com/mdhender/tnrpt/web/handlers"
	"github.com/mdhender/tnrpt/web/store"
)

func main() {
	addr := flag.String("addr", ":8787", "HTTP listen address")
	dataDir := flag.String("data", "testdata/sprint-13", "directory containing .docx turn reports")
	logWithDefaultFlags := flag.Bool("log-with-default-flags", false, "log with default flags")
	logWithShortFileName := flag.Bool("log-with-shortfile", true, "log with short file name")
	logWithTimestamp := flag.Bool("log-with-timestamp", false, "log with timestamp")
	staticDir := flag.String("static", "web/static", "static files directory")
	timeout := flag.Duration("timeout", 0, "auto-shutdown after duration (e.g., 5s, 1m)")
	showVersion := flag.Bool("version", false, "show version and exit")
	authAs := flag.String("auth-as", "", "auto-authenticate as clan (e.g., clan0500) for testing")
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

	sqliteStore, err := store.NewSQLiteStore()
	if err != nil {
		log.Fatalf("failed to create SQLite store: %v", err)
	}
	defer sqliteStore.Close()

	if err := store.LoadFromDir(sqliteStore, *dataDir); err != nil {
		log.Printf("warning: failed to load data: %v", err)
	}

	stats := sqliteStore.Stats()
	log.Printf("store: %d reports, %d units, %d acts, %d steps",
		stats.Reports, stats.Units, stats.Acts, stats.Steps)

	sessions := auth.NewSessionStore()
	h := handlers.New(sqliteStore, sessions)

	if *authAs != "" {
		h.SetAutoAuth(*authAs)
		log.Printf("auth: auto-authenticating as %s", *authAs)
	}

	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir(*staticDir))
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

	server := &http.Server{
		Addr:         *addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	if *timeout > 0 {
		go func() {
			log.Printf("server: will auto-shutdown in %v", *timeout)
			time.Sleep(*timeout)
			log.Printf("server: timeout reached, initiating shutdown")
			shutdown <- os.Interrupt
		}()
	}

	go func() {
		log.Printf("server: listening on %s", *addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	<-shutdown
	log.Printf("server: shutting down gracefully")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("server: shutdown error: %v", err)
	}

	log.Printf("server: stopped")
}

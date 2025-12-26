// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/mdhender/phrases/v2"
	"github.com/mdhender/tnrpt"
	"github.com/mdhender/tnrpt/adapters"
	"github.com/mdhender/tnrpt/model"
	"github.com/mdhender/tnrpt/parsers"
	"github.com/mdhender/tnrpt/pipelines/parsers/bistre"
	"github.com/mdhender/tnrpt/pipelines/parsers/docx"
	"github.com/mdhender/tnrpt/pipelines/parsers/report"
	"github.com/mdhender/tnrpt/pipelines/stages"
	sqlite "github.com/mdhender/tnrpt/stores/sqlite"
	"github.com/mdhender/tnrpt/walkers/anhinga"
	"github.com/mdhender/tnrpt/web/auth"
	"github.com/mdhender/tnrpt/web/handlers"
	"github.com/spf13/cobra"
)

func main() {
	addFlags := func(cmd *cobra.Command) error {
		cmd.PersistentFlags().Bool("debug", false, "log debugging information")
		cmd.PersistentFlags().Bool("log-with-default-flags", false, "log with default flags")
		cmd.PersistentFlags().Bool("log-with-shortfile", true, "log with short file name")
		cmd.PersistentFlags().Bool("log-with-timestamp", false, "log with timestamp")
		cmd.PersistentFlags().Bool("quiet", false, "log less information")
		cmd.PersistentFlags().Bool("show-version", false, "show version")
		cmd.PersistentFlags().Bool("verbose", false, "log more information")
		return nil
	}
	var cmdRoot = &cobra.Command{
		Use:   "tnrpt",
		Short: "TribeNet command line utility",
		Long:  `Run commands for TribeNet turn reports and maps`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			logWithDefaultFlags, _ := cmd.Flags().GetBool("log-with-default-flags")
			logWithShortFileName, _ := cmd.Flags().GetBool("log-with-shortfile")
			logWithTimestamp, _ := cmd.Flags().GetBool("log-with-timestamp")
			logFlags := 0
			if logWithShortFileName {
				logFlags |= log.Lshortfile
			}
			if logWithTimestamp {
				logFlags |= log.Ltime
			}
			if logWithDefaultFlags || logFlags == 0 {
				logFlags = log.LstdFlags
			}
			log.SetFlags(logFlags)

			if showVersion, _ := cmd.Flags().GetBool("show-version"); showVersion {
				fmt.Printf("tnrpt: version %q\n", tnrpt.Version().Core())
			}

			return nil
		},
	}
	cmdRoot.AddCommand(cmdDb())
	cmdRoot.AddCommand(cmdParse())
	cmdRoot.AddCommand(cmdPhrase())
	cmdRoot.AddCommand(cmdBistreParse())
	cmdRoot.AddCommand(cmdPipeline())
	cmdRoot.AddCommand(cmdUpload())
	cmdRoot.AddCommand(cmdWalk())
	cmdRoot.AddCommand(cmdVersion())
	if err := addFlags(cmdRoot); err != nil {
		log.Fatal(err)
	}

	if err := cmdRoot.Execute(); err != nil {
		os.Exit(1)
	}
}

func cmdDb() *cobra.Command {
	showBuildInfo := false
	addFlags := func(cmd *cobra.Command) error {
		cmd.Flags().BoolVar(&showBuildInfo, "build-info", showBuildInfo, "show build information")
		return nil
	}
	var cmd = &cobra.Command{
		Use:   "db",
		Short: "database tools",
	}
	cmd.AddCommand(cmdDbCompact())
	cmd.AddCommand(cmdDbInit())
	if err := addFlags(cmd); err != nil {
		log.Fatal(err)
	}
	return cmd
}

func cmdDbCompact() *cobra.Command {
	var cmd = &cobra.Command{
		Use:          "compact <database-path>",
		Short:        "Compact a SQLite database for backup/export",
		Long:         `Runs VACUUM and checkpoints WAL to create a single compact database file suitable for backup or export.`,
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath := args[0]

			log.Printf("db: compact: compacting database at %s", dbPath)

			if err := sqlite.CompactDatabase(dbPath); err != nil {
				return fmt.Errorf("compact database: %w", err)
			}

			log.Printf("db: compact: database compacted successfully")
			return nil
		},
	}
	return cmd
}

func cmdDbInit() *cobra.Command {
	var cmd = &cobra.Command{
		Use:          "initb <database-path>",
		Short:        "Create and initialize a new SQLite database",
		Long:         `Creates a new SQLite database file and initializes the schema. The database file must not already exist.`,
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath := args[0]

			log.Printf("db: initb: creating database at %s", dbPath)

			if err := sqlite.InitDatabase(dbPath); err != nil {
				return fmt.Errorf("init database: %w", err)
			}

			log.Printf("db: initb: database created successfully")
			log.Printf("db: initb: WAL mode enabled for concurrent access")
			return nil
		},
	}
	return cmd
}

func cmdParse() *cobra.Command {
	autoEOL := true
	stripCR := false
	var configFile string
	var excludeUnits []string
	var includeUnits []string
	var outputFile string
	addFlags := func(cmd *cobra.Command) error {
		cmd.Flags().BoolVar(&autoEOL, "auto-eol", autoEOL, "automatically convert line endings")
		cmd.Flags().StringVarP(&configFile, "config-file", "c", configFile, "load configuration from file")
		cmd.Flags().StringSliceVarP(&excludeUnits, "exclude", "e", excludeUnits, "exclude the unit")
		cmd.Flags().StringSliceVarP(&includeUnits, "include", "i", includeUnits, "include the unit")
		cmd.Flags().StringVarP(&outputFile, "output", "o", outputFile, "save parse to file")
		cmd.Flags().BoolVar(&stripCR, "strip-cr", stripCR, "strip CR from end-of-lines")
		return nil
	}
	var cmd = &cobra.Command{
		Use:          "parse <turn-report-file>",
		Short:        "parse a turn report file (text or Word)",
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1), // require path to turn report file
		RunE: func(cmd *cobra.Command, args []string) error {
			quiet, _ := cmd.Flags().GetBool("quiet")
			verbose, _ := cmd.Flags().GetBool("verbose")
			debug, _ := cmd.Flags().GetBool("debug")
			if quiet {
				verbose = false
			}

			if configFile != "" {
				return fmt.Errorf("error: --config-file is not implemented")
			}

			turn, err := parsers.ParseTurnReport(args[0], autoEOL, stripCR, quiet, verbose, debug)
			if err != nil {
				return err
			}
			at, err := adapters.AzulParserTurnToModel(args[0], turn)
			if err != nil {
				return err
			}
			if data, err := json.MarshalIndent(at, "", "  "); err != nil {
				log.Fatalf("json: %v\n", err)
			} else if outputFile == "" {
				log.Printf("%s\n", string(data))
			} else if err = os.WriteFile(outputFile, data, 0o644); err != nil {
				return err
			} else {
				log.Printf("%s: wrote %d bytes\n", outputFile, len(data))
			}

			return nil
		},
	}
	if err := addFlags(cmd); err != nil {
		log.Fatal(err)
	}
	return cmd
}

func cmdPhrase() *cobra.Command {
	length := 6
	addFlags := func(cmd *cobra.Command) error {
		cmd.Flags().IntVar(&length, "length", length, "number of words in phrase")
		return nil
	}
	var cmd = &cobra.Command{
		Use:   "phrase",
		Short: "random phrase",
		RunE: func(cmd *cobra.Command, args []string) error {
			if length < 1 {
				length = 1
			} else if length > 16 {
				length = 16
			}
			fmt.Println(phrases.Generate(length))
			return nil
		},
	}
	if err := addFlags(cmd); err != nil {
		log.Fatal(err)
	}
	return cmd
}

func cmdBistreParse() *cobra.Command {
	var docxFile string // := filepath.Join("testdata", "0301.0899-12.0987.docx")
	var textFile string // := filepath.Join("testdata", "0301.0899-12.0987.txt")
	var game, clanNo string
	normalizeCRLF, normalizeCR := true, true
	showDBStats := false
	showReportSections := false
	showReportSectionLines := false
	showText := false
	showTiming := false
	trimLeading, trimTrailing := true, true
	var serve bool
	var serveNoAuth bool
	var serveAddr string
	var staticDir string
	addFlags := func(cmd *cobra.Command) error {
		cmd.Flags().StringVar(&docxFile, "docx", docxFile, "import docx file")
		cmd.Flags().StringVar(&textFile, "text", textFile, "import text file")
		cmd.Flags().StringVar(&game, "game", game, "game identifier")
		cmd.Flags().StringVar(&clanNo, "clan", clanNo, "clan number")
		cmd.Flags().BoolVar(&normalizeCR, "normalize-cr", normalizeCR, "change CR to LF at end-of-line")
		cmd.Flags().BoolVar(&normalizeCRLF, "normalize-cr-lf", normalizeCRLF, "change CR+LF to LF at end-of-line")
		cmd.Flags().BoolVar(&showDBStats, "show-db-stats", showDBStats, "dump row counts from each table")
		cmd.Flags().BoolVar(&showReportSections, "show-report-sections", showReportSections, "show report sections")
		cmd.Flags().BoolVar(&showReportSectionLines, "show-section-lines", showReportSectionLines, "show report section lines")
		cmd.Flags().BoolVar(&showText, "show-text", showText, "show report text")
		cmd.Flags().BoolVar(&showTiming, "show-timing", showText, "show timing for each stage")
		cmd.Flags().BoolVar(&trimLeading, "trim-leading-spaces", trimLeading, "trim leading spaces on import")
		cmd.Flags().BoolVar(&trimLeading, "trim-trailing-spaces", trimTrailing, "trim trailing spaces on import")
		cmd.Flags().BoolVar(&serve, "serve", false, "start HTTP server after parsing")
		cmd.Flags().BoolVar(&serveNoAuth, "serve-no-auth", false, "start HTTP server without authentication")
		cmd.Flags().StringVar(&serveAddr, "serve-addr", ":8787", "HTTP server listen address")
		cmd.Flags().StringVar(&staticDir, "static", "web/static", "static files directory")
		return nil
	}
	var cmd = &cobra.Command{
		Use:          "bistre",
		Short:        "Run the bistre parser pipeline (legacy synchronous path)",
		SilenceUsage: true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// cascade show flags up
			if showReportSectionLines {
				showReportSections = true
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			quiet, _ := cmd.Flags().GetBool("quiet")
			verbose, _ := cmd.Flags().GetBool("verbose")
			debug, _ := cmd.Flags().GetBool("debug")
			if quiet {
				verbose = false
			}

			var err error

			// Open in-memory database
			store, err := sqlite.NewSQLiteStore()
			if err != nil {
				return fmt.Errorf("create store: %w", err)
			}
			defer store.Close()

			startedPipeline, startedStage := time.Now(), time.Now()
			var doc *docx.Docx
			if docxFile != "" {
				doc, err = docx.ParsePath(docxFile, trimLeading, trimTrailing, quiet, verbose, debug)
				if err != nil {
					return err
				}
				if showTiming {
					log.Printf("%s: parsed docx in %v\n", filepath.Base(doc.Source), time.Since(startedStage))
				}
			}
			if textFile != "" {
				doc = &docx.Docx{
					Source: textFile,
					Text:   nil,
				}
				doc.Text, err = os.ReadFile(textFile)
				if err != nil {
					return err
				}
				if showTiming {
					log.Printf("%s: parsed text in %v\n", filepath.Base(doc.Source), time.Since(startedStage))
				}
			}
			if doc == nil {
				return fmt.Errorf("missing import")
			}

			log.Printf("docx: %q\n", doc.Source)
			if showText {
				log.Printf("docx: %s\n", string(doc.Text))
			}

			startedStage = time.Now()
			rpt, err := report.ParseReportText(doc, normalizeCRLF, normalizeCR, quiet, verbose, debug)
			if err != nil {
				return err
			}
			if showTiming {
				log.Printf("%s: parse text  completed in %v\n", rpt.Name, time.Since(startedStage))
			}

			log.Printf("report: path %q\n", rpt.Path)
			log.Printf("report: name %q\n", rpt.Name)
			log.Printf("report: turn %q\n", rpt.TurnNo)
			log.Printf("report: sections %d\n", len(rpt.Sections))

			if showReportSections {
				for n, section := range rpt.Sections {
					log.Printf("section %3d: unit %q\n", n+1, section.UnitId)
					log.Printf("section %3d: turn %q\n", n+1, section.TurnNo)
					if showReportSectionLines {
						for no, line := range section.Lines {
							log.Printf("section %3d: %4d: %s\n", n+1, no+1, string(line))
						}
					}
				}
			}

			var text []byte
			for _, section := range rpt.Sections {
				text = append(text, bytes.Join(section.Lines, []byte{'\n'})...)
				text = append(text, '\n')
			}
			var acceptLoneDash, parserDebugFlag, sectionsDebugFlag, stepsDebugFlag, nodesDebugFlag, fleetMovementDebugFlag, splitTrailingUnits, cleanupScoutStill bool

			startedStage = time.Now()
			turn, err := bistre.ParseInput(rpt.Name, rpt.TurnNo, text, acceptLoneDash, parserDebugFlag, sectionsDebugFlag, stepsDebugFlag, nodesDebugFlag, fleetMovementDebugFlag, splitTrailingUnits, cleanupScoutStill, bistre.ParseConfig{})
			if err != nil {
				return err
			} else if turn == nil {
				return fmt.Errorf("parser returned nil, nil")
			}
			if showTiming {
				log.Printf("%s: parse input completed in %v\n", rpt.Name, time.Since(startedStage))
			}

			log.Printf("%s: parse: turn %q: %4d %2d\n", rpt.Name, turn.Id, turn.Year, turn.Month)
			if foundTurnNo := fmt.Sprintf("%d-%02d", turn.Year, turn.Month); rpt.TurnNo != foundTurnNo {
				if turn.Year == 0 && turn.Month == 0 {
					log.Printf("error: unable to locate turn information in file\n")
					log.Printf("error: this is usually caused by unexpected line endings in the file\n")
					log.Printf("error: try running with --auto-eol\n")
					return fmt.Errorf("unable to find current turn in source")
				}
				log.Printf("error: expected turn %q: got turn %q\n", rpt.TurnNo, foundTurnNo)
				return fmt.Errorf("unexpected current turn in source")
			}

			startedStage = time.Now()
			at, err := adapters.BistreParserTurnToModel(rpt.Name, turn)
			if err != nil {
				return err
			} else if at == nil {
				return fmt.Errorf("adapter returned nil, nil")
			}
			_ = at // retained for compatibility; new code uses Store
			if showTiming {
				log.Printf("%s: adapt turn  completed in %v\n", rpt.Name, time.Since(startedStage))
			}

			// Persist to in-memory database
			startedStage = time.Now()
			_, _, err = adapters.BistreTurnToStore(ctx, store, rpt.Name, turn, game, clanNo)
			if err != nil {
				return fmt.Errorf("persist to store: %w", err)
			}
			if showTiming {
				log.Printf("%s: store write completed in %v\n", rpt.Name, time.Since(startedStage))
			}

			// Show database stats if requested
			if showDBStats {
				stats, err := store.TableStats(ctx)
				if err != nil {
					return fmt.Errorf("get table stats: %w", err)
				}
				log.Println("database stats:")
				tables := make([]string, 0, len(stats))
				for table := range stats {
					tables = append(tables, table)
				}
				sort.Strings(tables)
				for _, table := range tables {
					if stats[table] > 0 {
						log.Printf("  %-20s %d rows\n", table, stats[table])
					}
				}
			}

			if showTiming {
				log.Printf("%s: pipeline    completed in %v\n", rpt.Name, time.Since(startedPipeline))
			}

			if serve || serveNoAuth {
				rx, err := adapters.BistreTurnToModelReportX(rpt.Name, turn, game, clanNo)
				if err != nil {
					return fmt.Errorf("adapt to model: %w", err)
				}

				sqliteStore, err := sqlite.NewSQLiteStore()
				if err != nil {
					return fmt.Errorf("create SQLite store: %w", err)
				}
				defer sqliteStore.Close()

				if err := sqliteStore.AddReport(rx); err != nil {
					return fmt.Errorf("add report to store: %w", err)
				}

				stats := sqliteStore.Stats()
				log.Printf("store: %d reports, %d units, %d acts, %d steps",
					stats.Reports, stats.Units, stats.Acts, stats.Steps)

				sessions := auth.NewSessionStore()
				h := handlers.New(sqliteStore, sessions)

				mux := http.NewServeMux()
				fs := http.FileServer(http.Dir(staticDir))
				mux.Handle("/static/", http.StripPrefix("/static/", fs))
				mux.HandleFunc("/", h.Index)

				if serveNoAuth {
					mux.HandleFunc("/units", h.UnitsNoAuth)
				} else {
					mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodPost {
							h.Login(w, r)
						} else {
							h.LoginPage(w, r)
						}
					})
					mux.HandleFunc("/logout", h.Logout)
					mux.HandleFunc("/units", h.RequireAuth(h.Units))
				}

				server := &http.Server{
					Addr:         serveAddr,
					Handler:      mux,
					ReadTimeout:  15 * time.Second,
					WriteTimeout: 15 * time.Second,
					IdleTimeout:  60 * time.Second,
				}

				shutdown := make(chan os.Signal, 1)
				signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

				go func() {
					log.Printf("server: listening on %s", serveAddr)
					if serveNoAuth {
						log.Printf("server: authentication disabled (--serve-no-auth)")
					}
					if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
						log.Fatalf("server: %v", err)
					}
				}()

				<-shutdown
				log.Printf("server: shutting down gracefully")

				shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				if err := server.Shutdown(shutdownCtx); err != nil {
					return fmt.Errorf("server shutdown: %w", err)
				}
				log.Printf("server: stopped")
			}

			return nil
		},
	}
	if err := addFlags(cmd); err != nil {
		log.Fatal(err)
	}
	return cmd
}

func cmdUpload() *cobra.Command {
	var dbPath string
	var file string
	var game string
	var turn string
	var clan string
	addFlags := func(cmd *cobra.Command) error {
		cmd.Flags().StringVar(&dbPath, "db", "", "path to SQLite database (required)")
		cmd.Flags().StringVar(&file, "file", "", "path to turn report file (.docx or .report.txt)")
		cmd.Flags().StringVar(&game, "game", "", "game ID (4-digit, e.g., 0301)")
		cmd.Flags().StringVar(&turn, "turn", "", "turn ID (YYYY-MM format, e.g., 0899-12)")
		cmd.Flags().StringVar(&clan, "clan", "", "clan number (0001-0999, extracted from filename if not provided)")
		cmd.MarkFlagRequired("db")
		cmd.MarkFlagRequired("file")
		cmd.MarkFlagRequired("game")
		cmd.MarkFlagRequired("turn")
		return nil
	}
	var cmd = &cobra.Command{
		Use:   "upload",
		Short: "Upload a turn report to the database",
		Long: `Upload a turn report file (.docx or .report.txt) to the database.
Uses the same parsing pipeline as the web upload handler.

File naming patterns:
  CCCC.docx                      - clan only (0001-0999)
  GGGG.YYYY-MM.CCCC.report.txt   - game, turn, clan

Examples:
  tnrpt upload --db data/tnrpt.db --file 0987.docx --game 0301 --turn 0899-12
  tnrpt upload --db data/tnrpt.db --file 0301.0899-12.0987.report.txt --game 0301 --turn 0899-12`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			filename := filepath.Base(file)
			fileClan, fileGame, fileTurn := parseUploadFilename(filename)

			if clan == "" {
				if fileClan == "" {
					return fmt.Errorf("clan not provided and could not be extracted from filename")
				}
				clan = fileClan
			}

			if fileGame != "" && fileGame != game {
				return fmt.Errorf("game in filename (%s) does not match --game (%s)", fileGame, game)
			}
			if fileTurn != "" && fileTurn != turn {
				return fmt.Errorf("turn in filename (%s) does not match --turn (%s)", fileTurn, turn)
			}

			data, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("read file: %w", err)
			}

			store, err := sqlite.NewSQLiteStoreWithConfig(sqlite.StoreConfig{Path: dbPath})
			if err != nil {
				return fmt.Errorf("open database: %w", err)
			}
			defer store.Close()

			var text []byte
			if strings.HasSuffix(strings.ToLower(filename), ".docx") {
				doc, err := docx.ParseReader(bytes.NewReader(data), true, true, true, false, false)
				if err != nil {
					return fmt.Errorf("parse docx: %w", err)
				}

				rpt, err := report.ParseReportText(doc, true, true, true, false, false)
				if err != nil {
					return fmt.Errorf("parse report: %w", err)
				}

				for _, section := range rpt.Sections {
					text = append(text, bytes.Join(section.Lines, []byte{'\n'})...)
					text = append(text, '\n')
				}
			} else {
				text = data
			}

			parsedTurn, err := bistre.ParseInput(filename, turn, text, false, false, false, false, false, false, false, false, bistre.ParseConfig{})
			if err != nil {
				return fmt.Errorf("parse turn report: %w", err)
			}
			if parsedTurn == nil {
				return fmt.Errorf("parser returned no data")
			}

			turnNo := 100*parsedTurn.Year + parsedTurn.Month
			now := time.Now().UTC()

			hash := sha256.Sum256(data)
			var mime string
			if strings.HasSuffix(strings.ToLower(filename), ".docx") {
				mime = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
			} else {
				mime = "text/plain"
			}

			rf := &model.ReportFile{
				Game:      game,
				ClanNo:    clan,
				TurnNo:    turnNo,
				Name:      filename,
				SHA256:    hex.EncodeToString(hash[:]),
				Mime:      mime,
				CreatedAt: now,
			}
			if err := store.AddReportFile(rf); err != nil {
				return fmt.Errorf("store report file: %w", err)
			}

			rx, err := adapters.BistreTurnToModelReportX(filename, parsedTurn, game, clan)
			if err != nil {
				return fmt.Errorf("convert report: %w", err)
			}
			rx.ReportFileID = rf.ID

			if err := store.AddReport(rx); err != nil {
				return fmt.Errorf("store report: %w", err)
			}

			units := len(rx.Units)
			acts := 0
			steps := 0
			for _, u := range rx.Units {
				acts += len(u.Acts)
				for _, a := range u.Acts {
					steps += len(a.Steps)
				}
			}

			log.Printf("upload: %s: game=%s turn=%s clan=%s units=%d acts=%d steps=%d",
				filename, game, turn, clan, units, acts, steps)

			return nil
		},
	}
	if err := addFlags(cmd); err != nil {
		log.Fatal(err)
	}
	return cmd
}

func parseUploadFilename(filename string) (clan, game, turn string) {
	docxRe := regexp.MustCompile(`^(0\d{3})\.docx$`)
	if matches := docxRe.FindStringSubmatch(filename); matches != nil {
		return matches[1], "", ""
	}

	txtRe := regexp.MustCompile(`^(\d{4})\.(\d{4}-\d{2})\.(0\d{3})\.report\.txt$`)
	if matches := txtRe.FindStringSubmatch(filename); matches != nil {
		return matches[3], matches[1], matches[2]
	}

	return "", "", ""
}

func cmdWalk() *cobra.Command {
	var excludeUnits []string
	var includeUnits []string
	var outputFile string
	addFlags := func(cmd *cobra.Command) error {
		cmd.Flags().StringSliceVarP(&excludeUnits, "exclude", "e", excludeUnits, "exclude the unit")
		cmd.Flags().StringSliceVarP(&includeUnits, "include", "i", includeUnits, "include the unit")
		cmd.Flags().StringVarP(&outputFile, "output", "o", outputFile, "save output to file")
		return nil
	}
	var cmd = &cobra.Command{
		Use:          "walk <turn-report.json> [<turn-report.json>...]",
		Short:        "Walk a parsed turn report, adding coordinates",
		SilenceUsage: true,
		Args:         cobra.MinimumNArgs(1), // require path to turn report file
		RunE: func(cmd *cobra.Command, args []string) error {
			quiet, _ := cmd.Flags().GetBool("quiet")
			verbose, _ := cmd.Flags().GetBool("verbose")
			debug, _ := cmd.Flags().GetBool("debug")
			if quiet {
				verbose = false
			}

			for _, input := range args {
				started, startedParser := time.Now(), time.Now()
				turn, err := parsers.ParseTurnReport(input, true, false, quiet, verbose, debug)
				if err != nil {
					return err
				}
				log.Printf("%s: parsed in %v\n", input, time.Since(startedParser))

				startedStage := time.Now()
				at, err := adapters.AzulParserTurnToModel(input, turn)
				if err != nil {
					return err
				}
				log.Printf("%s: adapted in %v\n", input, time.Since(startedStage))

				startedWalker := time.Now()
				_, err = anhinga.Walk(at, nil, quiet, verbose, debug)
				if err != nil {
					return err
				}
				log.Printf("%s: walked in %v\n", input, time.Since(startedWalker))

				log.Printf("%s: finished in %v\n", input, time.Since(started))
			}

			return nil
		},
	}
	if err := addFlags(cmd); err != nil {
		log.Fatal(err)
	}
	return cmd
}

func cmdPipeline() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pipeline",
		Short: "pipeline commands for report processing",
		Long:  "Commands for ingesting, processing, and tracking report files through the pipeline.",
	}
	cmd.AddCommand(cmdPipelineIngest())
	cmd.AddCommand(cmdPipelineStatus())
	cmd.AddCommand(cmdPipelineWork())
	return cmd
}

func cmdPipelineIngest() *cobra.Command {
	var dbPath string
	var dataDir string
	var game string
	var clan string
	var turn int

	cmd := &cobra.Command{
		Use:   "ingest <file>...",
		Short: "Ingest turn report files into the pipeline",
		Long: `Ingest one or more turn report files (.docx or .txt) into the pipeline.

Creates an upload batch and queues work items for processing:
  - DOCX files are queued for the 'extract' stage
  - TXT files are queued directly for the 'parse' stage

Files are copied to {data-dir}/batches/{batch_id}/ with standardized names.
Duplicate files (same SHA-256) are silently skipped (idempotent).

Examples:
  tnrpt pipeline ingest --db data/amp/tnrpt.db --data-dir data/amp --game 0301 --clan 0512 --turn 89912 *.docx
  tnrpt pipeline ingest --db data/amp/tnrpt.db --data-dir data/amp --game 0301 --clan 0512 --turn 89912 report.txt`,
		SilenceUsage: true,
		Args:         cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			store, err := sqlite.NewSQLiteStoreWithConfig(sqlite.StoreConfig{Path: dbPath})
			if err != nil {
				return fmt.Errorf("open database: %w", err)
			}
			defer store.Close()

			svc := stages.NewIngestService(store, dataDir)

			var files []stages.IngestRequest
			for _, path := range args {
				data, err := os.ReadFile(path)
				if err != nil {
					return fmt.Errorf("read %s: %w", path, err)
				}
				files = append(files, stages.IngestRequest{
					Filename: filepath.Base(path),
					Data:     data,
				})
			}

			createdBy := fmt.Sprintf("cli:%s", os.Getenv("USER"))
			batchID, results, err := svc.IngestBatch(ctx, game, clan, turn, createdBy, files)
			if err != nil {
				return fmt.Errorf("ingest batch: %w", err)
			}

			duplicates := 0
			ingested := 0
			for _, r := range results {
				if r.Duplicate {
					duplicates++
				} else {
					ingested++
				}
			}

			log.Printf("pipeline: ingest: batch=%d ingested=%d duplicates=%d", batchID, ingested, duplicates)
			return nil
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "path to SQLite database (required)")
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "data directory for file storage (required)")
	cmd.Flags().StringVar(&game, "game", "", "game ID (e.g., 0301)")
	cmd.Flags().StringVar(&clan, "clan", "", "clan number (e.g., 0512)")
	cmd.Flags().IntVar(&turn, "turn", 0, "turn number (e.g., 89912 for year 899, month 12)")
	cmd.MarkFlagRequired("db")
	cmd.MarkFlagRequired("data-dir")
	cmd.MarkFlagRequired("game")
	cmd.MarkFlagRequired("clan")
	cmd.MarkFlagRequired("turn")

	return cmd
}

func cmdPipelineStatus() *cobra.Command {
	var dbPath string
	var batchID int64
	var showFailed bool
	var stage string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show pipeline status",
		Long: `Show pipeline work queue status.

With --batch-id: shows summary for a specific batch
With --failed: lists all failed jobs
With --failed --stage: lists failed jobs for a specific stage

Examples:
  tnrpt pipeline status --db data/amp/tnrpt.db --batch-id 1
  tnrpt pipeline status --db data/amp/tnrpt.db --failed
  tnrpt pipeline status --db data/amp/tnrpt.db --failed --stage extract`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			store, err := sqlite.NewSQLiteStoreWithConfig(sqlite.StoreConfig{Path: dbPath})
			if err != nil {
				return fmt.Errorf("open database: %w", err)
			}
			defer store.Close()

			if showFailed {
				return showFailedJobs(ctx, store, stage)
			}

			if batchID > 0 {
				return showBatchStatus(ctx, store, batchID)
			}

			return fmt.Errorf("specify --batch-id or --failed")
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "path to SQLite database (required)")
	cmd.Flags().Int64Var(&batchID, "batch-id", 0, "show summary for specific batch")
	cmd.Flags().BoolVar(&showFailed, "failed", false, "list failed jobs")
	cmd.Flags().StringVar(&stage, "stage", "", "filter by stage (extract, parse)")
	cmd.MarkFlagRequired("db")

	return cmd
}

func showBatchStatus(ctx context.Context, store *sqlite.SQLiteStore, batchID int64) error {
	batch, err := store.GetUploadBatch(ctx, batchID)
	if err != nil {
		return fmt.Errorf("get batch: %w", err)
	}
	if batch == nil {
		return fmt.Errorf("batch %d not found", batchID)
	}

	fmt.Printf("Batch %d (game=%s, clan=%s, turn=%d)\n", batch.ID, batch.Game, batch.ClanNo, batch.TurnNo)
	fmt.Printf("Created: %s\n", batch.CreatedAt.Format(time.RFC3339))
	fmt.Println()

	summary, err := store.GetWorkSummaryByBatch(ctx, batchID)
	if err != nil {
		return fmt.Errorf("get work summary: %w", err)
	}

	fmt.Println("Work Summary:")
	for _, stage := range []string{"extract", "parse"} {
		statuses := summary[stage]
		if statuses == nil {
			statuses = make(map[string]int)
		}
		fmt.Printf("  %s: %d ok, %d running, %d queued, %d failed\n",
			stage,
			statuses["ok"],
			statuses["running"],
			statuses["queued"],
			statuses["failed"])
	}

	return nil
}

func showFailedJobs(ctx context.Context, store *sqlite.SQLiteStore, stage string) error {
	stages := []string{"extract", "parse"}
	if stage != "" {
		stages = []string{stage}
	}

	fmt.Println("Failed Jobs:")
	total := 0
	for _, s := range stages {
		jobs, err := store.GetFailedWork(ctx, s)
		if err != nil {
			return fmt.Errorf("get failed work: %w", err)
		}
		for _, j := range jobs {
			errCode := ""
			if j.ErrorCode != nil {
				errCode = *j.ErrorCode
			}
			fmt.Printf("  ID=%d  stage=%s  file_id=%d  error=%s\n", j.ID, j.Stage, j.ReportFileID, errCode)
			total++
		}
	}

	if total == 0 {
		fmt.Println("  (none)")
	} else {
		fmt.Println()
		fmt.Println("To retry: tnrpt pipeline work <stage> --retry-failed")
	}

	return nil
}

func cmdPipelineWork() *cobra.Command {
	var dbPath string
	var dataDir string
	var pollInterval time.Duration
	var retryFailed bool

	cmd := &cobra.Command{
		Use:   "work <stage>",
		Short: "Process pipeline work queue",
		Long: `Process jobs in the pipeline work queue.

Stages:
  extract  - Extract text from DOCX files
  parse    - Parse extracted text into model tables
  all      - Process extract then parse sequentially

The worker claims jobs atomically and processes them one at a time.
Use --poll-interval to run continuously, polling for new work.

Examples:
  tnrpt pipeline work --db data/amp/tnrpt.db --data-dir data/amp extract
  tnrpt pipeline work --db data/amp/tnrpt.db --data-dir data/amp parse --poll-interval 5s
  tnrpt pipeline work --db data/amp/tnrpt.db --data-dir data/amp all
  tnrpt pipeline work --db data/amp/tnrpt.db --data-dir data/amp extract --retry-failed`,
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			stage := args[0]

			if stage != "extract" && stage != "parse" && stage != "all" {
				return fmt.Errorf("invalid stage %q: must be extract, parse, or all", stage)
			}

			store, err := sqlite.NewSQLiteStoreWithConfig(sqlite.StoreConfig{Path: dbPath})
			if err != nil {
				return fmt.Errorf("open database: %w", err)
			}
			defer store.Close()

			worker := stages.NewWorkerService(store, dataDir, "")

			if retryFailed {
				return retryFailedJobs(ctx, store, stage)
			}

			if stage == "all" {
				return runAllStages(ctx, worker, pollInterval)
			}

			return runWorker(ctx, worker, stage, pollInterval)
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "path to SQLite database (required)")
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "data directory for file storage (required)")
	cmd.Flags().DurationVar(&pollInterval, "poll-interval", 0, "poll interval for continuous processing (0 = process once)")
	cmd.Flags().BoolVar(&retryFailed, "retry-failed", false, "reset failed jobs to queued and exit")
	cmd.MarkFlagRequired("db")
	cmd.MarkFlagRequired("data-dir")

	return cmd
}

func runWorker(ctx context.Context, worker *stages.WorkerService, stage string, pollInterval time.Duration) error {
	processed := 0
	failed := 0

	for {
		jobProcessed, err := worker.ProcessJob(ctx, stage)
		if err != nil {
			log.Printf("pipeline: work: %s: error: %v", stage, err)
			failed++
		}
		if jobProcessed {
			if err == nil {
				processed++
				log.Printf("pipeline: work: %s: processed job (total: %d)", stage, processed)
			}
		} else {
			if pollInterval == 0 {
				log.Printf("pipeline: work: %s: no more jobs (processed: %d, failed: %d)", stage, processed, failed)
				return nil
			}
			time.Sleep(pollInterval)
		}
	}
}

func runAllStages(ctx context.Context, worker *stages.WorkerService, pollInterval time.Duration) error {
	for _, stage := range []string{model.WorkStageExtract, model.WorkStageParse} {
		log.Printf("pipeline: work: processing %s stage", stage)
		if err := runWorker(ctx, worker, stage, 0); err != nil {
			return fmt.Errorf("%s: %w", stage, err)
		}
	}

	if pollInterval > 0 {
		log.Printf("pipeline: work: all stages complete, starting poll loop")
		for {
			for _, stage := range []string{model.WorkStageExtract, model.WorkStageParse} {
				_, err := worker.ProcessJob(ctx, stage)
				if err != nil {
					log.Printf("pipeline: work: %s: error: %v", stage, err)
				}
			}
			time.Sleep(pollInterval)
		}
	}

	return nil
}

func retryFailedJobs(ctx context.Context, store *sqlite.SQLiteStore, stage string) error {
	stages := []string{model.WorkStageExtract, model.WorkStageParse}
	if stage != "all" {
		stages = []string{stage}
	}

	total := 0
	for _, s := range stages {
		count, err := store.ResetFailedWork(ctx, s)
		if err != nil {
			return fmt.Errorf("reset failed %s jobs: %w", s, err)
		}
		if count > 0 {
			log.Printf("pipeline: work: reset %d failed %s jobs", count, s)
			total += count
		}
	}

	if total == 0 {
		log.Printf("pipeline: work: no failed jobs to reset")
	}
	return nil
}

func cmdVersion() *cobra.Command {
	showBuildInfo := false
	addFlags := func(cmd *cobra.Command) error {
		cmd.Flags().BoolVar(&showBuildInfo, "build-info", showBuildInfo, "show build information")
		return nil
	}
	var cmd = &cobra.Command{
		Use:   "version",
		Short: "display the application's version number",
		RunE: func(cmd *cobra.Command, args []string) error {
			if showBuildInfo {
				fmt.Println(tnrpt.Version().String())
				return nil
			}
			fmt.Println(tnrpt.Version().Core())
			return nil
		},
	}
	if err := addFlags(cmd); err != nil {
		log.Fatal(err)
	}
	return cmd
}

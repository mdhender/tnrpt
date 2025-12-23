// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"syscall"
	"time"

	"github.com/mdhender/tnrpt"
	"github.com/mdhender/tnrpt/adapters"
	"github.com/mdhender/tnrpt/model"
	"github.com/mdhender/tnrpt/parsers"
	"github.com/mdhender/tnrpt/pipelines/parsers/bistre"
	"github.com/mdhender/tnrpt/pipelines/parsers/docx"
	"github.com/mdhender/tnrpt/pipelines/parsers/report"
	"github.com/mdhender/tnrpt/walkers/anhinga"
	"github.com/mdhender/tnrpt/web/auth"
	"github.com/mdhender/tnrpt/web/handlers"
	webstore "github.com/mdhender/tnrpt/web/store"
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
	cmdRoot.AddCommand(cmdParse())
	cmdRoot.AddCommand(cmdPipeline())
	cmdRoot.AddCommand(cmdWalk())
	cmdRoot.AddCommand(cmdVersion())
	if err := addFlags(cmdRoot); err != nil {
		log.Fatal(err)
	}

	if err := cmdRoot.Execute(); err != nil {
		os.Exit(1)
	}
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

func cmdPipeline() *cobra.Command {
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
		Use:          "pipeline",
		Short:        "Run a pipeline",
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
			store, err := model.NewStore(ctx, ":memory:")
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

				sqliteStore, err := webstore.NewSQLiteStore()
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

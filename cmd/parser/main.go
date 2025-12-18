// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package main

import (
	"encoding/json"
	"fmt"

	"github.com/mdhender/tnrpt"
	"github.com/mdhender/tnrpt/adapters"
	"github.com/mdhender/tnrpt/renderer"
	"github.com/spf13/cobra"

	"log"
	"os"
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
		Use:   "ottoapp",
		Short: "OttoMap command runner",
		Long:  `OttoApp runs commands for OttoMap.`,
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
	var configFile string
	var excludeUnits []string
	var includeUnits []string
	var outputFile string
	addFlags := func(cmd *cobra.Command) error {
		cmd.Flags().BoolVar(&autoEOL, "auto-eol", true, "automatically convert line endings")
		cmd.Flags().StringVarP(&configFile, "config-file", "c", configFile, "load configuration from file")
		cmd.Flags().StringSliceVarP(&excludeUnits, "exclude", "e", excludeUnits, "exclude the unit")
		cmd.Flags().StringSliceVarP(&includeUnits, "include", "i", includeUnits, "include the unit")
		cmd.Flags().StringVarP(&outputFile, "output", "o", outputFile, "save parse to file")
		cmd.Flags().BoolVar(&autoEOL, "strip-cr", false, "strip CR from end-of-lines")
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

			r, err := renderer.New(args[0], quiet, verbose, debug)
			if err != nil {
				return err
			}
			pt, err := r.Run()
			if err != nil {
				return err
			}
			at, err := adapters.AdaptParserTurnToModel(pt)
			if err != nil {
				return err
			}
			if data, err := json.MarshalIndent(at, "", "  "); err != nil {
				log.Fatalf("json: %v\n", err)
			} else {
				log.Printf("turn: %s\n", string(data))
			}

			//ast, err := p.ParseInput()
			//if err != nil {
			//	return err
			//}
			//
			//data, err := json.MarshalIndent(ast, "", "  ")
			//if err != nil {
			//	return err
			//}
			//if outputFile == "" {
			//	log.Printf("%s\n", string(data))
			//} else if err = os.WriteFile(outputFile, data, 0o644); err != nil {
			//	return err
			//} else {
			//	log.Printf("%s: wrote %d bytes\n", outputFile, len(data))
			//}

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

// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package main

import (
	"encoding/json"
	"fmt"

	"github.com/mdhender/tnrpt"
	"github.com/mdhender/tnrpt/parser"
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
			if showVersion, _ := cmd.Flags().GetBool("show-version"); showVersion {
				fmt.Printf("tnrpt: version %q\n", tnrpt.Version().Core())
			}
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
	var configFile string
	var outputFile string
	addFlags := func(cmd *cobra.Command) error {
		cmd.Flags().StringVar(&configFile, "config-file", configFile, "load configuration from file")
		cmd.Flags().StringVar(&outputFile, "output", outputFile, "save parse to file")
		return nil
	}
	var cmd = &cobra.Command{
		Use:   "parse <turn-report-file>",
		Short: "parse a turn report file (text or Word)",
		Args:  cobra.ExactArgs(1), // require path to turn report file
		RunE: func(cmd *cobra.Command, args []string) error {
			if configFile != "" {
				return fmt.Errorf("error: --config-file is not implemented")
			}
			p, err := parser.New(args[0])
			if err != nil {
				return err
			}
			ast, err := p.Parse()
			if err != nil {
				return err
			}
			data, err := json.MarshalIndent(ast, "", "  ")
			if err != nil {
				return err
			}
			if outputFile == "" {
				fmt.Printf("%s\n", string(data))
			} else if err = os.WriteFile(outputFile, data, 0o644); err != nil {
				return err
			} else {
				fmt.Printf("%s: wrote %d bytes\n", outputFile, len(data))
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

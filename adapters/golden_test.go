// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package adapters_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/mdhender/tnrpt/adapters"
	"github.com/mdhender/tnrpt/parsers/azul"
)

var updateGolden = flag.Bool("update-golden", false, "update golden files")

func TestAzulParserTurnToModel_Golden(t *testing.T) {
	testCases := []struct {
		name       string
		inputFile  string
		goldenFile string
	}{
		{
			name:       "0899-12.0987",
			inputFile:  "0899-12.0987.report.txt",
			goldenFile: "0899-12.0987.azul.golden.json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			inputPath := filepath.Join(testdataPath, tc.inputFile)
			goldenPath := filepath.Join(testdataPath, tc.goldenFile)

			input, err := os.ReadFile(inputPath)
			if err != nil {
				t.Fatalf("read input: %v", err)
			}

			pt, err := azul.ParseInput(
				inputPath,          // fid
				"",                 // tid
				input,              // input
				false,              // acceptLoneDash
				false,              // debugParser
				false,              // debugSections
				false,              // debugSteps
				false,              // debugNodes
				false,              // debugFleetMovement
				false,              // experimentalUnitSplit
				false,              // experimentalScoutStill
				azul.ParseConfig{}, // cfg
			)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}

			at, err := adapters.AzulParserTurnToModel("<test-input>", pt)
			if err != nil {
				t.Fatalf("AzulParserTurnToModel: %v", err)
			}

			got, err := json.MarshalIndent(at, "", "  ")
			if err != nil {
				t.Fatalf("json.MarshalIndent: %v", err)
			}
			got = append(got, '\n')

			if *updateGolden {
				if err := os.WriteFile(goldenPath, got, 0644); err != nil {
					t.Fatalf("failed to update golden file: %v", err)
				}
				t.Logf("updated golden file: %s", goldenPath)
				return
			}

			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("failed to read golden file %q: %v\nRun with -update-golden to create it", goldenPath, err)
			}

			if !bytes.Equal(got, want) {
				t.Errorf("output differs from golden file %q\nRun with -update-golden to update", goldenPath)
				t.Errorf("got:\n%s", got)
				t.Errorf("want:\n%s", want)
			}
		})
	}
}

func TestBistreToModel_Golden(t *testing.T) {
	testCases := []struct {
		name       string
		inputFile  string
		goldenFile string
	}{
		{
			name:       "0899-12.0987",
			inputFile:  "0899-12.0987.report.txt",
			goldenFile: "0899-12.0987.bistre.golden.json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			inputPath := filepath.Join(testdataPath, tc.inputFile)
			goldenPath := filepath.Join(testdataPath, tc.goldenFile)

			input, err := os.ReadFile(inputPath)
			if err != nil {
				t.Fatalf("read input: %v", err)
			}

			pt, err := azul.ParseInput(
				inputPath,          // fid
				"",                 // tid
				input,              // input
				false,              // acceptLoneDash
				false,              // debugParser
				false,              // debugSections
				false,              // debugSteps
				false,              // debugNodes
				false,              // debugFleetMovement
				false,              // experimentalUnitSplit
				false,              // experimentalScoutStill
				azul.ParseConfig{}, // cfg
			)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}

			rf, rx, err := adapters.BistreToModel("<test-input>", pt)
			if err != nil {
				t.Fatalf("BistreToModel: %v", err)
			}

			// Zero out timestamps for deterministic output
			zeroTime := time.Time{}
			rf.CreatedAt = zeroTime
			rx.CreatedAt = zeroTime

			// Sort units by UnitID for deterministic output
			sort.Slice(rx.Units, func(i, j int) bool {
				return rx.Units[i].UnitID < rx.Units[j].UnitID
			})

			// Combine into a single output structure for golden comparison
			output := struct {
				ReportFile any `json:"reportFile"`
				ReportX    any `json:"reportX"`
			}{
				ReportFile: rf,
				ReportX:    rx,
			}

			got, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				t.Fatalf("json.MarshalIndent: %v", err)
			}
			got = append(got, '\n')

			if *updateGolden {
				if err := os.WriteFile(goldenPath, got, 0644); err != nil {
					t.Fatalf("failed to update golden file: %v", err)
				}
				t.Logf("updated golden file: %s", goldenPath)
				return
			}

			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("failed to read golden file %q: %v\nRun with -update-golden to create it", goldenPath, err)
			}

			if !bytes.Equal(got, want) {
				t.Errorf("output differs from golden file %q\nRun with -update-golden to update", goldenPath)
				t.Errorf("got:\n%s", got)
				t.Errorf("want:\n%s", want)
			}
		})
	}
}

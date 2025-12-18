// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package adapters_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/mdhender/tnrpt/adapters"
	"github.com/mdhender/tnrpt/renderer"
)

var updateGolden = flag.Bool("update-golden", false, "update golden files")

func TestAdaptParserTurnToModel_Golden(t *testing.T) {
	testCases := []struct {
		name       string
		inputFile  string
		goldenFile string
	}{
		{
			name:       "0899-12.0987",
			inputFile:  "0899-12.0987.report.txt",
			goldenFile: "0899-12.0987.adapter.golden.json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			inputPath := filepath.Join(testdataPath, tc.inputFile)
			goldenPath := filepath.Join(testdataPath, tc.goldenFile)

			r, err := renderer.New(inputPath, quiet, verbose, debug)
			if err != nil {
				t.Fatalf("renderer.New: %v", err)
			}

			pt, err := r.Run()
			if err != nil {
				t.Fatalf("renderer.Run: %v", err)
			}

			at, err := adapters.AdaptParserTurnToModel("<test-input>", pt)
			if err != nil {
				t.Fatalf("AdaptParserTurnToModel: %v", err)
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

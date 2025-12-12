// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/mdhender/tnrpt"
)

func main() {
	log.SetFlags(log.Lshortfile)

	for _, file := range []string{
		"0899-12.0987.report.txt",
		"0900-01.0987.report.txt",
	} {
		started := time.Now()
		if err := scan("testdata", file, false); err != nil {
			fmt.Printf("%s: failed %v\n", file, err)
			continue
		}
		fmt.Printf("%s: completed in %v\n", file, time.Since(started))
	}
}

func scan(path, file string, unknownOnly bool) error {
	input, err := os.ReadFile(filepath.Join("testdata", file))
	if err != nil {
		return err
	}
	s := tnrpt.NewLexer(context.Background(), file, input, slog.Default())
	tokenCounter, maxTokens := 0, len(input)+1
	for tokenCounter < maxTokens {
		tok := s.Scan()
		if tok == nil {
			panic("assert(s.scan != nil)")
		}
		tokenCounter++
		logToken := tok.Kind == tnrpt.UNKNOWN || !unknownOnly
		if logToken {
			fmt.Printf("%-35s %5d %-20s %q\n", fmt.Sprintf("%s:%d:%d:", file, tok.Line, tok.Column), tokenCounter, tok.Kind, tok.Lexeme(input))
		}
		if tok.Kind == tnrpt.EndOfInput {
			break
		}
	}

	return nil
}

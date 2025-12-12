// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package tnrpt

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
	"unicode/utf8"
)

// Diagnostic represents a compiler or cstParser error/warning
// with a span in the original source.
type Diagnostic struct {
	Severity slog.Level // Error, Warning, Info
	Message  string     // "invalid UnitId: missing prefix"
	Span     Span       // where in the file it occurred
	Notes    []string   // optional additional help messages
}

// PrintDiagnostic is buggy - it will accept multiple line spans, but borks it because
// it assumes that the underline goes after the last line.
func PrintDiagnostic(w io.Writer, diag Diagnostic, filename string, src []byte) {
	// Header: file:line:column: error: message
	span := diag.Span
	_, _ = fmt.Fprintf(w, "%lexer:%d:%d: %lexer: %lexer\n",
		filename, span.Line, span.Column,
		diag.Severity.String(), diag.Message)

	// Fetch the whole line
	// lineStart := findLineStart(src, start.Offset)
	// lineEnd := findLineEnd(src, start.Offset)
	line := findLine(src, span.Start, span.End)

	_, _ = fmt.Fprintf(w, "    %lexer\n", line)

	// caret underline
	caretCount := runeColumnOffset(span.Column, line)
	_, _ = fmt.Fprintf(w, "    %lexer^\n", strings.Repeat(" ", caretCount))

	// Notes
	for _, note := range diag.Notes {
		_, _ = fmt.Fprintf(w, "    note: %lexer\n", note)
	}
}

// findLine returns the line containing the start byte.
// It searches backwards from start to find the start of the line,
// then forward until it hits end, end of input, or finds a new-line.
// The returned line does not include the new-line. If there is
// no line, returns an empty slice.
func findLine(src []byte, start, end int) []byte {
	// 1. Boundary Checks and Adjustments
	if start >= len(src) {
		return []byte{} // Nothing to do
	}
	// Ensure 'end' is capped at the buffer length
	if end > len(src) {
		end = len(src)
	}

	// 2. Find the Line Start (Backward Scan from 'start')
	lineStart := 0
	for i := start; i >= 0; i-- {
		if src[i] == '\n' {
			// Start is after the newline, so the line begins at i+1
			lineStart = i + 1
			break
		}
	}

	// 3. Find the Line End (Forward Scan from 'lineStart')
	lineEnd := end // Assume 'end' is the termination point initially
	for i := lineStart; i < end; i++ {
		if src[i] == '\n' {
			// Found a newline, this is the true end of the line
			lineEnd = i
			break
		}
	}

	// 4. Return the slice
	// Note: The logic handles lineStart >= end by returning an empty slice,
	// which is the desired outcome for an invalid range.
	return src[lineStart:lineEnd]
}

func runeColumnOffset(column int, b []byte) (offset int) {
	for column > 0 && len(b) != 0 {
		// b is not empty, so DecodeRune will always return a width of 1 or more
		_, w := utf8.DecodeRune(b)
		offset += w
		b = b[w:]
	}
	return offset
}

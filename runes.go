// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package tnrpt

import (
	"unicode"
	"unicode/utf8"
)

const (
	// CR and LF are control characters, respectively coded 0x0D (13 decimal) and 0x0A (10 decimal).
	// Windows uses CR + LF, Unix/Mac uses LF, Classic Mac uses CR.
	// This package doesn't support Classic Mac, so stray CR characters are treated as spaces.

	// CR is 0x0D or '\r'
	CR rune = rune(13)

	// LF is 0x0A or '\n'
	LF rune = rune(10)

	// EOF is a sentinel for end of input
	EOF rune = rune(-1)
)

func init() {
	for _, ch := range []byte{' ', '\t', '\r'} {
		delimiters[ch] = true
	}
	for _, ch := range []byte{0, '\n', '\'', '"', '.', ',', '(', ')', '#', '+', '-', '*', '/', '=', '\\', '$', ':'} {
		delimiters[ch] = true
	}
}

var (
	delimiters = [256]bool{}
)

func isdelimiter(ch rune) bool {
	if 0 <= ch && ch <= utf8.RuneSelf {
		return delimiters[byte(ch)]
	}
	return false
}

func isspace(ch rune) bool {
	return ch != '\n' && unicode.IsSpace(ch)
}

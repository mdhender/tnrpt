// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package tnrpt

import (
	"context"
	"fmt"
	"log/slog"
	"unicode"
	"unicode/utf8"
)

// Lexer invariants and coordinate system
//
// The lexer treats input as an immutable UTF-8 byte slice.
//
// Fields:
//   input       - the original []byte
//   length      - len(input)
//
//   r           - the current rune, or EOF when we have read past the end.
//                 Line endings are normalized so that:
//                   * "\n"   (LF) stays "\n"
//                   * "\r\n" (CRLF) is seen as a single "\n" rune
//                   * stray "\r" is treated as a space-like rune by callers
//
//   posCurrRune - index into input of the first byte of r,
//                 or length when r == EOF.
//   posNextRune - index into input of the first byte of the *next* rune,
//                 or length when r == EOF.
//   posAnchor   - index into input where the current token starts.
//
// Invariants (must always hold):
//   0 <= posCurrRune <= posNextRune <= length
//
//   r == EOF  <=> posCurrRune == posNextRune == length
//
//   r != EOF  => posCurrRune < length && posNextRune > posCurrRune
//                and input[posCurrRune:posNextRune] encodes exactly r.
//
// `advance`:
//
//   - On entry, (r, posCurrRune, posNextRune) describe the current rune.
//   - On exit:
//       * if there is another rune:
//             r, posCurrRune, posNextRune describe that next rune;
//       * if there is no rune:
//             r == EOF, posCurrRune == posNextRune == length, endToken != nil.
//   - `advance` also updates line/col, counting LF as a line break and
//     normalizing CRLF to a single LF rune.
//
// Anchors and token spans:
//
//   - `setAnchor` sets anchorPos = posCurrRune. It MUST be called when
//     r is the first rune of the token (or, for zero-length tokens, when
//     posCurrRune is the desired start index).
//
//   - Scanners that produce a token should:
//       1. Check that the current rune is a valid start for that token;
//          if not, return nil.
//       2. Call setAnchor().
//       3. Repeatedly call advance() while r belongs to the token.
//          When the loop stops, r is the *first rune after the token*
//          (or EOF), and posCurrRune points at the first byte of that rune.
//       4. Slice the token'lexer value as input[anchorPos:posCurrRune].
//
//   - Helpers like copyFromAnchor must use posCurrRune (the index of the
//     first byte after the token) as the upper bound, so they never steal
//     bytes from the next token.

type Lexer struct {
	name        string // name of the input source
	r           rune   // current rune
	line        int    // line number of current rune
	column      int    // column number of current rune
	posCurrRune int    // position of current rune
	posNextRune int    // position of next rune
	length      int    // length of input buffer
	input       []byte

	anchorPos    int
	anchorLine   int
	anchorColumn int

	// returns a canonical end of input token to preserve
	// spaces at the end of the file
	endToken *Token

	// logging
	ctx        context.Context
	logger     *slog.Logger
	errorCount int
	tokenCount int
}

func NewLexer(ctx context.Context, path string, input []byte, logger *slog.Logger) *Lexer {
	l := &Lexer{
		name:   path,
		input:  input,
		length: len(input),
		line:   1,
		column: 1,
		ctx:    ctx,
		logger: logger,
	}
	// read the first character to initialize the lexer.
	l.advance()
	return l
}

// Scan returns the next token from the input buffer.
// Leading spaces are put in the token'lexer leading trivia.
//
// Once we reach end of input, we always return the same EOF token.
func (l *Lexer) Scan() *Token {
	if l.iseof() {
		if l.endToken == nil {
			l.seteof(nil)
		}
		return l.endToken
	}

	l.setAnchor()

	// capture leading spaces
	var leadingTrivia []*Token
	if l.scanSpaces() == SPACE {
		leadingTrivia = []*Token{&Token{
			Position: Position{
				Line:   l.anchorLine,
				Column: l.anchorColumn,
				Start:  l.anchorPos,
			},
			End:  l.posNextRune,
			Kind: SPACE,
		}}
		// check for end of input since we consumed some of the input
		if l.iseof() {
			//log.Printf("lexer: found end of input after spaces\n")
			l.seteof(leadingTrivia)
			return l.endToken
		}
		// reset the anchor
		l.setAnchor()
	}

	// next token should be end of line, a delimiter, or text,
	// but there might be bugs in advance or the scanner, so
	// we'll accept unknown tokens, too.
	switch l.peekChar() {
	case LF:
		l.advance()
		return &Token{
			Position: Position{
				Line:   l.anchorLine,
				Column: l.anchorColumn,
				Start:  l.anchorPos,
			},
			End:           l.posCurrRune,
			Kind:          EndOfLine,
			LeadingTrivia: leadingTrivia,
		}
	case '\\':
		l.advance()
		return &Token{
			Position: Position{
				Line:   l.anchorLine,
				Column: l.anchorColumn,
				Start:  l.anchorPos,
			},
			End:           l.posCurrRune,
			Kind:          BACKSLASH,
			LeadingTrivia: leadingTrivia,
		}
	case ':':
		l.advance()
		return &Token{
			Position: Position{
				Line:   l.anchorLine,
				Column: l.anchorColumn,
				Start:  l.anchorPos,
			},
			End:           l.posCurrRune,
			Kind:          COLON,
			LeadingTrivia: leadingTrivia,
		}
	case ',':
		l.advance()
		return &Token{
			Position: Position{
				Line:   l.anchorLine,
				Column: l.anchorColumn,
				Start:  l.anchorPos,
			},
			End:           l.posCurrRune,
			Kind:          COMMA,
			LeadingTrivia: leadingTrivia,
		}
	case '-':
		l.advance()
		return &Token{
			Position: Position{
				Line:   l.anchorLine,
				Column: l.anchorColumn,
				Start:  l.anchorPos,
			},
			End:           l.posCurrRune,
			Kind:          DASH,
			LeadingTrivia: leadingTrivia,
		}
	case '.':
		l.advance()
		return &Token{
			Position: Position{
				Line:   l.anchorLine,
				Column: l.anchorColumn,
				Start:  l.anchorPos,
			},
			End:           l.posCurrRune,
			Kind:          DOT,
			LeadingTrivia: leadingTrivia,
		}
	case '=':
		l.advance()
		return &Token{
			Position: Position{
				Line:   l.anchorLine,
				Column: l.anchorColumn,
				Start:  l.anchorPos,
			},
			End:           l.posCurrRune,
			Kind:          EQUALS,
			LeadingTrivia: leadingTrivia,
		}
	case '#':
		l.advance()
		if l.peekChar() == '#' {
			l.advance()
			return &Token{
				Position: Position{
					Line:   l.anchorLine,
					Column: l.anchorColumn,
					Start:  l.anchorPos,
				},
				End:           l.posCurrRune,
				Kind:          Grid,
				LeadingTrivia: leadingTrivia,
			}
		}
		return &Token{
			Position: Position{
				Line:   l.anchorLine,
				Column: l.anchorColumn,
				Start:  l.anchorPos,
			},
			End:           l.posCurrRune,
			Kind:          HASH,
			LeadingTrivia: leadingTrivia,
		}
	case '(':
		l.advance()
		return &Token{
			Position: Position{
				Line:   l.anchorLine,
				Column: l.anchorColumn,
				Start:  l.anchorPos,
			},
			End:           l.posCurrRune,
			Kind:          LEFTPAREN,
			LeadingTrivia: leadingTrivia,
		}
	case ')':
		l.advance()
		return &Token{
			Position: Position{
				Line:   l.anchorLine,
				Column: l.anchorColumn,
				Start:  l.anchorPos,
			},
			End:           l.posCurrRune,
			Kind:          RIGHTPAREN,
			LeadingTrivia: leadingTrivia,
		}
	case '\'':
		l.advance()
		return &Token{
			Position: Position{
				Line:   l.anchorLine,
				Column: l.anchorColumn,
				Start:  l.anchorPos,
			},
			End:           l.posCurrRune,
			Kind:          SINGLEQUOTE,
			LeadingTrivia: leadingTrivia,
		}
	case '/':
		l.advance()
		return &Token{
			Position: Position{
				Line:   l.anchorLine,
				Column: l.anchorColumn,
				Start:  l.anchorPos,
			},
			End:           l.posCurrRune,
			Kind:          SLASH,
			LeadingTrivia: leadingTrivia,
		}
	}

	if l.scanText() == Text {
		return &Token{
			Position: Position{
				Line:   l.anchorLine,
				Column: l.anchorColumn,
				Start:  l.anchorPos,
			},
			End:           l.posCurrRune,
			Kind:          Text,
			LeadingTrivia: leadingTrivia,
		}
	}

	// accept the next character as an unknown token.
	l.advance()
	return &Token{
		Position: Position{
			Line:   l.anchorLine,
			Column: l.anchorColumn,
			Start:  l.anchorPos,
		},
		End:           l.posCurrRune,
		Kind:          UNKNOWN,
		LeadingTrivia: leadingTrivia,
	}
}

// scanSpaces accepts a run of spaces and returns SPACE.
func (l *Lexer) scanSpaces() Kind {
	if l.peekChar() == LF || !unicode.IsSpace(l.peekChar()) {
		return UNKNOWN
	}

	// consume run of spaces (not including LF)
	for unicode.IsSpace(l.peekChar()) && l.peekChar() != LF {
		l.advance()
	}

	return SPACE
}

// scanText accepts a run of any non-space, non-delimiter and returns Text.
func (l *Lexer) scanText() Kind {
	if unicode.IsSpace(l.peekChar()) || isdelimiter(l.peekChar()) {
		return UNKNOWN
	}

	// consume run of non-space, non-delimiter characters
	for !l.iseof() && !unicode.IsSpace(l.peekChar()) && !isdelimiter(l.peekChar()) {
		l.advance()
	}

	return Text
}

// peekChar returns the current character without advancing the input.
func (l *Lexer) peekChar() rune {
	return l.r
}

// peekCharN returns the nth character without advancing the input.
// peekCharN(0) is the same as peek().
func (l *Lexer) peekCharN(numberOfChars int) rune {
	if numberOfChars < 0 {
		panic("assert(numberOfChars >= 0)")
	}
	ch := l.r

	posPeekRune := l.posNextRune
	for numberOfChars > 0 && posPeekRune < l.length {
		r, w := rune(l.input[posPeekRune]), 1
		if r == LF {
			ch, w = LF, 1
		} else if r == CR && posPeekRune+1 < l.length && rune(l.input[posPeekRune+1]) == LF {
			ch, w = LF, 2
		} else if r >= utf8.RuneSelf {
			// The current rune is not actually ASCII, so we have to decode it properly.
			ch, w = utf8.DecodeRune(l.input[posPeekRune:])
		} else {
			ch = r
		}
		posPeekRune += w
		numberOfChars--
	}

	if numberOfChars > 0 {
		// we reached end of input before peeking the requested number of characters
		ch = EOF
	}

	return ch
}

// setAnchor marks the start of the current token.
func (s *Lexer) setAnchor() {
	s.anchorPos = s.posCurrRune
	s.anchorLine = s.line
	s.anchorColumn = s.column
}

// advance moves to the next rune and updates line/col.
// It normalizes "\r\n" into a single LF rune.
// On end of input, it sets r == EOF and both positions to length and returns.
func (l *Lexer) advance() {
	// already at or past the end?
	if l.posNextRune >= l.length {
		// do we need to update the last rune'lexer location?
		if l.r == LF {
			l.line++
			l.column = 1
		} else if l.r != EOF {
			l.column++
		}
		l.posCurrRune, l.posNextRune = l.length, l.length
		l.r = EOF
		return
	}

	// update line/col wrt the *current* rune before stepping
	if l.r == LF {
		l.line++
		l.column = 1
	} else {
		l.column++
	}

	l.posCurrRune = l.posNextRune

	// read the next rune, optimizing for ASCII grammars.
	r, w := rune(l.input[l.posCurrRune]), 1
	if r == LF {
		r, w = LF, 1
	} else if r == CR && l.posCurrRune+1 < l.length && rune(l.input[l.posCurrRune+1]) == LF {
		// merge CR+LF into a single LF rune, but consume both bytes
		r, w = LF, 2
	} else if r >= utf8.RuneSelf {
		// the current rune must be decoded
		r, w = utf8.DecodeRune(l.input[l.posCurrRune:])
	}
	l.posNextRune = l.posCurrRune + w
	l.r = r
}

func (l *Lexer) iseof() bool {
	return l.r == EOF
}

func (l *Lexer) debug(format string, args ...any) {
	if l.logger == nil {
		return
	}
	l.logger.Debug(fmt.Sprintf("%lexer:%d:%d %lexer\n", l.name, l.line, l.column, fmt.Sprintf(format, args...)))
}

func (l *Lexer) error(format string, args ...any) {
	l.errorCount++
	if l.logger == nil {
		return
	}
	l.logger.Error(fmt.Sprintf("%lexer:%d:%d %lexer\n", l.name, l.line, l.column, fmt.Sprintf(format, args...)))
}

func (l *Lexer) info(format string, args ...any) {
	if l.logger == nil {
		return
	}
	l.logger.Info(fmt.Sprintf("%lexer:%d:%d %lexer\n", l.name, l.line, l.column, fmt.Sprintf(format, args...)))
}

// seteof updates the Lexer state to enforce the end of input invariants:
// * r is EOF
// * posCurrRune = posNextRune = length
// * endToken is set to the canonical EOF token
func (l *Lexer) seteof(leadingTrivia []*Token) {
	l.r = EOF
	l.posCurrRune = l.length
	l.posNextRune = l.length
	if l.endToken == nil {
		l.endToken = &Token{
			Position: Position{
				Line:   l.line,
				Column: l.column,
				Start:  l.posNextRune,
			},
			End:           l.posCurrRune,
			Kind:          EndOfInput,
			LeadingTrivia: leadingTrivia,
		}
	}
}

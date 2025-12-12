// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package tnrpt

import (
	"context"
	"log/slog"
)

/*
Invariants:
 * Initialization
   * `cst.Parse(ctx, lexer, logger)` constructs a `cstParser`, which:
      Stores `lexer` in `p.lexer`
      Primes `p.currToken` with the first token from the lexer (including trivia handling) before the top-level parse begins
   * After initialization, `p.currToken` is never `nil` (once EOF is reached, will always be EOF)

 * lexer is private:
   * Only the cstParser’lexer internal methods (`nextTokenFromSource` / `scanWithTrivia`) ever call `p.lexer.Scan()` and `p.lexer.Token()`.
   * All parsing code uses `advance()` / `peek()` only.
   * currToken is the lookahead:
     peek() → returns the token to be consumed next.
     advance() → returns the current token and moves lookahead forward.
   * Whitespace (Spaces) is never returned as a main token:
     Instead, it’lexer accumulated into LeadingTrivia of the next non-space token.
   * EOL & EOF are real tokens, not trivia:
     They are returned from advance() / peek().
     They may have LeadingTrivia with spaces.
   * LeadingTrivia is a flat slice of *Token, all with Kind == Spaces.

 * Token cursor semantics
   * Meaning of `currToken`:
     * `currToken` is the lookahead token, i.e. the “current” token that `peek()` returns and `advance()` will consume next
   * `peek()`:
      * Returns `p.currToken`.
      * Never changes any cstParser state (pure read).
   * `advance()`:
      * Returns the current token (`old := currToken`).
      * Updates `currToken` to the next token in the stream:
        * Prefer tokens in `p.q` (if not empty).
        * Otherwise, get the next token from the lexer (with trivia folding).
        * Does not skip EOL or EOF.
          (Those are real tokens; they can be sync points.)
   * Once EOF has been produced:
     * `currToken` points to an EOF token.
     * `peek()` always returns EOF.
     * `advance()` keeps returning EOF forever (keeps `currToken` at EOF)

 * Queue semantics
   * Queue is small and only used for lookahead/backtracking:
   * `p.q` is only read via nextTokenFromSource.
     * Queue meaning
       * `p.q` is a FIFO of tokens that should be seen after the current `currToken` but before scanning more.
       * `nextTokenFromSource()` consumes from the front of `p.q` if available, otherwise calls the lexer.
       * Any token placed into `p.q`:
         * Has already had its trivia set up (i.e. has correct `LeadingTrivia`).

 * Trivia semantics
   * lexer never returns tokens with leading or trailing trivia set
   * EOL and EOF are not trivia
   * Spaces are trivia only
   * Tokens of kind `EOL` and `EOF`:
     * Are returned as normal tokens (via `advance` / `peek`)
     * Are not merged into whitespace trivia
     * May have `LeadingTrivia` containing only `Spaces`
   * Tokens with kind `Spaces`:
     * Are never returned as `currToken` or via `advance()` / `peek()`
     * Are only found in `LeadingTrivia` (and possibly `TrailingTrivia` if nodes inject trivia).
   * LeadingTrivia shape
     * For every non-space token (including EOL and EOF):
       * `LeadingTrivia` is a **flat slice** of whitespace tokens (kind `Spaces`) that immediately precede that token.
       * No non-whitespace tokens appear in `LeadingTrivia`.

That gives us deterministic behavior for building the CST.
*/

type cstParser struct {
	ctx       context.Context
	logger    *slog.Logger
	lexer     Lexer
	q         []*Token // simple slice-based deque of upcoming tokens
	currToken *Token   // current lookahead
	eofToken  *Token   // canonical EOF token (optional but nice)
}

// ParseCST parses the token stream and returns a CST.
func ParseCST(lexer Lexer) *TurnReportNode {
	p := newCSTParser(context.Background(), lexer, slog.Default())
	_ = p
	return nil
}

// newCSTParser constructor returns an initialized CST parser
func newCSTParser(ctx context.Context, lexer Lexer, logger *slog.Logger) *cstParser {
	p := &cstParser{
		ctx:    context.Background(),
		logger: slog.Default(),
		lexer:  lexer,
	}
	// Prime the cursor with the first token.
	// After this, currToken is never nil (except in truly exceptional cases).
	p.currToken = p.nextTokenFromSource()
	return p
}

// Notes:
// * `pushFront` is O(n) because of the `copy`, but for typical cstParser lookahead queues this is totally fine.
// * `popFront` is O(1) because we just reslice instead of shifting the whole slice.
// * `pushBack` / `popBack` are O(1) amortized.

// pushBack adds a token at the back (right side).
func (p *cstParser) pushBack(tok *Token) {
	p.q = append(p.q, tok)
}

// pushFront adds a token at the front (left side).
func (p *cstParser) pushFront(tok *Token) {
	p.q = append(p.q, nil)
	copy(p.q[1:], p.q[:len(p.q)-1])
	p.q[0] = tok
}

// popFront removes and returns the front token.
// Returns nil if the queue is empty.
func (p *cstParser) popFront() *Token {
	if len(p.q) == 0 {
		return nil
	}
	tok := p.q[0]
	p.q[0] = nil // clear the token from for later GC
	p.q = p.q[1:]
	return tok
}

// nextTokenFromSource returns the next token from queue or lexer.
// (This is where the queue gets first dibs; otherwise we go to the lexer.)
func (p *cstParser) nextTokenFromSource() *Token {
	// 1. From queue if available.
	if tok := p.popFront(); tok != nil {
		return tok
	}
	// 2. From lexer.
	return p.scan()
}

// scan returns a token, ensuring that:
// * Spaces are never returned as main
// * EOL and EOF are real tokens (we don’t special-case them; they just aren’t `SPACE`).
// * LeadingTrivia is flat and only contains SPACE
//
// NB: this should be the only parser function that communicates with the lexer!
func (p *cstParser) scan() *Token {
	if p.eofToken != nil {
		return p.eofToken
	}
	tok := p.lexer.Scan()
	if tok == nil {
		// Defensive: if lexer ever returns nil.
		panic("assert(scan.token != nil)")
	}
	if tok.Kind == EndOfInput {
		// Canonical EOF: save the EOF token so that we only ever have one instance of it.
		// this ensures that we don't lose any leading trivia before EOF.
		p.eofToken = tok
	}
	return tok
}

// Invariants for peek() and advance():
//  * `peek()` == “current lookahead”.
//  * `advance()` == “return lookahead, then load next”.
//  * Queue is respected (`nextTokenFromSource`).
//  * Trivia is always applied consistently by `scan`.
//  * Spaces are only trivia; EOL/EOF are sync tokens with optional whitespace in `LeadingTrivia`.

// peek returns the current lookahead token without consuming it.
// After newCSTParser, this should never be nil under normal operation.
func (p *cstParser) peek() *Token {
	return p.currToken
}

// advance consumes and returns the current token, then updates the lookahead.
//
// Invariants:
//
//	-- currToken is always the token you’ll get if you call peek().
//	-- advance() always:
//	  -- Returns the current currToken.
//	  -- Updates currToken to the next token from queue or lexer.
//	-- EOF is returned repeatedly but the cursor doesn’t move past it (simple and parser-friendly).
//
// Will panic if the parser is not primed (somehow newCSTParser was bypassed).
func (p *cstParser) advance() *Token {
	// Safety net: if someone constructed cstParser manually and forgot to prime it.
	if p.currToken == nil {
		panic("assert(cstParser.currToken != nil)")
	}

	tok := p.currToken

	// If we've already reached EOF, stay there forever.
	if tok.Kind == EndOfInput {
		// Keep currToken at EOF so peek/advance are stable.
		return tok
	}

	// Otherwise, move to the next token.
	p.currToken = p.nextTokenFromSource()

	return tok
}

// match reports whether the current lookahead token matches the given kind.
func (p *cstParser) match(kind Kind) bool {
	tok := p.peek()
	if tok == nil {
		return false
	}
	return tok.Is(kind)
}

// matchOneOf reports whether the current lookahead token's Kind matches
// any of the provided kinds.
func (p *cstParser) matchOneOf(kinds ...Kind) bool {
	tok := p.peek()
	if tok == nil {
		return false
	}
	return tok.IsOneOf(kinds...)
}

// consume advances over the current token if its Kind equals kind.
// It returns true if a token was consumed, false otherwise.
func (p *cstParser) consume(kind Kind) bool {
	if p.match(kind) {
		p.advance()
		return true
	}
	return false
}

// consumeOneOf advances over the current token if its Kind matches any of
// the provided kinds. It returns true if a token was consumed, false otherwise.
func (p *cstParser) consumeOneOf(kinds ...Kind) bool {
	if p.matchOneOf(kinds...) {
		p.advance()
		return true
	}
	return false
}

// accept consumes and returns the current token if its Kind equals kind.
// It returns nil if the current token does not match.
func (p *cstParser) accept(kind Kind) *Token {
	if p.match(kind) {
		return p.advance()
	}
	return nil
}

// acceptOneOf consumes and returns the current token if its Kind matches
// any of the provided kinds. It returns nil if there is no match.
func (p *cstParser) acceptOneOf(kinds ...Kind) *Token {
	if p.matchOneOf(kinds...) {
		return p.advance()
	}
	return nil
}

// expect consumes and returns the current token if its Kind equals kind.
// If the current token does not match, it records an error and returns a
// synthesized error token (or whatever errorExpected produces).
func (p *cstParser) expect(kind Kind) *Token {
	tok := p.accept(kind)
	if tok == nil {
		// errorExpected should probably record the diagnostic and either
		// return a synthesized error token or the current token.
		return p.errorExpected(kind)
	}
	return tok
}

// expectOneOf consumes and returns the current token if its Kind matches
// any of the provided kinds. If there is no match, it records an error
// and returns a synthesized error token.
func (p *cstParser) expectOneOf(kinds ...Kind) *Token {
	tok := p.acceptOneOf(kinds...)
	if tok == nil {
		return p.errorExpectedOneOf(kinds...)
	}
	return p.advance()
}

// skipUntilSync consumes tokens until it reaches either EOF or a token whose
// Kind matches any of the provided sync kinds. The sync token itself is not
// consumed. It returns the list of skipped tokens.
//
// Typical sync tokens are EOL, EOF, and section headers.
//
// Panics if p.advance() returns nil since that'lexer a bug in the cstParser, not an
// issue with the input or grammar.
func (p *cstParser) skipUntilSync(kinds ...Kind) []*Token {
	var list []*Token
	for !p.isAtEnd() && !p.matchOneOf(kinds...) {
		tok := p.advance()
		if tok == nil {
			panic("assert(p.advance() != nil)")
		}
		list = append(list, tok)
	}
	return list
}

// isAtEnd reports whether the cstParser has reached EOF. It returns true if
// the current lookahead token is EOF. If currToken is nil (cstParser not
// initialized correctly), it returns false.
func (p *cstParser) isAtEnd() bool {
	return p.currToken != nil && p.currToken.Kind == EndOfInput
}

func (p *cstParser) errorExpected(kind Kind) *Token {
	panic("!implemented")
}

func (p *cstParser) errorExpectedOneOf(kinds ...Kind) *Token {
	panic("!implemented")
}

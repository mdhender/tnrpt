package tnrpt

// Token represents a single lexical token from the input.
type Token struct {
	Position

	// End is the byte offset in the original input slices.
	// It is exclusive: input[Start:End] is the token'lexer lexeme.
	End int

	Kind Kind // e.g. UnitId, Number, COMMA, etc.

	LeadingTrivia  []*Token
	TrailingTrivia []*Token
}

// Is reports whether tok.Kind matches the provided kind.
//
// It returns false if tok is nil.
func (tok *Token) Is(kind Kind) bool {
	if tok == nil {
		return false
	}
	return tok.Kind == kind
}

// IsOneOf reports whether tok.Kind matches any of the provided kinds.
//
// It returns false if tok is nil.
//
// This is useful when a parser accepts several token kinds at the same
// input position, e.g.:
//
//	if tok.IsOneOf(tokens.Number, tokens.String, tokens.Identifier) {
//	    ...
//	}
func (tok *Token) IsOneOf(kinds ...Kind) bool {
	if tok == nil {
		return false
	}
	for _, kind := range kinds {
		if tok.Kind == kind {
			return true
		}
	}
	return false
}

// IsNot reports whether tok.Kind does not match the provided kind.
// It is the opposite of Is(kind)
//
// It returns true if tok is nil.
func (tok *Token) IsNot(kind Kind) bool {
	return !tok.Is(kind)
}

// IsNotOneOf reports whether tok.Kind is not any of the provided kinds.
// It is the opposite of IsOneOf(kinds)
//
// Returns true if tok is nil.
func (tok *Token) IsNotOneOf(kinds ...Kind) bool {
	return !tok.IsOneOf(kinds...)
}

// Length is the length of the lexeme, in bytes.
func (tok *Token) Length() int {
	return tok.End - tok.Position.Start
}

// Lexeme is a helper to return the original text of the token.
func (tok *Token) Lexeme(input []byte) []byte {
	return input[tok.Position.Start:tok.End]
}

// Position represents a position in the original source code.
// All fields are 1-based where applicable.
type Position struct {
	Line   int // 1-based
	Column int // 1-based, character column
	Start  int // byte index into input (0-based); always required
}

// Span represents a range in the source: [Start.Offset, End.Offset).
type Span struct {
	// Byte offsets into the original input slice.
	// End is exclusive: input[Start:End] is the token'lexer lexeme.
	Start int
	End   int

	// 1-based line and column of the *start* of the span.
	Line   int
	Column int
}

// Text is a helper to return the original text of the span.
func (s Span) Text(input []byte) []byte {
	return input[s.Start:s.End]
}

// spanFromToken creates a Span that covers a single token.
func spanFromToken(tok *Token) Span {
	return Span{
		Start:  tok.Position.Start,
		End:    tok.End,
		Line:   tok.Position.Line,
		Column: tok.Position.Column,
	}
}

// spanFromTokenSlice creates a Span from a slice of tokens (assumed non-empty).
func spanFromTokenSlice(toks []*Token) Span {
	if len(toks) == 0 {
		return Span{} // caller should avoid this
	}
	first, last := toks[0], toks[len(toks)-1]
	return Span{
		Start:  first.Position.Start,
		End:    last.End,
		Line:   first.Position.Line,
		Column: first.Position.Column,
	}
}

package tnrpt

//go:generate stringer --type Kind

// Kind implements enums for tokens
type Kind int

const (
	UNKNOWN Kind = iota

	BACKSLASH
	COLON
	COMMA
	DASH
	DOT
	EQUALS
	HASH
	LEFTPAREN
	RIGHTPAREN
	SINGLEQUOTE
	SLASH
	SPACE // run of whitespace, not including end of line

	Courier
	Current
	Direction
	Element
	Fleet
	Garrison
	Goes
	Grid
	Hex
	Move
	Movement
	NA
	Next
	Note
	Number
	Previous
	Scout
	ScoutId
	Season
	Status
	TerrainCode
	Text // run of text that didn't match anything else
	To
	Tribe
	Turn
	TurnYearMonth
	UnitId
	Weather

	EndOfLine  // end of line (either LF or CR+LF)
	EndOfInput // end of input
)

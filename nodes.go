// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package tnrpt

import "strings"

// Node is the interface implemented by all CST nodes.
//
// Line and Column report the location of this node in the source text,
// usually the line and column of the node's first non-trivia token.
// Both are 1-based.
//
// Kind returns a short, stable identifier for the node type, such as
// "TurnReport", "ClanHeader", or "MovementLine".
//
// Source reconstructs the source code corresponding to this node and its
// children. For a lossless CST, concatenating Source() over the root node
// should reproduce the original input.
//
// Errors returns any parse errors that were attached to this node.
//
// Tokens returns the tokens directly associated with this node. Depending
// on how nodes are constructed, this may be a subset of the tokens that
// appear in Source() (children may own their own tokens).
type Node interface {
	Kind() string
	Line() int
	Column() int
	Span() Span

	Source(input []byte) string
	Tokens() []*Token
	Errors() []error
}

type BaseNode struct {
	kind   string
	span   Span // covers the entire subtree
	left   []Node
	right  []Node
	self   []*Token // tokens owned directly by this node
	errors []error
}

func (b *BaseNode) Kind() string    { return b.kind }
func (b *BaseNode) Span() Span      { return b.span }
func (b *BaseNode) Line() int       { return b.span.Line }
func (b *BaseNode) Column() int     { return b.span.Column }
func (b *BaseNode) Errors() []error { return b.errors }

func (b *BaseNode) Tokens() []*Token {
	var toks []*Token
	for _, ch := range b.left {
		toks = append(toks, ch.Tokens()...)
	}
	toks = append(toks, b.self...)
	for _, ch := range b.right {
		toks = append(toks, ch.Tokens()...)
	}
	return toks
}

func (b *BaseNode) Source(input []byte) string {
	// Simple/cheap: concatenate token lexemes.
	var sb strings.Builder
	for _, tok := range b.Tokens() {
		sb.Write(input[tok.Start:tok.End])
	}
	return sb.String()
}

type BadNode struct {
	BaseNode
}

func newBadNode(
	kind string,
	toks []*Token,
	err error,
) *BadNode {
	return &BadNode{
		BaseNode: BaseNode{
			kind:   kind, // e.g. "BadExpr", "BadUnitStatus"
			span:   spanFromTokenSlice(toks),
			self:   toks,
			errors: []error{err},
		},
	}
}

func newMissingNode(
	kind string,
	nextTok *Token,
	err error,
) *BadNode {
	return &BadNode{
		BaseNode: BaseNode{
			kind: kind, // e.g. "MissingStatusLine"
			span: Span{ // create zero-length span at insertion point
				Start:  nextTok.Start,
				End:    nextTok.Start,
				Line:   nextTok.Line,
				Column: nextTok.Column,
			},
			self:   nil, // no tokens; it was missing
			errors: []error{err},
		},
	}
}

type TurnReportNode struct{}

/*
unitLocationLine ::= Literal UnitId Comma Note? Comma longLocationNode Comma LeftParen longLocationNode RightParen EndOfLine

longLocationNode  ::= (Current|Previous) Hex Equal (Grid RowColumn | NA)

Examples:
	Tribe 0987, , Current Hex = QQ 0203, (Previous Hex = QQ 0101)
*/

type UnitLocationLineNode struct {
	BaseNode
	UnitTypeKw       Node // happy path: *LiteralNode
	UnitId           Node // happy path: *UnitIdNode
	Comma1           Node // happy path: *LiteralNode
	Note             Node // happy path: *LiteralNode
	Comma2           Node // happy path: *LiteralNode
	CurrentLocation  Node // happy path: *LongLocationNode
	Comma3           Node // happy path: *LiteralNode
	LeftParen        Node // happy path: *LiteralNode
	PreviousLocation Node // happy path: *LongLocationNode
	RightParen       Node // happy path: *LiteralNode
	EndOfLine        Node // happy path: *LiteralNode
}

func newUnitLocationLineNode(
	unitTypeKw,
	unitId,
	comma1,
	note,
	comma2,
	currentLocation,
	comma3,
	leftParen,
	previousLocation,
	rightParen,
	endOfLine Node,
) *UnitLocationLineNode {
	// build left/right slices for BaseNode
	left := []Node{
		unitTypeKw,
		unitId,
		comma1,
		note, // optional, may be nil
		comma2,
		currentLocation,
		comma3,
		leftParen,
		previousLocation,
		rightParen,
		endOfLine,
	}

	toks := subtreeTokens(left, nil, nil)

	return &UnitLocationLineNode{
		BaseNode: BaseNode{
			kind: "UnitLocationLine",
			span: spanFromTokenSlice(toks),
			left: left,
		},
		UnitTypeKw:       unitTypeKw,
		UnitId:           unitId,
		Comma1:           comma1,
		Note:             note,
		Comma2:           comma2,
		CurrentLocation:  currentLocation,
		Comma3:           comma3,
		LeftParen:        leftParen,
		PreviousLocation: previousLocation,
		RightParen:       rightParen,
		EndOfLine:        endOfLine,
	}
}

type LiteralNode struct {
	BaseNode
}

func newLiteralNode(
	tok *Token,
	kind string,
) *LiteralNode {
	return &LiteralNode{
		BaseNode: BaseNode{
			kind: kind, // Comma, Equal, SingleQuote
			span: spanFromToken(tok),
			self: []*Token{tok},
		},
	}
}

type UnitIdNode struct {
	BaseNode
}

func newUnitIdNode(tok *Token) *UnitIdNode {
	return &UnitIdNode{
		BaseNode: BaseNode{
			kind: "UnitId",
			span: spanFromToken(tok),
			self: []*Token{tok},
		},
	}
}

/*
longLocationNode ::= (Current|Previous) Hex Equal (NA | (Grid RowColumn))

Examples:
    Current Hex = QQ 0203
    Previous Hex = NA
*/

type LongLocationNode struct {
	BaseNode
	Prefix    Node // "Current" or "Previous" literal
	HexKw     Node // "Hex" literal
	Equal     Node // "=" literal
	NA        Node // literal "N/A" when applicable (mutually exclusive with Grid/RowColumn)
	Grid      Node // happy path: *LiteralNode (Grid) or nil if NA
	RowColumn Node // happy path: *LiteralNode (Row/Col) or nil if NA
}

func newLongLocationNode(
	prefix,
	hexKw,
	equal,
	grid,
	rowColumn,
	na Node,
) *LongLocationNode {
	left := []Node{ // child order in source
		prefix,
		hexKw,
		equal,
		grid,
		rowColumn,
		na,
	}
	toks := subtreeTokens(left, nil, nil)
	return &LongLocationNode{
		BaseNode: BaseNode{
			kind: "LongLocation",
			span: spanFromTokenSlice(toks),
			left: left,
		},
		Prefix:    prefix,
		HexKw:     hexKw,
		Equal:     equal,
		Grid:      grid,
		RowColumn: rowColumn,
		NA:        na,
	}
}

type LocationNode struct {
	BaseNode
}

func newLocationNode(tok *Token) *LocationNode {
	return &LocationNode{
		BaseNode: BaseNode{
			kind: "Location",
			span: spanFromToken(tok),
			self: []*Token{tok},
		},
	}
}

type UnitStatusLineNode struct {
	BaseNode
	// fields for direct children if you like:
	UnitID Node
	// Terrain Node, etc.
}

func newUnitStatusLineNode(
	unitId Node,
	selfTokens []*Token,
	rightChildren []Node,
) *UnitStatusLineNode {
	// Build left/right slices for BaseNode
	left := []Node{unitId}
	right := append([]Node(nil), rightChildren...)

	// Compute span from subtree tokens
	// (first token from left subtree, last token from right/self).
	toks := subtreeTokens(left, selfTokens, right)
	span := spanFromTokenSlice(toks)

	return &UnitStatusLineNode{
		BaseNode: BaseNode{
			kind:  "UnitStatusLine",
			span:  span,
			left:  left,
			self:  selfTokens,
			right: right,
		},
		UnitID: unitId,
	}
}

func subtreeTokens(
	left []Node,
	self []*Token,
	right []Node,
) []*Token {
	var toks []*Token
	for _, ch := range left {
		if ch == nil {
			continue
		}
		toks = append(toks, ch.Tokens()...)
	}
	toks = append(toks, self...)
	for _, ch := range right {
		if ch == nil {
			continue
		}
		toks = append(toks, ch.Tokens()...)
	}
	return toks
}

type TextNode struct {
	BaseNode
}

func newTextNode(tok *Token) *TextNode {
	return &TextNode{
		BaseNode: BaseNode{
			kind: "Text",
			span: spanFromToken(tok),
			self: []*Token{tok},
		},
	}
}

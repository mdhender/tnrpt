// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package tnrpt

// Last piece of the puzzle: AST nodes copy spans from CST nodes (you don’t recompute).

type UnitIdExpr struct {
	Span Span
	Text string
}

type HexLocation struct {
	Span   Span
	IsNA   bool   // true if "N/A", false if we have Grid/RowCol
	Grid   string // e.g. "QQ"
	RowCol string // e.g. "0203"
}

type UnitLocationLine struct {
	Span     Span
	UnitType string     // "Tribe", "Courier", etc
	UnitId   UnitIdExpr // parsed from UnitId node
	Note     string     // optional; empty string if omitted
	Current  HexLocation
	Previous HexLocation
}

// Now any semantic error involving a UnitIDExpr has:
// * A precise Span for diagnostics.
// * The original text (Text) for messages.
func buildUnitIdExpr(n *UnitIdNode, src []byte) UnitIdExpr {
	return UnitIdExpr{
		Span: n.Span(), // directly from CST
		Text: n.Source(src),
	}
}

// buildHexLocation builds a HexLocation AST value from a LongLocationNode.
//
// CST: (Current|Previous) Hex Equal (Grid RowColumn | NA)
//
// Examples of CST Source():
//
//	"Current Hex = QQ 0203"
//	"Previous Hex = NA"
//
// This is “happy-path with a bit of defensive coding”:
// * If NA exists, we mark IsNA = true.
// * If Grid/RowColumn exist, we pull their text.
//
// If the CST is malformed, we’ll get empty strings; later we’ll upgrade
// this to emit diagnostics rather than silently accept.
func buildHexLocation(n *LongLocationNode, src []byte) HexLocation {
	span := n.Span()

	// If NA is present, we treat it as IsNA and ignore Grid/RowCol.
	if n.NA != nil {
		return HexLocation{
			Span: span,
			IsNA: true,
			// Grid/RowCol left empty intentionally
		}
	}

	// Otherwise we expect Grid and RowColumn literals.
	gridText := ""
	rowColText := ""

	if n.Grid != nil {
		gridText = n.Grid.Source(src)
	}
	if n.RowColumn != nil {
		rowColText = n.RowColumn.Source(src)
	}

	return HexLocation{
		Span:   span,
		IsNA:   false,
		Grid:   gridText,
		RowCol: rowColText,
	}
}

// BuildUnitLocationLine builds an AST UnitLocationLine from a CST UnitLocationLineNode.
// src is the full input []byte used by the lexer (for Source()).
//
// “happy path with minimal guard rails” version:
//   - If the CST is exactly as expected, everything is populated cleanly.
//   - If something is off (e.g., UnitId is a BadNode), we still get a Span
//     and some text, so we can later generate diagnostics in the semantic pass.
func BuildUnitLocationLine(n *UnitLocationLineNode, src []byte) UnitLocationLine {
	// UnitType: usually "Tribe", "Courier", etc
	unitTypeText := ""
	if n.UnitTypeKw != nil {
		unitTypeText = n.UnitTypeKw.Source(src)
	}

	// UnitId: we expect a *UnitIdNode on the happy path
	unitIDExpr := UnitIdExpr{}
	if uid, ok := n.UnitId.(*UnitIdNode); ok {
		unitIDExpr = buildUnitIdExpr(uid, src)
	} else if n.UnitId != nil {
		// Fallback: use generic Node.Source and Span; we can refine this later.
		unitIDExpr = UnitIdExpr{
			Span: n.UnitId.Span(),
			Text: n.UnitId.Source(src),
		}
	}

	// Optional note
	noteText := ""
	if n.Note != nil {
		noteText = n.Note.Source(src)
	}

	// Current location
	currentLoc := HexLocation{}
	if curr, ok := n.CurrentLocation.(*LongLocationNode); ok && curr != nil {
		currentLoc = buildHexLocation(curr, src)
	}

	// Previous location
	previousLoc := HexLocation{}
	if prev, ok := n.PreviousLocation.(*LongLocationNode); ok && prev != nil {
		previousLoc = buildHexLocation(prev, src)
	}

	return UnitLocationLine{
		Span:     n.Span(),
		UnitType: unitTypeText,
		UnitId:   unitIDExpr,
		Note:     noteText,
		Current:  currentLoc,
		Previous: previousLoc,
	}
}

// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package coords


//// 9.0 Other Systems
//
//// 9.1 Compass Points
//
//// The compass points are approximations, but close to the rose.
//// ✅ = supported, 🚫 = not supported.
////
//// +------+-------+---------+---------+
//// | Name | Angle |  Flat?  | Pointy? |
//// +------+-------+---------+---------+
//// | N    |   0°  |   ✅    |   🚫    |
//// | NNE  |  30°  |   🚫    |   ✅    |
//// | ENE  |  60°  |   ✅    |   🚫    |
//// | E    |  90°  |   🚫    |   ✅    |
//// | ESE  | 120°  |   ✅    |   🚫    |
//// | SSE  | 150°  |   🚫    |   ✅    |
//// | S    | 180°  |   ✅    |   🚫    |
//// | SSW  | 210°  |   🚫    |   ✅    |
//// | WSW  | 240°  |   ✅    |   🚫    |
//// | W    | 270°  |   🚫    |   ✅    |
//// | WNW  | 300°  |   ✅    |   🚫    |
//// | NNW  | 330°  |   🚫    |   ✅    |
//// +------+-------+---------+---------+
////
//// Using an unsupported value for your layout may cause unexpected results.
//const (
//	N   int = 2
//	NNE int = 1
//	ENE int = 1
//	E   int = 0
//	ESE int = 0
//	SSE int = 5
//	S   int = 5
//	SSW int = 4
//	WSW int = 4
//	W   int = 3
//	WNW int = 3
//	NNW int = 2
//)
//
//var (
//	bearingToDirection = map[string]int{
//		"N":   2,
//		"NNE": 1,
//		"ENE": 1,
//		"E":   0,
//		"ESE": 0,
//		"SSE": 5,
//		"S":   5,
//		"SSW": 4,
//		"WSW": 4,
//		"W":   3,
//		"WNW": 3,
//		"NNW": 2,
//	}
//)
//
//// BearingToDirection returns the direction for a compass point.
//// Panics on invalid input.
//func BearingToDirection(p string) int {
//	dir, ok := bearingToDirection[p]
//	if !ok {
//		panic(fmt.Sprintf("assert(p != %q)", p))
//	}
//	return dir
//}
//
//var (
//	// horizontalDirectionToBearing maps a direction to the compass point for a horizontal layout
//	horizontalDirectionToBearing = []string{
//		"E", "NNE", "NNW", "W", "SSW", "SSE",
//	}
//	// verticalDirectionToBearing maps a direction to the compass point for a vertical layout
//	verticalDirectionToBearing = []string{
//		"ESE", "ENE", "N", "WNW", "WSW", "S",
//	}
//)
//
//// 9.2 TribeNet Coordinates
//
//// TribeNet coordinates are in the form "AB 0102":
//// - "A" (grid row) and "B" (grid column) identify a sub-map.
//// - "0102" is the in-submap position: column 01 (1-based) and row 02 (1-based).
//// Each sub-map is 30 columns wide and 21 rows tall,
//// with "0101" as the upper-left and "3021" as the lower-right corner.
////
//// The global map origin is (1,1) at the upper-left.
//// Even-numbered columns are vertically offset (odd-q layout), so (2,1) is southeast of (1,1).
////
//// TribeNet coordinates are converted to OffsetCoord using "odd-q" layout,
//// with the origin translated by (-1, -1) so "AA 0101" becomes (0,0).
//
//// NewLayoutTribeNet returns a layout with flat-top hexes for TribeNet.
////
//// All TribeNet maps are "odd-q,", origin is (1,1) and will
//// be translated to (0,0), sub-maps are 21 rows x 30 columns.
////
//// You may need to translate the origin from (0,0) to (1,1) when displaying TribeNet coordinates.
//func NewLayoutTribeNet() LayoutPointyTypeHorizontalOddRight_t {
//	return NewLayoutEvenQ(Point{1, 1}, Point{0, 0})
//}
//
//// HexFromCubeCoords returns a Hex initialized from cube coordinates.
//// Panics if q + r + s != 0.
//func HexFromCubeCoords(q, r, s int) Hex { // Cube constructor
//	if q+r+s != 0 {
//		panic("assert(q + r + s == 0)")
//	}
//	return Hex{q: q, r: r, s: -q - r}
//}


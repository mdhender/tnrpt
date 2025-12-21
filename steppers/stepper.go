// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package steppers

import (
	"github.com/maloquacious/hexg"
	"github.com/mdhender/tnrpt/model"
)

// Stepper is used by walkers to navigate between hexes. It defines
// methods to convert between TribeNet coordinates (e.g., "QQ 1010")
// and hexg.Hex (e.g., (15,16,-31). It also defines methods to find
// a hex's neighbor using direction code (N, NE, SE, S, SW, NW).
type Stepper interface {
	// CoordToHex maps a TribeNet coordinate to hexg.Hex coordinate.
	// Handles N/A and obscured grids.
	CoordToHex(coord model.TNCoord) (hexg.Hex, error)

	// HexToCoord converts a hex coordinate to a TribeNet coordinate string ("AB 0102").
	// Returns an error if the coordinate falls outside the supported 26×26 letter grid.
	HexToCoord(hex hexg.Hex) (model.TNCoord, error)

	// StepBackwardHex applies a direction to a current hex and returns
	// the hex stepped from. Used for reconstructing paths from adv steps.
	StepBackwardHex(cur hexg.Hex, dir string) (hexg.Hex, bool)

	// StepForwardHex applies a direction to a current hex and returns
	// the hex stepped into. Used for reconstructing paths from adv steps.
	StepForwardHex(cur hexg.Hex, dir string) (hexg.Hex, bool)
}

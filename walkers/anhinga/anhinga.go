// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package anhinga

import (
	"log"
	"sort"

	"github.com/mdhender/tnrpt"
	"github.com/mdhender/tnrpt/model"
	"github.com/mdhender/tnrpt/steppers"
)

type Walker struct{}

func Walk(input *tnrpt.Turn_t, nav steppers.Stepper, quiet, verbose, debug bool) ([]*model.Tile, error) {
	if !quiet {
		log.Printf("anhinga: walking %q\n", input.Source)
	}

	var unitMoves []*tnrpt.Moves_t
	for _, unit := range input.UnitMoves {
		unitMoves = append(unitMoves, unit)
	}
	sort.Slice(unitMoves, func(i, j int) bool {
		a, b := unitMoves[i], unitMoves[j]
		return a.UnitId < b.UnitId
	})

	for _, unit := range unitMoves {
		err := walkMoves(nav, unit)
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func walkMoves(nav steppers.Stepper, moves *tnrpt.Moves_t) error {
	currentHex, err := nav.CoordToHex(model.TNCoord(moves.CurrentHex))
	if err != nil {
		return err
	}
	log.Printf("walk: %s: curr %s: %q\n", moves.UnitId, moves.CurrentHex, currentHex.ConciseString())

	// walk all moves backwards
	for i := len(moves.Moves) - 1; i >= 0; i-- {
		move := moves.Moves[i]
		log.Printf("walk: %s: %3d: %4d %3d\n", moves.UnitId, i+1, move.LineNo, move.StepNo)
	}
	log.Printf("walk: %s: prev %s: %q\n", moves.UnitId, moves.PreviousHex, currentHex.ConciseString())

	return nil
}

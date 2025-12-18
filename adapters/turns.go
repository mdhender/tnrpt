// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package adapters

import (
	"github.com/mdhender/tnrpt"
	"github.com/mdhender/tnrpt/parser"
)

// AdaptParserTurnToModel does that.
func AdaptParserTurnToModel(pt *parser.Turn_t) (*tnrpt.Turn_t, error) {
	mt := &tnrpt.Turn_t{
		Id:           pt.Id,
		Year:         pt.Year,
		Month:        pt.Month,
		UnitMoves:    map[tnrpt.UnitId_t]*tnrpt.Moves_t{},
		SpecialNames: map[string]*tnrpt.Special_t{},
	}
	for k, v := range pt.UnitMoves {
		mt.UnitMoves[tnrpt.UnitId_t(k)] = &tnrpt.Moves_t{
			TurnId:      v.TurnId,
			UnitId:      tnrpt.UnitId_t(v.UnitId),
			Moves:       nil,
			Follows:     tnrpt.UnitId_t(v.Follows),
			GoesTo:      v.GoesTo,
			Scries:      nil,
			Scouts:      nil,
			FromHex:     v.FromHex,
			ToHex:       v.ToHex,
			Coordinates: v.Coordinates,
			Location:    v.Location,
		}
	}
	for k, v := range pt.SpecialNames {
		mt.SpecialNames[k] = &tnrpt.Special_t{
			TurnId: v.TurnId,
			Id:     v.Id,
			Name:   v.Name,
		}
	}
	return mt, nil
}

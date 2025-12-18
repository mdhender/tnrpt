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
		mt.UnitMoves[tnrpt.UnitId_t(k)] = adaptParserMoves(v)
	}

	//for _, v := range pt.SortedMoves {
	//	if mv, ok := mt.UnitMoves[tnrpt.UnitId_t(v.UnitId)]; ok {
	//		mt.SortedMoves = append(mt.SortedMoves, mv)
	//	}
	//}

	//for _, v := range pt.MovesSortedByElement {
	//	if mv, ok := mt.UnitMoves[tnrpt.UnitId_t(v.UnitId)]; ok {
	//		mt.MovesSortedByElement = append(mt.MovesSortedByElement, mv)
	//	}
	//}

	for k, v := range pt.SpecialNames {
		mt.SpecialNames[k] = &tnrpt.Special_t{
			TurnId: v.TurnId,
			Id:     v.Id,
			Name:   v.Name,
		}
	}

	return mt, nil
}

func adaptParserMoves(pm *parser.Moves_t) *tnrpt.Moves_t {
	if pm == nil {
		return nil
	}
	mm := &tnrpt.Moves_t{
		TurnId:      pm.TurnId,
		UnitId:      tnrpt.UnitId_t(pm.UnitId),
		Follows:     tnrpt.UnitId_t(pm.Follows),
		GoesTo:      pm.GoesTo,
		FromHex:     pm.FromHex,
		ToHex:       pm.ToHex,
		Coordinates: pm.Coordinates,
		Location:    pm.Location,
	}

	for _, v := range pm.Moves {
		mm.Moves = append(mm.Moves, adaptParserMove(v))
	}

	for _, v := range pm.Scries {
		mm.Scries = append(mm.Scries, adaptParserScry(v))
	}

	for _, v := range pm.Scouts {
		mm.Scouts = append(mm.Scouts, adaptParserScout(v))
	}

	return mm
}

func adaptParserMove(pm *parser.Move_t) *tnrpt.Move_t {
	if pm == nil {
		return nil
	}
	return &tnrpt.Move_t{
		UnitId:          tnrpt.UnitId_t(pm.UnitId),
		Advance:         pm.Advance,
		Follows:         tnrpt.UnitId_t(pm.Follows),
		GoesTo:          pm.GoesTo,
		Still:           pm.Still,
		Result:          pm.Result,
		Report:          adaptParserReport(pm.Report),
		LineNo:          pm.LineNo,
		StepNo:          pm.StepNo,
		Line:            pm.Line,
		TurnId:          pm.TurnId,
		CurrentHex:      pm.CurrentHex,
		FromCoordinates: pm.FromCoordinates,
		ToCoordinates:   pm.ToCoordinates,
		Location:        pm.Location,
	}
}

func adaptParserScry(ps *parser.Scry_t) *tnrpt.Scry_t {
	if ps == nil {
		return nil
	}
	ms := &tnrpt.Scry_t{
		UnitId:      tnrpt.UnitId_t(ps.UnitId),
		Type:        ps.Type,
		Origin:      ps.Origin,
		Coordinates: ps.Coordinates,
		Location:    ps.Location,
		Text:        ps.Text,
		Scouts:      adaptParserScout(ps.Scouts),
	}

	for _, v := range ps.Moves {
		ms.Moves = append(ms.Moves, adaptParserMove(v))
	}

	return ms
}

func adaptParserScout(ps *parser.Scout_t) *tnrpt.Scout_t {
	if ps == nil {
		return nil
	}
	ms := &tnrpt.Scout_t{
		No:     ps.No,
		TurnId: ps.TurnId,
		LineNo: ps.LineNo,
		Line:   ps.Line,
	}

	for _, v := range ps.Moves {
		ms.Moves = append(ms.Moves, adaptParserMove(v))
	}

	return ms
}

func adaptParserReport(pr *parser.Report_t) *tnrpt.Report_t {
	if pr == nil {
		return nil
	}
	mr := &tnrpt.Report_t{
		UnitId:        tnrpt.UnitId_t(pr.UnitId),
		Location:      pr.Location,
		TurnId:        pr.TurnId,
		ScoutedTurnId: pr.ScoutedTurnId,
		Terrain:       pr.Terrain,
		Resources:     pr.Resources,
		WasVisited:    pr.WasVisited,
		WasScouted:    pr.WasScouted,
	}

	for _, v := range pr.Borders {
		mr.Borders = append(mr.Borders, adaptParserBorder(v))
	}

	for _, v := range pr.Encounters {
		mr.Encounters = append(mr.Encounters, adaptParserEncounter(v))
	}

	for _, v := range pr.Items {
		mr.Items = append(mr.Items, adaptParserFoundItem(v))
	}

	for _, v := range pr.Settlements {
		mr.Settlements = append(mr.Settlements, adaptParserSettlement(v))
	}

	for _, v := range pr.FarHorizons {
		mr.FarHorizons = append(mr.FarHorizons, adaptParserFarHorizon(v))
	}

	return mr
}

func adaptParserBorder(pb *parser.Border_t) *tnrpt.Border_t {
	if pb == nil {
		return nil
	}
	return &tnrpt.Border_t{
		Direction: pb.Direction,
		Edge:      pb.Edge,
		Terrain:   pb.Terrain,
	}
}

func adaptParserEncounter(pe *parser.Encounter_t) *tnrpt.Encounter_t {
	if pe == nil {
		return nil
	}
	return &tnrpt.Encounter_t{
		TurnId:   pe.TurnId,
		UnitId:   tnrpt.UnitId_t(pe.UnitId),
		Friendly: pe.Friendly,
	}
}

func adaptParserFoundItem(pf *parser.FoundItem_t) *tnrpt.FoundItem_t {
	if pf == nil {
		return nil
	}
	return &tnrpt.FoundItem_t{
		Quantity: pf.Quantity,
		Item:     pf.Item,
	}
}

func adaptParserSettlement(ps *parser.Settlement_t) *tnrpt.Settlement_t {
	if ps == nil {
		return nil
	}
	return &tnrpt.Settlement_t{
		TurnId: ps.TurnId,
		Name:   ps.Name,
	}
}

func adaptParserFarHorizon(pf *parser.FarHorizon_t) *tnrpt.FarHorizon_t {
	if pf == nil {
		return nil
	}
	return &tnrpt.FarHorizon_t{
		Point:   pf.Point,
		Terrain: pf.Terrain,
	}
}

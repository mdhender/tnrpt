// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package adapters

import (
	"path/filepath"

	"github.com/mdhender/tnrpt"
	"github.com/mdhender/tnrpt/pipelines/parsers/bistre"
)

// BistreParserTurnToModel adapts bistre parser output to the old tnrpt model types.
func BistreParserTurnToModel(source string, pt *bistre.Turn_t) (*tnrpt.Turn_t, error) {
	mt := &tnrpt.Turn_t{
		Id:           pt.Id,
		Source:       filepath.Clean(source),
		Year:         pt.Year,
		Month:        pt.Month,
		UnitMoves:    map[tnrpt.UnitId_t]*tnrpt.Moves_t{},
		SpecialNames: map[string]*tnrpt.Special_t{},
	}

	for k, v := range pt.UnitMoves {
		mt.UnitMoves[tnrpt.UnitId_t(k)] = adaptBistreParserMoves(v)
	}

	for k, v := range pt.SpecialNames {
		mt.SpecialNames[k] = &tnrpt.Special_t{
			Id:   v.Id,
			Name: v.Name,
		}
	}

	return mt, nil
}

func adaptBistreParserMoves(pm *bistre.Moves_t) *tnrpt.Moves_t {
	if pm == nil {
		return nil
	}
	mm := &tnrpt.Moves_t{
		UnitId:      tnrpt.UnitId_t(pm.UnitId),
		Follows:     tnrpt.UnitId_t(pm.Follows),
		GoesTo:      pm.GoesTo,
		PreviousHex: pm.PreviousHex,
		CurrentHex:  pm.CurrentHex,
	}

	for _, v := range pm.Moves {
		mm.Moves = append(mm.Moves, adaptBistreParserMove(v))
	}

	for _, v := range pm.Scries {
		mm.Scries = append(mm.Scries, adaptBistreParserScry(v))
	}

	for _, v := range pm.Scouts {
		mm.Scouts = append(mm.Scouts, adaptBistreParserScout(v))
	}

	return mm
}

func adaptBistreParserMove(pm *bistre.Move_t) *tnrpt.Move_t {
	if pm == nil {
		return nil
	}
	return &tnrpt.Move_t{
		Advance:   pm.Advance,
		Follows:   tnrpt.UnitId_t(pm.Follows),
		GoesToHex: pm.GoesTo,
		Still:     pm.Still,
		Result:    pm.Result,
		Report:    adaptBistreParserReport(pm.Report),
		LineNo:    pm.LineNo,
		StepNo:    pm.StepNo,
		Line:      string(pm.Line),
	}
}

func adaptBistreParserScry(ps *bistre.Scry_t) *tnrpt.Scry_t {
	if ps == nil {
		return nil
	}
	ms := &tnrpt.Scry_t{
		Type:      ps.Type,
		OriginHex: ps.Origin,
		Text:      string(ps.Text),
		Scouts:    adaptBistreParserScout(ps.Scouts),
	}

	for _, v := range ps.Moves {
		ms.Moves = append(ms.Moves, adaptBistreParserMove(v))
	}

	return ms
}

func adaptBistreParserScout(ps *bistre.Scout_t) *tnrpt.Scout_t {
	if ps == nil {
		return nil
	}
	ms := &tnrpt.Scout_t{
		No:     ps.No,
		LineNo: ps.LineNo,
		Line:   string(ps.Line),
	}

	for _, v := range ps.Moves {
		ms.Moves = append(ms.Moves, adaptBistreParserMove(v))
	}

	return ms
}

func adaptBistreParserReport(pr *bistre.Report_t) *tnrpt.Report_t {
	if pr == nil {
		return nil
	}
	mr := &tnrpt.Report_t{
		Terrain:   pr.Terrain,
		Resources: pr.Resources,
	}

	for _, v := range pr.Borders {
		mr.Borders = append(mr.Borders, adaptBistreParserBorder(v))
	}

	for _, v := range pr.Encounters {
		mr.Encounters = append(mr.Encounters, adaptBistreParserEncounter(v))
	}

	for _, v := range pr.Items {
		mr.Items = append(mr.Items, adaptBistreParserFoundItem(v))
	}

	for _, v := range pr.Settlements {
		mr.Settlements = append(mr.Settlements, adaptBistreParserSettlement(v))
	}

	for _, v := range pr.FarHorizons {
		mr.FarHorizons = append(mr.FarHorizons, adaptBistreParserFarHorizon(v))
	}

	return mr
}

func adaptBistreParserBorder(pb *bistre.Border_t) *tnrpt.Border_t {
	if pb == nil {
		return nil
	}
	return &tnrpt.Border_t{
		Direction: pb.Direction,
		Edge:      pb.Edge,
		Terrain:   pb.Terrain,
	}
}

func adaptBistreParserEncounter(pe *bistre.Encounter_t) *tnrpt.Encounter_t {
	if pe == nil {
		return nil
	}
	return &tnrpt.Encounter_t{
		UnitId:   tnrpt.UnitId_t(pe.UnitId),
		Friendly: pe.Friendly,
	}
}

func adaptBistreParserFoundItem(pf *bistre.FoundItem_t) *tnrpt.FoundItem_t {
	if pf == nil {
		return nil
	}
	return &tnrpt.FoundItem_t{
		Quantity: pf.Quantity,
		Item:     pf.Item,
	}
}

func adaptBistreParserSettlement(ps *bistre.Settlement_t) *tnrpt.Settlement_t {
	if ps == nil {
		return nil
	}
	return &tnrpt.Settlement_t{
		Name: ps.Name,
	}
}

func adaptBistreParserFarHorizon(pf *bistre.FarHorizon_t) *tnrpt.FarHorizon_t {
	if pf == nil {
		return nil
	}
	return &tnrpt.FarHorizon_t{
		Point:   pf.Point,
		Terrain: pf.Terrain,
	}
}

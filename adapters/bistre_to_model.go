// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package adapters

import (
	"time"

	"github.com/mdhender/tnrpt/direction"
	"github.com/mdhender/tnrpt/model"
	"github.com/mdhender/tnrpt/pipelines/parsers/bistre"
	"github.com/mdhender/tnrpt/results"
)

// BistreTurnToModelReportX converts a bistre.Turn_t to model.ReportX without persisting.
// This is for in-memory use in the web spike.
func BistreTurnToModelReportX(source string, turn *bistre.Turn_t, game, clanNo string) (*model.ReportX, error) {
	now := time.Now().UTC()
	turnNo := 100*turn.Year + turn.Month

	rx := &model.ReportX{
		Game:      game,
		ClanNo:    clanNo,
		TurnNo:    turnNo,
		CreatedAt: now,
	}

	for unitId, moves := range turn.UnitMoves {
		ux := convertUnitMoves(turnNo, unitId, moves)
		rx.Units = append(rx.Units, ux)
	}

	return rx, nil
}

func convertUnitMoves(turnNo int, unitId bistre.UnitId_t, moves *bistre.Moves_t) *model.UnitX {
	ux := &model.UnitX{
		UnitID:  string(unitId),
		TurnNo:  turnNo,
		StartTN: model.TNCoord(moves.PreviousHex),
		EndTN:   model.TNCoord(moves.CurrentHex),
	}

	actSeq := 0

	if moves.Follows != "" {
		actSeq++
		act := &model.Act{
			Seq:          actSeq,
			Kind:         model.ActKindFollow,
			Ok:           true,
			TargetUnitID: string(moves.Follows),
		}
		ux.Acts = append(ux.Acts, act)
	}

	if moves.GoesTo != "" {
		actSeq++
		act := &model.Act{
			Seq:    actSeq,
			Kind:   model.ActKindGoto,
			Ok:     true,
			DestTN: model.TNCoord(moves.GoesTo),
		}
		ux.Acts = append(ux.Acts, act)
	}

	if len(moves.Moves) > 0 && moves.Follows == "" && moves.GoesTo == "" {
		actSeq++
		act := &model.Act{
			Seq:  actSeq,
			Kind: model.ActKindMove,
			Ok:   true,
		}

		for stepSeq, mv := range moves.Moves {
			step := convertMove(mv, stepSeq+1)
			act.Steps = append(act.Steps, step)
			if !step.Ok {
				act.Ok = false
			}
		}

		ux.Acts = append(ux.Acts, act)
	}

	for _, scout := range moves.Scouts {
		actSeq++
		act := &model.Act{
			Seq:  actSeq,
			Kind: model.ActKindScout,
			Ok:   true,
		}

		for stepSeq, mv := range scout.Moves {
			step := convertMove(mv, stepSeq+1)
			act.Steps = append(act.Steps, step)
			if !step.Ok {
				act.Ok = false
			}
		}

		ux.Acts = append(ux.Acts, act)
	}

	return ux
}

func convertMove(mv *bistre.Move_t, seq int) *model.Step {
	step := &model.Step{
		Seq: seq,
		Ok:  mv.Result == results.Succeeded || mv.Result == results.StayedInPlace,
	}

	if mv.Still {
		step.Kind = model.StepKindStill
	} else if mv.Advance != direction.Unknown {
		step.Kind = model.StepKindAdv
		step.Dir = mv.Advance.String()
		if !step.Ok {
			step.FailWhy = convertResultToFailWhy(mv.Result)
		}
	} else {
		step.Kind = model.StepKindObs
	}

	if mv.Report != nil {
		step.Terr = mv.Report.Terrain.String()

		if len(mv.Report.Encounters) > 0 || len(mv.Report.Settlements) > 0 || len(mv.Report.Resources) > 0 {
			enc := &model.Enc{}

			for _, e := range mv.Report.Encounters {
				enc.Units = append(enc.Units, &model.UnitSeen{
					UnitID: string(e.UnitId),
				})
			}

			for _, s := range mv.Report.Settlements {
				enc.Sets = append(enc.Sets, &model.SettleSeen{
					Name: s.Name,
				})
			}

			for _, r := range mv.Report.Resources {
				enc.Rsrc = append(enc.Rsrc, &model.RsrcSeen{
					Kind: r.String(),
				})
			}

			step.Enc = enc
		}

		for _, b := range mv.Report.Borders {
			obs := &model.BorderObs{
				Dir: b.Direction.String(),
			}
			if b.Edge != 0 {
				obs.Kind = b.Edge.String()
			} else if b.Terrain != 0 {
				obs.Kind = b.Terrain.String()
			}
			step.Borders = append(step.Borders, obs)
		}
	}

	return step
}

func convertResultToFailWhy(r results.Result_e) string {
	switch r {
	case results.Blocked:
		return "blocked"
	case results.ExhaustedMovementPoints:
		return "exhaust"
	case results.Prohibited:
		return "terrain"
	case results.Failed:
		return "unknown"
	default:
		return ""
	}
}

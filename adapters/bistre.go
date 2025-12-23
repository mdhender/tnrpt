// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package adapters

import (
	"time"

	"github.com/mdhender/tnrpt/direction"
	"github.com/mdhender/tnrpt/model"
	"github.com/mdhender/tnrpt/parsers/azul"
	"github.com/mdhender/tnrpt/results"
)

// BistreToModel adapts azul parser output to the new model types.
// It returns a ReportFile (the source document metadata) and a ReportX (the extracted data).
func BistreToModel(source string, pt *azul.Turn_t) (*model.ReportFile, *model.ReportX, error) {
	now := time.Now().UTC()
	turnNo := 100*pt.Year + pt.Month

	rf := &model.ReportFile{
		ID:        0,  // caller assigns
		Game:      "", // caller assigns
		ClanNo:    "", // caller assigns
		TurnNo:    turnNo,
		Name:      source,
		CreatedAt: now,
	}

	rx := &model.ReportX{
		ID:           0, // caller assigns
		ReportFileID: rf.ID,
		Game:         rf.Game,
		ClanNo:       rf.ClanNo,
		TurnNo:       rf.TurnNo,
		CreatedAt:    now,
	}

	// Convert each unit's moves to UnitX
	var actSeq, stepSeq int
	for unitId, moves := range pt.UnitMoves {
		ux := &model.UnitX{
			ID:        0, // caller assigns
			ReportXID: rx.ID,
			UnitID:    string(unitId),
			TurnNo:    turnNo,
			StartTN:   model.TNCoord(moves.PreviousHex),
			EndTN:     model.TNCoord(moves.CurrentHex),
		}

		actSeq = 0

		// Handle follows
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

		// Handle goes-to
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

		// Handle regular moves
		if len(moves.Moves) > 0 && moves.Follows == "" && moves.GoesTo == "" {
			actSeq++
			act := &model.Act{
				Seq:  actSeq,
				Kind: model.ActKindMove,
				Ok:   true,
			}

			stepSeq = 0
			for _, mv := range moves.Moves {
				stepSeq++
				step := adaptMove(mv, stepSeq)
				act.Steps = append(act.Steps, step)
				if !step.Ok {
					act.Ok = false
				}
			}

			ux.Acts = append(ux.Acts, act)
		}

		// Handle scouts
		for _, scout := range moves.Scouts {
			actSeq++
			act := &model.Act{
				Seq:  actSeq,
				Kind: model.ActKindScout,
				Ok:   true,
			}

			stepSeq = 0
			for _, mv := range scout.Moves {
				stepSeq++
				step := adaptMove(mv, stepSeq)
				act.Steps = append(act.Steps, step)
				if !step.Ok {
					act.Ok = false
				}
			}

			ux.Acts = append(ux.Acts, act)
		}

		rx.Units = append(rx.Units, ux)
	}

	return rf, rx, nil
}

// adaptMove converts an azul.Move_t to a model.Step.
func adaptMove(mv *azul.Move_t, seq int) *model.Step {
	step := &model.Step{
		Seq: seq,
		Ok:  mv.Result == results.Succeeded || mv.Result == results.StayedInPlace,
	}

	// Determine step kind
	if mv.Still {
		step.Kind = model.StepKindStill
	} else if mv.Advance != direction.Unknown {
		step.Kind = model.StepKindAdv
		step.Dir = mv.Advance.String()
		if !step.Ok {
			step.FailWhy = resultToFailWhy(mv.Result)
		}
	} else {
		step.Kind = model.StepKindObs
	}

	// Extract observations from report
	if mv.Report != nil {
		step.Terr = mv.Report.Terrain.String()

		// Build encounters
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

		// Build borders
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

// resultToFailWhy maps results.Result_e to a failure reason string.
func resultToFailWhy(r results.Result_e) string {
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

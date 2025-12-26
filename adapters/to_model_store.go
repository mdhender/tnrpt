// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package adapters

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/mdhender/tnrpt/direction"
	"github.com/mdhender/tnrpt/model"
	"github.com/mdhender/tnrpt/pipelines/parsers/bistre"
	"github.com/mdhender/tnrpt/results"
)

// ParseStore defines the minimal store interface needed for parsing operations.
type ParseStore interface {
	InsertReportFile(ctx context.Context, rf *model.ReportFile) (int64, error)
	InsertReportExtract(ctx context.Context, rx *model.ReportX) (int64, error)
	InsertUnitExtract(ctx context.Context, ux *model.UnitX) (int64, error)
	InsertAct(ctx context.Context, act *model.Act) (int64, error)
	InsertStep(ctx context.Context, step *model.Step) (int64, error)
}

// BistreTurnToStore converts a bistre.Turn_t to model types and persists them via Store.
// Returns the ReportFile ID and ReportX ID that were inserted.
func BistreTurnToStore(ctx context.Context, store ParseStore, source string, turn *bistre.Turn_t, game, clanNo string) (int64, int64, error) {
	now := time.Now().UTC()
	turnNo := 100*turn.Year + turn.Month

	// Insert ReportFile
	rf := &model.ReportFile{
		Game:      game,
		ClanNo:    clanNo,
		TurnNo:    turnNo,
		Name:      source,
		SHA256:    computeSHA256(source),
		Mime:      "application/octet-stream",
		CreatedAt: now,
	}
	rfID, err := store.InsertReportFile(ctx, rf)
	if err != nil {
		return 0, 0, fmt.Errorf("insert report file: %w", err)
	}

	// Insert ReportExtract
	rx := &model.ReportX{
		ReportFileID: rfID,
		Game:         game,
		ClanNo:       clanNo,
		TurnNo:       turnNo,
		CreatedAt:    now,
	}
	rxID, err := store.InsertReportExtract(ctx, rx)
	if err != nil {
		return 0, 0, fmt.Errorf("insert report extract: %w", err)
	}

	// Convert and insert each unit's moves
	for unitId, moves := range turn.UnitMoves {
		if err := insertUnitMoves(ctx, store, rxID, rfID, turnNo, unitId, moves); err != nil {
			return 0, 0, fmt.Errorf("insert unit %s: %w", unitId, err)
		}
	}

	return rfID, rxID, nil
}

// ParseStoreMinimal defines the minimal store interface for parsing when ReportFile already exists.
type ParseStoreMinimal interface {
	InsertReportExtract(ctx context.Context, rx *model.ReportX) (int64, error)
	InsertUnitExtract(ctx context.Context, ux *model.UnitX) (int64, error)
	InsertAct(ctx context.Context, act *model.Act) (int64, error)
	InsertStep(ctx context.Context, step *model.Step) (int64, error)
}

// BistreTurnToStoreWithReportFile converts a bistre.Turn_t to model types and persists them,
// using an existing ReportFile. Returns the ReportX ID that was inserted.
func BistreTurnToStoreWithReportFile(ctx context.Context, store ParseStoreMinimal, rf *model.ReportFile, turn *bistre.Turn_t) (int64, error) {
	now := time.Now().UTC()
	turnNo := 100*turn.Year + turn.Month

	// Insert ReportExtract
	rx := &model.ReportX{
		ReportFileID: rf.ID,
		Game:         rf.Game,
		ClanNo:       rf.ClanNo,
		TurnNo:       turnNo,
		CreatedAt:    now,
	}
	rxID, err := store.InsertReportExtract(ctx, rx)
	if err != nil {
		return 0, fmt.Errorf("insert report extract: %w", err)
	}

	// Convert and insert each unit's moves
	for unitId, moves := range turn.UnitMoves {
		if err := insertUnitMovesMinimal(ctx, store, rxID, rf.ID, turnNo, unitId, moves); err != nil {
			return 0, fmt.Errorf("insert unit %s: %w", unitId, err)
		}
	}

	return rxID, nil
}

func insertUnitMoves(ctx context.Context, store ParseStore, rxID, rfID int64, turnNo int, unitId bistre.UnitId_t, moves *bistre.Moves_t) error {
	ux := &model.UnitX{
		ReportXID: rxID,
		UnitID:    string(unitId),
		ClanID:    extractClanID(string(unitId)),
		TurnNo:    turnNo,
		StartTN:   model.TNCoord(moves.PreviousHex),
		EndTN:     model.TNCoord(moves.CurrentHex),
		Src: &model.SrcRef{
			DocID:  rfID,
			UnitID: string(unitId),
			TurnNo: turnNo,
		},
	}

	uxID, err := store.InsertUnitExtract(ctx, ux)
	if err != nil {
		return err
	}

	actSeq := 0

	// Handle follows
	if moves.Follows != "" {
		actSeq++
		act := &model.Act{
			UnitXID:      uxID,
			Seq:          actSeq,
			Kind:         model.ActKindFollow,
			Ok:           true,
			TargetUnitID: string(moves.Follows),
			Src: &model.SrcRef{
				DocID:  rfID,
				UnitID: string(unitId),
				TurnNo: turnNo,
				ActSeq: actSeq,
			},
		}
		if _, err := store.InsertAct(ctx, act); err != nil {
			return err
		}
	}

	// Handle goes-to
	if moves.GoesTo != "" {
		actSeq++
		act := &model.Act{
			UnitXID: uxID,
			Seq:     actSeq,
			Kind:    model.ActKindGoto,
			Ok:      true,
			DestTN:  model.TNCoord(moves.GoesTo),
			Src: &model.SrcRef{
				DocID:  rfID,
				UnitID: string(unitId),
				TurnNo: turnNo,
				ActSeq: actSeq,
			},
		}
		if _, err := store.InsertAct(ctx, act); err != nil {
			return err
		}
	}

	// Handle regular moves
	if len(moves.Moves) > 0 && moves.Follows == "" && moves.GoesTo == "" {
		actSeq++
		act := &model.Act{
			UnitXID: uxID,
			Seq:     actSeq,
			Kind:    model.ActKindMove,
			Ok:      true,
			Src: &model.SrcRef{
				DocID:  rfID,
				UnitID: string(unitId),
				TurnNo: turnNo,
				ActSeq: actSeq,
			},
		}

		actID, err := store.InsertAct(ctx, act)
		if err != nil {
			return err
		}

		stepSeq := 0
		for _, mv := range moves.Moves {
			stepSeq++
			step := adaptBistreMove(mv, actID, stepSeq)
			step.Src = &model.SrcRef{
				DocID:   rfID,
				UnitID:  string(unitId),
				TurnNo:  turnNo,
				ActSeq:  actSeq,
				StepSeq: stepSeq,
			}
			if _, err := store.InsertStep(ctx, step); err != nil {
				return err
			}
		}
	}

	// Handle scouts
	for _, scout := range moves.Scouts {
		actSeq++
		act := &model.Act{
			UnitXID: uxID,
			Seq:     actSeq,
			Kind:    model.ActKindScout,
			Ok:      true,
			Src: &model.SrcRef{
				DocID:  rfID,
				UnitID: string(unitId),
				TurnNo: turnNo,
				ActSeq: actSeq,
			},
		}

		actID, err := store.InsertAct(ctx, act)
		if err != nil {
			return err
		}

		stepSeq := 0
		for _, mv := range scout.Moves {
			stepSeq++
			step := adaptBistreMove(mv, actID, stepSeq)
			step.Src = &model.SrcRef{
				DocID:   rfID,
				UnitID:  string(unitId),
				TurnNo:  turnNo,
				ActSeq:  actSeq,
				StepSeq: stepSeq,
			}
			if _, err := store.InsertStep(ctx, step); err != nil {
				return err
			}
		}
	}

	return nil
}

func insertUnitMovesMinimal(ctx context.Context, store ParseStoreMinimal, rxID, rfID int64, turnNo int, unitId bistre.UnitId_t, moves *bistre.Moves_t) error {
	ux := &model.UnitX{
		ReportXID: rxID,
		UnitID:    string(unitId),
		ClanID:    extractClanID(string(unitId)),
		TurnNo:    turnNo,
		StartTN:   model.TNCoord(moves.PreviousHex),
		EndTN:     model.TNCoord(moves.CurrentHex),
		Src: &model.SrcRef{
			DocID:  rfID,
			UnitID: string(unitId),
			TurnNo: turnNo,
		},
	}

	uxID, err := store.InsertUnitExtract(ctx, ux)
	if err != nil {
		return err
	}

	actSeq := 0

	// Handle follows
	if moves.Follows != "" {
		actSeq++
		act := &model.Act{
			UnitXID:      uxID,
			Seq:          actSeq,
			Kind:         model.ActKindFollow,
			Ok:           true,
			TargetUnitID: string(moves.Follows),
			Src: &model.SrcRef{
				DocID:  rfID,
				UnitID: string(unitId),
				TurnNo: turnNo,
				ActSeq: actSeq,
			},
		}
		if _, err := store.InsertAct(ctx, act); err != nil {
			return err
		}
	}

	// Handle goes-to
	if moves.GoesTo != "" {
		actSeq++
		act := &model.Act{
			UnitXID: uxID,
			Seq:     actSeq,
			Kind:    model.ActKindGoto,
			Ok:      true,
			DestTN:  model.TNCoord(moves.GoesTo),
			Src: &model.SrcRef{
				DocID:  rfID,
				UnitID: string(unitId),
				TurnNo: turnNo,
				ActSeq: actSeq,
			},
		}
		if _, err := store.InsertAct(ctx, act); err != nil {
			return err
		}
	}

	// Handle regular moves
	if len(moves.Moves) > 0 && moves.Follows == "" && moves.GoesTo == "" {
		actSeq++
		act := &model.Act{
			UnitXID: uxID,
			Seq:     actSeq,
			Kind:    model.ActKindMove,
			Ok:      true,
			Src: &model.SrcRef{
				DocID:  rfID,
				UnitID: string(unitId),
				TurnNo: turnNo,
				ActSeq: actSeq,
			},
		}

		actID, err := store.InsertAct(ctx, act)
		if err != nil {
			return err
		}

		stepSeq := 0
		for _, mv := range moves.Moves {
			stepSeq++
			step := adaptBistreMove(mv, actID, stepSeq)
			step.Src = &model.SrcRef{
				DocID:   rfID,
				UnitID:  string(unitId),
				TurnNo:  turnNo,
				ActSeq:  actSeq,
				StepSeq: stepSeq,
			}
			if _, err := store.InsertStep(ctx, step); err != nil {
				return err
			}
		}
	}

	// Handle scouts
	for _, scout := range moves.Scouts {
		actSeq++
		act := &model.Act{
			UnitXID: uxID,
			Seq:     actSeq,
			Kind:    model.ActKindScout,
			Ok:      true,
			Src: &model.SrcRef{
				DocID:  rfID,
				UnitID: string(unitId),
				TurnNo: turnNo,
				ActSeq: actSeq,
			},
		}

		actID, err := store.InsertAct(ctx, act)
		if err != nil {
			return err
		}

		stepSeq := 0
		for _, mv := range scout.Moves {
			stepSeq++
			step := adaptBistreMove(mv, actID, stepSeq)
			step.Src = &model.SrcRef{
				DocID:   rfID,
				UnitID:  string(unitId),
				TurnNo:  turnNo,
				ActSeq:  actSeq,
				StepSeq: stepSeq,
			}
			if _, err := store.InsertStep(ctx, step); err != nil {
				return err
			}
		}
	}

	return nil
}

func adaptBistreMove(mv *bistre.Move_t, actID int64, seq int) *model.Step {
	step := &model.Step{
		ActID: actID,
		Seq:   seq,
		Ok:    mv.Result == results.Succeeded || mv.Result == results.StayedInPlace,
	}

	// Determine step kind
	if mv.Still {
		step.Kind = model.StepKindStill
	} else if mv.Advance != direction.Unknown {
		step.Kind = model.StepKindAdv
		step.Dir = mv.Advance.String()
		if !step.Ok {
			step.FailWhy = bistreResultToFailWhy(mv.Result)
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

func bistreResultToFailWhy(r results.Result_e) string {
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

func computeSHA256(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)
}

// extractClanID extracts the clan ID from a unit ID.
// Unit IDs are formatted as "CCCC" (tribal unit) or "CCCCsN" (sub-unit).
// Returns the first 4 characters as the clan ID.
func extractClanID(unitID string) string {
	if len(unitID) >= 4 {
		return unitID[:4]
	}
	return unitID
}

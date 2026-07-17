package grade_compute

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/gradecompute"
)

// CheckRecomputeEligibility reports whether saving a numeric cell on the given
// job phase should drive a scaled-summary recompute, and — when it should — the
// set of outcome_criteria ids that participate in the resolved scheme's active
// component graph.
//
// It reuses the exact scheme-resolution ladder the roll-up itself walks (phase →
// template-phase scheme → score scale → component-criteria junction), so the
// classification can never disagree with what ComputePhaseOutcome would attempt.
// A phase whose scheme resolves no score scale (a ledger scheme) or has no scoped
// criteria drives nothing: it returns (false, nil, nil) so the caller acks the
// cell as not-applicable instead of running a roll-up that would fail loud.
//
// The distinction between "not eligible" and "error" is deliberate: a phase that
// simply is not gradable is a normal outcome (false, nil, nil), while only a
// genuine repository read failure returns a non-nil error. This read is an
// internal classification over the same config the grade grid already exposes to
// the acting principal, so it carries no additional action gate.
func (uc *ComputePhaseOutcomeUseCase) CheckRecomputeEligibility(ctx context.Context, jobPhaseID string) (bool, map[string]bool, error) {
	if jobPhaseID == "" {
		return false, nil, nil
	}

	phase, err := uc.readJobPhase(ctx, jobPhaseID)
	if err != nil {
		return false, nil, err
	}

	var templatePhaseSchemeID *string
	if phase.TemplatePhaseId != nil && *phase.TemplatePhaseId != "" {
		tplPhase, terr := uc.readJobTemplatePhase(ctx, *phase.TemplatePhaseId)
		if terr != nil {
			return false, nil, terr
		}
		if tplPhase != nil {
			templatePhaseSchemeID = tplPhase.ScoringSchemeId
		}
	}

	schemeID, rerr := gradecompute.ResolveScoringScheme(phase.ScoringSchemeId, templatePhaseSchemeID)
	if rerr != nil {
		// No scheme resolves → the phase produces no scaled summary.
		return false, nil, nil
	}

	scheme, err := uc.readScoringScheme(ctx, schemeID)
	if err != nil {
		return false, nil, err
	}
	if scheme.ScoreScaleId == nil || *scheme.ScoreScaleId == "" {
		// A scheme with no score scale is a ledger, not a graded band.
		return false, nil, nil
	}

	inScope, err := uc.inScopeCriteria(ctx, schemeID)
	if err != nil {
		return false, nil, err
	}
	if len(inScope) == 0 {
		return false, nil, nil
	}

	return true, inScope, nil
}

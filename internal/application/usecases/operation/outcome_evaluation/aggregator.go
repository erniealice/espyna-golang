package outcome_evaluation

import (
	"fmt"

	enums "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	jobpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary"
	phasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/phase_outcome_summary"
	outcomepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

// aggregatePhase computes a PhaseOutcomeSummary from a set of task outcomes.
// It does NOT set identity fields (Id, JobPhaseId, JobId) — those are assigned
// by the caller after persisting.
func aggregatePhase(
	outcomes []*outcomepb.TaskOutcome,
	scoringMethod enums.ScoringMethod,
) (*phasepb.PhaseOutcomeSummary, error) {
	counts := countDeterminations(outcomes)
	evaluableCount := counts.pass + counts.fail + counts.conditional

	var (
		score          *float64
		phaseDet       enums.OverallDetermination
	)

	switch scoringMethod {
	case enums.ScoringMethod_SCORING_METHOD_EQUAL_WEIGHT:
		if evaluableCount > 0 {
			s := float64(counts.pass+counts.conditional) / float64(evaluableCount)
			score = float64Ptr(s)
			phaseDet = overallFromScore(s)
		} else {
			phaseDet = enums.OverallDetermination_OVERALL_DETERMINATION_IN_PROGRESS
		}

	case enums.ScoringMethod_SCORING_METHOD_WEIGHTED_AVERAGE:
		var sumWeighted, sumWeights float64
		for _, o := range outcomes {
			if !isEvaluable(o.Determination) {
				continue
			}
			w := outcomeWeight(o)
			sumWeights += w
			if isPass(o.Determination) {
				sumWeighted += w
			}
		}
		if sumWeights > 0 {
			s := sumWeighted / sumWeights
			score = float64Ptr(s)
			phaseDet = overallFromScore(s)
		} else {
			phaseDet = enums.OverallDetermination_OVERALL_DETERMINATION_IN_PROGRESS
		}

	case enums.ScoringMethod_SCORING_METHOD_MINIMUM_DETERMINATION:
		dets := make([]enums.Determination, 0, len(outcomes))
		for _, o := range outcomes {
			dets = append(dets, o.Determination)
		}
		worst := worstDetermination(dets)
		phaseDet = determinationToOverall(worst)

	case enums.ScoringMethod_SCORING_METHOD_PERCENTAGE_PASS:
		if evaluableCount > 0 {
			pct := float64(counts.pass+counts.conditional) / float64(evaluableCount)
			score = float64Ptr(pct)
			if pct >= 1.0 {
				phaseDet = enums.OverallDetermination_OVERALL_DETERMINATION_ACCEPTED
			} else if pct > 0 {
				phaseDet = enums.OverallDetermination_OVERALL_DETERMINATION_CONDITIONALLY_ACCEPTED
			} else {
				phaseDet = enums.OverallDetermination_OVERALL_DETERMINATION_REJECTED
			}
		} else {
			phaseDet = enums.OverallDetermination_OVERALL_DETERMINATION_IN_PROGRESS
		}

	default:
		return nil, fmt.Errorf("unsupported scoring method: %v", scoringMethod)
	}

	summary := &phasepb.PhaseOutcomeSummary{
		ScoringMethod:      scoringMethod,
		PhaseDetermination: phaseDet,
		SummaryScore:       score,
		TotalCriteriaCount: int32(len(outcomes)),
		PassCount:          int32(counts.pass),
		FailCount:          int32(counts.fail),
		ConditionalCount:   int32(counts.conditional),
		DeferredCount:      int32(counts.deferred),
		NaCount:            int32(counts.na),
	}
	return summary, nil
}

// aggregateJob computes a JobOutcomeSummary.
// When phaseSummaries is non-empty, it aggregates across phase-level scores.
// When phaseSummaries is empty, it falls back to aggregating direct task outcomes.
func aggregateJob(
	phaseSummaries []*phasepb.PhaseOutcomeSummary,
	outcomes []*outcomepb.TaskOutcome,
	scoringMethod enums.ScoringMethod,
) (*jobpb.JobOutcomeSummary, error) {
	if len(phaseSummaries) > 0 {
		return aggregateJobFromPhases(phaseSummaries, scoringMethod)
	}
	// No phases — flatten directly from task outcomes
	phaseSummary, err := aggregatePhase(outcomes, scoringMethod)
	if err != nil {
		return nil, err
	}
	return &jobpb.JobOutcomeSummary{
		ScoringMethod:        scoringMethod,
		OverallDetermination: phaseSummaryToOverall(phaseSummary),
		SummaryScore:         phaseSummary.SummaryScore,
		TotalCriteriaCount:   phaseSummary.TotalCriteriaCount,
		PassCount:            phaseSummary.PassCount,
		FailCount:            phaseSummary.FailCount,
		ConditionalCount:     phaseSummary.ConditionalCount,
		DeferredCount:        phaseSummary.DeferredCount,
		NaCount:              phaseSummary.NaCount,
	}, nil
}

// aggregateJobFromPhases rolls up existing phase summaries into a job summary.
func aggregateJobFromPhases(
	phases []*phasepb.PhaseOutcomeSummary,
	scoringMethod enums.ScoringMethod,
) (*jobpb.JobOutcomeSummary, error) {
	var (
		totalCriteria, pass, fail, conditional, deferred, na int32
		sumScore, sumWeights                                  float64
		overallDet                                            enums.OverallDetermination
	)

	for _, p := range phases {
		totalCriteria += p.TotalCriteriaCount
		pass += p.PassCount
		fail += p.FailCount
		conditional += p.ConditionalCount
		deferred += p.DeferredCount
		na += p.NaCount

		if p.SummaryScore != nil {
			// Each phase is weighted equally unless we later add per-phase weights
			sumScore += *p.SummaryScore
			sumWeights++
		}
	}

	var score *float64
	evaluablePhases := sumWeights

	switch scoringMethod {
	case enums.ScoringMethod_SCORING_METHOD_EQUAL_WEIGHT,
		enums.ScoringMethod_SCORING_METHOD_WEIGHTED_AVERAGE,
		enums.ScoringMethod_SCORING_METHOD_PERCENTAGE_PASS:
		if evaluablePhases > 0 {
			s := sumScore / evaluablePhases
			score = float64Ptr(s)
			overallDet = overallFromScore(s)
		} else {
			overallDet = enums.OverallDetermination_OVERALL_DETERMINATION_IN_PROGRESS
		}

	case enums.ScoringMethod_SCORING_METHOD_MINIMUM_DETERMINATION:
		phaseDets := make([]enums.OverallDetermination, 0, len(phases))
		for _, p := range phases {
			phaseDets = append(phaseDets, p.PhaseDetermination)
		}
		overallDet = worstOverallDetermination(phaseDets)

	default:
		return nil, fmt.Errorf("unsupported scoring method: %v", scoringMethod)
	}

	return &jobpb.JobOutcomeSummary{
		ScoringMethod:        scoringMethod,
		OverallDetermination: overallDet,
		SummaryScore:         score,
		TotalCriteriaCount:   totalCriteria,
		PassCount:            pass,
		FailCount:            fail,
		ConditionalCount:     conditional,
		DeferredCount:        deferred,
		NaCount:              na,
	}, nil
}

// determinationCounts holds bucketed outcome counts for aggregation.
type determinationCounts struct {
	pass        int
	fail        int
	conditional int
	deferred    int
	na          int
}

func countDeterminations(outcomes []*outcomepb.TaskOutcome) determinationCounts {
	var c determinationCounts
	for _, o := range outcomes {
		switch o.Determination {
		case enums.Determination_DETERMINATION_PASS:
			c.pass++
		case enums.Determination_DETERMINATION_FAIL:
			c.fail++
		case enums.Determination_DETERMINATION_PASS_WITH_CONDITION:
			c.conditional++
		case enums.Determination_DETERMINATION_DEFERRED:
			c.deferred++
		case enums.Determination_DETERMINATION_NOT_APPLICABLE:
			c.na++
		}
	}
	return c
}

// outcomeWeight returns the weight of a task outcome's criteria.
// Falls back to 1.0 when no weight is available on the outcome itself.
func outcomeWeight(o *outcomepb.TaskOutcome) float64 {
	if o.CriteriaVersion != nil && o.CriteriaVersion.Weight > 0 {
		return o.CriteriaVersion.Weight
	}
	return 1.0
}

// determinationToOverall maps a task-level Determination to an OverallDetermination.
func determinationToOverall(d enums.Determination) enums.OverallDetermination {
	switch d {
	case enums.Determination_DETERMINATION_PASS:
		return enums.OverallDetermination_OVERALL_DETERMINATION_ACCEPTED
	case enums.Determination_DETERMINATION_PASS_WITH_CONDITION:
		return enums.OverallDetermination_OVERALL_DETERMINATION_CONDITIONALLY_ACCEPTED
	case enums.Determination_DETERMINATION_FAIL:
		return enums.OverallDetermination_OVERALL_DETERMINATION_REJECTED
	case enums.Determination_DETERMINATION_DEFERRED:
		return enums.OverallDetermination_OVERALL_DETERMINATION_DEFERRED
	default:
		return enums.OverallDetermination_OVERALL_DETERMINATION_IN_PROGRESS
	}
}

// phaseSummaryToOverall extracts the OverallDetermination from a PhaseOutcomeSummary.
func phaseSummaryToOverall(p *phasepb.PhaseOutcomeSummary) enums.OverallDetermination {
	return p.PhaseDetermination
}

// worstOverallDetermination returns the worst (most severe) OverallDetermination.
func worstOverallDetermination(ds []enums.OverallDetermination) enums.OverallDetermination {
	rank := func(d enums.OverallDetermination) int {
		switch d {
		case enums.OverallDetermination_OVERALL_DETERMINATION_REJECTED:
			return 0
		case enums.OverallDetermination_OVERALL_DETERMINATION_CONDITIONALLY_ACCEPTED:
			return 1
		case enums.OverallDetermination_OVERALL_DETERMINATION_DEFERRED:
			return 2
		case enums.OverallDetermination_OVERALL_DETERMINATION_ACCEPTED:
			return 3
		default:
			return -1
		}
	}

	worst := enums.OverallDetermination_OVERALL_DETERMINATION_IN_PROGRESS
	initialized := false
	for _, d := range ds {
		r := rank(d)
		if r < 0 {
			continue
		}
		if !initialized || r < rank(worst) {
			worst = d
			initialized = true
		}
	}
	return worst
}

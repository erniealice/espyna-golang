package outcome_evaluation

import (
	enums "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
)

// threshold returns the threshold value for a given role, and a bool indicating presence.
func threshold(tmap map[enums.ThresholdRole]float64, role enums.ThresholdRole) (float64, bool) {
	v, ok := tmap[role]
	return v, ok
}

// determinationRank returns an ordinal so that the "worst" determination can be found.
// Lower rank = worse outcome. NOT_APPLICABLE and NOT_EVALUATED are excluded from ranking
// (treated as non-evaluable).
func determinationRank(d enums.Determination) int {
	switch d {
	case enums.Determination_DETERMINATION_FAIL:
		return 0
	case enums.Determination_DETERMINATION_PASS_WITH_CONDITION:
		return 1
	case enums.Determination_DETERMINATION_DEFERRED:
		return 2
	case enums.Determination_DETERMINATION_PASS:
		return 3
	default:
		// NOT_EVALUATED, NOT_APPLICABLE, UNSPECIFIED — treated as non-evaluable
		return -1
	}
}

// worstDetermination returns the determination with the lowest rank among
// the supplied slice, ignoring non-evaluable values.
func worstDetermination(ds []enums.Determination) enums.Determination {
	worst := enums.Determination_DETERMINATION_PASS
	initialized := false
	for _, d := range ds {
		rank := determinationRank(d)
		if rank < 0 {
			continue
		}
		if !initialized || rank < determinationRank(worst) {
			worst = d
			initialized = true
		}
	}
	if !initialized {
		return enums.Determination_DETERMINATION_NOT_EVALUATED
	}
	return worst
}

// isEvaluable returns true for determinations that count toward pass/fail scoring.
func isEvaluable(d enums.Determination) bool {
	switch d {
	case enums.Determination_DETERMINATION_PASS,
		enums.Determination_DETERMINATION_FAIL,
		enums.Determination_DETERMINATION_PASS_WITH_CONDITION:
		return true
	default:
		return false
	}
}

// isPass returns true for determinations considered passing for percentage calculations.
func isPass(d enums.Determination) bool {
	return d == enums.Determination_DETERMINATION_PASS ||
		d == enums.Determination_DETERMINATION_PASS_WITH_CONDITION
}

// overallFromScore maps a normalized 0..1 score to an OverallDetermination.
// The thresholds here are conventional — conditionally accepted is ≥ 0.5.
func overallFromScore(score float64) enums.OverallDetermination {
	if score >= 1.0 {
		return enums.OverallDetermination_OVERALL_DETERMINATION_ACCEPTED
	}
	if score >= 0.5 {
		return enums.OverallDetermination_OVERALL_DETERMINATION_CONDITIONALLY_ACCEPTED
	}
	return enums.OverallDetermination_OVERALL_DETERMINATION_REJECTED
}

// float64Ptr is a convenience helper to take the address of a float64 literal.
func float64Ptr(v float64) *float64 { return &v }

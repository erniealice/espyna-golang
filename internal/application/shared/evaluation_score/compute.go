// Package evaluationscore provides the single PURE evaluation-score projection —
// ComputeEvaluationScore — a weighted average over snapshotted response values.
//
// The rule (Q-EVAL-SCORE-1):
//
//	score = Σ(criteria_weight × numeric_value) / Σ(criteria_weight)
//
//	- a negative weight is rejected (returns an error)
//	- a zero weight is excluded from BOTH the numerator and the denominator
//	- a response whose snapshotted criteria_type is NOT numeric
//	  (∉ {numeric_range, numeric_score}) is excluded from BOTH sums — paired with
//	  the template-activation guard that already rejects a weighted non-numeric
//	  dimension, so a non-numeric criterion can never silently deflate the score
//	- an empty input (no numeric, non-zero-weight responses) yields nil (no score)
//
// It is computed from the SNAPSHOT columns on each evaluation_response, never the
// live OutcomeCriteria rubric, so editing a rubric after submit never moves a
// frozen overall_score.
//
// Charter — this package MUST NOT import:
//   - proto entity types (esqyma/...) — it takes Go primitives only; the calling
//     use case (SubmitEvaluation) handles all proto translation
//   - DB drivers or adapter packages
//   - anything under internal/application/usecases/... (no upward use-case imports)
//   - internal/application/ports
//
// Depends only on the Go standard library plus the local Response type below.
//
// Consumers (keep in sync):
//   - usecases/domain/operation/evaluation/submit_evaluation.go
package evaluationscore

import "errors"

// ErrNegativeWeight is returned when any included response carries a negative
// snapshotted criteria_weight. A negative weight is a data-integrity violation
// (the DB CHECK rejects it at write time; this is the belt-and-suspenders gate).
var ErrNegativeWeight = errors.New("evaluation score: criteria_weight cannot be negative")

// Response is the minimal, proto-free shape the calculator needs from one
// snapshotted evaluation_response row. It deliberately mirrors only the four
// score-relevant snapshot columns so the leaf never grows a proto import.
type Response struct {
	// IsNumeric is true iff the snapshotted criteria_type ∈
	// {numeric_range, numeric_score}. Non-numeric responses are excluded from
	// BOTH sums (the caller derives this from the snapshotted CriteriaType).
	IsNumeric bool
	// Weight is the snapshotted criteria_weight (nil = unweighted → excluded).
	Weight *float64
	// NumericValue is the snapshotted numeric answer (nil = no answer → excluded).
	NumericValue *float64
}

// ComputeEvaluationScore returns the weighted-average score over the numeric,
// positively-weighted responses, or nil when no such response exists.
//
// Returns ErrNegativeWeight if any included response has a negative weight.
func ComputeEvaluationScore(responses []Response) (*float64, error) {
	var weightedSum float64
	var weightSum float64

	for _, r := range responses {
		// Exclude non-numeric criteria from BOTH sums.
		if !r.IsNumeric {
			continue
		}
		// A missing value or missing weight cannot contribute.
		if r.Weight == nil || r.NumericValue == nil {
			continue
		}
		w := *r.Weight
		if w < 0 {
			return nil, ErrNegativeWeight
		}
		// Zero weight is excluded from BOTH sums (no influence, no denominator).
		if w == 0 {
			continue
		}
		weightedSum += w * (*r.NumericValue)
		weightSum += w
	}

	// Empty → no score.
	if weightSum == 0 {
		return nil, nil
	}

	score := weightedSum / weightSum
	return &score, nil
}

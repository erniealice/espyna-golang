package outcome_evaluation

import (
	"fmt"

	portsdomain "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	enums "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	outcomepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

// numericScoreEvaluator evaluates CRITERIA_TYPE_NUMERIC_SCORE outcomes.
//
// Score logic:
//  1. Validate value is within criteria.MinScore..MaxScore
//  2. If DISTINCTION_THRESHOLD threshold exists and value >= it → PASS (distinction noted)
//  3. If PASS_THRESHOLD exists and value >= it → PASS
//  4. Else → FAIL
type numericScoreEvaluator struct{}

func (e *numericScoreEvaluator) Evaluate(
	outcome *outcomepb.TaskOutcome,
	ctx *EvaluationContext,
) (*portsdomain.EvaluationResult, error) {
	if outcome.NumericValue == nil {
		return nil, fmt.Errorf("numeric_score evaluation requires a numeric_value")
	}
	v := *outcome.NumericValue

	// Validate within criteria score bounds
	if ctx.Criteria.MinScore != nil {
		min := float64(*ctx.Criteria.MinScore)
		if v < min {
			return nil, fmt.Errorf("value %.4g is below minimum score %.4g", v, min)
		}
	}
	if ctx.Criteria.MaxScore != nil {
		max := float64(*ctx.Criteria.MaxScore)
		if v > max {
			return nil, fmt.Errorf("value %.4g exceeds maximum score %.4g", v, max)
		}
	}

	// Distinction threshold — a special grade of PASS
	if dist, ok := threshold(ctx.Thresholds, enums.ThresholdRole_THRESHOLD_ROLE_DISTINCTION_THRESHOLD); ok {
		if v >= dist {
			return &portsdomain.EvaluationResult{
				Determination: enums.Determination_DETERMINATION_PASS,
				Source:        enums.DeterminationSource_DETERMINATION_SOURCE_AUTO_COMPUTED,
				Notes:         fmt.Sprintf("score %.4g meets distinction threshold (>= %.4g)", v, dist),
			}, nil
		}
	}

	// Standard pass threshold
	if passThresh, ok := threshold(ctx.Thresholds, enums.ThresholdRole_THRESHOLD_ROLE_PASS_THRESHOLD); ok {
		if v >= passThresh {
			return &portsdomain.EvaluationResult{
				Determination: enums.Determination_DETERMINATION_PASS,
				Source:        enums.DeterminationSource_DETERMINATION_SOURCE_AUTO_COMPUTED,
			}, nil
		}
		return &portsdomain.EvaluationResult{
			Determination: enums.Determination_DETERMINATION_FAIL,
			Source:        enums.DeterminationSource_DETERMINATION_SOURCE_AUTO_COMPUTED,
			Notes:         fmt.Sprintf("score %.4g is below pass threshold %.4g", v, passThresh),
		}, nil
	}

	// No threshold configured — cannot auto-determine
	return &portsdomain.EvaluationResult{
		Determination: enums.Determination_DETERMINATION_NOT_EVALUATED,
		Source:        enums.DeterminationSource_DETERMINATION_SOURCE_AUTO_COMPUTED,
		Notes:         "no pass threshold configured for numeric score criteria",
	}, nil
}

package outcome_evaluation

import (
	"fmt"

	enums "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	criteriapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
	thresholdpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/criteria_threshold"
	optionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/criteria_option"
	checkpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome_check"
	outcomepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
	portsdomain "github.com/erniealice/espyna-golang/internal/application/ports/domain"
)

// evaluator is the internal interface all criterion evaluators implement.
// Each evaluator is a pure function — it receives pre-fetched data and
// returns a result without touching the database.
type evaluator interface {
	Evaluate(outcome *outcomepb.TaskOutcome, ctx *EvaluationContext) (*portsdomain.EvaluationResult, error)
}

// EvaluationContext carries all pre-fetched configuration needed during evaluation.
type EvaluationContext struct {
	Criteria   *criteriapb.OutcomeCriteria
	Thresholds map[enums.ThresholdRole]float64
	Options    []*optionpb.CriteriaOption
	Checks     []*checkpb.TaskOutcomeCheck
}

// newEvaluationContext builds an EvaluationContext, indexing thresholds by role for O(1) lookup.
func newEvaluationContext(
	criteria *criteriapb.OutcomeCriteria,
	thresholds []*thresholdpb.CriteriaThreshold,
	options []*optionpb.CriteriaOption,
	checks []*checkpb.TaskOutcomeCheck,
) *EvaluationContext {
	tmap := make(map[enums.ThresholdRole]float64, len(thresholds))
	for _, t := range thresholds {
		tmap[t.ThresholdRole] = t.Value
	}
	return &EvaluationContext{
		Criteria:   criteria,
		Thresholds: tmap,
		Options:    options,
		Checks:     checks,
	}
}

// newEvaluator returns the correct evaluator for the given CriteriaType.
func newEvaluator(ct enums.CriteriaType) (evaluator, error) {
	switch ct {
	case enums.CriteriaType_CRITERIA_TYPE_NUMERIC_RANGE:
		return &numericRangeEvaluator{}, nil
	case enums.CriteriaType_CRITERIA_TYPE_NUMERIC_SCORE:
		return &numericScoreEvaluator{}, nil
	case enums.CriteriaType_CRITERIA_TYPE_PASS_FAIL:
		return &passFailEvaluator{}, nil
	case enums.CriteriaType_CRITERIA_TYPE_CATEGORICAL:
		return &categoricalEvaluator{}, nil
	case enums.CriteriaType_CRITERIA_TYPE_TEXT:
		return &textEvaluator{}, nil
	case enums.CriteriaType_CRITERIA_TYPE_MULTI_CHECK:
		return &multiCheckEvaluator{}, nil
	default:
		return nil, fmt.Errorf("unsupported criteria type: %v", ct)
	}
}

package outcome_evaluation

import (
	"fmt"

	enums "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	outcomepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
	portsdomain "github.com/erniealice/espyna-golang/internal/application/ports/domain"
)

// passFailEvaluator evaluates CRITERIA_TYPE_PASS_FAIL outcomes.
// true → PASS, false → FAIL.
type passFailEvaluator struct{}

func (e *passFailEvaluator) Evaluate(
	outcome *outcomepb.TaskOutcome,
	_ *EvaluationContext,
) (*portsdomain.EvaluationResult, error) {
	if outcome.PassFailValue == nil {
		return nil, fmt.Errorf("pass_fail evaluation requires a pass_fail_value")
	}

	if *outcome.PassFailValue {
		return &portsdomain.EvaluationResult{
			Determination: enums.Determination_DETERMINATION_PASS,
			Source:        enums.DeterminationSource_DETERMINATION_SOURCE_AUTO_COMPUTED,
		}, nil
	}

	return &portsdomain.EvaluationResult{
		Determination: enums.Determination_DETERMINATION_FAIL,
		Source:        enums.DeterminationSource_DETERMINATION_SOURCE_AUTO_COMPUTED,
	}, nil
}

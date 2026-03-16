package outcome_evaluation

import (
	enums "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	outcomepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
	portsdomain "github.com/erniealice/espyna-golang/internal/application/ports/domain"
)

// textEvaluator evaluates CRITERIA_TYPE_TEXT outcomes.
// Text responses cannot be auto-evaluated — always returns NOT_EVALUATED with
// source HUMAN_ASSIGNED to signal that a reviewer must assign a determination.
type textEvaluator struct{}

func (e *textEvaluator) Evaluate(
	_ *outcomepb.TaskOutcome,
	_ *EvaluationContext,
) (*portsdomain.EvaluationResult, error) {
	return &portsdomain.EvaluationResult{
		Determination: enums.Determination_DETERMINATION_NOT_EVALUATED,
		Source:        enums.DeterminationSource_DETERMINATION_SOURCE_HUMAN_ASSIGNED,
		Notes:         "text responses require human review",
	}, nil
}

package domain

import (
	"context"

	optionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/criteria_option"
	thresholdpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/criteria_threshold"
	enums "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	jobpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary"
	criteriapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
	phasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/phase_outcome_summary"
	outcomepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
	checkpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome_check"
)

// OutcomeEvaluationService defines the contract for outcome evaluation operations.
// Unlike entity services which implement gRPC server interfaces,
// evaluation services use a custom interface because determinations are
// computed aggregates, not stored entities.
type OutcomeEvaluationService interface {
	EvaluateOutcome(ctx context.Context, outcome *outcomepb.TaskOutcome, criteria *criteriapb.OutcomeCriteria, thresholds []*thresholdpb.CriteriaThreshold, options []*optionpb.CriteriaOption, checks []*checkpb.TaskOutcomeCheck) (*EvaluationResult, error)
	AggregatePhase(ctx context.Context, outcomes []*outcomepb.TaskOutcome, scoringMethod enums.ScoringMethod) (*phasepb.PhaseOutcomeSummary, error)
	AggregateJob(ctx context.Context, phaseSummaries []*phasepb.PhaseOutcomeSummary, outcomes []*outcomepb.TaskOutcome, scoringMethod enums.ScoringMethod) (*jobpb.JobOutcomeSummary, error)
}

// EvaluationResult holds the computed outcome of an evaluation.
type EvaluationResult struct {
	Determination enums.Determination
	Source        enums.DeterminationSource
	Notes         string
}

package outcome_evaluation

import (
	"context"

	portsdomain "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	optionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/criteria_option"
	thresholdpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/criteria_threshold"
	enums "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	jobpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary"
	criteriapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
	phasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/phase_outcome_summary"
	outcomepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
	checkpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome_check"
)

// OutcomeEvaluationServiceImpl is the concrete implementation of OutcomeEvaluationService.
// All methods are pure — no database calls are made here; data arrives pre-fetched.
type OutcomeEvaluationServiceImpl struct{}

// NewOutcomeEvaluationService creates a new evaluation service.
func NewOutcomeEvaluationService() *OutcomeEvaluationServiceImpl {
	return &OutcomeEvaluationServiceImpl{}
}

// EvaluateOutcome computes the determination for a single task outcome.
func (s *OutcomeEvaluationServiceImpl) EvaluateOutcome(
	_ context.Context,
	outcome *outcomepb.TaskOutcome,
	criteria *criteriapb.OutcomeCriteria,
	thresholds []*thresholdpb.CriteriaThreshold,
	options []*optionpb.CriteriaOption,
	checks []*checkpb.TaskOutcomeCheck,
) (*portsdomain.EvaluationResult, error) {
	evalCtx := newEvaluationContext(criteria, thresholds, options, checks)

	ev, err := newEvaluator(criteria.CriteriaType)
	if err != nil {
		return nil, err
	}

	return ev.Evaluate(outcome, evalCtx)
}

// AggregatePhase computes a PhaseOutcomeSummary from a slice of task outcomes.
func (s *OutcomeEvaluationServiceImpl) AggregatePhase(
	_ context.Context,
	outcomes []*outcomepb.TaskOutcome,
	scoringMethod enums.ScoringMethod,
) (*phasepb.PhaseOutcomeSummary, error) {
	return aggregatePhase(outcomes, scoringMethod)
}

// AggregateJob computes a JobOutcomeSummary from phase summaries (or direct outcomes when
// the job has no phases).
func (s *OutcomeEvaluationServiceImpl) AggregateJob(
	_ context.Context,
	phaseSummaries []*phasepb.PhaseOutcomeSummary,
	outcomes []*outcomepb.TaskOutcome,
	scoringMethod enums.ScoringMethod,
) (*jobpb.JobOutcomeSummary, error) {
	return aggregateJob(phaseSummaries, outcomes, scoringMethod)
}

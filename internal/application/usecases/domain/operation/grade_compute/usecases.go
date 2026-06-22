// Package grade_compute is the orchestration use-case layer for the education
// job-grading roll-up. It does the DB data-plumbing around the pure
// gradecompute algorithm (within-criterion MAX -> SUM composite -> score_scale
// transmutation) and persists the transmuted grade onto phase_outcome_summary.
//
// This is the "genuine build" sibling to the L7 CRUD use-cases: it READS the
// already-tested config (scoring_scheme / scoring_component_criteria /
// score_scale / score_scale_band) + the recorded task_outcomes, runs the math,
// and UPSERTS the report-card grade. It is additive and fail-loud: a missing
// scheme, missing in-scope criteria, or a transmutation gap is an error, not a
// silent zero.
package grade_compute

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"

	jobphasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
	jobtemplatephasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
	phaseoutcomesummarypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/phase_outcome_summary"
	scorescalepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale"
	scorescalebandpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale_band"
	scoringcomponentcriteriapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/scoring_component_criteria"
	scoringschemepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/scoring_scheme"
	taskoutcomepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

// Repositories groups the read/write repository dependencies the grade roll-up
// needs. Every field is a Layer-7 DomainServiceServer reached via the operation
// provider — no raw SQL, no adapter imports. The single write target is
// PhaseOutcomeSummary; everything else is read-only config + recorded outcomes.
type Repositories struct {
	JobPhase                 jobphasepb.JobPhaseDomainServiceServer
	JobTemplatePhase         jobtemplatephasepb.JobTemplatePhaseDomainServiceServer
	ScoringScheme            scoringschemepb.ScoringSchemeDomainServiceServer
	ScoringComponentCriteria scoringcomponentcriteriapb.ScoringComponentCriteriaDomainServiceServer
	ScoreScale               scorescalepb.ScoreScaleDomainServiceServer
	ScoreScaleBand           scorescalebandpb.ScoreScaleBandDomainServiceServer
	TaskOutcome              taskoutcomepb.TaskOutcomeDomainServiceServer
	PhaseOutcomeSummary      phaseoutcomesummarypb.PhaseOutcomeSummaryDomainServiceServer
}

// Services groups the cross-cutting services (auth gate, transactions, i18n, id).
type Services struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator      ports.IDGenerator
}

// UseCases contains all grade-compute orchestration use cases.
type UseCases struct {
	ComputePhaseOutcome *ComputePhaseOutcomeUseCase
}

// NewUseCases constructs the grade-compute use-case sub-aggregate.
func NewUseCases(repositories Repositories, services Services) *UseCases {
	return &UseCases{
		ComputePhaseOutcome: NewComputePhaseOutcomeUseCase(repositories, services),
	}
}

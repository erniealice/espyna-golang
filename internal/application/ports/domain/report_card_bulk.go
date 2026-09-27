package domain

import (
	"context"

	jobtemplatephasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
	phaseoutcomesummarypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/phase_outcome_summary"
)

// ReportCardPhaseSummaries is an optional repository capability. The trusted
// request context supplies workspace and principal scope; ids cannot select a
// different workspace. Implementations return newest revisions first.
type ReportCardPhaseSummaries interface {
	ListByJobs(context.Context, []string) ([]*phaseoutcomesummarypb.PhaseOutcomeSummary, error)
}

// ReportCardTemplatePhases is the template-side bulk read used by a card.
type ReportCardTemplatePhases interface {
	ListByTemplates(context.Context, []string) ([]*jobtemplatephasepb.JobTemplatePhase, error)
}

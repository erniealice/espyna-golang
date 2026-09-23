package subscription_group_outcome_export

import (
	"context"
	"errors"
	"strings"

	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	jobdoctmplpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary_document_template"
)

// ResolvePublishedReportCardTemplateUseCase resolves the single applicable,
// PUBLISHED report-card template binding for RENDER-TIME use (R3 / DEC-3
// LOCKED).
//
// The domain package's job_outcome_summary_document_template.FindApplicableUseCase
// gates on job_outcome_summary_document_template:list — a MANAGEMENT
// permission held only by Superadmin (principal types {1,2}). A STAFF
// principal (type 7) holding subscription_group_outcome_export:read cannot
// pass that gate, so apps' resolveTemplateBytes closures previously swallowed
// the denial into (nil, nil): phase downloads 503'd, Year Final silently used
// the embedded fallback, and grade-sheet headers silently omitted the
// document name.
//
// This use case authorizes with THIS package's existing report scope
// (Services.reportScope — subscription_group_outcome_export:read + the
// established principal-type allowlist) and discards the returned scope
// value; only the authorization check matters here (the binding resolver
// itself is workspace-scoped in SQL from trusted context, not by the caller's
// group/servicing grant). It then calls the josdt REPOSITORY's
// FindApplicableJobOutcomeSummaryDocumentTemplate directly — the SAME
// repository the domain use case wraps — bypassing the list-gated use case
// entirely. The repository/adapter already enforces the workspace boundary,
// published-only visibility, the validity window, and phase-scope ambiguity
// (LIMIT 2), so no additional filtering is required here.
//
// The returned binding is minimized before it leaves this use case: audit
// fields (created_by, published_by) are cleared, since this seam exists only
// to hand render code enough to fetch bytes (document_template_id +
// storage locator), never to expose who authored/published the binding to a
// STAFF principal that could not have listed it directly.
type ResolvePublishedReportCardTemplateUseCase struct {
	repositories Repositories
	services     Services
}

func (uc *ResolvePublishedReportCardTemplateUseCase) Execute(
	ctx context.Context,
	req *jobdoctmplpb.FindApplicableJobOutcomeSummaryDocumentTemplateRequest,
) (*jobdoctmplpb.FindApplicableJobOutcomeSummaryDocumentTemplateResponse, error) {
	// Discard the scope value: this resolver is not group-scoped by the
	// caller's servicing/membership grant, only gated by the permission +
	// principal-type check reportScope performs.
	if _, err := uc.services.reportScope(ctx); err != nil {
		return nil, err
	}
	if err := validateResolveReportCardTemplateRequest(req); err != nil {
		return nil, err
	}
	if uc.repositories.JobOutcomeSummaryDocumentTemplate == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"subscription_group_outcome_export.errors.unavailable", "report card template resolver is unavailable"))
	}
	response, err := uc.repositories.JobOutcomeSummaryDocumentTemplate.FindApplicableJobOutcomeSummaryDocumentTemplate(ctx, req)
	if err != nil {
		return nil, err
	}
	return minimizeReportCardTemplateResponse(response), nil
}

// validateResolveReportCardTemplateRequest requires price_schedule_id and
// job_template_phase_code, when set, to be trimmed/canonical; an absent
// price_schedule_id is a deliberate fallback-only lookup (see the domain
// request's comment), and an absent phase code selects the whole-year
// binding — both remain valid.
func validateResolveReportCardTemplateRequest(req *jobdoctmplpb.FindApplicableJobOutcomeSummaryDocumentTemplateRequest) error {
	if req == nil {
		return errors.New("report card template resolve request is required")
	}
	if req.PriceScheduleId != nil {
		trimmed := strings.TrimSpace(req.GetPriceScheduleId())
		if trimmed == "" || trimmed != req.GetPriceScheduleId() {
			return errors.New("price_schedule_id must be nonempty and canonical when supplied")
		}
	}
	if req.JobTemplatePhaseCode != nil {
		trimmed := strings.TrimSpace(req.GetJobTemplatePhaseCode())
		if trimmed != req.GetJobTemplatePhaseCode() || !canonicalPhaseCode.MatchString(trimmed) {
			return errors.New("job_template_phase_code must be a canonical path segment when supplied")
		}
	}
	return nil
}

// minimizeReportCardTemplateResponse clears audit-only fields on the
// resolved binding before it leaves the render-scoped use case. Nil-safe:
// a miss (no binding) or a nil response pass through unchanged.
func minimizeReportCardTemplateResponse(
	response *jobdoctmplpb.FindApplicableJobOutcomeSummaryDocumentTemplateResponse,
) *jobdoctmplpb.FindApplicableJobOutcomeSummaryDocumentTemplateResponse {
	if response == nil || response.GetBinding() == nil {
		return response
	}
	binding := response.GetBinding()
	binding.CreatedBy = nil
	binding.PublishedBy = nil
	return response
}

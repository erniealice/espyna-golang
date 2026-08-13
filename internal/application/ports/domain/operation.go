package domain

import (
	"context"

	documenttemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/document/template"
	jobcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_category"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	exportpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/subscription_group_outcome_export"
)

// JobListTabSupportRequest carries the server-derived per-kind include flags for
// the job-list tabstrip support read (the 20260718 courses-list-perf Rank-1
// counter-plan). Workspace is NEVER a field here — the adapter takes it from the
// SESSION identity (identity.FromContext), so tenancy cannot be spoofed. The two
// flags are bound by the use case from two INDEPENDENT ActionGatekeeper checks
// (job_category:list and job_template:list); a denied kind arrives false and its
// UNION-ALL branch is not executed, so its rows are simply omitted (matching the
// pre-existing component-level degradation — no silent partial, no fail-open).
type JobListTabSupportRequest struct {
	// IncludeCategories gates the job_category branch (job_category:list).
	IncludeCategories bool
	// IncludeTemplates gates the active job_template stub branch (job_template:list).
	IncludeTemplates bool
}

// JobListTabSupportResponse wraps BOTH row kinds returned by the single
// UNION-ALL statement. Each slice carries only the narrow columns the job-list
// tabstrip needs (proto domain messages reused as lightweight DTO carriers, NOT
// a new proto service contract — the proto-less internal-service shape, mirroring
// ListPendingActivitiesForAssigneeResponse):
//   - Categories: id, name, active, sort_order (ACTIVE and INACTIVE) — the tab rows.
//   - Templates:  id, name, job_category_id (ACTIVE only) — the template→category
//     map + the deportment template-grain fallback rows.
//
// A denied (or non-postgres) kind is an empty slice, never a nil-vs-empty signal.
type JobListTabSupportResponse struct {
	Categories []*jobcategorypb.JobCategory
	Templates  []*jobtemplatepb.JobTemplate
}

// JobListTabSupportQueryService is a read-only query port that collapses the
// job-list tabstrip's category + active-template reads into ONE SQL statement
// (replacing the former 12 generic-List statements: 2 category calls + ~4 paged
// template calls, each COUNT+SELECT). It is a SEPARATE hand-written port (no
// generated proto ServiceServer) so no new esqyma service contract is required.
//
// Fail-closed: an empty session WorkspaceID, or a request with BOTH include
// flags false, returns an empty response with no SQL executed.
type JobListTabSupportQueryService interface {
	ListJobListTabSupport(ctx context.Context, req *JobListTabSupportRequest) (*JobListTabSupportResponse, error)
}

// SubscriptionGroupOutcomeExportScope is derived by the report-authorized
// application use case. It is deliberately absent from the generated request:
// a transport caller cannot self-select a wider principal scope.
type SubscriptionGroupOutcomeExportScope struct {
	WorkspaceWide bool
}

// SubscriptionGroupOutcomeExportQueryService is the internal typed port for
// the one-statement group outcome matrix and the minimal render locator. The
// adapter also implements the generated RPC server method, but application code
// uses these scoped methods so the workspace-wide decision is server-derived.
type SubscriptionGroupOutcomeExportQueryService interface {
	GetSubscriptionGroupOutcomeExportScoped(
		ctx context.Context,
		req *exportpb.GetSubscriptionGroupOutcomeExportRequest,
		scope SubscriptionGroupOutcomeExportScope,
	) (*exportpb.GetSubscriptionGroupOutcomeExportResponse, error)
	ResolveSubscriptionGroupOutcomeDocumentForRenderScoped(
		ctx context.Context,
		req *exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderRequest,
		scope SubscriptionGroupOutcomeExportScope,
	) (*exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderResponse, error)
}

// Backward-compatible names for the canonical provider-neutral Esqyma landing
// messages. These are aliases, not application-owned DTOs; new signatures use
// the generated names directly.
type SubscriptionGroupOutcomeLandingRequest = exportpb.ListSubscriptionGroupOutcomeLandingRequest
type SubscriptionGroupOutcomeLandingRow = exportpb.SubscriptionGroupOutcomeLandingRow
type SubscriptionGroupOutcomeLandingResponse = exportpb.ListSubscriptionGroupOutcomeLandingResponse

// SubscriptionGroupOutcomeLandingQueryService is the scoped, one-statement
// query port for the report landing aggregate.
type SubscriptionGroupOutcomeLandingQueryService interface {
	ListSubscriptionGroupOutcomeLandingScoped(
		ctx context.Context,
		req *exportpb.ListSubscriptionGroupOutcomeLandingRequest,
		scope SubscriptionGroupOutcomeExportScope,
	) (*exportpb.ListSubscriptionGroupOutcomeLandingResponse, error)
}

// SubscriptionGroupDocumentTemplateDraftPairDeleter atomically soft-deletes
// one active DRAFT binding and its unshared document-template artifact. The
// returned artifact carries the exact committed storage locator so the caller
// can delete the object only after the database transaction commits.
type SubscriptionGroupDocumentTemplateDraftPairDeleter interface {
	DeleteDraftPair(ctx context.Context, bindingID string) (*documenttemplatepb.DocumentTemplate, error)
}

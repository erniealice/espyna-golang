package domain

import (
	"context"

	jobcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_category"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
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

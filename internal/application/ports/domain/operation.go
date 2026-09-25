package domain

import (
	"context"
	"errors"

	documenttemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/document/template"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	jobcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_category"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	ratingdescriptionsetpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set"
	ratingdescriptionsetproductplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_product_plan"
	exportpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/subscription_group_outcome_export"
)

// ErrClientReportNotFound intentionally does not distinguish a missing group,
// foreign client, or absent membership at the report projection boundary.
var ErrClientReportNotFound = errors.New("client report not found")

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

// SubscriptionGroupClientReportCardQueryService is an in-process, one-client
// projection port. It is deliberately separate from the generated gRPC server
// contract so the sensitive typed render source cannot become a transport
// endpoint by adding this query.
type SubscriptionGroupClientReportCardQueryService interface {
	GetSubscriptionGroupClientReportCardScoped(
		ctx context.Context,
		req *exportpb.GetSubscriptionGroupClientReportCardRequest,
		scope SubscriptionGroupOutcomeExportScope,
	) (*exportpb.GetSubscriptionGroupClientReportCardResponse, error)
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

// RatingDescriptionSetLifecycleRepository is the conditional-write port for
// the rating_description_set status transitions and lock-then-write
// operations that are NOT part of the generated
// RatingDescriptionSetDomainServiceServer interface — Publish, Deprecate,
// LockForUpdate and the DRAFT-only Update are plain use-case-owned writes,
// not proto RPCs (interfaces.md §2; docs/plan/20260925-criterion-descriptors-
// by-program-year, schema-proposal.md §9.2). All four methods require an
// ambient transaction on ctx, failing closed otherwise. The lifecycle use
// cases wrap the call(s) in services.Transactor.ExecuteInTransaction and
// type-assert the injected repository against this interface, mirroring
// SubscriptionGroupDocumentTemplateDraftPairDeleter's constructor pattern.
//
// LockRatingDescriptionSetForUpdate + UpdateRatingDescriptionSetIfDraft close
// two W3 follow-up findings (2026-09-25, codex-review-impl1.out.md): Publish
// must count entries AFTER taking the parent row lock (call LockForUpdate
// first, then count, inside the same tx), and a generic Update must never
// write version_status/version/supersedes_id back from a pre-read snapshot —
// UpdateRatingDescriptionSetIfDraft strips those fields unconditionally and
// writes under the SAME lock LockForUpdate already took.
type RatingDescriptionSetLifecycleRepository interface {
	PublishRatingDescriptionSetIfDraft(ctx context.Context, id string) (*ratingdescriptionsetpb.RatingDescriptionSet, error)
	DeprecateRatingDescriptionSetIfPublished(ctx context.Context, id string) (*ratingdescriptionsetpb.RatingDescriptionSet, error)
	LockRatingDescriptionSetForUpdate(ctx context.Context, id string) (enumspb.VersionStatus, error)
	UpdateRatingDescriptionSetIfDraft(ctx context.Context, req *ratingdescriptionsetpb.UpdateRatingDescriptionSetRequest) (*ratingdescriptionsetpb.RatingDescriptionSet, error)
}

// RatingDescriptionSetProductPlanRelinker is the conditional-write port for
// the offering<->AY link transaction (RelinkRatingDescriptionSetProductPlan /
// UnlinkRatingDescriptionSetProductPlan — plain use-case messages, not proto
// RPCs, interfaces.md §2). RelinkLocked performs the full transaction body:
// lock the offering pair, compare expectedCurrentLinkID (nil = expect no
// current link), require the target set PUBLISHED + same workspace,
// deactivate the old active link if any, insert newLink. UnlinkLocked
// deactivates one link with the same lock discipline. Both require an
// ambient transaction on ctx and fail closed otherwise. Both also write a
// durable semantic audit event inside that same transaction before returning
// (schema-proposal.md §9.4, codex-review-impl2.out.md finding 9) — reason is
// the caller-supplied request's reason field, persisted on that event.
type RatingDescriptionSetProductPlanRelinker interface {
	RelinkLocked(ctx context.Context, newLink *ratingdescriptionsetproductplanpb.RatingDescriptionSetProductPlan, expectedCurrentLinkID *string, reason string) (*ratingdescriptionsetproductplanpb.RatingDescriptionSetProductPlan, error)
	UnlinkLocked(ctx context.Context, linkID string, reason string) (*ratingdescriptionsetproductplanpb.RatingDescriptionSetProductPlan, error)
}

// WorkspaceScopedProductPlanReader is a read-only query port for listing
// product_plan rows scoped through BOTH parent product.workspace_id AND
// plan.workspace_id (docs/plan/20260925-criterion-descriptors-by-program-year,
// codex-review-impl3.out.md finding #3). product_plan itself carries NO
// workspace_id column — the generic ListProductPlans/dbOps.List path has no
// predicate for this column-less tenant table (contrib/postgres/internal/
// adapter/core/workspace_operations.go's columnLessTenantTables comment: "a
// column-less TENANT list returns rows across ALL workspaces"). Used by the
// rating_description_set_product_plan offering picker (Relink drawer) and
// the assignment-list page data, replacing a direct, unscoped call to the
// generic ProductPlanDomainServiceServer.ListProductPlans.
//
// Workspace is NEVER a caller-supplied field — mirroring
// JobListTabSupportQueryService above, the adapter derives it from the
// session identity (identity.FromContext), so tenancy cannot be spoofed.
type WorkspaceScopedProductPlanReader interface {
	ListWorkspaceScopedProductPlans(ctx context.Context) ([]*productplanpb.ProductPlan, error)
}

// RatingDescriptionSetEntryCounter is a read-only aggregation port for the
// rating_description_set LIST page's Entries column
// (docs/plan/20260925-criterion-descriptors-by-program-year,
// codex-review-impl4.out.md round-3/round-4 disposition #2: the previous
// implementation paged through every ACTIVE entry in the workspace,
// silently capped at 5,000 rows (listSummaryMaxPages * listSummaryPageLimit),
// undercounting larger workspaces). Returns active-entry counts GROUP BY
// rating_description_set_id, scoped to EXACTLY the caller-supplied set ids
// (already workspace-filtered by the caller's own RatingDescriptionSet
// read) — never a workspace-wide scan. A nil/zero-length id slice returns an
// empty map with no SQL executed.
type RatingDescriptionSetEntryCounter interface {
	CountActiveRatingDescriptionSetEntriesBySet(ctx context.Context, ratingDescriptionSetIds []string) (map[string]int32, error)
}

// RatingDescriptionSetProductPlanCounter is the identical scoped-aggregation
// port for the LIST page's Links column (same finding as
// RatingDescriptionSetEntryCounter above).
type RatingDescriptionSetProductPlanCounter interface {
	CountActiveRatingDescriptionSetProductPlansBySet(ctx context.Context, ratingDescriptionSetIds []string) (map[string]int32, error)
}

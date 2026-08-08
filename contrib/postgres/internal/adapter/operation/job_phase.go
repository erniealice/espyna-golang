//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/lib/pq"
	"google.golang.org/protobuf/encoding/protojson"

	auditadapter "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/audit"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	infraports "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
	"github.com/erniealice/espyna-golang/internal/application/shared/approvalctx"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
)

// jobPhaseApprovalMutableKeys are the protojson (camelCase) map keys for the
// server-owned approval lifecycle. Generic create/delete-from + generic update
// strip these so ONLY the dedicated transition RPCs can stamp them (codex CRITICAL
// finding: generic create/update must not accept forged approval/audit fields).
// The *_string mirrors are (db).ignore and never persist, but are stripped too
// for completeness.
var jobPhaseApprovalMutableKeys = []string{
	"approvalStatus",
	"submittedBy", "submittedAt", "submittedAtString",
	"verifiedBy", "verifiedAt", "verifiedAtString",
	"publishedBy", "publishedAt", "publishedAtString",
	"returnReason",
	"returnedBy", "returnedAt", "returnedAtString",
}

// jobPhaseMembershipKeys are the protojson (camelCase) map keys for the sheet
// membership anchors + the active flag. A generic UPDATE strips them so an
// ordinary edit can never REPARENT a phase (change job_id / template_phase_id) or
// toggle active — either would let a concurrent write insert/move a member after
// the transition captured its locked set S, defeating the sorted-RETURNING
// comparison (codex §4 CRITICAL: membership must not be generically mutable
// outside the parent mutex). Reparent/active changes belong only to the dedicated
// parent-locked seams.
var jobPhaseMembershipKeys = []string{"jobId", "templatePhaseId", "active"}

// stripJobPhaseApprovalKeys deletes every approval/audit lifecycle key from a
// generic write payload. On CREATE this drops approval_status from the INSERT so
// the DB default ('PHASE_APPROVAL_STATUS_IN_PROGRESS') applies and the audit
// columns stay NULL; on UPDATE it leaves those columns untouched.
func stripJobPhaseApprovalKeys(data map[string]any) {
	for _, k := range jobPhaseApprovalMutableKeys {
		delete(data, k)
	}
}

// stripJobPhaseMembershipKeys deletes the membership anchors + active flag from a
// generic UPDATE payload so they stay immutable (see jobPhaseMembershipKeys).
func stripJobPhaseMembershipKeys(data map[string]any) {
	for _, k := range jobPhaseMembershipKeys {
		delete(data, k)
	}
}

// jobWorkspaceScope returns the SHADOW-INDEPENDENT tenant-ancestry JOIN fragment and
// its bind arg for the explicit job_phase projections (FIX-4 / codex §4 HIGH).
// job_phase has no workspace_id column, so its owning workspace is derived via
// job.workspace_id. The three explicit projections (list/item/ListByJob) previously
// read the raw DB with NO tenant join, and the workspace decorator only scopes
// job_phase reads through the parent-JOIN probe gated on the global AUTHZ_ENFORCE
// flag (SHADOW by default = log-but-pass). This binds the trusted parent ancestry
// ALWAYS when a workspace is present — regardless of that flag — mirroring the coded
// task_outcome reads (ListCodedTaskOutcomeValuesByJob binds j.workspace_id in the SQL
// predicate unconditionally). placeholderNum is the $N index the workspace bind
// occupies. When no trusted workspace is in context (service-to-service /
// unauthenticated — e.g. a CLI or the login path) the scope is empty (pass-through),
// exactly like the decorator's own wsID=="" short-circuit, so system callers are
// unaffected. Only THIS entity's reads change; other entities' shadow behavior is
// untouched.
func jobWorkspaceScope(ctx context.Context, placeholderNum int) (joinSQL string, arg string, scoped bool) {
	id, ok := identity.FromContext(ctx)
	if !ok || id == nil || id.WorkspaceID == "" {
		return "", "", false
	}
	joinSQL = "\n\t\t\tJOIN " + entityid.Job + " j ON j.id = jp.job_id AND j.workspace_id = " + fmt.Sprintf("$%d", placeholderNum)
	return joinSQL, id.WorkspaceID, true
}

// requireTrustedJobWorkspace proves that the given job belongs to the caller's
// TRUSTED context workspace, SHADOW-INDEPENDENTLY (codex P3 §A4: generic Create
// must prove the supplied Job belongs to the trusted workspace, not merely inject
// it). Returns nil when there is no trusted workspace in ctx (service-to-service /
// CLI pass-through, mirroring jobWorkspaceScope). Fails closed when a workspace IS
// present and the job is missing / owned by another tenant.
func (r *PostgresJobPhaseRepository) requireTrustedJobWorkspace(ctx context.Context, jobID string) error {
	id, ok := identity.FromContext(ctx)
	if !ok || id == nil || id.WorkspaceID == "" {
		return nil // no trusted workspace → pass-through (system/CLI)
	}
	if jobID == "" {
		return fmt.Errorf("job_phase create: no owning job_id to prove workspace (fail closed)")
	}
	var exists bool
	const q = `SELECT EXISTS(SELECT 1 FROM ` + entityid.Job + ` WHERE id = $1 AND workspace_id = $2 AND active = true)`
	if err := r.readExecutor(ctx).QueryRowContext(ctx, q, jobID, id.WorkspaceID).Scan(&exists); err != nil {
		return fmt.Errorf("job_phase: owning-job workspace probe: %w", err)
	}
	if !exists {
		return fmt.Errorf("job_phase: owning job not in trusted workspace — denied")
	}
	return nil
}

// resolvePhaseInTrustedWorkspace resolves a phase's owning job_id + (coalesced)
// template_phase_id, SHADOW-INDEPENDENTLY binding the phase's owning job to the
// trusted context workspace (codex P3 §A4: generic Read/Update/Delete must enforce
// Job ancestry independent of the SHADOW-sensitive generic decorator). found=false
// means the phase is missing or owned by another tenant → the caller MUST fail
// closed. When no trusted workspace is present the resolve is unscoped
// (pass-through for system/CLI callers), found reflects mere existence.
func (r *PostgresJobPhaseRepository) resolvePhaseInTrustedWorkspace(ctx context.Context, phaseID string) (jobID, templatePhaseID string, found bool, err error) {
	id, ok := identity.FromContext(ctx)
	wsPresent := ok && id != nil && id.WorkspaceID != ""
	q := `SELECT jp.job_id, COALESCE(jp.template_phase_id, '')
		FROM ` + entityid.JobPhase + ` jp
		JOIN ` + entityid.Job + ` j ON j.id = jp.job_id
		WHERE jp.id = $1`
	args := []any{phaseID}
	if wsPresent {
		q += ` AND j.workspace_id = $2`
		args = append(args, id.WorkspaceID)
	}
	err = r.readExecutor(ctx).QueryRowContext(ctx, q, args...).Scan(&jobID, &templatePhaseID)
	if err == sql.ErrNoRows {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, fmt.Errorf("job_phase: phase ancestry probe: %w", err)
	}
	return jobID, templatePhaseID, true, nil
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.JobPhase, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres job_phase repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresJobPhaseRepository(dbOps, tableName), nil
	})
}

// PostgresJobPhaseRepository implements job_phase CRUD operations using PostgreSQL
type PostgresJobPhaseRepository struct {
	pb.UnimplementedJobPhaseDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
	// audit writes the append-only sheet transition event inside each approval
	// transition's business transaction (ambient-tx aware). Nil-safe: nil skips
	// the audit write (mock/test paths).
	audit infraports.AuditService
	// recompute is the injected submit-time freshness barrier (FIX-3): it finalizes
	// the phase + job outcome summaries for the locked sheet on the AMBIENT
	// transition transaction, post-lock/pre-flip. It is wired in operation/usecases.go
	// (grade_compute.UseCases.SheetRecompute) via SetSheetRecompute — the adapter
	// cannot import the use-case layer, so the closure is injected as a bare func.
	// SubmitJobPhaseApproval fails CLOSED when it is nil (refusing to advance a
	// possibly-stale sheet). phaseIDs are the locked job_phase ids; jobIDs their
	// distinct owning job ids. A non-nil return rolls the transition back.
	recompute func(ctx context.Context, phaseIDs, jobIDs []string) error
}

// SetSheetRecompute injects the submit-time summary-recompute freshness barrier
// (FIX-3). The application initializer calls this once with
// grade_compute.UseCases.SheetRecompute after both the job_phase adapter and the
// grade-compute use cases exist (the adapter is built by a registry factory that has
// no access to the use-case layer, so the port is injected post-construction — the
// same shape as the audit dependency). The bare func type matches exactly across
// packages, so no shared named type is required.
func (r *PostgresJobPhaseRepository) SetSheetRecompute(fn func(ctx context.Context, phaseIDs, jobIDs []string) error) {
	r.recompute = fn
}

// NewPostgresJobPhaseRepository creates a new PostgreSQL job_phase repository
func NewPostgresJobPhaseRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.JobPhaseDomainServiceServer {
	if tableName == "" {
		tableName = "job_phase"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	var auditSvc infraports.AuditService
	if db != nil {
		// The approval transition audit is a security-critical durable record, so
		// it is always written (the audit adapter is ambient-tx aware and joins the
		// active transition transaction via ctx).
		auditSvc = auditadapter.New(db)
	}

	return &PostgresJobPhaseRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
		audit:     auditSvc,
	}
}

// CreateJobPhase creates a new job phase record
func (r *PostgresJobPhaseRepository) CreateJobPhase(ctx context.Context, req *pb.CreateJobPhaseRequest) (*pb.CreateJobPhaseResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("job phase data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	// Membership-phantom guard (codex §4 CRITICAL): a template-backed phase
	// (template_phase_id set) may be created ONLY by the trusted W-SPAWN seam,
	// which pre-locks the parent job_template_phase FOR UPDATE so the insert
	// serializes against in-flight transitions. A generic create that sets
	// template_phase_id is rejected — otherwise a concurrent insert could add a
	// sheet member after a transition captured its locked set S. The internal
	// spawn marker cannot be forged by any downstream module.
	if tpid, ok := data["templatePhaseId"].(string); ok && tpid != "" && !approvalctx.IsTrustedSpawn(ctx) {
		return nil, fmt.Errorf("job_phase create: template-backed membership (template_phase_id) may only be created through the W-SPAWN seam — generic create rejected")
	}

	// Tenant proof (codex P3 §A4): prove the supplied job_id belongs to the trusted
	// workspace — a generic create must not attach a phase to another tenant's job.
	// The trusted spawn seam supplies internally-resolved same-workspace jobs, so it
	// passes; a forged cross-tenant job_id is rejected. No-workspace (system/CLI)
	// contexts pass through.
	if jid, _ := data["jobId"].(string); true {
		if err := r.requireTrustedJobWorkspace(ctx, jid); err != nil {
			return nil, err
		}
	}

	// Server-owned lifecycle (defense-in-depth vs the use-case strip): a generic
	// create can never seed approval_status or the audit stamps. Dropping
	// approval_status lets the DB default (IN_PROGRESS) apply; the audit columns
	// stay NULL. Only the transition RPCs stamp them.
	stripJobPhaseApprovalKeys(data)

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create job phase: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	phase := &pb.JobPhase{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, phase); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.CreateJobPhaseResponse{
		Success: true,
		Data:    []*pb.JobPhase{phase},
	}, nil
}

// ReadJobPhase retrieves a job phase record by ID
func (r *PostgresJobPhaseRepository) ReadJobPhase(ctx context.Context, req *pb.ReadJobPhaseRequest) (*pb.ReadJobPhaseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job phase ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read job phase: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	phase := &pb.JobPhase{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, phase); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	// Tenant ancestry (codex P3 §A4): the generic dbOps.Read by id has no WHERE
	// seam, so enforce the SHADOW-INDEPENDENT parent scope post-read — a phase whose
	// owning job is not in the trusted workspace is not-found. No-workspace
	// (system/CLI) contexts pass through.
	if err := r.requireTrustedJobWorkspace(ctx, phase.JobId); err != nil {
		return nil, fmt.Errorf("job phase with ID '%s' not found", req.Data.Id)
	}

	return &pb.ReadJobPhaseResponse{
		Success: true,
		Data:    []*pb.JobPhase{phase},
	}, nil
}

// UpdateJobPhase updates a job phase record
func (r *PostgresJobPhaseRepository) UpdateJobPhase(ctx context.Context, req *pb.UpdateJobPhaseRequest) (*pb.UpdateJobPhaseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job phase ID is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	// Tenant ancestry (codex P3 §A4): prove the target phase's owning job is in the
	// trusted workspace, SHADOW-INDEPENDENTLY, BEFORE the generic decorator update —
	// a cross-tenant phase id fails closed to not-found. No-workspace contexts pass
	// through.
	if _, _, found, aerr := r.resolvePhaseInTrustedWorkspace(ctx, req.Data.Id); aerr != nil {
		return nil, aerr
	} else if !found {
		return nil, fmt.Errorf("job phase with ID '%s' not found", req.Data.Id)
	}

	// Server-owned lifecycle (defense-in-depth vs the use-case strip): a generic
	// update can never mutate approval_status or any audit stamp, NOR reparent the
	// phase (job_id / template_phase_id) or toggle active. Stripping the keys
	// leaves those columns untouched; only the transition RPCs move the ladder and
	// only the parent-locked seams change membership.
	stripJobPhaseApprovalKeys(data)
	stripJobPhaseMembershipKeys(data)

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update job phase: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	phase := &pb.JobPhase{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, phase); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.UpdateJobPhaseResponse{
		Success: true,
		Data:    []*pb.JobPhase{phase},
	}, nil
}

// DeleteJobPhase deletes a job phase record (soft delete)
func (r *PostgresJobPhaseRepository) DeleteJobPhase(ctx context.Context, req *pb.DeleteJobPhaseRequest) (*pb.DeleteJobPhaseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job phase ID is required")
	}

	// Tenant ancestry + membership guard (codex P3 §A4): resolve the phase's owning
	// job under the trusted workspace (SHADOW-independent); a cross-tenant id fails
	// closed. A TEMPLATE-BACKED phase is a sheet member — deactivating it defeats a
	// transition's locked-set-S comparison, so it may be removed ONLY through a
	// parent-locked seam, never generic delete. No-workspace (system/CLI) contexts
	// still resolve (unscoped) so the template-backed guard applies uniformly.
	_, templatePhaseID, found, aerr := r.resolvePhaseInTrustedWorkspace(ctx, req.Data.Id)
	if aerr != nil {
		return nil, aerr
	}
	if !found {
		return nil, fmt.Errorf("job phase with ID '%s' not found", req.Data.Id)
	}
	if templatePhaseID != "" {
		return nil, fmt.Errorf("job_phase delete: template-backed sheet member (template_phase_id set) cannot be removed via generic delete — use the parent-locked seam")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete job phase: %w", err)
	}

	return &pb.DeleteJobPhaseResponse{
		Success: true,
	}, nil
}

// ListJobPhases lists job phase records with optional filters and an optional
// delivery-group narrow (req.subscription_group_id).
//
// The narrow is OPT-IN and OFF by default: an absent/empty group id runs the
// pre-20260731 path unchanged. When it IS set, the listing is restricted to the
// phases whose owning job belongs to that group via the existing
// groupNarrowPredicate, and any state in which the narrow could not be APPLIED
// returns an error rather than a quietly unnarrowed success (see
// narrowPhasesToGroup).
//
// ⚠ CALLER CONTRACT FOR PAGED READS. The narrow is applied to the page, AFTER
// LIMIT/OFFSET — it filters what the page returned, it does not change which
// rows the page selected. So under a narrow a SHORT PAGE IS NOT EXHAUSTION: a
// full 100-row page may yield 3 in-group rows, and a paging loop that breaks on
// `len(batch) < limit` would stop mid-sheet and silently under-read. A paged
// caller MUST drive its loop from the UNNARROWED page size (page until an
// unnarrowed page comes back short, or page a bounded id set to exhaustion),
// never from the narrowed row count. ListJobPhasesResponse carries no pagination
// block, so the adapter cannot signal this — it is the caller's obligation.
func (r *PostgresJobPhaseRepository) ListJobPhases(ctx context.Context, req *pb.ListJobPhasesRequest) (*pb.ListJobPhasesResponse, error) {
	var params *interfaces.ListParams
	// Forward Filters, Pagination AND Sort. The M8 row-cap fix forwarded
	// Pagination in ListJobTasks/ListTaskOutcomes but MISSED this method (it only
	// forwarded Filters): a paged caller then silently re-received the same
	// default-capped first 100 rows every page. Bulk callers (the report-card
	// enrollment-evidence walk) page an entity set far larger than the 100-row
	// cap; they also pass a unique id-sort so OFFSET paging over a tied
	// date_created is deterministic and never drops/duplicates a row.
	if req != nil && (req.Filters != nil || req.Pagination != nil || req.Sort != nil) {
		params = &interfaces.ListParams{Filters: req.Filters, Pagination: req.Pagination, Sort: req.Sort}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list job phases: %w", err)
	}

	var phases []*pb.JobPhase
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal job_phase row: %v", err)
			continue
		}

		phase := &pb.JobPhase{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, phase); err != nil {
			log.Printf("WARN: protojson unmarshal job_phase: %v", err)
			continue
		}
		phases = append(phases, phase)
	}

	// FIX-4: the generic List routes through the workspace decorator, but job_phase
	// is column-less so the decorator only SHADOW-logs the unscoped tenant list (it
	// cannot express a parent-JOIN predicate in a StringFilter). Enforce trusted
	// parent ancestry here instead — SHADOW-independent — by dropping any phase whose
	// owning job is not in the caller's workspace (one batched job probe). A caller
	// with no trusted workspace (service-to-service / unauthenticated) is passed
	// through unchanged. Same-tenant data is never dropped, so pagination is
	// unaffected for legitimate callers; only cross-tenant leakage is removed.
	scoped, err := r.filterPhasesByWorkspace(ctx, phases)
	if err != nil {
		return nil, err
	}

	// Delivery-group narrow (20260731, strictly additive). ABSENT or EMPTY
	// subscription_group_id takes NO branch: the ListParams built above, the
	// dbOps.List call, the workspace ancestry filter and the response are the
	// byte-identical pre-change path — not one extra query, not one changed arg.
	// Only a caller that explicitly sets the field pays for, or is affected by,
	// the narrow. Nothing in the tree sets it yet.
	if groupID := req.GetSubscriptionGroupId(); groupID != "" {
		scoped, err = r.narrowPhasesToGroup(ctx, scoped, groupID)
		if err != nil {
			return nil, err
		}
	}

	return &pb.ListJobPhasesResponse{
		Success: true,
		Data:    scoped,
	}, nil
}

// jobGroupNarrowProbeSQL builds the job-grain probe that applies the EXISTING
// delivery-group narrow (groupNarrowPredicate, job_phase_approval.go:442 — shared
// verbatim with the four approval transitions and the outcome-matrix roll-up, so
// a read can never disagree with a write about which jobs belong to a group).
//
// The probe runs at JOB grain, not phase grain, because group membership is a
// property of the job's (client_id, origin_id) pair — exactly what the predicate
// keys on. That also dedupes: a job with twelve phases costs one probed id.
//
// Placeholders: $1 = job id array, $2 = trusted workspace (bound BOTH to
// j.workspace_id and, through the predicate's wsArgN, to sgm_g.workspace_id),
// $3 = the group id appended by the predicate. Returns ok=false when the
// predicate declined to emit — the caller MUST fail closed rather than run the
// residual unnarrowed query, which is the R-1 fail-open this seam exists to make
// unrepresentable.
//
// Deliberately absent: any jp.active / status term. The narrow narrows by GROUP
// and by nothing else, so it can never quietly change a caller's activity or
// lifecycle semantics.
func jobGroupNarrowProbeSQL(groupID string) (query string, narrowArgs []any, ok bool) {
	narrow, narrowArgs := groupNarrowPredicate(groupID, 3, 2)
	if narrow == "" {
		return "", nil, false
	}
	return `SELECT j.id FROM ` + entityid.Job + ` j
			WHERE j.id = ANY($1)
			  AND j.workspace_id = $2` + narrow, narrowArgs, true
}

// narrowPhasesToGroup restricts an already-listed, already-workspace-scoped phase
// set to ONE delivery group. It mirrors filterPhasesByWorkspace's shape (one
// batched job probe on the ambient executor) with ONE deliberate divergence:
//
//	filterPhasesByWorkspace PASSES THROUGH when the context carries no trusted
//	workspace. A narrow must NOT. The predicate binds sgm_g.workspace_id, so
//	without a trusted workspace there is nothing to bind it to, and passing
//	through would return the UNNARROWED set under a success response — the
//	caller would read "the group has these phases" from a set that was never
//	narrowed. That is the R-1 fail-open (docs/plan/20260729-report-card-render-
//	gate-group-grain/progress.md). Every path here that cannot APPLY the narrow
//	returns an error instead, so on this adapter "narrow not applied" is not a
//	representable success state — see the report's requirement-4 answer.
//
// A phase whose owning job cannot be resolved or proven in-group is dropped
// (fail closed), never kept.
func (r *PostgresJobPhaseRepository) narrowPhasesToGroup(ctx context.Context, phases []*pb.JobPhase, groupID string) ([]*pb.JobPhase, error) {
	if groupID == "" {
		// Unreachable from ListJobPhases (which tests the field first), and kept
		// unreachable on purpose: an empty group must never silently mean "no
		// narrow" on a seam whose caller asked for one.
		return nil, fmt.Errorf("job_phase list: group narrow requested with an empty subscription_group_id (fail closed)")
	}
	id, ok := identity.FromContext(ctx)
	if !ok || id == nil || id.WorkspaceID == "" {
		return nil, fmt.Errorf("job_phase list: group narrow requires a trusted workspace in context — refusing to return an unnarrowed set (fail closed)")
	}
	if len(phases) == 0 {
		// The narrow of an empty set is empty. No probe, and no fail-open risk:
		// there is nothing that could have been wrongly kept.
		return []*pb.JobPhase{}, nil
	}

	seen := make(map[string]struct{}, len(phases))
	jobIDs := make([]string, 0, len(phases))
	for _, p := range phases {
		if p == nil || p.JobId == "" {
			continue
		}
		if _, dup := seen[p.JobId]; dup {
			continue
		}
		seen[p.JobId] = struct{}{}
		jobIDs = append(jobIDs, p.JobId)
	}
	if len(jobIDs) == 0 {
		// No resolvable owning job on any row → group membership is unprovable →
		// fail closed (same disposition as the workspace ancestry probe).
		return []*pb.JobPhase{}, nil
	}

	query, narrowArgs, ok := jobGroupNarrowProbeSQL(groupID)
	if !ok {
		return nil, fmt.Errorf("job_phase list: group narrow predicate declined to emit — refusing to run the unnarrowed residual (fail closed)")
	}
	args := append([]any{pq.Array(jobIDs), id.WorkspaceID}, narrowArgs...)

	inGroup := make(map[string]bool, len(jobIDs))
	rows, err := r.readExecutor(ctx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("job_phase list: group narrow probe: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var jid string
		if err := rows.Scan(&jid); err != nil {
			return nil, fmt.Errorf("job_phase list: scan group narrow: %w", err)
		}
		inGroup[jid] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("job_phase list: iterate group narrow: %w", err)
	}

	out := make([]*pb.JobPhase, 0, len(phases))
	for _, p := range phases {
		if p != nil && inGroup[p.JobId] {
			out = append(out, p)
		}
	}
	return out, nil
}

// filterPhasesByWorkspace drops phases whose owning job is not in the trusted
// context workspace (FIX-4). It is SHADOW-independent (always filters when a
// workspace is present) and passes through unchanged when the context carries no
// workspace. It runs one batched job-ancestry probe on the ambient executor (the
// active *sql.Tx when inside a transaction, else the pool), so a phase whose job the
// caller cannot prove it owns is excluded fail-closed.
func (r *PostgresJobPhaseRepository) filterPhasesByWorkspace(ctx context.Context, phases []*pb.JobPhase) ([]*pb.JobPhase, error) {
	id, ok := identity.FromContext(ctx)
	if !ok || id == nil || id.WorkspaceID == "" {
		return phases, nil // no trusted workspace → pass-through (service-to-service / CLI)
	}
	if len(phases) == 0 {
		return phases, nil
	}
	seen := make(map[string]struct{}, len(phases))
	jobIDs := make([]string, 0, len(phases))
	for _, p := range phases {
		if p == nil || p.JobId == "" {
			continue
		}
		if _, dup := seen[p.JobId]; dup {
			continue
		}
		seen[p.JobId] = struct{}{}
		jobIDs = append(jobIDs, p.JobId)
	}
	if len(jobIDs) == 0 {
		// No resolvable owning job on any row → cannot prove tenancy → fail closed.
		return []*pb.JobPhase{}, nil
	}
	allowed := make(map[string]bool, len(jobIDs))
	const q = `SELECT id FROM ` + entityid.Job + ` WHERE id = ANY($1) AND workspace_id = $2`
	rows, err := r.readExecutor(ctx).QueryContext(ctx, q, pq.Array(jobIDs), id.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("job_phase list: workspace ancestry probe: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var jid string
		if err := rows.Scan(&jid); err != nil {
			return nil, fmt.Errorf("job_phase list: scan workspace ancestry: %w", err)
		}
		allowed[jid] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("job_phase list: iterate workspace ancestry: %w", err)
	}
	out := make([]*pb.JobPhase, 0, len(phases))
	for _, p := range phases {
		if p != nil && allowed[p.JobId] {
			out = append(out, p)
		}
	}
	return out, nil
}

// jobPhaseSortableSQLCols is the fail-closed sort whitelist for
// GetJobPhaseListPageData. Only columns projected by the CTE SELECT are included
// so ORDER BY can never reference an unprojected/injected identifier.
var jobPhaseSortableSQLCols = []string{
	"id", "date_created", "date_modified", "active", "job_id",
	"name", "phase_order", "status", "approval_status",
}

// jobPhaseApprovalCols is the ordered SELECT fragment (leading comma) for the
// P1 approval read surface, appended after jp.status in every explicit
// projection (list/item/ListByJob). Kept in one place so the column order stays
// in lockstep with applyJobPhaseApprovalScan's Scan target order.
const jobPhaseApprovalCols = `,
				jp.approval_status,
				jp.submitted_by,
				jp.submitted_at,
				jp.verified_by,
				jp.verified_at,
				jp.published_by,
				jp.published_at,
				jp.return_reason,
				jp.returned_by,
				jp.returned_at`

// jobPhaseApprovalScan holds the raw scan targets for the approval read surface.
// approval_status is NOT NULL (enum name); the audit pairs + return_reason are
// nullable. Declared as a struct so each projection can scan the same address
// set in the same order as jobPhaseApprovalCols.
type jobPhaseApprovalScan struct {
	approvalStatus string
	submittedBy    sql.NullString
	submittedAt    sql.NullInt64
	verifiedBy     sql.NullString
	verifiedAt     sql.NullInt64
	publishedBy    sql.NullString
	publishedAt    sql.NullInt64
	returnReason   sql.NullString
	returnedBy     sql.NullString
	returnedAt     sql.NullInt64
}

// scanDest returns the Scan target pointers in jobPhaseApprovalCols order.
func (a *jobPhaseApprovalScan) scanDest() []any {
	return []any{
		&a.approvalStatus,
		&a.submittedBy, &a.submittedAt,
		&a.verifiedBy, &a.verifiedAt,
		&a.publishedBy, &a.publishedAt,
		&a.returnReason,
		&a.returnedBy, &a.returnedAt,
	}
}

// apply maps the scanned approval columns onto the proto. UNSPECIFIED/unknown
// tokens leave ApprovalStatus at its zero value (fail-soft on read); the DB
// CHECK guarantees only the four persisted tokens exist.
func (a *jobPhaseApprovalScan) apply(phase *pb.JobPhase) {
	if v, ok := pb.PhaseApprovalStatus_value[a.approvalStatus]; ok {
		phase.ApprovalStatus = pb.PhaseApprovalStatus(v)
	}
	if a.submittedBy.Valid {
		phase.SubmittedBy = &a.submittedBy.String
	}
	if a.submittedAt.Valid {
		phase.SubmittedAt = &a.submittedAt.Int64
	}
	if a.verifiedBy.Valid {
		phase.VerifiedBy = &a.verifiedBy.String
	}
	if a.verifiedAt.Valid {
		phase.VerifiedAt = &a.verifiedAt.Int64
	}
	if a.publishedBy.Valid {
		phase.PublishedBy = &a.publishedBy.String
	}
	if a.publishedAt.Valid {
		phase.PublishedAt = &a.publishedAt.Int64
	}
	if a.returnReason.Valid {
		phase.ReturnReason = &a.returnReason.String
	}
	if a.returnedBy.Valid {
		phase.ReturnedBy = &a.returnedBy.String
	}
	if a.returnedAt.Valid {
		phase.ReturnedAt = &a.returnedAt.Int64
	}
}

// GetJobPhaseListPageData retrieves job phases with pagination, filtering, sorting, and search
func (r *PostgresJobPhaseRepository) GetJobPhaseListPageData(
	ctx context.Context,
	req *pb.GetJobPhaseListPageDataRequest,
) (*pb.GetJobPhaseListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get job phase list page data request is required")
	}

	searchPattern, err := postgresCore.BoundedContainsSearchPattern(req.GetSearch())
	if err != nil {
		return nil, fmt.Errorf("invalid list search: %w", err)
	}

	limit, offset, page, err := postgresCore.BoundedOffsetPagination(req.GetPagination(), 50)
	if err != nil {
		return nil, fmt.Errorf("invalid list pagination: %w", err)
	}

	// Sort — fail-closed against the per-entity whitelist (A2 guard). The default
	// references the outer enriched projection (phase_order) since the page rows
	// are selected via "SELECT e.* FROM enriched e". An unknown sort column now
	// errors instead of being interpolated verbatim into ORDER BY.
	orderByClause, err := postgresCore.BuildOrderBy(jobPhaseSortableSQLCols, req.GetSort(), "phase_order ASC")
	if err != nil {
		return nil, err
	}

	// FIX-4: bind trusted job.workspace_id ancestry unconditionally (workspace = $4)
	// when the context carries a workspace — SHADOW-independent tenant scoping for
	// this page projection.
	wsJoin, wsID, wsScoped := jobWorkspaceScope(ctx, 4)
	queryArgs := []any{searchPattern, limit, offset}
	if wsScoped {
		queryArgs = append(queryArgs, wsID)
	}

	query := `
		WITH enriched AS (
			SELECT
				jp.id,
				jp.date_created,
				jp.date_modified,
				jp.active,
				jp.job_id,
				jp.name,
				jp.phase_order,
				jp.status` + jobPhaseApprovalCols + `
			FROM ` + entityid.JobPhase + ` jp` + wsJoin + `
			WHERE jp.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       jp.name ILIKE $1)
		)
		-- A3 (Q-PAGE-COUNT default tier): COUNT(*) OVER () computes the total in the
		-- same scan as the page rows (the prior counted CTE forced a second scan).
		SELECT
			e.*,
			COUNT(*) OVER () AS total
		FROM enriched e
		` + orderByClause + `
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query job phase list page data: %w", err)
	}
	defer rows.Close()

	var phases []*pb.JobPhase
	var totalCount int64

	for rows.Next() {
		var (
			id           string
			dateCreated  time.Time
			dateModified time.Time
			active       bool
			jobID        string
			name         string
			phaseOrder   int32
			status       string
			approval     jobPhaseApprovalScan
			total        int64
		)

		// Scan order must match the CTE SELECT (e.* then COUNT(*) OVER () AS total):
		// base cols, jobPhaseApprovalCols, then total.
		dest := []any{&id, &dateCreated, &dateModified, &active, &jobID, &name, &phaseOrder, &status}
		dest = append(dest, approval.scanDest()...)
		dest = append(dest, &total)
		if err := rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("failed to scan job phase row: %w", err)
		}

		totalCount = total

		phase := &pb.JobPhase{
			Id:         id,
			Active:     active,
			JobId:      jobID,
			Name:       name,
			PhaseOrder: phaseOrder,
		}

		// Map enum string to proto enum
		if v, ok := pb.PhaseStatus_value[status]; ok {
			phase.Status = pb.PhaseStatus(v)
		}
		approval.apply(phase)

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			phase.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			phase.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			phase.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			phase.DateModifiedString = &dmStr
		}

		phases = append(phases, phase)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating job phase rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &pb.GetJobPhaseListPageDataResponse{
		JobPhaseList: phases,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  int32(totalCount),
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     hasNext,
			HasPrev:     hasPrev,
		},
		Success: true,
	}, nil
}

// GetJobPhaseItemPageData retrieves a single job phase with enriched data
func (r *PostgresJobPhaseRepository) GetJobPhaseItemPageData(
	ctx context.Context,
	req *pb.GetJobPhaseItemPageDataRequest,
) (*pb.GetJobPhaseItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get job phase item page data request is required")
	}
	if req.JobPhaseId == "" {
		return nil, fmt.Errorf("job phase ID is required")
	}

	// FIX-4: bind trusted job.workspace_id ancestry unconditionally (workspace = $2)
	// when present — a cross-tenant by-id item read now fails closed to not-found
	// regardless of the global shadow/enforce flag.
	wsJoin, wsID, wsScoped := jobWorkspaceScope(ctx, 2)
	itemArgs := []any{req.JobPhaseId}
	if wsScoped {
		itemArgs = append(itemArgs, wsID)
	}

	query := `
		SELECT
			jp.id,
			jp.date_created,
			jp.date_modified,
			jp.active,
			jp.job_id,
			jp.name,
			jp.phase_order,
			jp.status` + jobPhaseApprovalCols + `
		FROM ` + entityid.JobPhase + ` jp` + wsJoin + `
		WHERE jp.id = $1 AND jp.active = true
	`

	row := r.db.QueryRowContext(ctx, query, itemArgs...)

	var (
		id           string
		dateCreated  time.Time
		dateModified time.Time
		active       bool
		jobID        string
		name         string
		phaseOrder   int32
		status       string
		approval     jobPhaseApprovalScan
	)

	dest := []any{&id, &dateCreated, &dateModified, &active, &jobID, &name, &phaseOrder, &status}
	dest = append(dest, approval.scanDest()...)
	err := row.Scan(dest...)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job phase with ID '%s' not found", req.JobPhaseId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query job phase item page data: %w", err)
	}

	phase := &pb.JobPhase{
		Id:         id,
		Active:     active,
		JobId:      jobID,
		Name:       name,
		PhaseOrder: phaseOrder,
	}

	if v, ok := pb.PhaseStatus_value[status]; ok {
		phase.Status = pb.PhaseStatus(v)
	}
	approval.apply(phase)

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		phase.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		phase.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		phase.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		phase.DateModifiedString = &dmStr
	}

	return &pb.GetJobPhaseItemPageDataResponse{
		JobPhase: phase,
		Success:  true,
	}, nil
}

// ListByJob retrieves all phases for a given job, ordered by phase_order
func (r *PostgresJobPhaseRepository) ListByJob(
	ctx context.Context,
	req *pb.ListJobPhasesByJobRequest,
) (*pb.ListJobPhasesByJobResponse, error) {
	if req == nil || req.JobId == "" {
		return nil, fmt.Errorf("job ID is required")
	}

	// FIX-4: bind trusted job.workspace_id ancestry unconditionally (workspace = $2)
	// when present — ListByJob no longer returns another tenant's phases regardless
	// of the global shadow/enforce flag. A system/CLI caller with no workspace in
	// context keeps the prior unscoped behavior (pass-through).
	wsJoin, wsID, wsScoped := jobWorkspaceScope(ctx, 2)
	listArgs := []any{req.JobId}
	if wsScoped {
		listArgs = append(listArgs, wsID)
	}

	query := `
		SELECT
			jp.id,
			jp.date_created,
			jp.date_modified,
			jp.active,
			jp.job_id,
			jp.name,
			jp.phase_order,
			jp.status` + jobPhaseApprovalCols + `
		FROM ` + entityid.JobPhase + ` jp` + wsJoin + `
		WHERE jp.job_id = $1 AND jp.active = true
		ORDER BY jp.phase_order ASC
	`

	rows, err := r.db.QueryContext(ctx, query, listArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to list job phases by job: %w", err)
	}
	defer rows.Close()

	var phases []*pb.JobPhase
	for rows.Next() {
		var (
			id           string
			dateCreated  time.Time
			dateModified time.Time
			active       bool
			jobID        string
			name         string
			phaseOrder   int32
			status       string
			approval     jobPhaseApprovalScan
		)

		dest := []any{&id, &dateCreated, &dateModified, &active, &jobID, &name, &phaseOrder, &status}
		dest = append(dest, approval.scanDest()...)
		if err := rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("failed to scan job phase row: %w", err)
		}

		phase := &pb.JobPhase{
			Id:         id,
			Active:     active,
			JobId:      jobID,
			Name:       name,
			PhaseOrder: phaseOrder,
		}

		if v, ok := pb.PhaseStatus_value[status]; ok {
			phase.Status = pb.PhaseStatus(v)
		}
		approval.apply(phase)

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			phase.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			phase.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			phase.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			phase.DateModifiedString = &dmStr
		}

		phases = append(phases, phase)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating job phase rows: %w", err)
	}

	return &pb.ListJobPhasesByJobResponse{
		JobPhases: phases,
		Success:   true,
	}, nil
}

// NewJobPhaseRepository creates a new PostgreSQL job_phase repository (old-style constructor)
func NewJobPhaseRepository(db *sql.DB, tableName string) pb.JobPhaseDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresJobPhaseRepository(dbOps, tableName)
}

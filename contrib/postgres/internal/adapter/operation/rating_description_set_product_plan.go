//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"fmt"
	"time"

	auditadapter "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/audit"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	portsdomain "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	infraports "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	sqlexec "github.com/erniealice/espyna-golang/shared/database/sqlexec"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_product_plan"
	"github.com/lib/pq"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.RatingDescriptionSetProductPlan, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres rating_description_set_product_plan repository requires *sql.DB, got %T", conn)
		}
		// schema-proposal.md §9.4 / codex-review-impl2.out.md finding 9: join the
		// ambient transaction so Create/Update/Delete diff-audits roll back with
		// the write (see rating_description_set.go's init() for the full note).
		dbOps := postgresCore.NewAuditedWorkspaceAwareOperations(db, auditadapter.New(db))
		return NewPostgresRatingDescriptionSetProductPlanRepository(dbOps, tableName), nil
	})
}

// PostgresRatingDescriptionSetProductPlanRepository implements
// rating_description_set_product_plan CRUD via PostgreSQL, plus the page-data
// read and two conditional writes (RelinkLocked, UnlinkLocked) that are NOT
// part of the generated RatingDescriptionSetProductPlanDomainServiceServer
// interface — Relink/Unlink are plain use-case messages (interfaces.md §2),
// consumed by the lifecycle use cases via a domain-port type assertion
// (internal/application/ports/domain/operation.go), same pattern as
// PostgresRatingDescriptionSetRepository's Publish/Deprecate.
//
// ProposeRatingDescriptionSetProductPlansFromPreviousPriceSchedule IS part of
// the generated interface but is on this agent's Defer list (BulkRelink,
// Propose, Import, audited-ops wrapper — see W3-ESPYNA.done); it is left to
// the embedded UnimplementedRatingDescriptionSetProductPlanDomainServiceServer,
// which returns a clean codes.Unimplemented error rather than panicking.
type PostgresRatingDescriptionSetProductPlanRepository struct {
	pb.UnimplementedRatingDescriptionSetProductPlanDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
	// audit writes the append-only Relink/Unlink transition event inside the
	// same ambient transaction as the lock-and-swap SQL (writeLifecycleAudit;
	// schema-proposal.md §9.4). writeLifecycleAudit itself fails closed on a
	// nil audit, mirroring PostgresJobPhaseRepository.writeTransitionAudit
	// (job_phase.go).
	audit infraports.AuditService
}

func NewPostgresRatingDescriptionSetProductPlanRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.RatingDescriptionSetProductPlanDomainServiceServer {
	if tableName == "" {
		tableName = entityid.RatingDescriptionSetProductPlan
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	var auditSvc infraports.AuditService
	if db != nil {
		// The Relink/Unlink transition audit is a security-critical durable
		// record (codex-review-impl2.out.md finding 9), so it is always written
		// when a real connection is available — same posture as
		// PostgresJobPhaseRepository (job_phase.go).
		auditSvc = auditadapter.New(db)
	}
	return &PostgresRatingDescriptionSetProductPlanRepository{dbOps: dbOps, db: db, tableName: tableName, audit: auditSvc}
}

// writeLifecycleAudit records a Relink/Unlink link transition as a durable
// semantic audit event inside the SAME ambient transaction as the
// lock-and-swap SQL (schema-proposal.md §9.4; codex-review-impl2.out.md
// finding 9). Mirrors PostgresRatingDescriptionSetRepository's twin (and
// PostgresJobPhaseRepository.writeTransitionAudit, job_phase.go): AUDIT IS
// MANDATORY — a nil audit dependency fails the transition, and the actor is
// derived from the trusted authenticated identity.
func (r *PostgresRatingDescriptionSetProductPlanRepository) writeLifecycleAudit(ctx context.Context, workspaceID, linkID, permCode, useCase, reason string, changes []infraports.AuditFieldChange) error {
	if r.audit == nil {
		return fmt.Errorf("rating description set product plan %s: audit dependency is absent — refusing to transition without a durable audit event (fail closed)", useCase)
	}
	idn, ok := identity.FromContext(ctx)
	if !ok || idn == nil || idn.UserID == "" {
		return fmt.Errorf("rating description set product plan %s: no trusted actor identity for the audit event (fail closed)", useCase)
	}
	ac, _ := infraports.GetAuditContext(ctx)
	ac.ActorID = idn.UserID
	ac.ActorType = "user"
	ctx = infraports.WithAuditContext(ctx, ac)
	return r.audit.LogEntry(ctx, &infraports.AuditLogRequest{
		WorkspaceID:    workspaceID,
		EntityType:     entityid.RatingDescriptionSetProductPlan,
		EntityID:       linkID,
		Domain:         "espyna",
		Action:         2, // AUDIT_ACTION_UPDATE
		PermissionCode: permCode,
		UseCase:        useCase,
		Reason:         reason,
		MethodName:     useCase,
		FieldChanges:   changes,
	})
}

func (r *PostgresRatingDescriptionSetProductPlanRepository) executor(ctx context.Context) sqlexec.DBExecutor {
	if ep, ok := r.dbOps.(interface {
		GetExecutor(ctx context.Context) sqlexec.DBExecutor
	}); ok {
		if e := ep.GetExecutor(ctx); e != nil {
			return e
		}
	}
	if r.db != nil {
		return r.db
	}
	return nil
}

func (r *PostgresRatingDescriptionSetProductPlanRepository) CreateRatingDescriptionSetProductPlan(ctx context.Context, req *pb.CreateRatingDescriptionSetProductPlanRequest) (*pb.CreateRatingDescriptionSetProductPlanResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("rating description set product plan data is required")
	}
	data, err := protoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create rating description set product plan link: %w", err)
	}
	item, err := ratingDescriptionSetProductPlanFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.CreateRatingDescriptionSetProductPlanResponse{Data: []*pb.RatingDescriptionSetProductPlan{item}, Success: true}, nil
}

func (r *PostgresRatingDescriptionSetProductPlanRepository) ReadRatingDescriptionSetProductPlan(ctx context.Context, req *pb.ReadRatingDescriptionSetProductPlanRequest) (*pb.ReadRatingDescriptionSetProductPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("rating description set product plan ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read rating description set product plan link: %w", err)
	}
	item, err := ratingDescriptionSetProductPlanFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.ReadRatingDescriptionSetProductPlanResponse{Data: []*pb.RatingDescriptionSetProductPlan{item}, Success: true}, nil
}

func (r *PostgresRatingDescriptionSetProductPlanRepository) UpdateRatingDescriptionSetProductPlan(ctx context.Context, req *pb.UpdateRatingDescriptionSetProductPlanRequest) (*pb.UpdateRatingDescriptionSetProductPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("rating description set product plan ID is required")
	}
	data, err := protoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update rating description set product plan link: %w", err)
	}
	item, err := ratingDescriptionSetProductPlanFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateRatingDescriptionSetProductPlanResponse{Data: []*pb.RatingDescriptionSetProductPlan{item}, Success: true}, nil
}

func (r *PostgresRatingDescriptionSetProductPlanRepository) DeleteRatingDescriptionSetProductPlan(ctx context.Context, req *pb.DeleteRatingDescriptionSetProductPlanRequest) (*pb.DeleteRatingDescriptionSetProductPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("rating description set product plan ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete rating description set product plan link: %w", err)
	}
	return &pb.DeleteRatingDescriptionSetProductPlanResponse{Success: true}, nil
}

func (r *PostgresRatingDescriptionSetProductPlanRepository) ListRatingDescriptionSetProductPlans(ctx context.Context, req *pb.ListRatingDescriptionSetProductPlansRequest) (*pb.ListRatingDescriptionSetProductPlansResponse, error) {
	// fix3-backend (codex-review-impl3 #2): forward search/filters/sort/
	// pagination (see ratingDescriptionListParams in rating_description_set.go).
	params, err := ratingDescriptionListParams(req.GetSearch(), req.GetFilters(), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	items, _, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &pb.ListRatingDescriptionSetProductPlansResponse{Data: items, Success: true}, nil
}

// GetRatingDescriptionSetProductPlanListPageData is the AY setup read
// (interfaces.md §4 route rating_description_set_product_plan.list): every
// rating_description_set_product_plan row (any status — active AND inactive,
// so history/relink trail is visible) for the given price_schedule_id, plus
// any additional caller filters/sort/pagination/search.
//
// NOTE (frozen-proto constraint, recorded for the fayna views agent): the
// generated GetRatingDescriptionSetProductPlanListPageDataResponse (esqyma
// CHECKPOINT-2, frozen for this wave) returns
// []*RatingDescriptionSetProductPlan — the LINK entity only. It has no field
// for offerings that have NEVER been linked (no row exists yet), so a true
// "offerings x AY" grid (interfaces.md's phrasing) that also lists UNLINKED
// product_plans is not representable by this RPC's response shape. The fayna
// view layer must compose this link list against a separate product_plan
// read (e.g. ListProductPlans) to render the full unlinked/linked/invalid
// grid — or a later checkpoint widens the proto. Not a regression: nothing in
// W1/W2/CHECKPOINT-2 defined a richer response type for this RPC.
func (r *PostgresRatingDescriptionSetProductPlanRepository) GetRatingDescriptionSetProductPlanListPageData(ctx context.Context, req *pb.GetRatingDescriptionSetProductPlanListPageDataRequest) (*pb.GetRatingDescriptionSetProductPlanListPageDataResponse, error) {
	if req == nil || req.PriceScheduleId == "" {
		return nil, fmt.Errorf("price_schedule_id is required")
	}
	// OR logic is refused BEFORE the mandatory scope filter is appended —
	// otherwise the price_schedule_id predicate would be OR-ed with the
	// caller's filters instead of AND-ed.
	if _, err := ratingDescriptionListParams(nil, req.GetFilters(), nil, nil); err != nil {
		return nil, err
	}
	filters := mergeEqualsFilter(req.GetFilters(), "price_schedule_id", req.PriceScheduleId)
	params, err := ratingDescriptionListParams(req.GetSearch(), filters, req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	// Total/has_next from the adapter's COUNT(*) (codex-review-impl3 #2) —
	// no longer an in-memory slice of a 100-row-capped read.
	items, pagination, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &pb.GetRatingDescriptionSetProductPlanListPageDataResponse{RatingDescriptionSetProductPlanList: items, Pagination: pagination, Success: true}, nil
}

var _ portsdomain.RatingDescriptionSetProductPlanCounter = (*PostgresRatingDescriptionSetProductPlanRepository)(nil)

// CountActiveRatingDescriptionSetProductPlansBySet implements
// portsdomain.RatingDescriptionSetProductPlanCounter (codex-review-impl4.out.md
// round-3/round-4 disposition #2). Replaces the former workspace-wide,
// 5,000-row-capped page loop with one scoped GROUP BY aggregation.
func (r *PostgresRatingDescriptionSetProductPlanRepository) CountActiveRatingDescriptionSetProductPlansBySet(ctx context.Context, ratingDescriptionSetIds []string) (map[string]int32, error) {
	counts := map[string]int32{}
	if len(ratingDescriptionSetIds) == 0 {
		return counts, nil
	}
	exec := r.executor(ctx)
	if exec == nil {
		return nil, fmt.Errorf("count active rating description set product plans: no SQL executor available")
	}
	rows, err := exec.QueryContext(ctx,
		`SELECT rating_description_set_id, COUNT(*) FROM `+entityid.RatingDescriptionSetProductPlan+`
		 WHERE active = true AND rating_description_set_id = ANY($1)
		 GROUP BY rating_description_set_id`,
		pq.Array(ratingDescriptionSetIds))
	if err != nil {
		return nil, fmt.Errorf("count active rating description set product plans: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var setID string
		var count int32
		if err := rows.Scan(&setID, &count); err != nil {
			return nil, fmt.Errorf("count active rating description set product plans: scan: %w", err)
		}
		counts[setID] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("count active rating description set product plans: %w", err)
	}
	return counts, nil
}

// RelinkLocked implements the relink transaction body (schema-proposal.md
// §9.2, espyna-golang.md "Lifecycle and relink rules"): validate offering
// (product_plan) and AY (price_schedule) tenant ownership via their parents
// (W3 follow-up finding, 2026-09-25 — see the inline step-0 doc comment
// below), lock the offering pair (advisory lock so two concurrent FIRST
// links serialize even with no row to FOR UPDATE yet, then FOR UPDATE on any
// existing active row), compare expectedCurrentLinkID (stale-request guard:
// nil means "expect no current link"), require the target set PUBLISHED +
// same workspace (FOR SHARE, so a concurrent Deprecate's row lock serializes
// against it), deactivate the old active link if any, insert newLink.
// Requires an ambient transaction — fails closed otherwise. newLink must
// already carry Id/WorkspaceId/DateCreated (the use case enriches it,
// mirroring CreateScoreScaleUseCase.enrich). reason is the caller-supplied
// RelinkRatingDescriptionSetProductPlanRequest.reason, persisted on the
// semantic audit event this method writes before returning (schema-
// proposal.md §9.4, codex-review-impl2.out.md finding 9).
func (r *PostgresRatingDescriptionSetProductPlanRepository) RelinkLocked(ctx context.Context, newLink *pb.RatingDescriptionSetProductPlan, expectedCurrentLinkID *string, reason string) (*pb.RatingDescriptionSetProductPlan, error) {
	if newLink == nil || newLink.ProductPlanId == "" || newLink.PriceScheduleId == "" || newLink.RatingDescriptionSetId == "" {
		return nil, fmt.Errorf("relink: product_plan_id, price_schedule_id, rating_description_set_id are required")
	}
	idn, ok := identity.FromContext(ctx)
	if !ok || idn == nil || idn.WorkspaceID == "" {
		return nil, fmt.Errorf("relink: no trusted workspace in context (fail closed)")
	}
	exec := r.executor(ctx)
	if exec == nil {
		return nil, fmt.Errorf("relink: no SQL executor available")
	}
	if _, isTx := exec.(*sql.Tx); !isTx {
		return nil, fmt.Errorf("relink: requires an ambient transaction (fail closed)")
	}

	// 0. Validate BOTH the offering (product_plan) and the academic year
	// (price_schedule) belong to the caller's workspace, BEFORE taking any
	// lock or inserting (W3 follow-up finding, 2026-09-25 —
	// codex-review-impl1.out.md: "RelinkLocked ... allows an initially
	// unlinked foreign offering/AY pair to receive a caller-owned link. That
	// occupies the global unique pair and prevents its rightful workspace
	// from linking it."). product_plan carries NO workspace_id column of its
	// own (confirmed via read-only psql \d on education2clone20260925a), so
	// ownership is derived through ITS parents: product.workspace_id AND
	// plan.workspace_id must both equal the caller's workspace (NULL or
	// mismatched on either side fails closed — a legacy/cross-tenant
	// product_plan can never be relinked). price_schedule.workspace_id is a
	// direct column and is checked the same fail-closed way.
	var productWorkspaceID, planWorkspaceID sql.NullString
	if err := exec.QueryRowContext(ctx,
		`SELECT p.workspace_id, pl.workspace_id
		 FROM `+entityid.ProductPlan+` pp
		 JOIN `+entityid.Product+` p ON p.id = pp.product_id
		 JOIN `+entityid.Plan+` pl ON pl.id = pp.plan_id
		 WHERE pp.id = $1`,
		newLink.ProductPlanId).Scan(&productWorkspaceID, &planWorkspaceID); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("relink: product_plan not found (or its product/plan parent is missing) — fail closed")
		}
		return nil, fmt.Errorf("relink: resolve product_plan tenant ownership: %w", err)
	}
	if !productWorkspaceID.Valid || productWorkspaceID.String != idn.WorkspaceID ||
		!planWorkspaceID.Valid || planWorkspaceID.String != idn.WorkspaceID {
		return nil, fmt.Errorf("forbidden: product_plan is not owned by the caller's workspace")
	}

	var scheduleWorkspaceID sql.NullString
	if err := exec.QueryRowContext(ctx,
		`SELECT workspace_id FROM `+entityid.PriceSchedule+` WHERE id = $1`,
		newLink.PriceScheduleId).Scan(&scheduleWorkspaceID); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("relink: price_schedule not found — fail closed")
		}
		return nil, fmt.Errorf("relink: resolve price_schedule tenant ownership: %w", err)
	}
	if !scheduleWorkspaceID.Valid || scheduleWorkspaceID.String != idn.WorkspaceID {
		return nil, fmt.Errorf("forbidden: price_schedule is not owned by the caller's workspace")
	}

	// 1. Advisory transaction lock on the (product_plan_id, price_schedule_id)
	// pair — serializes concurrent FIRST links, which have no row to lock yet.
	if _, err := exec.ExecContext(ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`,
		newLink.ProductPlanId+"|"+newLink.PriceScheduleId); err != nil {
		return nil, fmt.Errorf("relink: advisory lock: %w", err)
	}

	// 2. Find + lock the current active link for this pair, if any.
	var currentID, currentWorkspace string
	err := exec.QueryRowContext(ctx,
		`SELECT id, workspace_id FROM `+entityid.RatingDescriptionSetProductPlan+`
		 WHERE product_plan_id = $1 AND price_schedule_id = $2 AND active = true FOR UPDATE`,
		newLink.ProductPlanId, newLink.PriceScheduleId).Scan(&currentID, &currentWorkspace)
	switch {
	case err == sql.ErrNoRows:
		currentID = ""
	case err != nil:
		return nil, fmt.Errorf("relink: lock current link: %w", err)
	default:
		if currentWorkspace != idn.WorkspaceID {
			return nil, fmt.Errorf("relink: current link is in a different workspace — fail closed")
		}
	}

	// 3. Stale-request guard.
	switch {
	case expectedCurrentLinkID != nil && *expectedCurrentLinkID != currentID:
		return nil, fmt.Errorf("STALE_LINK: expected current link %q but found %q", *expectedCurrentLinkID, currentID)
	case expectedCurrentLinkID == nil && currentID != "":
		return nil, fmt.Errorf("STALE_LINK: expected no current link but found %q", currentID)
	}

	// 4. Target set must be PUBLISHED + same workspace; FOR SHARE so it
	// serializes against a concurrent Deprecate (which locks the set row for
	// its own conditional UPDATE) — "Linking vs deprecating" in
	// espyna-golang.md.
	var setStatus, setWorkspace string
	if err := exec.QueryRowContext(ctx,
		`SELECT version_status, workspace_id FROM `+entityid.RatingDescriptionSet+` WHERE id = $1 FOR SHARE`,
		newLink.RatingDescriptionSetId).Scan(&setStatus, &setWorkspace); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("relink: target rating description set not found")
		}
		return nil, fmt.Errorf("relink: lock target set FOR SHARE: %w", err)
	}
	if setWorkspace != idn.WorkspaceID {
		return nil, fmt.Errorf("relink: target rating description set is in a different workspace — fail closed")
	}
	if setStatus != versionStatusPublishedValue {
		return nil, fmt.Errorf("SET_NOT_PUBLISHED: target rating description set is not PUBLISHED")
	}

	// 5. Deactivate the current link, if any.
	if currentID != "" {
		if _, err := exec.ExecContext(ctx,
			`UPDATE `+entityid.RatingDescriptionSetProductPlan+` SET active = false, date_modified = $1 WHERE id = $2`,
			time.Now().UnixMilli(), currentID); err != nil {
			return nil, fmt.Errorf("relink: deactivate current link: %w", err)
		}
	}

	// 6. Insert the new link via the standard write path (workspace stamping
	// + column canonicalization) — dbOps.Create's GetExecutor(ctx) resolves to
	// the SAME ambient *sql.Tx used above.
	newLink.Active = true
	data, err := protoToMap(newLink)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("relink: insert new link: %w", err)
	}
	item, err := ratingDescriptionSetProductPlanFromResult(result)
	if err != nil {
		return nil, err
	}

	// 7. Semantic audit event, inside the SAME ambient transaction — a
	// failure here returns an error, which the calling use case's
	// services.Transactor.ExecuteInTransaction rolls back (steps 5+6
	// included; schema-proposal.md §9.4 "audit failure rolls back the
	// operation").
	if err := r.writeLifecycleAudit(ctx, idn.WorkspaceID, item.GetId(),
		entityid.EntityPermission(entityid.RatingDescriptionSetProductPlan, relinkAuditAction(currentID)),
		"RelinkRatingDescriptionSetProductPlan", reason, []infraports.AuditFieldChange{
			{FieldName: "product_plan_id", FieldType: 1, OldValue: "", NewValue: newLink.ProductPlanId},
			{FieldName: "price_schedule_id", FieldType: 1, OldValue: "", NewValue: newLink.PriceScheduleId},
			{FieldName: "rating_description_set_id", FieldType: 1, OldValue: "", NewValue: newLink.RatingDescriptionSetId},
			{FieldName: "old_link_id", FieldType: 1, OldValue: "", NewValue: currentID},
			{FieldName: "new_link_id", FieldType: 1, OldValue: "", NewValue: item.GetId()},
		}); err != nil {
		return nil, fmt.Errorf("relink: %w", err)
	}
	return item, nil
}

// relinkAuditAction mirrors the use case's own action choice
// (RelinkRatingDescriptionSetProductPlanUseCase.Execute — entityid.ActionCreate
// when there was no current link, entityid.ActionUpdate for a replacement) so
// the audit permission_code always matches the ActionGatekeeper check that
// authorized this call.
func relinkAuditAction(currentID string) string {
	if currentID == "" {
		return entityid.ActionCreate
	}
	return entityid.ActionUpdate
}

// UnlinkLocked deactivates the given link with the same lock discipline as
// RelinkLocked (FOR UPDATE on the link row, workspace check, fail closed
// without an ambient transaction). The offering becomes NO_LINK for the
// resolver (schema-proposal.md §4). reason is the caller-supplied
// UnlinkRatingDescriptionSetProductPlanRequest.reason, persisted on the
// semantic audit event this method writes before returning (schema-
// proposal.md §9.4, codex-review-impl2.out.md finding 9).
func (r *PostgresRatingDescriptionSetProductPlanRepository) UnlinkLocked(ctx context.Context, linkID string, reason string) (*pb.RatingDescriptionSetProductPlan, error) {
	if linkID == "" {
		return nil, fmt.Errorf("unlink: link_id is required")
	}
	idn, ok := identity.FromContext(ctx)
	if !ok || idn == nil || idn.WorkspaceID == "" {
		return nil, fmt.Errorf("unlink: no trusted workspace in context (fail closed)")
	}
	exec := r.executor(ctx)
	if exec == nil {
		return nil, fmt.Errorf("unlink: no SQL executor available")
	}
	if _, isTx := exec.(*sql.Tx); !isTx {
		return nil, fmt.Errorf("unlink: requires an ambient transaction (fail closed)")
	}
	var workspaceID, productPlanID, priceScheduleID, setID string
	var active bool
	if err := exec.QueryRowContext(ctx,
		`SELECT workspace_id, active, product_plan_id, price_schedule_id, rating_description_set_id
		 FROM `+entityid.RatingDescriptionSetProductPlan+` WHERE id = $1 FOR UPDATE`,
		linkID).Scan(&workspaceID, &active, &productPlanID, &priceScheduleID, &setID); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("unlink: link not found")
		}
		return nil, fmt.Errorf("unlink: lock link FOR UPDATE: %w", err)
	}
	if workspaceID != idn.WorkspaceID {
		return nil, fmt.Errorf("unlink: link is in a different workspace — fail closed")
	}
	if !active {
		return nil, fmt.Errorf("STALE_LINK: link is already inactive")
	}
	if _, err := exec.ExecContext(ctx,
		`UPDATE `+entityid.RatingDescriptionSetProductPlan+` SET active = false, date_modified = $1 WHERE id = $2`,
		time.Now().UnixMilli(), linkID); err != nil {
		return nil, fmt.Errorf("unlink: %w", err)
	}

	// Semantic audit event, inside the SAME ambient transaction — a failure
	// here returns an error, which the calling use case's
	// services.Transactor.ExecuteInTransaction rolls back the deactivate above
	// (schema-proposal.md §9.4 "audit failure rolls back the operation").
	if err := r.writeLifecycleAudit(ctx, workspaceID, linkID,
		entityid.EntityPermission(entityid.RatingDescriptionSetProductPlan, entityid.ActionDelete),
		"UnlinkRatingDescriptionSetProductPlan", reason, []infraports.AuditFieldChange{
			{FieldName: "product_plan_id", FieldType: 1, OldValue: productPlanID, NewValue: ""},
			{FieldName: "price_schedule_id", FieldType: 1, OldValue: priceScheduleID, NewValue: ""},
			{FieldName: "rating_description_set_id", FieldType: 1, OldValue: setID, NewValue: ""},
			{FieldName: "old_link_id", FieldType: 1, OldValue: linkID, NewValue: ""},
			{FieldName: "active", FieldType: 1, OldValue: "true", NewValue: "false"},
		}); err != nil {
		return nil, fmt.Errorf("unlink: %w", err)
	}

	result, err := r.dbOps.Read(ctx, r.tableName, linkID)
	if err != nil {
		return nil, fmt.Errorf("unlink: read-back: %w", err)
	}
	return ratingDescriptionSetProductPlanFromResult(result)
}

func (r *PostgresRatingDescriptionSetProductPlanRepository) listPage(ctx context.Context, params *interfaces.ListParams) ([]*pb.RatingDescriptionSetProductPlan, *commonpb.PaginationResponse, error) {
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list rating description set product plan links: %w", err)
	}
	var items []*pb.RatingDescriptionSetProductPlan
	for _, row := range listResult.Data {
		rj, err := json.Marshal(row)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal rating description set product plan row: %w", err)
		}
		item := &pb.RatingDescriptionSetProductPlan{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
			return nil, nil, fmt.Errorf("failed to decode rating description set product plan row: %w", err)
		}
		items = append(items, item)
	}
	return items, listResult.Pagination, nil
}

func ratingDescriptionSetProductPlanFromResult(result any) (*pb.RatingDescriptionSetProductPlan, error) {
	rj, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	item := &pb.RatingDescriptionSetProductPlan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to proto: %w", err)
	}
	return item, nil
}

// mergeEqualsFilter appends a mandatory case-sensitive equals filter (the
// price_schedule_id scope) onto whatever filters the caller already supplied.
func mergeEqualsFilter(base *commonpb.FilterRequest, field, value string) *commonpb.FilterRequest {
	extra := &commonpb.TypedFilter{
		Field: field,
		FilterType: &commonpb.TypedFilter_StringFilter{
			StringFilter: &commonpb.StringFilter{
				Value:         value,
				Operator:      commonpb.StringOperator_STRING_EQUALS,
				CaseSensitive: true,
			},
		},
	}
	if base == nil {
		return &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{extra}}
	}
	merged := &commonpb.FilterRequest{
		Filters: append(append([]*commonpb.TypedFilter{}, base.Filters...), extra),
		Logic:   base.Logic,
	}
	return merged
}

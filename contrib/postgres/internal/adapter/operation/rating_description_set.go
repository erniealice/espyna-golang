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
	infraports "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	sqlexec "github.com/erniealice/espyna-golang/shared/database/sqlexec"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set"
	"google.golang.org/protobuf/encoding/protojson"
)

// Version-status text literals as protojson serializes the VersionStatus enum
// (name form, not number). Mirrors outcome_matrix_rating_resolution_query.go's
// versionStatusPublishedValue/versionStatusDeprecatedValue (same package) —
// only the DRAFT literal is new here.
const ratingDescriptionSetVersionStatusDraft = "VERSION_STATUS_DRAFT"

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.RatingDescriptionSet, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres rating_description_set repository requires *sql.DB, got %T", conn)
		}
		// schema-proposal.md §9.4 / codex-review-impl2.out.md finding 9: the
		// generic Create/Update/Delete diff-audit must join the ambient
		// transaction — NewAuditedWorkspaceAwareOperations wraps
		// NewPostgresOperationsWithAudit so DiffAndLog runs on the SAME *sql.Tx
		// as the caller's write (contrib/postgres/internal/adapter/core/
		// workspace_operations.go). auditadapter.New(db) is stateless besides db,
		// so this and the constructor's own copy below are interchangeable.
		dbOps := postgresCore.NewAuditedWorkspaceAwareOperations(db, auditadapter.New(db))
		return NewPostgresRatingDescriptionSetRepository(dbOps, tableName), nil
	})
}

// PostgresRatingDescriptionSetRepository implements rating_description_set CRUD
// via PostgreSQL, modeled on PostgresScoreScaleRepository. It additionally
// exposes two conditional lifecycle writes — PublishRatingDescriptionSetIfDraft
// and DeprecateRatingDescriptionSetIfPublished (schema-proposal.md §9.2) — that
// are NOT part of the generated RatingDescriptionSetDomainServiceServer
// interface (Publish/Deprecate are plain use-case messages, interfaces.md §2,
// not proto RPCs). Those two methods are consumed by the domain-port
// interfaces in internal/application/ports/domain/operation.go via a type
// assertion inside the lifecycle use cases' own constructors — the same
// pattern already used by
// subscription_group_document_template.DeleteDraftPairUseCase for
// SubscriptionGroupDocumentTemplateDraftPairDeleter.
type PostgresRatingDescriptionSetRepository struct {
	pb.UnimplementedRatingDescriptionSetDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
	// audit writes the append-only Publish/Deprecate transition event inside
	// the same ambient transaction as the conditional status UPDATE
	// (writeLifecycleAudit; schema-proposal.md §9.4). Nil-safe at the
	// constructor boundary only when db itself is nil (mock/test paths);
	// writeLifecycleAudit itself fails closed on a nil audit, mirroring
	// PostgresJobPhaseRepository.writeTransitionAudit (job_phase.go).
	audit infraports.AuditService
}

func NewPostgresRatingDescriptionSetRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.RatingDescriptionSetDomainServiceServer {
	if tableName == "" {
		tableName = entityid.RatingDescriptionSet
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	var auditSvc infraports.AuditService
	if db != nil {
		// The Publish/Deprecate transition audit is a security-critical durable
		// record (codex-review-impl2.out.md finding 9), so it is always written
		// when a real connection is available — same posture as
		// PostgresJobPhaseRepository (job_phase.go).
		auditSvc = auditadapter.New(db)
	}
	return &PostgresRatingDescriptionSetRepository{dbOps: dbOps, db: db, tableName: tableName, audit: auditSvc}
}

// executor returns the transaction-aware SQL executor: the active *sql.Tx when
// one is present on ctx, else the pooled *sql.DB. Mirrors
// PostgresTaskOutcomeRepository.executor (task_outcome.go).
func (r *PostgresRatingDescriptionSetRepository) executor(ctx context.Context) sqlexec.DBExecutor {
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

func (r *PostgresRatingDescriptionSetRepository) CreateRatingDescriptionSet(ctx context.Context, req *pb.CreateRatingDescriptionSetRequest) (*pb.CreateRatingDescriptionSetResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("rating description set data is required")
	}
	data, err := protoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create rating description set: %w", err)
	}
	item, err := ratingDescriptionSetFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.CreateRatingDescriptionSetResponse{Data: []*pb.RatingDescriptionSet{item}, Success: true}, nil
}

func (r *PostgresRatingDescriptionSetRepository) ReadRatingDescriptionSet(ctx context.Context, req *pb.ReadRatingDescriptionSetRequest) (*pb.ReadRatingDescriptionSetResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("rating description set ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read rating description set: %w", err)
	}
	item, err := ratingDescriptionSetFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.ReadRatingDescriptionSetResponse{Data: []*pb.RatingDescriptionSet{item}, Success: true}, nil
}

func (r *PostgresRatingDescriptionSetRepository) UpdateRatingDescriptionSet(ctx context.Context, req *pb.UpdateRatingDescriptionSetRequest) (*pb.UpdateRatingDescriptionSetResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("rating description set ID is required")
	}
	data, err := protoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update rating description set: %w", err)
	}
	item, err := ratingDescriptionSetFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateRatingDescriptionSetResponse{Data: []*pb.RatingDescriptionSet{item}, Success: true}, nil
}

func (r *PostgresRatingDescriptionSetRepository) DeleteRatingDescriptionSet(ctx context.Context, req *pb.DeleteRatingDescriptionSetRequest) (*pb.DeleteRatingDescriptionSetResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("rating description set ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete rating description set: %w", err)
	}
	return &pb.DeleteRatingDescriptionSetResponse{Success: true}, nil
}

func (r *PostgresRatingDescriptionSetRepository) ListRatingDescriptionSets(ctx context.Context, req *pb.ListRatingDescriptionSetsRequest) (*pb.ListRatingDescriptionSetsResponse, error) {
	// fix3-backend (codex-review-impl3 #2): forward search/filters/sort/
	// pagination to the query — callers page with offset pagination and stop
	// on a short page, which only terminates correctly when the adapter
	// actually applies LIMIT/OFFSET.
	params, err := ratingDescriptionListParams(req.GetSearch(), req.GetFilters(), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	items, _, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &pb.ListRatingDescriptionSetsResponse{Data: items, Success: true}, nil
}

func (r *PostgresRatingDescriptionSetRepository) GetRatingDescriptionSetListPageData(ctx context.Context, req *pb.GetRatingDescriptionSetListPageDataRequest) (*pb.GetRatingDescriptionSetListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	params, err := ratingDescriptionListParams(req.GetSearch(), req.GetFilters(), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	// Pagination metadata (total/has_next) comes from the adapter's COUNT(*)
	// over the same predicate — never from an in-memory slice of a capped read.
	items, pagination, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &pb.GetRatingDescriptionSetListPageDataResponse{RatingDescriptionSetList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresRatingDescriptionSetRepository) GetRatingDescriptionSetItemPageData(ctx context.Context, req *pb.GetRatingDescriptionSetItemPageDataRequest) (*pb.GetRatingDescriptionSetItemPageDataResponse, error) {
	if req == nil || req.RatingDescriptionSetId == "" {
		return nil, fmt.Errorf("rating description set ID required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.RatingDescriptionSetId)
	if err != nil {
		return nil, fmt.Errorf("failed to read rating description set: %w", err)
	}
	item, err := ratingDescriptionSetFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.GetRatingDescriptionSetItemPageDataResponse{RatingDescriptionSet: item, Success: true}, nil
}

// LockRatingDescriptionSetForUpdate takes the set row lock FOR UPDATE inside
// the ambient transaction and returns its current version_status literal
// (schema-proposal.md §9.2; W3 follow-up findings, 2026-09-25 — "Publish
// counts entries before locking" and "Set update reads DRAFT outside a
// transaction and writes that status back"). Callers (Update, Publish) MUST
// perform any state-dependent read (an entry count, a field merge) AFTER this
// call returns, inside the SAME transaction — only then does the lock
// actually serialize against concurrent writers: a concurrent entry
// create/update/delete takes FOR SHARE on this same row
// (guardRatingDescriptionSetEntryWrite) and a concurrent Deprecate takes the
// row's own UPDATE lock (conditionalStatusTransition), so both block until
// this transaction commits or rolls back. Requires an ambient transaction on
// ctx; fails closed otherwise, mirroring the other conditional writes on this
// adapter.
func (r *PostgresRatingDescriptionSetRepository) LockRatingDescriptionSetForUpdate(ctx context.Context, id string) (enumspb.VersionStatus, error) {
	unspecified := enumspb.VersionStatus_VERSION_STATUS_UNSPECIFIED
	if id == "" {
		return unspecified, fmt.Errorf("rating description set ID is required")
	}
	idn, ok := identity.FromContext(ctx)
	if !ok || idn == nil || idn.WorkspaceID == "" {
		return unspecified, fmt.Errorf("rating description set lock: no trusted workspace in context (fail closed)")
	}
	exec := r.executor(ctx)
	if exec == nil {
		return unspecified, fmt.Errorf("rating description set lock: no SQL executor available")
	}
	if _, isTx := exec.(*sql.Tx); !isTx {
		return unspecified, fmt.Errorf("rating description set lock: requires an ambient transaction (fail closed)")
	}
	var status, wsID string
	err := exec.QueryRowContext(ctx,
		`SELECT version_status, workspace_id FROM `+entityid.RatingDescriptionSet+` WHERE id = $1 FOR UPDATE`,
		id).Scan(&status, &wsID)
	if err != nil {
		if err == sql.ErrNoRows {
			return unspecified, fmt.Errorf("rating description set lock: not found — fail closed")
		}
		return unspecified, fmt.Errorf("rating description set lock: %w", err)
	}
	if wsID != idn.WorkspaceID {
		return unspecified, fmt.Errorf("rating description set lock: not found — fail closed")
	}
	// Parse the raw DB literal (protojson enum-name spelling) back to the
	// typed enum. An unrecognized literal fails closed as UNSPECIFIED, which
	// never equals DRAFT/PUBLISHED downstream.
	if n, ok := enumspb.VersionStatus_value[status]; ok {
		return enumspb.VersionStatus(n), nil
	}
	return unspecified, nil
}

// UpdateRatingDescriptionSetIfDraft writes a set's header fields under the
// SAME row lock/transaction as the DRAFT check (W3 follow-up finding,
// 2026-09-25): the caller must have already obtained the lock via
// LockRatingDescriptionSetForUpdate on this same txCtx and confirmed DRAFT.
// version_status, version and supersedes_id are ALWAYS stripped from the
// write payload here — never merely reset to a pre-read snapshot — so a
// concurrent Publish/Deprecate that committed status is impossible anyway
// (blocked by the row lock) and can never be clobbered by a stale value even
// if a future caller forgets to strip the fields itself.
func (r *PostgresRatingDescriptionSetRepository) UpdateRatingDescriptionSetIfDraft(ctx context.Context, req *pb.UpdateRatingDescriptionSetRequest) (*pb.RatingDescriptionSet, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("rating description set ID is required")
	}
	exec := r.executor(ctx)
	if exec == nil {
		return nil, fmt.Errorf("rating description set update: no SQL executor available")
	}
	if _, isTx := exec.(*sql.Tx); !isTx {
		return nil, fmt.Errorf("rating description set update: requires an ambient transaction (fail closed)")
	}
	data, err := protoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	delete(data, "version_status")
	delete(data, "versionStatus")
	delete(data, "version")
	delete(data, "supersedes_id")
	delete(data, "supersedesId")
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update rating description set: %w", err)
	}
	return ratingDescriptionSetFromResult(result)
}

// PublishRatingDescriptionSetIfDraft is the conditional DRAFT->PUBLISHED
// transition (schema-proposal.md §9.2): one UPDATE ... WHERE version_status =
// DRAFT (row lock), no non-transactional fallback. The caller (the use case)
// checks "≥1 entry" separately before calling this (it needs the entry
// repository, which this adapter does not depend on). Requires an ambient
// transaction on ctx (services.Transactor.ExecuteInTransaction) — fails closed
// otherwise, mirroring PostgresTaskOutcomeRepository.GuardCellWrite.
func (r *PostgresRatingDescriptionSetRepository) PublishRatingDescriptionSetIfDraft(ctx context.Context, id string) (*pb.RatingDescriptionSet, error) {
	return r.conditionalStatusTransition(ctx, id, ratingDescriptionSetVersionStatusDraft, versionStatusPublishedValue, "SET_NOT_DRAFT: rating description set is not DRAFT (or not found / foreign workspace) — publish rejected",
		entityid.EntityPermission(entityid.RatingDescriptionSet, entityid.ActionPublish), "PublishRatingDescriptionSet")
}

// DeprecateRatingDescriptionSetIfPublished is the conditional PUBLISHED->
// DEPRECATED transition. It takes the SAME set-row lock (FOR UPDATE, via the
// conditional UPDATE's row lock) as Relink's FOR SHARE re-check on the target
// set, so a concurrent relink cannot land a new link on a set that just
// became DEPRECATED (schema-proposal.md §9.2 "Linking vs deprecating").
// Existing links to a DEPRECATED set keep resolving (Q19) — this method never
// touches rating_description_set_product_plan.
func (r *PostgresRatingDescriptionSetRepository) DeprecateRatingDescriptionSetIfPublished(ctx context.Context, id string) (*pb.RatingDescriptionSet, error) {
	return r.conditionalStatusTransition(ctx, id, versionStatusPublishedValue, versionStatusDeprecatedValue, "SET_NOT_PUBLISHED: rating description set is not PUBLISHED (or not found / foreign workspace) — deprecate rejected",
		entityid.EntityPermission(entityid.RatingDescriptionSet, entityid.ActionDeprecate), "DeprecateRatingDescriptionSet")
}

// conditionalStatusTransition performs the row-locked conditional UPDATE
// shared by Publish and Deprecate, then — inside the SAME ambient transaction
// — writes the durable semantic audit event for the transition
// (writeLifecycleAudit; schema-proposal.md §9.4, codex-review-impl2.out.md
// finding 9). A failure to write the audit event returns an error from this
// method, which the calling use case's services.Transactor.ExecuteInTransaction
// propagates as a non-nil closure error — rolling back the whole transaction,
// including the conditional UPDATE above (schema-proposal.md §9.4 "audit
// failure rolls back the operation").
func (r *PostgresRatingDescriptionSetRepository) conditionalStatusTransition(ctx context.Context, id, fromStatus, toStatus, conflictMessage, permCode, useCase string) (*pb.RatingDescriptionSet, error) {
	if id == "" {
		return nil, fmt.Errorf("rating description set ID is required")
	}
	idn, ok := identity.FromContext(ctx)
	if !ok || idn == nil || idn.WorkspaceID == "" {
		return nil, fmt.Errorf("rating description set status transition: no trusted workspace in context (fail closed)")
	}
	exec := r.executor(ctx)
	if exec == nil {
		return nil, fmt.Errorf("rating description set status transition: no SQL executor available")
	}
	if _, isTx := exec.(*sql.Tx); !isTx {
		return nil, fmt.Errorf("rating description set status transition: requires an ambient transaction (fail closed)")
	}
	res, err := exec.ExecContext(ctx,
		`UPDATE `+entityid.RatingDescriptionSet+` SET version_status = $1, date_modified = $2
		 WHERE id = $3 AND workspace_id = $4 AND version_status = $5`,
		toStatus, time.Now().UnixMilli(), id, idn.WorkspaceID, fromStatus)
	if err != nil {
		return nil, fmt.Errorf("rating description set status transition: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("rating description set status transition: %w", err)
	}
	if n == 0 {
		return nil, fmt.Errorf("%s", conflictMessage)
	}
	if err := r.writeLifecycleAudit(ctx, idn.WorkspaceID, id, permCode, useCase, fromStatus, toStatus); err != nil {
		return nil, fmt.Errorf("rating description set status transition: %w", err)
	}
	result, err := r.dbOps.Read(ctx, r.tableName, id)
	if err != nil {
		return nil, fmt.Errorf("rating description set status transition: read-back failed: %w", err)
	}
	return ratingDescriptionSetFromResult(result)
}

// writeLifecycleAudit records a Publish/Deprecate status transition as a
// durable semantic audit event inside the SAME ambient transaction as the
// conditional UPDATE (schema-proposal.md §9.4; codex-review-impl2.out.md
// finding 9). Mirrors PostgresJobPhaseRepository.writeTransitionAudit
// (job_phase.go): AUDIT IS MANDATORY — a nil audit dependency fails the
// transition rather than silently skipping the durable record, and the actor
// is derived from the trusted authenticated identity (never an
// unauthenticated/empty middleware-supplied actor for a security-critical
// event).
func (r *PostgresRatingDescriptionSetRepository) writeLifecycleAudit(ctx context.Context, workspaceID, id, permCode, useCase, fromStatus, toStatus string) error {
	if r.audit == nil {
		return fmt.Errorf("rating description set %s: audit dependency is absent — refusing to transition without a durable audit event (fail closed)", useCase)
	}
	idn, ok := identity.FromContext(ctx)
	if !ok || idn == nil || idn.UserID == "" {
		return fmt.Errorf("rating description set %s: no trusted actor identity for the audit event (fail closed)", useCase)
	}
	ac, _ := infraports.GetAuditContext(ctx)
	ac.ActorID = idn.UserID
	ac.ActorType = "user"
	ctx = infraports.WithAuditContext(ctx, ac)
	return r.audit.LogEntry(ctx, &infraports.AuditLogRequest{
		WorkspaceID:    workspaceID,
		EntityType:     entityid.RatingDescriptionSet,
		EntityID:       id,
		Domain:         "espyna",
		Action:         2, // AUDIT_ACTION_UPDATE
		PermissionCode: permCode,
		UseCase:        useCase,
		MethodName:     useCase,
		FieldChanges: []infraports.AuditFieldChange{
			{FieldName: "version_status", FieldType: 1, OldValue: fromStatus, NewValue: toStatus},
			{FieldName: "id", FieldType: 1, OldValue: "", NewValue: id},
		},
	})
}

func (r *PostgresRatingDescriptionSetRepository) listPage(ctx context.Context, params *interfaces.ListParams) ([]*pb.RatingDescriptionSet, *commonpb.PaginationResponse, error) {
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list rating description sets: %w", err)
	}
	var items []*pb.RatingDescriptionSet
	for _, row := range listResult.Data {
		rj, err := json.Marshal(row)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal rating description set row: %w", err)
		}
		item := &pb.RatingDescriptionSet{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
			return nil, nil, fmt.Errorf("failed to decode rating description set row: %w", err)
		}
		items = append(items, item)
	}
	return items, listResult.Pagination, nil
}

// ratingDescriptionListParams builds the generic-List parameters for the three
// rating_description_set* adapters (fix3-backend, codex-review-impl3 #2),
// forwarding search, filters, sort AND pagination so LIMIT/OFFSET, ORDER BY
// (with the id tiebreaker) and COUNT(*) all run in SQL. Filter logic OR is
// refused (fail closed): the workspace decorator (and
// GetRatingDescriptionSetProductPlanListPageData's mandatory
// price_schedule_id scope) APPEND their predicate to the caller's filter list,
// and under OR grouping that predicate would be OR-ed away instead of AND-ed.
func ratingDescriptionListParams(search *commonpb.SearchRequest, filters *commonpb.FilterRequest, sort *commonpb.SortRequest, pagination *commonpb.PaginationRequest) (*interfaces.ListParams, error) {
	if filters != nil && filters.GetLogic() == commonpb.FilterLogic_OR && len(filters.GetFilters()) > 0 {
		return nil, fmt.Errorf("INVALID_LIST_REQUEST: OR filter logic is not supported for rating description lists (tenant scope must be AND-ed)")
	}
	if search == nil && filters == nil && sort == nil && pagination == nil {
		return nil, nil
	}
	return &interfaces.ListParams{Search: search, Filters: filters, Sort: sort, Pagination: pagination}, nil
}

func ratingDescriptionSetFromResult(result any) (*pb.RatingDescriptionSet, error) {
	rj, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	item := &pb.RatingDescriptionSet{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to proto: %w", err)
	}
	return item, nil
}

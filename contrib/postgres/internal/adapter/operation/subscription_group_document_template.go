//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"errors"
	"fmt"
	"strings"
	"time"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	sqlexec "github.com/erniealice/espyna-golang/shared/database/sqlexec"
	"github.com/erniealice/espyna-golang/shared/identity"
	documenttemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/document/template"
	jobcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_category"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/subscription_group_document_template"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	subscriptionGroupOutcomeSummaryDocumentPurpose = "subscription_group_outcome_summary"
)

// NOTE: versionStatusDraft/Published/Deprecated, nextBindingVersion,
// filterToMutableBindingFields, and pbVersionStatusValue are shared across
// operation adapters in this package.

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.SubscriptionGroupDocumentTemplate, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres subscription_group_document_template repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresSubscriptionGroupDocumentTemplateRepository(dbOps, tableName), nil
	})
}

// PostgresSubscriptionGroupDocumentTemplateRepository implements the section/group
// template-binding CRUD + the applicable resolver + publish transaction.
//
// Server-owned lifecycle (RA2 P1): DRAFT rows are born unversioned; only Publish
// moves a row to PUBLISHED and allocates a new version. Update is narrowed to
// active+date_modified only.
//
// Profile support is intentionally locked to the registered contract set.
//
// Resolver and publish run through raw SQL under workspace identity, with tenant
// equality on every join and half-open validity filtering.
type PostgresSubscriptionGroupDocumentTemplateRepository struct {
	pb.UnimplementedSubscriptionGroupDocumentTemplateDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func NewPostgresSubscriptionGroupDocumentTemplateRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.SubscriptionGroupDocumentTemplateDomainServiceServer {
	if tableName == "" {
		tableName = entityid.SubscriptionGroupDocumentTemplate
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresSubscriptionGroupDocumentTemplateRepository{dbOps: dbOps, db: db, tableName: tableName}
}

// --- CRUD -----------------------------------------------------------------

func requireWorkspaceIDFromContext(ctx context.Context) (string, error) {
	id, ok := identity.FromContext(ctx)
	if !ok || id.WorkspaceID == "" {
		return "", fmt.Errorf("workspace identity required")
	}
	return id.WorkspaceID, nil
}

func normalizedOptionalID(id string) string {
	return strings.TrimSpace(id)
}

type documentTemplateLockMode uint8

const (
	documentTemplateLockNone documentTemplateLockMode = iota
	documentTemplateLockForKeyShare
	documentTemplateLockForUpdate
)

func profileRequiresExactCategory(profile pb.RenderProfile) bool {
	return profile == pb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1
}

func profileIsSupported(profile pb.RenderProfile) bool {
	return profile == pb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1
}

func (r *PostgresSubscriptionGroupDocumentTemplateRepository) executor(ctx context.Context) sqlexec.DBExecutor {
	if ep, ok := r.dbOps.(interface {
		GetExecutor(context.Context) sqlexec.DBExecutor
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

func validateReferencedEntityWorkspaceWithExecutor(ctx context.Context, ex sqlexec.DBExecutor, tableName, entityID, workspaceID string) error {
	if ex == nil {
		return fmt.Errorf("query executor required to validate %s", tableName)
	}
	if entityID == "" {
		return nil
	}
	var found bool
	if err := ex.QueryRowContext(ctx, fmt.Sprintf(`SELECT EXISTS (SELECT 1 FROM %s WHERE id = $1 AND workspace_id = $2)`, tableName), entityID, workspaceID).Scan(&found); err != nil {
		return fmt.Errorf("failed to validate %s=%q in workspace=%q: %w", tableName, entityID, workspaceID, err)
	}
	if !found {
		return fmt.Errorf("%s=%q not found in workspace=%q", tableName, entityID, workspaceID)
	}
	return nil
}

func (r *PostgresSubscriptionGroupDocumentTemplateRepository) validateReferencedEntityWorkspace(ctx context.Context, tableName, entityID, workspaceID string) error {
	return validateReferencedEntityWorkspaceWithExecutor(ctx, r.executor(ctx), tableName, entityID, workspaceID)
}

func validateTemplateDocumentForProfileWithExecutor(ctx context.Context, ex sqlexec.DBExecutor, workspaceID, documentTemplateID string, lockMode documentTemplateLockMode) error {
	if ex == nil {
		return fmt.Errorf("query executor required to validate document_template=%q", documentTemplateID)
	}
	lockClause := ""
	switch lockMode {
	case documentTemplateLockNone:
	case documentTemplateLockForKeyShare:
		lockClause = " FOR KEY SHARE"
	case documentTemplateLockForUpdate:
		lockClause = " FOR UPDATE"
	default:
		return fmt.Errorf("unsupported document template lock mode")
	}
	var foundID string
	if err := ex.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT id
			  FROM %s
			 WHERE id = $1
			   AND workspace_id = $2
			   AND document_purpose = $3
			   AND template_type = 'docx'
			   AND active = true
			   AND status = 'active'
			   AND NULLIF(btrim(storage_container), '') IS NOT NULL
			   AND NULLIF(btrim(storage_key), '') IS NOT NULL%s`, entityid.DocumentTemplate, lockClause),
		documentTemplateID, workspaceID, subscriptionGroupOutcomeSummaryDocumentPurpose).Scan(&foundID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("document template=%q does not match required workspace/purpose/type constraints", documentTemplateID)
		}
		return fmt.Errorf("failed to validate document template=%q: %w", documentTemplateID, err)
	}
	if foundID == "" {
		return fmt.Errorf("document template=%q does not match required workspace/purpose/type constraints", documentTemplateID)
	}
	return nil
}

func validateBindingReferencesWithExecutor(ctx context.Context, ex sqlexec.DBExecutor, workspaceID string, req *pb.SubscriptionGroupDocumentTemplate) error {
	if !profileIsSupported(req.RenderProfile) {
		return fmt.Errorf("unsupported render profile: %q", req.GetRenderProfile())
	}
	if profileRequiresExactCategory(req.RenderProfile) && normalizedOptionalID(req.GetJobCategoryId()) == "" {
		return fmt.Errorf("job_category_id is required for render_profile=%s", pb.RenderProfile_name[int32(req.RenderProfile)])
	}
	if normalizedOptionalID(req.GetPriceScheduleId()) != "" {
		if err := validateReferencedEntityWorkspaceWithExecutor(ctx, ex, entityid.PriceSchedule, req.GetPriceScheduleId(), workspaceID); err != nil {
			return err
		}
	}
	if normalizedOptionalID(req.GetPlanId()) != "" {
		if err := validateReferencedEntityWorkspaceWithExecutor(ctx, ex, entityid.Plan, req.GetPlanId(), workspaceID); err != nil {
			return err
		}
	}
	if normalizedOptionalID(req.GetJobCategoryId()) != "" {
		if err := validateReferencedEntityWorkspaceWithExecutor(ctx, ex, entityid.JobCategory, req.GetJobCategoryId(), workspaceID); err != nil {
			return err
		}
	}
	if err := validateReferencedEntityWorkspaceWithExecutor(ctx, ex, entityid.DocumentTemplate, req.GetDocumentTemplateId(), workspaceID); err != nil {
		return err
	}
	if err := validateTemplateDocumentForProfileWithExecutor(ctx, ex, workspaceID, req.GetDocumentTemplateId(), documentTemplateLockForKeyShare); err != nil {
		return err
	}
	return nil
}

func (r *PostgresSubscriptionGroupDocumentTemplateRepository) CreateSubscriptionGroupDocumentTemplate(ctx context.Context, req *pb.CreateSubscriptionGroupDocumentTemplateRequest) (*pb.CreateSubscriptionGroupDocumentTemplateResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("binding data is required")
	}
	wsID, err := requireWorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if req.Data.DocumentTemplateId == "" {
		return nil, fmt.Errorf("document_template_id is required")
	}
	ex := r.executor(ctx)
	tx, ok := ex.(*sql.Tx)
	if !ok || tx == nil {
		return nil, fmt.Errorf("binding create requires an active transaction")
	}
	if err := validateBindingReferencesWithExecutor(ctx, tx, wsID, req.Data); err != nil {
		return nil, err
	}

	data, err := protoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	// hydrate-only fields never persist
	delete(data, "document_template")
	delete(data, "price_schedule")
	delete(data, "plan")
	delete(data, "job_category")
	stripClientWorkspaceKeys(data)

	// Server-owned lifecycle.
	data["version_status"] = versionStatusDraft
	data["version"] = 0
	delete(data, "published_at")
	delete(data, "published_at_string")
	delete(data, "published_by")
	delete(data, "supersedes_binding_id")
	delete(data, "date_created")
	delete(data, "date_created_string")
	delete(data, "date_modified")
	delete(data, "date_modified_string")
	data["active"] = true
	if requestIdentity, ok := identity.FromContext(ctx); ok && strings.TrimSpace(requestIdentity.UserID) != "" {
		data["created_by"] = requestIdentity.UserID
	} else {
		delete(data, "created_by")
	}

	// Empty optional FK values become NULL for FK integrity.
	if raw, ok := data["price_schedule_id"].(string); ok {
		if v := normalizedOptionalID(raw); v == "" {
			data["price_schedule_id"] = nil
		} else {
			data["price_schedule_id"] = v
		}
	}
	if raw, ok := data["plan_id"].(string); ok {
		if v := normalizedOptionalID(raw); v == "" {
			data["plan_id"] = nil
		} else {
			data["plan_id"] = v
		}
	}
	if raw, ok := data["job_category_id"].(string); ok {
		if v := normalizedOptionalID(raw); v == "" {
			data["job_category_id"] = nil
		} else {
			data["job_category_id"] = v
		}
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create binding: %w", err)
	}
	item, err := subscriptionGroupDocumentTemplateFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.CreateSubscriptionGroupDocumentTemplateResponse{Data: []*pb.SubscriptionGroupDocumentTemplate{item}, Success: true}, nil
}

func (r *PostgresSubscriptionGroupDocumentTemplateRepository) ReadSubscriptionGroupDocumentTemplate(ctx context.Context, req *pb.ReadSubscriptionGroupDocumentTemplateRequest) (*pb.ReadSubscriptionGroupDocumentTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("binding ID is required")
	}
	wsID, err := requireWorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	item, found, err := r.readByIDWithWorkspace(ctx, wsID, req.Data.Id)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("binding not found")
	}
	return &pb.ReadSubscriptionGroupDocumentTemplateResponse{Data: []*pb.SubscriptionGroupDocumentTemplate{item}, Success: true}, nil
}

func (r *PostgresSubscriptionGroupDocumentTemplateRepository) UpdateSubscriptionGroupDocumentTemplate(ctx context.Context, req *pb.UpdateSubscriptionGroupDocumentTemplateRequest) (*pb.UpdateSubscriptionGroupDocumentTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("binding ID is required")
	}
	wsID, err := requireWorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	_, found, err := r.readByIDWithWorkspace(ctx, wsID, req.Data.Id)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("binding not found")
	}
	return nil, fmt.Errorf("generic binding update is not supported; use publish or draft delete")
}

func (r *PostgresSubscriptionGroupDocumentTemplateRepository) DeleteSubscriptionGroupDocumentTemplate(ctx context.Context, req *pb.DeleteSubscriptionGroupDocumentTemplateRequest) (*pb.DeleteSubscriptionGroupDocumentTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("binding ID is required")
	}
	wsID, err := requireWorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	nowMillis := now.UnixMilli()
	ex := r.executor(ctx)
	if ex == nil {
		return nil, fmt.Errorf("delete requires direct SQL access")
	}
	res, err := ex.ExecContext(ctx,
		fmt.Sprintf(`UPDATE %s
		            SET active = false, date_modified = $1
		          WHERE id = $2 AND workspace_id = $3 AND version_status = $4`,
			entityid.SubscriptionGroupDocumentTemplate),
		nowMillis, req.Data.Id, wsID, versionStatusDraft)
	if err != nil {
		return nil, fmt.Errorf("failed to delete binding: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("delete affected-row check: %w", err)
	}
	if affected != 1 {
		return nil, fmt.Errorf("only a draft binding can be deleted")
	}
	return &pb.DeleteSubscriptionGroupDocumentTemplateResponse{Success: true}, nil
}

// DeleteDraftPair soft-deletes one DRAFT binding and its unshared artifact in
// the caller's ambient transaction. The returned locator is consumed only
// after the use case observes a successful commit.
func (r *PostgresSubscriptionGroupDocumentTemplateRepository) DeleteDraftPair(ctx context.Context, bindingID string) (*documenttemplatepb.DocumentTemplate, error) {
	bindingID = strings.TrimSpace(bindingID)
	if bindingID == "" {
		return nil, fmt.Errorf("binding ID is required")
	}
	wsID, err := requireWorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	ex := r.executor(ctx)
	tx, ok := ex.(*sql.Tx)
	if !ok || tx == nil {
		return nil, fmt.Errorf("draft-pair delete requires an active transaction")
	}

	var (
		documentTemplateID sql.NullString
		versionStatus      sql.NullString
	)
	if err := tx.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT document_template_id, version_status
		               FROM %s
		              WHERE id = $1
		                AND workspace_id = $2
		                AND active = true
		                AND version_status = $3
		              FOR UPDATE`, entityid.SubscriptionGroupDocumentTemplate),
		bindingID, wsID, versionStatusDraft,
	).Scan(&documentTemplateID, &versionStatus); errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("active binding not found")
	} else if err != nil {
		return nil, fmt.Errorf("lock draft binding: %w", err)
	}
	if !versionStatus.Valid || versionStatus.String != versionStatusDraft {
		return nil, fmt.Errorf("only an active draft binding can be deleted")
	}
	if !documentTemplateID.Valid || strings.TrimSpace(documentTemplateID.String) == "" {
		return nil, fmt.Errorf("draft binding has no document template")
	}

	var storageContainer, storageKey string
	if err := tx.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT storage_container, storage_key
		               FROM %s
		              WHERE id = $1
		                AND workspace_id = $2
		                AND active = true
		                AND status = 'active'
		                AND template_type = 'docx'
		                AND document_purpose = $3
		                AND NULLIF(btrim(storage_container), '') IS NOT NULL
		                AND NULLIF(btrim(storage_key), '') IS NOT NULL
		              FOR UPDATE`, entityid.DocumentTemplate),
		documentTemplateID.String, wsID, subscriptionGroupOutcomeSummaryDocumentPurpose,
	).Scan(&storageContainer, &storageKey); errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("active document template artifact not found")
	} else if err != nil {
		return nil, fmt.Errorf("lock document template artifact: %w", err)
	}

	var shared bool
	if err := tx.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT EXISTS (
			SELECT 1
			  FROM %s
			 WHERE workspace_id = $1
			   AND document_template_id = $2
			   AND id <> $3
			   AND active = true
		)`, entityid.SubscriptionGroupDocumentTemplate),
		wsID, documentTemplateID.String, bindingID,
	).Scan(&shared); err != nil {
		return nil, fmt.Errorf("check document template references: %w", err)
	}
	if shared {
		return nil, fmt.Errorf("document template artifact is referenced by another active binding")
	}

	now := time.Now().UTC()
	nowMillis := now.UnixMilli()
	bindingResult, err := tx.ExecContext(ctx,
		fmt.Sprintf(`UPDATE %s
		               SET active = false, date_modified = $1
		             WHERE id = $2
		               AND workspace_id = $3
		               AND document_template_id = $4
		               AND active = true
		               AND version_status = $5`, entityid.SubscriptionGroupDocumentTemplate),
		nowMillis, bindingID, wsID, documentTemplateID.String, versionStatusDraft,
	)
	if err != nil {
		return nil, fmt.Errorf("soft-delete draft binding: %w", err)
	}
	bindingAffected, err := bindingResult.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("draft binding delete affected-row check: %w", err)
	}
	if bindingAffected != 1 {
		return nil, fmt.Errorf("draft binding changed while locked")
	}

	artifactResult, err := tx.ExecContext(ctx,
		fmt.Sprintf(`UPDATE %s
		               SET active = false, date_modified = $1
		             WHERE id = $2
		               AND workspace_id = $3
		               AND active = true
		               AND status = 'active'
		               AND document_purpose = $4
		               AND storage_container = $5
		               AND storage_key = $6`, entityid.DocumentTemplate),
		now, documentTemplateID.String, wsID, subscriptionGroupOutcomeSummaryDocumentPurpose, storageContainer, storageKey,
	)
	if err != nil {
		return nil, fmt.Errorf("soft-delete document template artifact: %w", err)
	}
	artifactAffected, err := artifactResult.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("document template delete affected-row check: %w", err)
	}
	if artifactAffected != 1 {
		return nil, fmt.Errorf("document template artifact changed while locked")
	}

	workspaceID := wsID
	container := storageContainer
	key := storageKey
	return &documenttemplatepb.DocumentTemplate{
		Id:               documentTemplateID.String,
		WorkspaceId:      &workspaceID,
		TemplateType:     "docx",
		DocumentPurpose:  subscriptionGroupOutcomeSummaryDocumentPurpose,
		StorageContainer: &container,
		StorageKey:       &key,
		Status:           "active",
		Active:           false,
		DateModified:     &nowMillis,
	}, nil
}

func (r *PostgresSubscriptionGroupDocumentTemplateRepository) ListSubscriptionGroupDocumentTemplates(ctx context.Context, req *pb.ListSubscriptionGroupDocumentTemplatesRequest) (*pb.ListSubscriptionGroupDocumentTemplatesResponse, error) {
	wsID, err := requireWorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	var params *interfaces.ListParams
	if req != nil && (req.Filters != nil || req.Pagination != nil || req.Search != nil || req.Sort != nil) {
		params = &interfaces.ListParams{Filters: req.Filters, Pagination: req.Pagination, Search: req.Search, Sort: req.Sort}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list bindings: %w", err)
	}
	var items []*pb.SubscriptionGroupDocumentTemplate
	for _, row := range listResult.Data {
		rowData := row
		rawID, ok := rowData["id"].(string)
		if !ok || rawID == "" {
			continue
		}
		item, found, err := r.readByIDWithWorkspace(ctx, wsID, rawID)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		items = append(items, item)
	}
	return &pb.ListSubscriptionGroupDocumentTemplatesResponse{Data: items, Success: true}, nil
}

func subscriptionGroupDocumentTemplateFromResult(result any) (*pb.SubscriptionGroupDocumentTemplate, error) {
	m, ok := result.(map[string]any)
	if ok {
		postgresCore.ConvertMillisToRFC3339(m, "validity_start", "validity_end")
	}
	rj, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	item := &pb.SubscriptionGroupDocumentTemplate{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to proto: %w", err)
	}
	return item, nil
}

// --- Resolver ---------------------------------------------------------------

func findApplicableSubscriptionGroupDocumentTemplateSQL() string {
	return fmt.Sprintf(`
		WITH requested_scope AS (
			SELECT NULLIF($2, '') AS price_schedule_id,
			       NULLIF($3, '') AS plan_id,
			       NULLIF($4, '') AS job_category_id
			 WHERE (NULLIF($2, '') IS NULL
			        OR EXISTS (SELECT 1 FROM %s ps WHERE ps.id = NULLIF($2, '') AND ps.workspace_id = $1))
			   AND (NULLIF($3, '') IS NULL
			        OR EXISTS (SELECT 1 FROM %s pl WHERE pl.id = NULLIF($3, '') AND pl.workspace_id = $1))
			   AND (NULLIF($4, '') IS NULL
			        OR EXISTS (SELECT 1 FROM %s jc WHERE jc.id = NULLIF($4, '') AND jc.workspace_id = $1))
		)
		SELECT
			b.id, b.workspace_id, b.document_template_id, b.render_profile, b.price_schedule_id, b.plan_id, b.job_category_id, b.version,
			b.version_status, b.validity_start, b.validity_end, b.supersedes_binding_id,
			b.active, b.created_by, b.published_at, b.published_by, b.date_created, b.date_modified,
			dt.id, dt.name, dt.description, dt.active, dt.workspace_id, dt.template_type,
			dt.document_purpose, dt.storage_container, dt.storage_key, dt.original_filename,
			dt.file_size_bytes, dt.is_default, dt.created_by, dt.status, dt.module_key,
			ps.id, ps.name, ps.active, ps.workspace_id,
			pl.id, pl.name, pl.active, pl.workspace_id,
			jc.id, jc.name, jc.code, jc.active, jc.workspace_id,
			CASE
				WHEN b.plan_id = rs.plan_id
				 AND b.price_schedule_id = rs.price_schedule_id THEN 0
				WHEN b.plan_id = rs.plan_id
				 AND b.price_schedule_id IS NULL THEN 1
				WHEN b.plan_id IS NULL
				 AND b.price_schedule_id = rs.price_schedule_id THEN 2
				WHEN b.plan_id IS NULL
				 AND b.price_schedule_id IS NULL THEN 3
			END AS match_rank
		FROM %s b
		CROSS JOIN requested_scope rs
		JOIN %s dt
		  ON dt.id = b.document_template_id AND dt.workspace_id = b.workspace_id
		LEFT JOIN %s ps
		  ON ps.id = b.price_schedule_id AND ps.workspace_id = b.workspace_id
		LEFT JOIN %s pl
		  ON pl.id = b.plan_id AND pl.workspace_id = b.workspace_id
		LEFT JOIN %s jc
		  ON jc.id = b.job_category_id AND jc.workspace_id = b.workspace_id
		WHERE b.workspace_id = $1
		  AND b.active = true
		  AND b.version_status = $5
		  AND b.render_profile = $6
		  AND b.active = true
		  AND dt.active = true
		  AND dt.status = 'active'
		  AND dt.template_type = 'docx'
			  AND NULLIF(btrim(dt.storage_container), '') IS NOT NULL
			  AND NULLIF(btrim(dt.storage_key), '') IS NOT NULL
		  AND dt.document_purpose = $8
		  AND (b.validity_start IS NULL OR b.validity_start <= $7)
		  AND (b.validity_end IS NULL OR $7 < b.validity_end)
		  AND b.job_category_id IS NOT NULL
		  AND (b.job_category_id = rs.job_category_id)
		  AND (b.plan_id = rs.plan_id OR b.plan_id IS NULL)
		  AND (b.price_schedule_id = rs.price_schedule_id OR b.price_schedule_id IS NULL)
		ORDER BY match_rank, b.version DESC
		LIMIT 2`,
		entityid.PriceSchedule, entityid.Plan, entityid.JobCategory, entityid.SubscriptionGroupDocumentTemplate,
		entityid.DocumentTemplate, entityid.PriceSchedule, entityid.Plan, entityid.JobCategory)
}

func (r *PostgresSubscriptionGroupDocumentTemplateRepository) readByIDWithWorkspace(ctx context.Context, workspaceID, bindingID string) (*pb.SubscriptionGroupDocumentTemplate, bool, error) {
	ex := r.executor(ctx)
	if ex == nil {
		return nil, false, fmt.Errorf("binding resolver requires direct SQL access")
	}
	if bindingID == "" {
		return nil, false, nil
	}

	row := ex.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT
			b.id, b.workspace_id, b.document_template_id, b.render_profile, b.price_schedule_id, b.plan_id, b.job_category_id, b.version,
			b.version_status, b.validity_start, b.validity_end, b.supersedes_binding_id,
			b.active, b.created_by, b.published_at, b.published_by, b.date_created, b.date_modified,
			dt.id, dt.name, dt.description, dt.active, dt.workspace_id, dt.template_type,
			dt.document_purpose, dt.storage_container, dt.storage_key, dt.original_filename,
			dt.file_size_bytes, dt.is_default, dt.created_by, dt.status, dt.module_key,
			ps.id, ps.name, ps.active, ps.workspace_id,
			pl.id, pl.name, pl.active, pl.workspace_id,
			jc.id, jc.name, jc.code, jc.active, jc.workspace_id
		FROM %s b
		JOIN %s dt
		  ON dt.id = b.document_template_id AND dt.workspace_id = b.workspace_id
		LEFT JOIN %s ps
		  ON ps.id = b.price_schedule_id AND ps.workspace_id = b.workspace_id
		LEFT JOIN %s pl
		  ON pl.id = b.plan_id AND pl.workspace_id = b.workspace_id
		LEFT JOIN %s jc
		  ON jc.id = b.job_category_id AND jc.workspace_id = b.workspace_id
		WHERE b.id = $1
		  AND b.workspace_id = $2`,
			entityid.SubscriptionGroupDocumentTemplate, entityid.DocumentTemplate, entityid.PriceSchedule, entityid.Plan, entityid.JobCategory), bindingID, workspaceID)

	var (
		bID, bWorkspaceID, bDocTemplateID                         string
		bRenderProfile, bPriceScheduleID, bPlanID, bJobCategoryID sql.NullString
		bVersionStatus, bSupersedes, bCreatedBy                   sql.NullString
		bPublishedBy                                              sql.NullString
		bVersion                                                  sql.NullInt32
		bActive                                                   sql.NullBool
		bValidityStart, bValidityEnd                              sql.NullTime
		bPublishedAt, bDateCreated, bDateModified                 sql.NullInt64

		dtID                                                   string
		dtName, dtDescription, dtWorkspaceID, dtTemplateType   sql.NullString
		dtDocumentPurpose, dtStorageContainer, dtStorageKey    sql.NullString
		dtOriginalFilename, dtCreatedBy, dtStatus, dtModuleKey sql.NullString
		dtActive, dtIsDefault                                  sql.NullBool
		dtFileSizeBytes                                        sql.NullInt64

		psID, psName, psWorkspaceID sql.NullString
		psActive                    sql.NullBool

		plID, plName, plWorkspaceID sql.NullString
		plActive                    sql.NullBool

		jcID, jcName, jcCode, jcWorkspaceID sql.NullString
		jcActive                            sql.NullBool
	)
	if err := row.Scan(
		&bID, &bWorkspaceID, &bDocTemplateID, &bRenderProfile, &bPriceScheduleID, &bPlanID, &bJobCategoryID, &bVersion,
		&bVersionStatus, &bValidityStart, &bValidityEnd, &bSupersedes,
		&bActive, &bCreatedBy, &bPublishedAt, &bPublishedBy, &bDateCreated, &bDateModified,
		&dtID, &dtName, &dtDescription, &dtActive, &dtWorkspaceID, &dtTemplateType,
		&dtDocumentPurpose, &dtStorageContainer, &dtStorageKey, &dtOriginalFilename,
		&dtFileSizeBytes, &dtIsDefault, &dtCreatedBy, &dtStatus, &dtModuleKey,
		&psID, &psName, &psActive, &psWorkspaceID,
		&plID, &plName, &plActive, &plWorkspaceID,
		&jcID, &jcName, &jcCode, &jcActive, &jcWorkspaceID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("failed to read binding: %w", err)
	}

	binding := &pb.SubscriptionGroupDocumentTemplate{
		Id:                 bID,
		WorkspaceId:        bWorkspaceID,
		DocumentTemplateId: bDocTemplateID,
	}
	if bRenderProfile.Valid {
		if profile, ok := pb.RenderProfile_value[bRenderProfile.String]; ok {
			binding.RenderProfile = pb.RenderProfile(profile)
		}
	}
	if bVersion.Valid {
		binding.Version = bVersion.Int32
	}
	if bVersionStatus.Valid {
		if v, ok := pbVersionStatusValue(bVersionStatus.String); ok {
			binding.VersionStatus = v
		}
	}
	if bPriceScheduleID.Valid {
		binding.PriceScheduleId = &bPriceScheduleID.String
	}
	if bPlanID.Valid {
		binding.PlanId = &bPlanID.String
	}
	if bJobCategoryID.Valid {
		binding.JobCategoryId = &bJobCategoryID.String
	}
	if bValidityStart.Valid {
		binding.ValidityStart = timestamppb.New(bValidityStart.Time)
	}
	if bValidityEnd.Valid {
		binding.ValidityEnd = timestamppb.New(bValidityEnd.Time)
	}
	if bSupersedes.Valid {
		binding.SupersedesBindingId = &bSupersedes.String
	}
	if bActive.Valid {
		binding.Active = bActive.Bool
	}
	if bCreatedBy.Valid {
		binding.CreatedBy = &bCreatedBy.String
	}
	if bPublishedAt.Valid {
		binding.PublishedAt = &bPublishedAt.Int64
	}
	if bPublishedBy.Valid {
		binding.PublishedBy = &bPublishedBy.String
	}
	if bDateCreated.Valid {
		binding.DateCreated = &bDateCreated.Int64
	}
	if bDateModified.Valid {
		binding.DateModified = &bDateModified.Int64
	}

	dt := &documenttemplatepb.DocumentTemplate{Id: dtID}
	if dtActive.Valid {
		dt.Active = dtActive.Bool
	}
	if dtName.Valid {
		dt.Name = dtName.String
	}
	if dtDescription.Valid {
		dt.Description = &dtDescription.String
	}
	if dtWorkspaceID.Valid {
		dt.WorkspaceId = &dtWorkspaceID.String
	}
	if dtTemplateType.Valid {
		dt.TemplateType = dtTemplateType.String
	}
	if dtDocumentPurpose.Valid {
		dt.DocumentPurpose = dtDocumentPurpose.String
	}
	if dtStorageContainer.Valid {
		dt.StorageContainer = &dtStorageContainer.String
	}
	if dtStorageKey.Valid {
		dt.StorageKey = &dtStorageKey.String
	}
	if dtOriginalFilename.Valid {
		dt.OriginalFilename = &dtOriginalFilename.String
	}
	if dtFileSizeBytes.Valid {
		dt.FileSizeBytes = &dtFileSizeBytes.Int64
	}
	if dtIsDefault.Valid {
		dt.IsDefault = &dtIsDefault.Bool
	}
	if dtCreatedBy.Valid {
		dt.CreatedBy = &dtCreatedBy.String
	}
	if dtStatus.Valid {
		dt.Status = dtStatus.String
	}
	if dtModuleKey.Valid {
		dt.ModuleKey = &dtModuleKey.String
	}
	binding.DocumentTemplate = dt

	if psID.Valid {
		ps := &priceschedulepb.PriceSchedule{Id: psID.String}
		if psName.Valid {
			ps.Name = psName.String
		}
		if psActive.Valid {
			ps.Active = psActive.Bool
		}
		if psWorkspaceID.Valid {
			ps.WorkspaceId = &psWorkspaceID.String
		}
		binding.PriceSchedule = ps
	}

	if plID.Valid {
		pl := &planpb.Plan{Id: &plID.String}
		if plName.Valid {
			pl.Name = plName.String
		}
		if plActive.Valid {
			pl.Active = plActive.Bool
		}
		if plWorkspaceID.Valid {
			pl.WorkspaceId = &plWorkspaceID.String
		}
		binding.Plan = pl
	}

	if jcID.Valid {
		jc := &jobcategorypb.JobCategory{Id: jcID.String}
		if jcName.Valid {
			jc.Name = jcName.String
		}
		if jcCode.Valid {
			jc.Code = &jcCode.String
		}
		if jcActive.Valid {
			jc.Active = jcActive.Bool
		}
		if jcWorkspaceID.Valid {
			jc.WorkspaceId = &jcWorkspaceID.String
		}
		binding.JobCategory = jc
	}

	return binding, true, nil
}

func (r *PostgresSubscriptionGroupDocumentTemplateRepository) FindApplicableSubscriptionGroupDocumentTemplate(ctx context.Context, req *pb.FindApplicableSubscriptionGroupDocumentTemplateRequest) (*pb.FindApplicableSubscriptionGroupDocumentTemplateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	ex := r.executor(ctx)
	if ex == nil {
		return nil, fmt.Errorf("binding resolver requires direct SQL access")
	}

	wsID, err := requireWorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if !profileIsSupported(req.GetRenderProfile()) {
		return nil, fmt.Errorf("unsupported render profile: %q", req.GetRenderProfile())
	}
	if profileRequiresExactCategory(req.GetRenderProfile()) && normalizedOptionalID(req.GetJobCategoryId()) == "" {
		return nil, fmt.Errorf("job_category_id is required for render_profile=%s", pb.RenderProfile_name[int32(req.GetRenderProfile())])
	}
	if strings.TrimSpace(req.GetDocumentPurpose()) != subscriptionGroupOutcomeSummaryDocumentPurpose {
		return nil, fmt.Errorf("document_purpose must be %q", subscriptionGroupOutcomeSummaryDocumentPurpose)
	}
	priceScheduleID := normalizedOptionalID(req.GetPriceScheduleId())
	planID := normalizedOptionalID(req.GetPlanId())
	jobCategoryID := normalizedOptionalID(req.GetJobCategoryId())
	if err := r.validateReferencedEntityWorkspace(ctx, entityid.JobCategory, jobCategoryID, wsID); err != nil {
		return nil, err
	}
	if err := r.validateReferencedEntityWorkspace(ctx, entityid.Plan, planID, wsID); err != nil {
		return nil, err
	}
	if err := r.validateReferencedEntityWorkspace(ctx, entityid.PriceSchedule, priceScheduleID, wsID); err != nil {
		return nil, err
	}

	asOf := time.Now().UTC()
	if req.AsOf != nil {
		asOf = req.AsOf.AsTime().UTC()
	}

	rows, err := ex.QueryContext(
		ctx,
		findApplicableSubscriptionGroupDocumentTemplateSQL(),
		wsID, priceScheduleID, planID, jobCategoryID,
		versionStatusPublished, req.GetRenderProfile().String(), asOf, subscriptionGroupOutcomeSummaryDocumentPurpose)
	if err != nil {
		return nil, fmt.Errorf("resolver query failed: %w", err)
	}
	defer rows.Close()

	var results []scannedSubscriptionGroupTemplate
	for rows.Next() {
		item, err := r.readSubscriptionGroupTemplateRowForResolver(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("resolver rows error: %w", err)
	}
	if len(results) == 0 {
		return &pb.FindApplicableSubscriptionGroupDocumentTemplateResponse{Found: false, Success: true}, nil
	}
	if len(results) == 2 && results[0].matchRank == results[1].matchRank {
		return nil, fmt.Errorf("ambiguous applicable binding: two equal-ranked bindings resolved")
	}

	return &pb.FindApplicableSubscriptionGroupDocumentTemplateResponse{Binding: results[0].binding, Found: true, Success: true}, nil
}

func (r *PostgresSubscriptionGroupDocumentTemplateRepository) readSubscriptionGroupTemplateRowForResolver(rows *sql.Rows) (scanned scannedSubscriptionGroupTemplate, err error) {
	var (
		bID, bWorkspaceID, bDocTemplateID                         string
		bRenderProfile, bPriceScheduleID, bPlanID, bJobCategoryID sql.NullString
		bVersionStatus, bSupersedes, bCreatedBy                   sql.NullString
		bPublishedBy                                              sql.NullString
		bVersion                                                  sql.NullInt32
		bActive                                                   sql.NullBool
		bValidityStart, bValidityEnd                              sql.NullTime
		bPublishedAt, bDateCreated, bDateModified                 sql.NullInt64

		dtID                                                   string
		dtName, dtDescription, dtWorkspaceID, dtTemplateType   sql.NullString
		dtDocumentPurpose, dtStorageContainer, dtStorageKey    sql.NullString
		dtOriginalFilename, dtCreatedBy, dtStatus, dtModuleKey sql.NullString
		dtActive, dtIsDefault                                  sql.NullBool
		dtFileSizeBytes                                        sql.NullInt64

		psID, psName, psWorkspaceID         sql.NullString
		psActive                            sql.NullBool
		plID, plName, plWorkspaceID         sql.NullString
		plActive                            sql.NullBool
		jcID, jcName, jcCode, jcWorkspaceID sql.NullString
		jcActive                            sql.NullBool
		matchRank                           int
	)
	if err = rows.Scan(
		&bID, &bWorkspaceID, &bDocTemplateID, &bRenderProfile, &bPriceScheduleID, &bPlanID, &bJobCategoryID, &bVersion,
		&bVersionStatus, &bValidityStart, &bValidityEnd, &bSupersedes,
		&bActive, &bCreatedBy, &bPublishedAt, &bPublishedBy, &bDateCreated, &bDateModified,
		&dtID, &dtName, &dtDescription, &dtActive, &dtWorkspaceID, &dtTemplateType,
		&dtDocumentPurpose, &dtStorageContainer, &dtStorageKey, &dtOriginalFilename,
		&dtFileSizeBytes, &dtIsDefault, &dtCreatedBy, &dtStatus, &dtModuleKey,
		&psID, &psName, &psActive, &psWorkspaceID,
		&plID, &plName, &plActive, &plWorkspaceID,
		&jcID, &jcName, &jcCode, &jcActive, &jcWorkspaceID,
		&matchRank,
	); err != nil {
		return scannedSubscriptionGroupTemplate{}, fmt.Errorf("resolver scan failed: %w", err)
	}

	binding := &pb.SubscriptionGroupDocumentTemplate{
		Id:                 bID,
		WorkspaceId:        bWorkspaceID,
		DocumentTemplateId: bDocTemplateID,
	}
	if bVersion.Valid {
		binding.Version = bVersion.Int32
	}
	if bRenderProfile.Valid {
		if value, ok := pb.RenderProfile_value[bRenderProfile.String]; ok {
			binding.RenderProfile = pb.RenderProfile(value)
		}
	}
	if bVersionStatus.Valid {
		if v, ok := pbVersionStatusValue(bVersionStatus.String); ok {
			binding.VersionStatus = v
		}
	}
	if bPriceScheduleID.Valid {
		binding.PriceScheduleId = &bPriceScheduleID.String
	}
	if bPlanID.Valid {
		binding.PlanId = &bPlanID.String
	}
	if bJobCategoryID.Valid {
		binding.JobCategoryId = &bJobCategoryID.String
	}
	if bValidityStart.Valid {
		binding.ValidityStart = timestamppb.New(bValidityStart.Time)
	}
	if bValidityEnd.Valid {
		binding.ValidityEnd = timestamppb.New(bValidityEnd.Time)
	}
	if bSupersedes.Valid {
		binding.SupersedesBindingId = &bSupersedes.String
	}
	if bActive.Valid {
		binding.Active = bActive.Bool
	}
	if bCreatedBy.Valid {
		binding.CreatedBy = &bCreatedBy.String
	}
	if bPublishedAt.Valid {
		binding.PublishedAt = &bPublishedAt.Int64
	}
	if bPublishedBy.Valid {
		binding.PublishedBy = &bPublishedBy.String
	}
	if bDateCreated.Valid {
		binding.DateCreated = &bDateCreated.Int64
	}
	if bDateModified.Valid {
		binding.DateModified = &bDateModified.Int64
	}

	dt := &documenttemplatepb.DocumentTemplate{Id: dtID}
	if dtActive.Valid {
		dt.Active = dtActive.Bool
	}
	if dtName.Valid {
		dt.Name = dtName.String
	}
	if dtDescription.Valid {
		dt.Description = &dtDescription.String
	}
	if dtWorkspaceID.Valid {
		dt.WorkspaceId = &dtWorkspaceID.String
	}
	if dtTemplateType.Valid {
		dt.TemplateType = dtTemplateType.String
	}
	if dtDocumentPurpose.Valid {
		dt.DocumentPurpose = dtDocumentPurpose.String
	}
	if dtStorageContainer.Valid {
		dt.StorageContainer = &dtStorageContainer.String
	}
	if dtStorageKey.Valid {
		dt.StorageKey = &dtStorageKey.String
	}
	if dtOriginalFilename.Valid {
		dt.OriginalFilename = &dtOriginalFilename.String
	}
	if dtFileSizeBytes.Valid {
		dt.FileSizeBytes = &dtFileSizeBytes.Int64
	}
	if dtIsDefault.Valid {
		dt.IsDefault = &dtIsDefault.Bool
	}
	if dtCreatedBy.Valid {
		dt.CreatedBy = &dtCreatedBy.String
	}
	if dtStatus.Valid {
		dt.Status = dtStatus.String
	}
	if dtModuleKey.Valid {
		dt.ModuleKey = &dtModuleKey.String
	}
	binding.DocumentTemplate = dt

	if psID.Valid {
		ps := &priceschedulepb.PriceSchedule{Id: psID.String}
		if psName.Valid {
			ps.Name = psName.String
		}
		if psActive.Valid {
			ps.Active = psActive.Bool
		}
		if psWorkspaceID.Valid {
			ps.WorkspaceId = &psWorkspaceID.String
		}
		binding.PriceSchedule = ps
	}
	if plID.Valid {
		pl := &planpb.Plan{Id: &plID.String}
		if plName.Valid {
			pl.Name = plName.String
		}
		if plActive.Valid {
			pl.Active = plActive.Bool
		}
		if plWorkspaceID.Valid {
			pl.WorkspaceId = &plWorkspaceID.String
		}
		binding.Plan = pl
	}
	if jcID.Valid {
		jc := &jobcategorypb.JobCategory{Id: jcID.String}
		if jcName.Valid {
			jc.Name = jcName.String
		}
		if jcCode.Valid {
			jc.Code = &jcCode.String
		}
		if jcActive.Valid {
			jc.Active = jcActive.Bool
		}
		if jcWorkspaceID.Valid {
			jc.WorkspaceId = &jcWorkspaceID.String
		}
		binding.JobCategory = jc
	}

	return scannedSubscriptionGroupTemplate{binding: binding, matchRank: matchRank, version: binding.Version}, nil
}

type scannedSubscriptionGroupTemplate struct {
	binding   *pb.SubscriptionGroupDocumentTemplate
	matchRank int
	version   int32
}

// --- Publish ----------------------------------------------------------------

func subscriptionGroupDocumentTemplatePublishFlipSQL() string {
	return fmt.Sprintf(`UPDATE %s
			  SET version_status = $1, version = $2, published_at = $3, published_by = $4,
			      date_modified = $3, validity_start = COALESCE(validity_start, $5),
			      supersedes_binding_id = $6
			WHERE id = $7 AND workspace_id = $8 AND version_status = $9`,
		entityid.SubscriptionGroupDocumentTemplate)
}

func (r *PostgresSubscriptionGroupDocumentTemplateRepository) PublishSubscriptionGroupDocumentTemplate(ctx context.Context, req *pb.PublishSubscriptionGroupDocumentTemplateRequest) (*pb.PublishSubscriptionGroupDocumentTemplateResponse, error) {
	if req == nil || req.Id == "" {
		return nil, fmt.Errorf("binding ID is required")
	}
	wsID, err := requireWorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	ex := r.executor(ctx)
	tx, ok := ex.(*sql.Tx)
	if !ok || tx == nil {
		return nil, fmt.Errorf("publish requires an active transaction")
	}
	publishedBy := ""
	if id, ok := identity.FromContext(ctx); ok {
		publishedBy = strings.TrimSpace(id.UserID)
	}
	if publishedBy == "" {
		return nil, fmt.Errorf("trusted publishing user identity is required")
	}
	nowMillis := time.Now().UTC().UnixMilli()
	tbl := entityid.SubscriptionGroupDocumentTemplate

	var (
		targetDocTemplateID                                      sql.NullString
		targetRenderProfile, targetVersionStatus                 sql.NullString
		targetPriceScheduleID, targetPlanID, targetJobCategoryID sql.NullString
		targetValidityStart, targetValidityEnd                   sql.NullTime
		targetActive                                             sql.NullBool
	)
	if err = tx.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT document_template_id, render_profile, price_schedule_id, plan_id, job_category_id,
		                      validity_start, validity_end, version_status, active
		               FROM %s
		              WHERE id = $1 AND workspace_id = $2
		              FOR UPDATE`, tbl), req.Id, wsID).Scan(
		&targetDocTemplateID, &targetRenderProfile, &targetPriceScheduleID, &targetPlanID, &targetJobCategoryID,
		&targetValidityStart, &targetValidityEnd, &targetVersionStatus, &targetActive,
	); err == sql.ErrNoRows {
		return nil, fmt.Errorf("binding not found")
	} else if err != nil {
		return nil, fmt.Errorf("load target: %w", err)
	}
	if !targetActive.Valid || !targetActive.Bool {
		return nil, fmt.Errorf("only an active draft binding can be published")
	}
	if !targetVersionStatus.Valid || targetVersionStatus.String != versionStatusDraft {
		return nil, fmt.Errorf("only a draft binding can be published (current status: %q)", targetVersionStatus.String)
	}
	if !targetRenderProfile.Valid {
		return nil, fmt.Errorf("target render_profile is required")
	}
	targetRenderProfileValue, ok := pb.RenderProfile_value[targetRenderProfile.String]
	if !ok {
		return nil, fmt.Errorf("unsupported render profile: %q", targetRenderProfile.String)
	}
	targetProfile := pb.RenderProfile(targetRenderProfileValue)
	if !profileIsSupported(targetProfile) {
		return nil, fmt.Errorf("unsupported render profile: %q", targetRenderProfile.String)
	}
	if profileRequiresExactCategory(targetProfile) && (!targetJobCategoryID.Valid || targetJobCategoryID.String == "") {
		return nil, fmt.Errorf("target job_category_id is required for render_profile=%s", targetRenderProfile.String)
	}
	if !targetDocTemplateID.Valid || strings.TrimSpace(targetDocTemplateID.String) == "" {
		return nil, fmt.Errorf("target document_template_id is required")
	}
	if err := validateReferencedEntityWorkspaceWithExecutor(ctx, tx, entityid.DocumentTemplate, targetDocTemplateID.String, wsID); err != nil {
		return nil, err
	}
	if err := validateTemplateDocumentForProfileWithExecutor(ctx, tx, wsID, targetDocTemplateID.String, documentTemplateLockForUpdate); err != nil {
		return nil, err
	}
	if err := validateReferencedEntityWorkspaceWithExecutor(ctx, tx, entityid.JobCategory, targetJobCategoryID.String, wsID); err != nil {
		return nil, err
	}
	if err := validateReferencedEntityWorkspaceWithExecutor(ctx, tx, entityid.Plan, targetPlanID.String, wsID); err != nil {
		return nil, err
	}
	if err := validateReferencedEntityWorkspaceWithExecutor(ctx, tx, entityid.PriceSchedule, targetPriceScheduleID.String, wsID); err != nil {
		return nil, err
	}

	bucketLockKey := strings.Join([]string{
		wsID,
		targetRenderProfile.String,
		targetPriceScheduleID.String,
		targetPlanID.String,
		targetJobCategoryID.String,
	}, "\x1f")
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1::text, 0))`, bucketLockKey); err != nil {
		return nil, fmt.Errorf("lock publication bucket: %w", err)
	}

	closeAt := targetValidityStart
	if !closeAt.Valid {
		closeAt = sql.NullTime{Time: time.Now().UTC(), Valid: true}
	}

	var laterPublishedSiblingID string
	err = tx.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT id
		               FROM %s
		              WHERE workspace_id = $1
		                AND render_profile = $2
		                AND COALESCE(price_schedule_id, '') = COALESCE($3, '')
		                AND COALESCE(plan_id, '') = COALESCE($4, '')
		                AND COALESCE(job_category_id, '') = COALESCE($5, '')
		                AND version_status = $6
		                AND active = true
		                AND id <> $7
		                AND validity_start > $8
		                AND ($9::timestamptz IS NULL OR validity_start < $9)
		              ORDER BY validity_start, id
		              LIMIT 1
		              FOR UPDATE`, tbl),
		wsID, targetRenderProfile.String,
		subscriptionGroupNullStringValue(targetPriceScheduleID),
		subscriptionGroupNullStringValue(targetPlanID),
		subscriptionGroupNullStringValue(targetJobCategoryID),
		versionStatusPublished, req.Id, closeAt.Time, targetValidityEnd).Scan(&laterPublishedSiblingID)
	if err == nil {
		return nil, fmt.Errorf("publication interval overlaps a later published sibling")
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("lock later published sibling: %w", err)
	}

	var maxVersion sql.NullInt32
	if err = tx.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT MAX(version)
	               FROM %s
	              WHERE workspace_id = $1
	                AND render_profile = $2
	                AND COALESCE(price_schedule_id, '') = COALESCE($3, '')
	                AND COALESCE(plan_id, '') = COALESCE($4, '')
	                AND COALESCE(job_category_id, '') = COALESCE($5, '')
	                AND version_status = $6`, tbl),
		wsID, targetRenderProfile.String,
		subscriptionGroupNullStringValue(targetPriceScheduleID),
		subscriptionGroupNullStringValue(targetPlanID),
		subscriptionGroupNullStringValue(targetJobCategoryID),
		versionStatusPublished).Scan(&maxVersion); err != nil {
		return nil, fmt.Errorf("compute next version: %w", err)
	}
	newVersion := nextBindingVersion(maxVersion)

	predecessorRows, err := tx.QueryContext(ctx,
		fmt.Sprintf(`SELECT id
			             FROM %s
			            WHERE workspace_id = $1
			              AND render_profile = $2
			              AND COALESCE(price_schedule_id, '') = COALESCE($3, '')
			              AND COALESCE(plan_id, '') = COALESCE($4, '')
			              AND COALESCE(job_category_id, '') = COALESCE($5, '')
			              AND version_status = $6
			              AND active = true
			              AND id <> $7
			              AND (validity_start IS NULL OR validity_start <= $8)
			              AND (validity_end IS NULL OR $8 < validity_end)
			            ORDER BY version DESC, date_created DESC NULLS LAST, id DESC
			            LIMIT 2
			            FOR UPDATE`, tbl),
		wsID, targetRenderProfile.String,
		subscriptionGroupNullStringValue(targetPriceScheduleID),
		subscriptionGroupNullStringValue(targetPlanID),
		subscriptionGroupNullStringValue(targetJobCategoryID),
		versionStatusPublished, req.Id, closeAt.Time)
	if err != nil {
		return nil, fmt.Errorf("lock predecessor: %w", err)
	}
	var predecessorIDs []string
	for predecessorRows.Next() {
		var predecessorID string
		if err := predecessorRows.Scan(&predecessorID); err != nil {
			predecessorRows.Close()
			return nil, fmt.Errorf("scan predecessor: %w", err)
		}
		predecessorIDs = append(predecessorIDs, predecessorID)
	}
	if err := predecessorRows.Err(); err != nil {
		predecessorRows.Close()
		return nil, fmt.Errorf("iterate predecessors: %w", err)
	}
	predecessorRows.Close()
	if len(predecessorIDs) > 1 {
		return nil, fmt.Errorf("ambiguous publication bucket: %d published predecessors overlap the target boundary", len(predecessorIDs))
	}

	var supersedes sql.NullString
	if len(predecessorIDs) == 1 {
		supersedes = sql.NullString{String: predecessorIDs[0], Valid: true}
		res, err := tx.ExecContext(ctx,
			fmt.Sprintf(`UPDATE %s
				           SET validity_end = $1, date_modified = $2
				         WHERE id = $3
				           AND workspace_id = $4
				           AND version_status = $5
				           AND active = true
				           AND (validity_start IS NULL OR validity_start <= $1)
				           AND (validity_end IS NULL OR $1 < validity_end)`, tbl),
			closeAt.Time, nowMillis, predecessorIDs[0], wsID, versionStatusPublished)
		if err != nil {
			return nil, fmt.Errorf("close prior sibling: %w", err)
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("close prior sibling affected-row check: %w", err)
		}
		if affected != 1 {
			return nil, fmt.Errorf("published predecessor changed while locked")
		}
	}

	res, err := tx.ExecContext(ctx, subscriptionGroupDocumentTemplatePublishFlipSQL(),
		versionStatusPublished, newVersion, nowMillis, publishedBy, closeAt.Time, supersedes, req.Id, wsID, versionStatusDraft)
	if err != nil {
		return nil, fmt.Errorf("publish target: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("publish affected-row check: %w", err)
	}
	if affected != 1 {
		return nil, fmt.Errorf("publish did not apply: binding is no longer a draft (concurrent publish?)")
	}

	item, found, err := r.readByIDWithWorkspace(ctx, wsID, req.Id)
	if err != nil {
		return nil, fmt.Errorf("re-read published binding: %w", err)
	}
	if !found {
		return nil, fmt.Errorf("binding not found after publish")
	}
	return &pb.PublishSubscriptionGroupDocumentTemplateResponse{Data: item, Success: true}, nil
}

// subscriptionGroupNullStringValue converts a nullable ID to a query parameter
// value that preserves the same bucket semantics as COALESCE(..., ”).
func subscriptionGroupNullStringValue(v sql.NullString) any {
	if v.Valid {
		return v.String
	}
	return nil
}

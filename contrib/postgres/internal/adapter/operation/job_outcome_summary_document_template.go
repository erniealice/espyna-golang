//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/shared/identity"
	documenttemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/document/template"
	enums "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary_document_template"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// versionStatusPublished is the persisted (protojson) name of the operation
// VersionStatus PUBLISHED value. The resolver gates on it; CRUD round-trips
// version_status via protojson's native enum-name serialization.
const versionStatusPublished = "VERSION_STATUS_PUBLISHED"

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.JobOutcomeSummaryDocumentTemplate, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres job_outcome_summary_document_template repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresJobOutcomeSummaryDocumentTemplateRepository(dbOps, tableName), nil
	})
}

// PostgresJobOutcomeSummaryDocumentTemplateRepository implements the report-card
// template-binding CRUD + the applicability resolver + the publish transaction.
// The resolver and publish paths use the raw *sql.DB (CTE + multi-statement TX);
// CRUD delegates to the workspace-aware dbOps decorator.
type PostgresJobOutcomeSummaryDocumentTemplateRepository struct {
	pb.UnimplementedJobOutcomeSummaryDocumentTemplateDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func NewPostgresJobOutcomeSummaryDocumentTemplateRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.JobOutcomeSummaryDocumentTemplateDomainServiceServer {
	if tableName == "" {
		tableName = entityid.JobOutcomeSummaryDocumentTemplate
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresJobOutcomeSummaryDocumentTemplateRepository{dbOps: dbOps, db: db, tableName: tableName}
}

// --- CRUD -----------------------------------------------------------------

func (r *PostgresJobOutcomeSummaryDocumentTemplateRepository) CreateJobOutcomeSummaryDocumentTemplate(ctx context.Context, req *pb.CreateJobOutcomeSummaryDocumentTemplateRequest) (*pb.CreateJobOutcomeSummaryDocumentTemplateResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("binding data is required")
	}
	data, err := protoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	// hydrate-only fields never persist.
	delete(data, "documentTemplate")
	delete(data, "priceSchedule")
	// empty optional FK ("" from a form) → SQL NULL so the FK constraint holds.
	if v, ok := data["priceScheduleId"].(string); ok && v == "" {
		data["priceScheduleId"] = nil
	}
	if v, ok := data["supersedesBindingId"].(string); ok && v == "" {
		data["supersedesBindingId"] = nil
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create binding: %w", err)
	}
	item, err := jobOutcomeSummaryDocumentTemplateFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.CreateJobOutcomeSummaryDocumentTemplateResponse{Data: []*pb.JobOutcomeSummaryDocumentTemplate{item}, Success: true}, nil
}

func (r *PostgresJobOutcomeSummaryDocumentTemplateRepository) ReadJobOutcomeSummaryDocumentTemplate(ctx context.Context, req *pb.ReadJobOutcomeSummaryDocumentTemplateRequest) (*pb.ReadJobOutcomeSummaryDocumentTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("binding ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read binding: %w", err)
	}
	item, err := jobOutcomeSummaryDocumentTemplateFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.ReadJobOutcomeSummaryDocumentTemplateResponse{Data: []*pb.JobOutcomeSummaryDocumentTemplate{item}, Success: true}, nil
}

func (r *PostgresJobOutcomeSummaryDocumentTemplateRepository) UpdateJobOutcomeSummaryDocumentTemplate(ctx context.Context, req *pb.UpdateJobOutcomeSummaryDocumentTemplateRequest) (*pb.UpdateJobOutcomeSummaryDocumentTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("binding ID is required")
	}
	data, err := protoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	delete(data, "documentTemplate")
	delete(data, "priceSchedule")
	if v, ok := data["priceScheduleId"].(string); ok && v == "" {
		data["priceScheduleId"] = nil
	}
	if v, ok := data["supersedesBindingId"].(string); ok && v == "" {
		data["supersedesBindingId"] = nil
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update binding: %w", err)
	}
	item, err := jobOutcomeSummaryDocumentTemplateFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateJobOutcomeSummaryDocumentTemplateResponse{Data: []*pb.JobOutcomeSummaryDocumentTemplate{item}, Success: true}, nil
}

func (r *PostgresJobOutcomeSummaryDocumentTemplateRepository) DeleteJobOutcomeSummaryDocumentTemplate(ctx context.Context, req *pb.DeleteJobOutcomeSummaryDocumentTemplateRequest) (*pb.DeleteJobOutcomeSummaryDocumentTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("binding ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete binding: %w", err)
	}
	return &pb.DeleteJobOutcomeSummaryDocumentTemplateResponse{Success: true}, nil
}

func (r *PostgresJobOutcomeSummaryDocumentTemplateRepository) ListJobOutcomeSummaryDocumentTemplates(ctx context.Context, req *pb.ListJobOutcomeSummaryDocumentTemplatesRequest) (*pb.ListJobOutcomeSummaryDocumentTemplatesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && (req.Filters != nil || req.Pagination != nil) {
		params = &interfaces.ListParams{Filters: req.Filters, Pagination: req.Pagination}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list bindings: %w", err)
	}
	var items []*pb.JobOutcomeSummaryDocumentTemplate
	for _, row := range listResult.Data {
		item, err := jobOutcomeSummaryDocumentTemplateFromResult(row)
		if err != nil {
			continue
		}
		items = append(items, item)
	}
	return &pb.ListJobOutcomeSummaryDocumentTemplatesResponse{Data: items, Success: true}, nil
}

func jobOutcomeSummaryDocumentTemplateFromResult(result any) (*pb.JobOutcomeSummaryDocumentTemplate, error) {
	m, ok := result.(map[string]any)
	if ok {
		// validity_* are timestamptz → normalizeValue returns int64 millis;
		// convert back to RFC3339 so protojson can decode the Timestamp fields.
		postgresCore.ConvertMillisToRFC3339(m, "validity_start", "validity_end")
	}
	rj, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	item := &pb.JobOutcomeSummaryDocumentTemplate{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to proto: %w", err)
	}
	return item, nil
}

// --- Resolver -------------------------------------------------------------

// FindApplicableJobOutcomeSummaryDocumentTemplate resolves the single applicable,
// published, active binding for (price_schedule_id, as_of). Tenant isolation is
// enforced IN THE SQL PREDICATE (workspace_id = $1, sourced from trusted context)
// — never from the request. Most-specific-wins: an AY-bound binding outranks the
// workspace-wide (price_schedule_id IS NULL) fallback; newest version breaks
// ties. A cross-workspace price_schedule yields no result (never a fallback). Two
// equal-ranked, equal-version rows fail closed (LIMIT 2 ambiguity guard). No
// match → found=false, success=true.
func (r *PostgresJobOutcomeSummaryDocumentTemplateRepository) FindApplicableJobOutcomeSummaryDocumentTemplate(ctx context.Context, req *pb.FindApplicableJobOutcomeSummaryDocumentTemplateRequest) (*pb.FindApplicableJobOutcomeSummaryDocumentTemplateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	if r.db == nil {
		return nil, fmt.Errorf("binding resolver requires direct *sql.DB access")
	}

	// Tenant isolation: scope to the caller's workspace from trusted context.
	// An absent/empty workspace resolves nothing (fail-closed).
	id, ok := identity.FromContext(ctx)
	if !ok || id.WorkspaceID == "" {
		return &pb.FindApplicableJobOutcomeSummaryDocumentTemplateResponse{Found: false, Success: true}, nil
	}
	wsID := id.WorkspaceID

	asOf := time.Now().UTC()
	if req.AsOf != nil {
		asOf = req.AsOf.AsTime().UTC()
	}

	query := `
		WITH requested_scope AS (
			SELECT NULLIF($2, '') AS price_schedule_id
			WHERE NULLIF($2, '') IS NULL
			   OR EXISTS (SELECT 1 FROM price_schedule ps
			              WHERE ps.id = NULLIF($2, '') AND ps.workspace_id = $1)
		)
		SELECT
			b.id, b.workspace_id, b.document_template_id, b.price_schedule_id, b.version,
			b.version_status, b.validity_start, b.validity_end, b.supersedes_binding_id,
			b.active, b.created_by, b.published_at, b.published_by, b.date_created, b.date_modified,
			dt.id, dt.name, dt.description, dt.active, dt.workspace_id, dt.template_type,
			dt.document_purpose, dt.storage_container, dt.storage_key, dt.original_filename,
			dt.file_size_bytes, dt.is_default, dt.created_by, dt.status, dt.module_key,
			ps.id, ps.name, ps.active, ps.workspace_id,
			CASE WHEN rs.price_schedule_id IS NOT NULL AND b.price_schedule_id = rs.price_schedule_id THEN 0 ELSE 1 END AS match_rank
		FROM job_outcome_summary_document_template b
		CROSS JOIN requested_scope rs
		JOIN document_template dt
			ON dt.id = b.document_template_id AND dt.workspace_id = b.workspace_id
		LEFT JOIN price_schedule ps
			ON ps.id = b.price_schedule_id AND ps.workspace_id = b.workspace_id
		WHERE b.workspace_id = $1
			AND b.active = true
			AND b.version_status = $3
			AND dt.active = true
			AND dt.status = 'active'
			AND dt.template_type = 'docx'
			AND NULLIF(dt.storage_key, '') IS NOT NULL
			AND (b.validity_start IS NULL OR b.validity_start <= $4)
			AND (b.validity_end IS NULL OR $4 < b.validity_end)
			AND (b.price_schedule_id = rs.price_schedule_id OR b.price_schedule_id IS NULL)
		ORDER BY match_rank, b.version DESC
		LIMIT 2`

	rows, err := r.db.QueryContext(ctx, query, wsID, req.GetPriceScheduleId(), versionStatusPublished, asOf)
	if err != nil {
		return nil, fmt.Errorf("resolver query failed: %w", err)
	}
	defer rows.Close()

	type scanned struct {
		binding   *pb.JobOutcomeSummaryDocumentTemplate
		matchRank int
		version   int32
	}
	var results []scanned
	for rows.Next() {
		var (
			bID, bWorkspaceID, bDocTmplID                            string
			bPriceScheduleID, bVersionStatus, bSupersedes, bCreatedBy sql.NullString
			bPublishedBy                                             sql.NullString
			bVersion                                                 sql.NullInt32
			bActive                                                  sql.NullBool
			bValidityStart, bValidityEnd                             sql.NullTime
			bPublishedAt, bDateCreated, bDateModified                sql.NullInt64

			dtID                                                     string
			dtName, dtDescription, dtWorkspaceID, dtTemplateType     sql.NullString
			dtDocumentPurpose, dtStorageContainer, dtStorageKey      sql.NullString
			dtOriginalFilename, dtCreatedBy, dtStatus, dtModuleKey   sql.NullString
			dtActive, dtIsDefault                                    sql.NullBool
			dtFileSizeBytes                                          sql.NullInt64

			psID, psName, psWorkspaceID sql.NullString
			psActive                    sql.NullBool

			matchRank int
		)
		if err := rows.Scan(
			&bID, &bWorkspaceID, &bDocTmplID, &bPriceScheduleID, &bVersion,
			&bVersionStatus, &bValidityStart, &bValidityEnd, &bSupersedes,
			&bActive, &bCreatedBy, &bPublishedAt, &bPublishedBy, &bDateCreated, &bDateModified,
			&dtID, &dtName, &dtDescription, &dtActive, &dtWorkspaceID, &dtTemplateType,
			&dtDocumentPurpose, &dtStorageContainer, &dtStorageKey, &dtOriginalFilename,
			&dtFileSizeBytes, &dtIsDefault, &dtCreatedBy, &dtStatus, &dtModuleKey,
			&psID, &psName, &psActive, &psWorkspaceID,
			&matchRank,
		); err != nil {
			return nil, fmt.Errorf("resolver scan failed: %w", err)
		}

		binding := &pb.JobOutcomeSummaryDocumentTemplate{
			Id:                 bID,
			WorkspaceId:        bWorkspaceID,
			DocumentTemplateId: bDocTmplID,
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

		// Hydrate the bound DocumentTemplate (storage locator drives the download).
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

		// Hydrate PriceSchedule status-agnostically (archived AYs still render).
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

		results = append(results, scanned{binding: binding, matchRank: matchRank, version: binding.Version})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("resolver rows error: %w", err)
	}

	if len(results) == 0 {
		return &pb.FindApplicableJobOutcomeSummaryDocumentTemplateResponse{Found: false, Success: true}, nil
	}
	// Ambiguity guard: two equal-ranked, equal-version rows must not be a
	// coin-flip. (The null-safe unique index makes this structurally impossible;
	// this is defense-in-depth.)
	if len(results) == 2 && results[0].matchRank == results[1].matchRank && results[0].version == results[1].version {
		return nil, fmt.Errorf("ambiguous applicable binding: two equal-ranked versions resolved")
	}

	return &pb.FindApplicableJobOutcomeSummaryDocumentTemplateResponse{
		Binding: results[0].binding,
		Found:   true,
		Success: true,
	}, nil
}

// --- Publish --------------------------------------------------------------

// PublishJobOutcomeSummaryDocumentTemplate flips a DRAFT binding to PUBLISHED and
// closes the prior published sibling's validity_end in ONE transaction
// (publish-flips-sibling). The prior sibling stays PUBLISHED so historical as_of
// resolution still finds it. version is re-allocated MAX(version)+1 in the
// lineage under the transaction. Workspace comes from trusted context.
func (r *PostgresJobOutcomeSummaryDocumentTemplateRepository) PublishJobOutcomeSummaryDocumentTemplate(ctx context.Context, req *pb.PublishJobOutcomeSummaryDocumentTemplateRequest) (*pb.PublishJobOutcomeSummaryDocumentTemplateResponse, error) {
	if req == nil || req.Id == "" {
		return nil, fmt.Errorf("binding ID is required")
	}
	if r.db == nil {
		return nil, fmt.Errorf("publish requires direct *sql.DB access")
	}
	id, ok := identity.FromContext(ctx)
	if !ok || id.WorkspaceID == "" {
		return nil, fmt.Errorf("workspace identity required to publish")
	}
	wsID := id.WorkspaceID
	publishedBy := id.UserID
	nowMillis := time.Now().UTC().UnixMilli()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Load the target (workspace-scoped).
	var (
		targetPriceScheduleID sql.NullString
		targetValidityStart   sql.NullTime
	)
	err = tx.QueryRowContext(ctx,
		`SELECT price_schedule_id, validity_start
		   FROM job_outcome_summary_document_template
		  WHERE id = $1 AND workspace_id = $2`, req.Id, wsID).
		Scan(&targetPriceScheduleID, &targetValidityStart)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("binding not found")
	}
	if err != nil {
		return nil, fmt.Errorf("load target: %w", err)
	}

	// Allocate the next version in the lineage (workspace + price_schedule bucket),
	// excluding the target itself.
	var maxVersion sql.NullInt32
	err = tx.QueryRowContext(ctx,
		`SELECT MAX(version)
		   FROM job_outcome_summary_document_template
		  WHERE workspace_id = $1
		    AND COALESCE(price_schedule_id, '') = COALESCE($2, '')
		    AND id <> $3`, wsID, targetPriceScheduleID, req.Id).
		Scan(&maxVersion)
	if err != nil {
		return nil, fmt.Errorf("compute next version: %w", err)
	}
	newVersion := int32(1)
	if maxVersion.Valid {
		newVersion = maxVersion.Int32 + 1
	}

	// Close the prior published sibling (leave it PUBLISHED for historical as_of).
	closeAt := targetValidityStart
	if !closeAt.Valid {
		closeAt = sql.NullTime{Time: time.Now().UTC(), Valid: true}
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE job_outcome_summary_document_template
		    SET validity_end = $1, date_modified = $2
		  WHERE workspace_id = $3
		    AND COALESCE(price_schedule_id, '') = COALESCE($4, '')
		    AND id <> $5
		    AND version_status = $6
		    AND (validity_end IS NULL OR validity_end > $1)`,
		closeAt.Time, nowMillis, wsID, targetPriceScheduleID, req.Id, versionStatusPublished); err != nil {
		return nil, fmt.Errorf("close prior sibling: %w", err)
	}

	// Flip the target to PUBLISHED.
	if _, err := tx.ExecContext(ctx,
		`UPDATE job_outcome_summary_document_template
		    SET version_status = $1, version = $2, published_at = $3, published_by = $4, date_modified = $3
		  WHERE id = $5 AND workspace_id = $6`,
		versionStatusPublished, newVersion, nowMillis, publishedBy, req.Id, wsID); err != nil {
		return nil, fmt.Errorf("publish target: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	// Re-read the published row for the response.
	result, err := r.dbOps.Read(ctx, r.tableName, req.Id)
	if err != nil {
		return nil, fmt.Errorf("re-read published binding: %w", err)
	}
	item, err := jobOutcomeSummaryDocumentTemplateFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.PublishJobOutcomeSummaryDocumentTemplateResponse{Data: item, Success: true}, nil
}

// pbVersionStatusValue maps the persisted enum name to the operation VersionStatus
// enum value carried by the binding proto.
func pbVersionStatusValue(name string) (enums.VersionStatus, bool) {
	if v, ok := enums.VersionStatus_value[name]; ok {
		return enums.VersionStatus(v), true
	}
	return enums.VersionStatus_VERSION_STATUS_UNSPECIFIED, false
}

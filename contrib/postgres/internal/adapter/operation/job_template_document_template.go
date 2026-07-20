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
	sqlexec "github.com/erniealice/espyna-golang/shared/database/sqlexec"
	"github.com/erniealice/espyna-golang/shared/identity"
	documenttemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/document/template"
	jobcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_category"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_document_template"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// NOTE: the versionStatus{Draft,Published,Deprecated} consts, plus the shared
// helpers filterToMutableBindingFields / nextBindingVersion / pbVersionStatusValue
// and protoToMap / stripClientWorkspaceKeys, are declared once at package scope
// (job_outcome_summary_document_template.go / proto_map.go) and reused here — this
// file is the JOSDT sibling in the same `operation` package.

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.JobTemplateDocumentTemplate, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres job_template_document_template repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresJobTemplateDocumentTemplateRepository(dbOps, tableName), nil
	})
}

// PostgresJobTemplateDocumentTemplateRepository implements the sheet-family
// template-binding CRUD + the applicability resolver + the publish transaction.
// It is the JOSDT sibling (20260720): it binds the job_template rendering surface
// (the outcome-matrix "grade sheet") to a document_template, scoped by workspace +
// (optional) price_schedule + (optional) job_category — the sheet COLUMN SHAPE
// axis. The resolver and publish paths use the raw *sql.DB (CTE + multi-statement
// TX); CRUD delegates to the workspace-aware dbOps decorator.
//
// Server-owned lifecycle (RA2 P1). A binding is born DRAFT + unversioned
// (version=0); the only path that flips it to PUBLISHED and allocates its real
// version is the Publish transaction. Create forces DRAFT; Update filters immutable
// fields once a row is PUBLISHED/DEPRECATED; Publish is a draft-only,
// affected-row-checked, publish-flips-sibling transaction. These persistence-layer
// guards are the belt to the use-case suspenders — a bypassed use case can still
// never mint a published row or mutate a published lineage through CRUD.
type PostgresJobTemplateDocumentTemplateRepository struct {
	pb.UnimplementedJobTemplateDocumentTemplateDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func NewPostgresJobTemplateDocumentTemplateRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.JobTemplateDocumentTemplateDomainServiceServer {
	if tableName == "" {
		tableName = entityid.JobTemplateDocumentTemplate
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresJobTemplateDocumentTemplateRepository{dbOps: dbOps, db: db, tableName: tableName}
}

// --- CRUD -----------------------------------------------------------------

func (r *PostgresJobTemplateDocumentTemplateRepository) CreateJobTemplateDocumentTemplate(ctx context.Context, req *pb.CreateJobTemplateDocumentTemplateRequest) (*pb.CreateJobTemplateDocumentTemplateResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("binding data is required")
	}
	data, err := protoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	// protoToMap canonicalizes keys to snake_case (gate H1), so hydrate-only and
	// FK keys are matched by their column spelling here.
	// hydrate-only fields never persist.
	delete(data, "document_template")
	delete(data, "price_schedule")
	delete(data, "job_category")
	// Tenancy is owned by the workspace-aware decorator, which injects the trusted
	// workspace_id on Create. Strip any client-supplied workspace key so it can
	// never win a key-normalization collision (gate H1).
	stripClientWorkspaceKeys(data)
	// RA2 P1 — server-owned lifecycle. A binding is ALWAYS born DRAFT and
	// unversioned/provisional (version=0). Never honor a client-supplied
	// version_status, version, or publish audit; only the Publish transaction may
	// flip version_status→PUBLISHED and allocate the real version. This is the
	// persistence-layer belt to the use-case suspenders.
	data["version_status"] = versionStatusDraft
	data["version"] = 0
	delete(data, "published_at")
	delete(data, "published_at_string")
	delete(data, "published_by")
	// empty optional FK ("" from a form) → SQL NULL so the FK constraint holds.
	if v, ok := data["price_schedule_id"].(string); ok && v == "" {
		data["price_schedule_id"] = nil
	}
	if v, ok := data["job_category_id"].(string); ok && v == "" {
		data["job_category_id"] = nil
	}
	if v, ok := data["supersedes_binding_id"].(string); ok && v == "" {
		data["supersedes_binding_id"] = nil
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create binding: %w", err)
	}
	item, err := jobTemplateDocumentTemplateFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.CreateJobTemplateDocumentTemplateResponse{Data: []*pb.JobTemplateDocumentTemplate{item}, Success: true}, nil
}

func (r *PostgresJobTemplateDocumentTemplateRepository) ReadJobTemplateDocumentTemplate(ctx context.Context, req *pb.ReadJobTemplateDocumentTemplateRequest) (*pb.ReadJobTemplateDocumentTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("binding ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read binding: %w", err)
	}
	item, err := jobTemplateDocumentTemplateFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.ReadJobTemplateDocumentTemplateResponse{Data: []*pb.JobTemplateDocumentTemplate{item}, Success: true}, nil
}

func (r *PostgresJobTemplateDocumentTemplateRepository) UpdateJobTemplateDocumentTemplate(ctx context.Context, req *pb.UpdateJobTemplateDocumentTemplateRequest) (*pb.UpdateJobTemplateDocumentTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("binding ID is required")
	}
	data, err := protoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	// RA2 P1 + Q3 (attendance-v2 follow-up) — the generic Update route is a
	// narrow administrative surface: the ONLY generically mutable fields are the
	// admin gate (active) + the audit stamp (date_modified). Everything else —
	// scope fields (document_template_id, price_schedule_id, job_category_id,
	// validity_start, validity_end, supersedes_binding_id), lifecycle (version,
	// version_status), publish audit, tenant keys, hydrate-only nests — is stripped
	// UNCONDITIONALLY, for DRAFT rows too. The previous draft-exempt filter
	// depended on an UNLOCKED lifecycle read, so a Publish interleaving between
	// that read and the write could let scope fields mutate a just-published row
	// (codex wave-b MED, scope-field TOCTOU). Stripping without reading closes
	// the race structurally; legitimate scope changes go through
	// delete-draft + re-create (the settings-UI flow, which never wires Update)
	// or a future dedicated administrative path.
	filterToMutableBindingFields(data)
	// active=true is refused as well: protojson omits the false zero-value, so
	// the generic route could only ever SET active — i.e. resurrect a
	// soft-deleted draft through the public update surface (codex wave-b MED).
	// Deactivation stays the draft-only Delete transaction's job; reactivation
	// has no generic route (fail-closed).
	if v, ok := data["active"].(bool); ok && v {
		delete(data, "active")
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("update payload contains no generically mutable binding fields")
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update binding: %w", err)
	}
	item, err := jobTemplateDocumentTemplateFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateJobTemplateDocumentTemplateResponse{Data: []*pb.JobTemplateDocumentTemplate{item}, Success: true}, nil
}

// DeleteJobTemplateDocumentTemplate soft-deletes a binding, but ONLY while it is a
// DRAFT. A PUBLISHED (or DEPRECATED) binding is part of the immutable version
// history — historical as_of renders must keep resolving it — so its removal is
// refused. This is the persistence-layer belt to the use-case suspenders: the
// guard is a status- and workspace-scoped soft-delete with an affected-row check,
// so a bypassed use case (or a concurrent publish) can never remove a non-draft
// lineage. Idempotent on a draft (re-stamps active=false).
func (r *PostgresJobTemplateDocumentTemplateRepository) DeleteJobTemplateDocumentTemplate(ctx context.Context, req *pb.DeleteJobTemplateDocumentTemplateRequest) (*pb.DeleteJobTemplateDocumentTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("binding ID is required")
	}
	id, ok := identity.FromContext(ctx)
	if !ok || id.WorkspaceID == "" {
		return nil, fmt.Errorf("workspace identity required to delete")
	}
	nowMillis := time.Now().UTC().UnixMilli()
	res, err := r.db.ExecContext(ctx,
		fmt.Sprintf(`UPDATE %s
			    SET active = false, date_modified = $1
			  WHERE id = $2 AND workspace_id = $3 AND version_status = $4`,
			entityid.JobTemplateDocumentTemplate),
		nowMillis, req.Data.Id, id.WorkspaceID, versionStatusDraft)
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
	return &pb.DeleteJobTemplateDocumentTemplateResponse{Success: true}, nil
}

func (r *PostgresJobTemplateDocumentTemplateRepository) ListJobTemplateDocumentTemplates(ctx context.Context, req *pb.ListJobTemplateDocumentTemplatesRequest) (*pb.ListJobTemplateDocumentTemplatesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && (req.Filters != nil || req.Pagination != nil) {
		params = &interfaces.ListParams{Filters: req.Filters, Pagination: req.Pagination}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list bindings: %w", err)
	}
	var items []*pb.JobTemplateDocumentTemplate
	for _, row := range listResult.Data {
		item, err := jobTemplateDocumentTemplateFromResult(row)
		if err != nil {
			continue
		}
		items = append(items, item)
	}
	return &pb.ListJobTemplateDocumentTemplatesResponse{Data: items, Success: true}, nil
}

func jobTemplateDocumentTemplateFromResult(result any) (*pb.JobTemplateDocumentTemplate, error) {
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
	item := &pb.JobTemplateDocumentTemplate{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to proto: %w", err)
	}
	return item, nil
}

// executor returns the transaction-aware SQL executor: the active *sql.Tx when
// one is present on ctx (so the resolver participates in an ambient use-case /
// test transaction and sees its uncommitted writes), else the pooled *sql.DB.
func (r *PostgresJobTemplateDocumentTemplateRepository) executor(ctx context.Context) sqlexec.DBExecutor {
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

// --- Resolver -------------------------------------------------------------

// findApplicableJobTemplateDocumentTemplateSQL builds the resolver query. Table
// identifiers come from the registry/entityid constants (table-name single-source
// invariant, Q-TABLE-NAMES) — never bare literals. Extracted so the predicate
// shape (tenant gate, published gate, half-open validity, most-specific-wins
// ordering across the category+schedule axes, LIMIT 2 ambiguity guard) is
// unit-testable without a live DB.
//
// Params: $1=workspace(ctx), $2=price_schedule_id, $3='VERSION_STATUS_PUBLISHED',
// $4=as_of, $5=job_category_id.
//
// 4-tier most-specific-wins match_rank (category ≻ schedule, per Q2):
//
//	rank 0 = category exact  + schedule exact
//	rank 1 = category exact  + schedule fallback (b.price_schedule_id IS NULL)
//	rank 2 = category fallback (b.job_category_id IS NULL) + schedule exact
//	rank 3 = category fallback + schedule fallback (workspace-wide)
func findApplicableJobTemplateDocumentTemplateSQL() string {
	return fmt.Sprintf(`
		WITH requested_scope AS (
			SELECT NULLIF($2, '') AS price_schedule_id, NULLIF($5, '') AS job_category_id
			WHERE (NULLIF($2, '') IS NULL
			       OR EXISTS (SELECT 1 FROM %[3]s ps
			                  WHERE ps.id = NULLIF($2, '') AND ps.workspace_id = $1))
			  AND (NULLIF($5, '') IS NULL
			       OR EXISTS (SELECT 1 FROM %[4]s jc
			                  WHERE jc.id = NULLIF($5, '') AND jc.workspace_id = $1))
		)
		SELECT
			b.id, b.workspace_id, b.document_template_id, b.price_schedule_id, b.job_category_id, b.version,
			b.version_status, b.validity_start, b.validity_end, b.supersedes_binding_id,
			b.active, b.created_by, b.published_at, b.published_by, b.date_created, b.date_modified,
			dt.id, dt.name, dt.description, dt.active, dt.workspace_id, dt.template_type,
			dt.document_purpose, dt.storage_container, dt.storage_key, dt.original_filename,
			dt.file_size_bytes, dt.is_default, dt.created_by, dt.status, dt.module_key,
			ps.id, ps.name, ps.active, ps.workspace_id,
			jc.id, jc.name, jc.code, jc.active, jc.workspace_id,
			CASE
				WHEN rs.job_category_id IS NOT NULL AND b.job_category_id = rs.job_category_id
				     AND rs.price_schedule_id IS NOT NULL AND b.price_schedule_id = rs.price_schedule_id THEN 0
				WHEN rs.job_category_id IS NOT NULL AND b.job_category_id = rs.job_category_id
				     AND b.price_schedule_id IS NULL THEN 1
				WHEN b.job_category_id IS NULL
				     AND rs.price_schedule_id IS NOT NULL AND b.price_schedule_id = rs.price_schedule_id THEN 2
				ELSE 3
			END AS match_rank
		FROM %[1]s b
		CROSS JOIN requested_scope rs
		JOIN %[2]s dt
			ON dt.id = b.document_template_id AND dt.workspace_id = b.workspace_id
		LEFT JOIN %[3]s ps
			ON ps.id = b.price_schedule_id AND ps.workspace_id = b.workspace_id
		LEFT JOIN %[4]s jc
			ON jc.id = b.job_category_id AND jc.workspace_id = b.workspace_id
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
			AND (b.job_category_id = rs.job_category_id OR b.job_category_id IS NULL)
		ORDER BY match_rank, b.version DESC
		LIMIT 2`,
		entityid.JobTemplateDocumentTemplate, entityid.DocumentTemplate, entityid.PriceSchedule, entityid.JobCategory)
}

// FindApplicableJobTemplateDocumentTemplate resolves the single applicable,
// published, active binding for (job_category_id, price_schedule_id, as_of).
// Tenant isolation is enforced IN THE SQL PREDICATE (workspace_id = $1, sourced
// from trusted context) — never from the request. Most-specific-wins across the
// category + schedule axes (category ≻ schedule, per Q2); newest version breaks
// ties. A cross-workspace price_schedule or job_category yields no result (never a
// fallback). Two equal-ranked, equal-version rows fail closed (LIMIT 2 ambiguity
// guard). No match → found=false, success=true.
func (r *PostgresJobTemplateDocumentTemplateRepository) FindApplicableJobTemplateDocumentTemplate(ctx context.Context, req *pb.FindApplicableJobTemplateDocumentTemplateRequest) (*pb.FindApplicableJobTemplateDocumentTemplateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	ex := r.executor(ctx)
	if ex == nil {
		return nil, fmt.Errorf("binding resolver requires direct SQL access")
	}

	// Tenant isolation: scope to the caller's workspace from trusted context.
	// An absent/empty workspace resolves nothing (fail-closed).
	id, ok := identity.FromContext(ctx)
	if !ok || id.WorkspaceID == "" {
		return &pb.FindApplicableJobTemplateDocumentTemplateResponse{Found: false, Success: true}, nil
	}
	wsID := id.WorkspaceID

	asOf := time.Now().UTC()
	if req.AsOf != nil {
		asOf = req.AsOf.AsTime().UTC()
	}

	rows, err := ex.QueryContext(ctx, findApplicableJobTemplateDocumentTemplateSQL(), wsID, req.GetPriceScheduleId(), versionStatusPublished, asOf, req.GetJobCategoryId())
	if err != nil {
		return nil, fmt.Errorf("resolver query failed: %w", err)
	}
	defer rows.Close()

	type scanned struct {
		binding   *pb.JobTemplateDocumentTemplate
		matchRank int
		version   int32
	}
	var results []scanned
	for rows.Next() {
		var (
			bID, bWorkspaceID, bDocTmplID                             string
			bPriceScheduleID, bJobCategoryID                          sql.NullString
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

			jcID, jcName, jcCode, jcWorkspaceID sql.NullString
			jcActive                            sql.NullBool

			matchRank int
		)
		if err := rows.Scan(
			&bID, &bWorkspaceID, &bDocTmplID, &bPriceScheduleID, &bJobCategoryID, &bVersion,
			&bVersionStatus, &bValidityStart, &bValidityEnd, &bSupersedes,
			&bActive, &bCreatedBy, &bPublishedAt, &bPublishedBy, &bDateCreated, &bDateModified,
			&dtID, &dtName, &dtDescription, &dtActive, &dtWorkspaceID, &dtTemplateType,
			&dtDocumentPurpose, &dtStorageContainer, &dtStorageKey, &dtOriginalFilename,
			&dtFileSizeBytes, &dtIsDefault, &dtCreatedBy, &dtStatus, &dtModuleKey,
			&psID, &psName, &psActive, &psWorkspaceID,
			&jcID, &jcName, &jcCode, &jcActive, &jcWorkspaceID,
			&matchRank,
		); err != nil {
			return nil, fmt.Errorf("resolver scan failed: %w", err)
		}

		binding := &pb.JobTemplateDocumentTemplate{
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

		// Hydrate JobCategory (the sheet-shape axis) when the binding is
		// category-scoped.
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

		results = append(results, scanned{binding: binding, matchRank: matchRank, version: binding.Version})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("resolver rows error: %w", err)
	}

	if len(results) == 0 {
		return &pb.FindApplicableJobTemplateDocumentTemplateResponse{Found: false, Success: true}, nil
	}
	// Ambiguity guard: two equal-ranked, equal-version rows must not be a
	// coin-flip. (The null-safe unique index makes this structurally impossible;
	// this is defense-in-depth.)
	if len(results) == 2 && results[0].matchRank == results[1].matchRank && results[0].version == results[1].version {
		return nil, fmt.Errorf("ambiguous applicable binding: two equal-ranked versions resolved")
	}

	return &pb.FindApplicableJobTemplateDocumentTemplateResponse{
		Binding: results[0].binding,
		Found:   true,
		Success: true,
	}, nil
}

// --- Publish --------------------------------------------------------------

// jobTemplateDocumentTemplatePublishFlipSQL is the guarded flip that promotes the
// target DRAFT to PUBLISHED. The WHERE clause carries the draft-only predicate
// ($7); the caller checks RowsAffected == 1 so a concurrent publish (or a
// non-draft target) fails closed instead of double-publishing. Table identifier
// from entityid (Q-TABLE-NAMES).
func jobTemplateDocumentTemplatePublishFlipSQL() string {
	return fmt.Sprintf(`UPDATE %s
		    SET version_status = $1, version = $2, published_at = $3, published_by = $4, date_modified = $3
		  WHERE id = $5 AND workspace_id = $6 AND version_status = $7`,
		entityid.JobTemplateDocumentTemplate)
}

// PublishJobTemplateDocumentTemplate flips a DRAFT binding to PUBLISHED and closes
// the prior published sibling's validity_end in ONE transaction
// (publish-flips-sibling). The prior sibling stays PUBLISHED so historical as_of
// resolution still finds it. version is re-allocated MAX(published.version)+1 in
// the lineage under the transaction. The lineage bucket is
// (workspace, schedule-or-fallback, category-or-fallback) — the same tuple as the
// partial UNIQUE index. Workspace comes from trusted context.
//
// Draft-only + concurrency-safe (RA2 P1): the target is loaded FOR UPDATE and must
// be an active DRAFT; the flip UPDATE re-asserts the DRAFT predicate and the
// affected-row count is verified == 1, so a published/deprecated/inactive row can
// never be republished and two concurrent publishes cannot both win.
func (r *PostgresJobTemplateDocumentTemplateRepository) PublishJobTemplateDocumentTemplate(ctx context.Context, req *pb.PublishJobTemplateDocumentTemplateRequest) (*pb.PublishJobTemplateDocumentTemplateResponse, error) {
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

	tbl := entityid.JobTemplateDocumentTemplate

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Load + LOCK the target (workspace-scoped). FOR UPDATE serializes concurrent
	// publishes of the same binding; the DRAFT/active gate rejects re-publishing a
	// published/deprecated/inactive row.
	var (
		targetPriceScheduleID sql.NullString
		targetJobCategoryID   sql.NullString
		targetValidityStart   sql.NullTime
		targetVersionStatus   sql.NullString
		targetActive          sql.NullBool
	)
	err = tx.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT price_schedule_id, job_category_id, validity_start, version_status, active
			   FROM %s
			  WHERE id = $1 AND workspace_id = $2
			  FOR UPDATE`, tbl), req.Id, wsID).
		Scan(&targetPriceScheduleID, &targetJobCategoryID, &targetValidityStart, &targetVersionStatus, &targetActive)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("binding not found")
	}
	if err != nil {
		return nil, fmt.Errorf("load target: %w", err)
	}
	if !targetActive.Valid || !targetActive.Bool {
		return nil, fmt.Errorf("only an active draft binding can be published")
	}
	if !targetVersionStatus.Valid || targetVersionStatus.String != versionStatusDraft {
		return nil, fmt.Errorf("only a draft binding can be published (current status: %q)", targetVersionStatus.String)
	}

	// Allocate the next version in the lineage (workspace + schedule + category
	// bucket), counting PUBLISHED siblings only — drafts are unversioned/provisional.
	var maxVersion sql.NullInt32
	err = tx.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT MAX(version)
			   FROM %s
			  WHERE workspace_id = $1
			    AND COALESCE(price_schedule_id, '') = COALESCE($2, '')
			    AND COALESCE(job_category_id, '') = COALESCE($4, '')
			    AND version_status = $3`, tbl), wsID, targetPriceScheduleID, versionStatusPublished, targetJobCategoryID).
		Scan(&maxVersion)
	if err != nil {
		return nil, fmt.Errorf("compute next version: %w", err)
	}
	newVersion := nextBindingVersion(maxVersion)

	// Close the prior published sibling (leave it PUBLISHED for historical as_of).
	closeAt := targetValidityStart
	if !closeAt.Valid {
		closeAt = sql.NullTime{Time: time.Now().UTC(), Valid: true}
	}
	if _, err := tx.ExecContext(ctx,
		fmt.Sprintf(`UPDATE %s
			    SET validity_end = $1, date_modified = $2
			  WHERE workspace_id = $3
			    AND COALESCE(price_schedule_id, '') = COALESCE($4, '')
			    AND COALESCE(job_category_id, '') = COALESCE($7, '')
			    AND id <> $5
			    AND version_status = $6
			    AND (validity_end IS NULL OR validity_end > $1)`, tbl),
		closeAt.Time, nowMillis, wsID, targetPriceScheduleID, req.Id, versionStatusPublished, targetJobCategoryID); err != nil {
		return nil, fmt.Errorf("close prior sibling: %w", err)
	}

	// Flip the target to PUBLISHED — guarded by the DRAFT predicate + affected-row
	// check so a concurrent publish cannot double-apply.
	res, err := tx.ExecContext(ctx, jobTemplateDocumentTemplatePublishFlipSQL(),
		versionStatusPublished, newVersion, nowMillis, publishedBy, req.Id, wsID, versionStatusDraft)
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

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	// Re-read the published row for the response.
	result, err := r.dbOps.Read(ctx, r.tableName, req.Id)
	if err != nil {
		return nil, fmt.Errorf("re-read published binding: %w", err)
	}
	item, err := jobTemplateDocumentTemplateFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.PublishJobTemplateDocumentTemplateResponse{Data: item, Success: true}, nil
}

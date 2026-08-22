//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	sqlexec "github.com/erniealice/espyna-golang/shared/database/sqlexec"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
)

// job_template_phase has NO workspace_id column of its own, so the generic
// workspace-aware decorator cannot scope it and the raw-SQL paths below would
// otherwise return or mutate any tenant's rows. Every read/write is confined to
// the caller's workspace by deriving ownership through the parent job_template
// (jt.workspace_id). The predicate spelling ($N::text = '' OR jt.workspace_id =
// $N::text) mirrors the sibling job/job_template adapters: a bound workspace
// scopes the rows; an empty workspace (a service context with no tenant bound)
// is left unscoped.

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.JobTemplatePhase, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres job_template_phase repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresJobTemplatePhaseRepository(dbOps, tableName), nil
	})
}

// PostgresJobTemplatePhaseRepository implements job_template_phase CRUD operations using PostgreSQL
type PostgresJobTemplatePhaseRepository struct {
	pb.UnimplementedJobTemplatePhaseDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresJobTemplatePhaseRepository creates a new PostgreSQL job_template_phase repository
func NewPostgresJobTemplatePhaseRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.JobTemplatePhaseDomainServiceServer {
	if tableName == "" {
		tableName = "job_template_phase"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresJobTemplatePhaseRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// executor returns the transaction-aware SQL executor: the active *sql.Tx when
// one is present on ctx (so the W-SPAWN graph enumeration reads the parents it is
// about to lock ON the same transaction — codex P3 §A5), else the pooled *sql.DB.
func (r *PostgresJobTemplatePhaseRepository) executor(ctx context.Context) sqlexec.DBExecutor {
	if ep, ok := r.dbOps.(interface {
		GetExecutor(ctx context.Context) sqlexec.DBExecutor
	}); ok {
		if e := ep.GetExecutor(ctx); e != nil {
			return e
		}
	}
	return r.db
}

// CreateJobTemplatePhase creates a new job template phase record
func (r *PostgresJobTemplatePhaseRepository) CreateJobTemplatePhase(ctx context.Context, req *pb.CreateJobTemplatePhaseRequest) (*pb.CreateJobTemplatePhaseResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("job template phase data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Convert millis timestamps to time.Time for postgres timestamp columns
	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	// Cross-tenant guard: the parent job_template must belong to the caller's
	// workspace, otherwise a phase could be attached to another tenant's template.
	if err := r.ensureTemplateInWorkspace(ctx, req.Data.GetJobTemplateId()); err != nil {
		return nil, err
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create job template phase: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	phase := &pb.JobTemplatePhase{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, phase); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.CreateJobTemplatePhaseResponse{
		Success: true,
		Data:    []*pb.JobTemplatePhase{phase},
	}, nil
}

// ReadJobTemplatePhase retrieves a job template phase record by ID
func (r *PostgresJobTemplatePhaseRepository) ReadJobTemplatePhase(ctx context.Context, req *pb.ReadJobTemplatePhaseRequest) (*pb.ReadJobTemplatePhaseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job template phase ID is required")
	}

	// Cross-tenant guard: reject a by-id read of a phase owned by another tenant.
	if err := r.ensurePhaseInWorkspace(ctx, req.Data.Id); err != nil {
		return nil, err
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read job template phase: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	phase := &pb.JobTemplatePhase{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, phase); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.ReadJobTemplatePhaseResponse{
		Success: true,
		Data:    []*pb.JobTemplatePhase{phase},
	}, nil
}

// UpdateJobTemplatePhase updates a job template phase record
func (r *PostgresJobTemplatePhaseRepository) UpdateJobTemplatePhase(ctx context.Context, req *pb.UpdateJobTemplatePhaseRequest) (*pb.UpdateJobTemplatePhaseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job template phase ID is required")
	}

	// Cross-tenant guard: reject an update to a phase owned by another tenant.
	if err := r.ensurePhaseInWorkspace(ctx, req.Data.Id); err != nil {
		return nil, err
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Convert millis timestamps to time.Time for postgres timestamp columns
	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update job template phase: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	phase := &pb.JobTemplatePhase{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, phase); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.UpdateJobTemplatePhaseResponse{
		Success: true,
		Data:    []*pb.JobTemplatePhase{phase},
	}, nil
}

// DeleteJobTemplatePhase deletes a job template phase record (soft delete)
func (r *PostgresJobTemplatePhaseRepository) DeleteJobTemplatePhase(ctx context.Context, req *pb.DeleteJobTemplatePhaseRequest) (*pb.DeleteJobTemplatePhaseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job template phase ID is required")
	}

	// Cross-tenant guard: reject a delete of a phase owned by another tenant.
	if err := r.ensurePhaseInWorkspace(ctx, req.Data.Id); err != nil {
		return nil, err
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete job template phase: %w", err)
	}

	return &pb.DeleteJobTemplatePhaseResponse{
		Success: true,
	}, nil
}

// ListJobTemplatePhases lists job template phase records with optional filters
func (r *PostgresJobTemplatePhaseRepository) ListJobTemplatePhases(ctx context.Context, req *pb.ListJobTemplatePhasesRequest) (*pb.ListJobTemplatePhasesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list job template phases: %w", err)
	}

	var phases []*pb.JobTemplatePhase
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal job_template_phase row: %v", err)
			continue
		}

		phase := &pb.JobTemplatePhase{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, phase); err != nil {
			log.Printf("WARN: protojson unmarshal job_template_phase: %v", err)
			continue
		}
		phases = append(phases, phase)
	}

	// Cross-tenant scope: the generic list flows through the workspace-aware
	// decorator, which cannot scope this column-less child table, so it returns
	// rows across every tenant. Confine them to the caller's workspace via the
	// parent job_template.
	phases, err = r.filterPhasesByWorkspace(ctx, phases)
	if err != nil {
		return nil, err
	}

	return &pb.ListJobTemplatePhasesResponse{
		Success: true,
		Data:    phases,
	}, nil
}

var jobTemplatePhaseSortableSQLCols = []string{
	"id", "date_created", "date_modified", "active", "job_template_id",
	"name", "phase_order",
}

// GetJobTemplatePhaseListPageData retrieves phases with pagination, filtering, sorting, and search
func (r *PostgresJobTemplatePhaseRepository) GetJobTemplatePhaseListPageData(
	ctx context.Context,
	req *pb.GetJobTemplatePhaseListPageDataRequest,
) (*pb.GetJobTemplatePhaseListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get job template phase list page data request is required")
	}

	searchPattern, err := postgresCore.BoundedContainsSearchPattern(req.GetSearch())
	if err != nil {
		return nil, fmt.Errorf("invalid list search: %w", err)
	}

	limit, offset, page, err := postgresCore.BoundedOffsetPagination(req.GetPagination(), 50)
	if err != nil {
		return nil, fmt.Errorf("invalid list pagination: %w", err)
	}

	// Sort — fail-closed against the per-entity whitelist (A2 guard). The outer
	// SELECT projects the enriched columns unprefixed (e.*), so the ORDER BY
	// references unprefixed whitelist columns. An unknown column errors instead
	// of being interpolated verbatim into ORDER BY.
	orderByClause, err := postgresCore.BuildOrderBy(jobTemplatePhaseSortableSQLCols, req.GetSort(), "phase_order ASC")
	if err != nil {
		return nil, err
	}

	// Cross-tenant scope: the drawer list is reached only through the session, so
	// a missing identity is a middleware fault and fails closed (Must). $4 carries
	// the workspace_id, derived through the parent job_template.
	wsID := identity.Must(ctx).WorkspaceID
	query := jobTemplatePhaseListPageDataSQL(orderByClause)

	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset, wsID)
	if err != nil {
		return nil, fmt.Errorf("failed to query job template phase list page data: %w", err)
	}
	defer rows.Close()

	var phases []*pb.JobTemplatePhase
	var totalCount int64

	for rows.Next() {
		var (
			id            string
			dateCreated   time.Time
			dateModified  time.Time
			active        bool
			jobTemplateID string
			name          string
			phaseOrder    sql.NullInt32
			code          sql.NullString
			total         int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&jobTemplateID,
			&name,
			&phaseOrder,
			&code,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job template phase row: %w", err)
		}

		totalCount = total

		phase := &pb.JobTemplatePhase{
			Id:            id,
			Active:        active,
			JobTemplateId: jobTemplateID,
			Name:          name,
			PhaseOrder:    phaseOrder.Int32,
		}
		if code.Valid {
			v := code.String
			phase.Code = &v
		}

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
		return nil, fmt.Errorf("error iterating job template phase rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &pb.GetJobTemplatePhaseListPageDataResponse{
		JobTemplatePhaseList: phases,
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

// GetJobTemplatePhaseItemPageData retrieves a single phase with enriched data
func (r *PostgresJobTemplatePhaseRepository) GetJobTemplatePhaseItemPageData(
	ctx context.Context,
	req *pb.GetJobTemplatePhaseItemPageDataRequest,
) (*pb.GetJobTemplatePhaseItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get job template phase item page data request is required")
	}
	if req.JobTemplatePhaseId == "" {
		return nil, fmt.Errorf("job template phase ID is required")
	}

	// Cross-tenant scope: drawer item reached only through the session; a missing
	// identity fails closed (Must). $2 carries the workspace_id, derived through
	// the parent job_template.
	wsID := identity.Must(ctx).WorkspaceID
	query := jobTemplatePhaseItemPageDataSQL()

	row := r.db.QueryRowContext(ctx, query, req.JobTemplatePhaseId, wsID)

	var (
		id            string
		dateCreated   time.Time
		dateModified  time.Time
		active        bool
		jobTemplateID string
		name          string
		phaseOrder    sql.NullInt32
		code          sql.NullString
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&jobTemplateID,
		&name,
		&phaseOrder,
		&code,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job template phase with ID '%s' not found", req.JobTemplatePhaseId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query job template phase item page data: %w", err)
	}

	phase := &pb.JobTemplatePhase{
		Id:            id,
		Active:        active,
		JobTemplateId: jobTemplateID,
		Name:          name,
		PhaseOrder:    phaseOrder.Int32,
	}
	if code.Valid {
		v := code.String
		phase.Code = &v
	}

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

	return &pb.GetJobTemplatePhaseItemPageDataResponse{
		JobTemplatePhase: phase,
		Success:          true,
	}, nil
}

// jobTemplatePhaseListByTemplateSQL returns the SELECT that ListByJobTemplate
// runs. It is a package-level builder (not an inline literal) purely so the
// SQL projection is unit-testable via a shape test — mirroring the
// jobTemplateSummarySelectFrom() / jobOutcomeSummaryListPageDataSQL() precedent
// in this package. The emitted SQL is byte-identical to the former inline literal.
//
// Root-cause note (why the shape test that reads this is load-bearing): commit
// f2c80100 shipped a silent production bug because this SELECT and its
// positional rows.Scan(...) were SYMMETRICALLY missing scoring_scheme_id — a
// dropped column, not a reorder, so database/sql never raised a column-count
// mismatch; the field just came back nil forever. Item #3 restored
// predecessor_template_phase_id the same way. Every column named here MUST have
// a matching Scan destination in ListByJobTemplate, in the SAME order.
//
// Scope note (item #3, Q3-A/Q3-B): the billing_* columns (triggers_billing,
// billing_percent_bps, billing_amount, billing_currency) are DELIBERATELY NOT
// projected here. Projecting them activates the milestone-billing event
// materializer (materialize_billing_events_for_job.go:258 gates on
// GetTriggersBilling()), a revenue surface that is (a) out of this grading
// PR's scope and (b) not yet safe end-to-end (JobPhase.ListByJob does not
// populate TemplatePhaseId, so spawned milestone events would have no
// job_phase_id and never release; and phase create/update do not enforce the
// billing_percent_bps vs billing_amount mutual-exclusion contract). Restoring
// them belongs to the separate billing follow-up ticket (plan Q3-B).
func jobTemplatePhaseListByTemplateSQL() string {
	return `
		SELECT
			jtp.id,
			jtp.date_created,
			jtp.date_modified,
			jtp.active,
			jtp.job_template_id,
			jtp.name,
			jtp.phase_order,
			jtp.scoring_scheme_id,
			jtp.predecessor_template_phase_id,
			jtp.code
		FROM ` + entityid.JobTemplatePhase + ` jtp
		JOIN ` + entityid.JobTemplate + ` jt ON jt.id = jtp.job_template_id
		WHERE jtp.job_template_id = $1 AND jtp.active = true
		  AND ($2::text = '' OR jt.workspace_id = $2::text)
		ORDER BY jtp.phase_order ASC
	`
}

// jobTemplatePhaseItemPageDataSQL returns the enriched single-phase read, scoped
// to the caller's workspace through the parent job_template ($2 = workspace_id).
func jobTemplatePhaseItemPageDataSQL() string {
	return `
		SELECT
			jtp.id,
			jtp.date_created,
			jtp.date_modified,
			jtp.active,
			jtp.job_template_id,
			jtp.name,
			jtp.phase_order,
			jtp.code
		FROM ` + entityid.JobTemplatePhase + ` jtp
		JOIN ` + entityid.JobTemplate + ` jt ON jt.id = jtp.job_template_id
		WHERE jtp.id = $1 AND jtp.active = true
		  AND ($2::text = '' OR jt.workspace_id = $2::text)
	`
}

// jobTemplatePhaseListPageDataSQL returns the paginated phase list, scoped to the
// caller's workspace through the parent job_template ($4 = workspace_id).
func jobTemplatePhaseListPageDataSQL(orderByClause string) string {
	return `
		WITH enriched AS (
			SELECT
				jtp.id,
				jtp.date_created,
				jtp.date_modified,
				jtp.active,
				jtp.job_template_id,
				jtp.name,
				jtp.phase_order,
				jtp.code
			FROM ` + entityid.JobTemplatePhase + ` jtp
			JOIN ` + entityid.JobTemplate + ` jt ON jt.id = jtp.job_template_id
			WHERE jtp.active = true
			  AND ($4::text = '' OR jt.workspace_id = $4::text)
			  AND ($1::text IS NULL OR $1::text = '' OR
			       jtp.name ILIKE $1)
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
}

// jobTemplatePhaseWorkspaceOwnedSQL reports whether a phase (by id, $1) hangs off
// a job_template owned by the workspace ($2). Backs the generic read/update/
// delete tenant guards.
func jobTemplatePhaseWorkspaceOwnedSQL() string {
	return `SELECT EXISTS (
		SELECT 1 FROM ` + entityid.JobTemplatePhase + ` jtp
		JOIN ` + entityid.JobTemplate + ` jt ON jt.id = jtp.job_template_id
		WHERE jtp.id = $1 AND ($2::text = '' OR jt.workspace_id = $2::text))`
}

// jobTemplateInWorkspaceSQL reports whether a job_template (by id, $1) belongs to
// the workspace ($2). Backs the phase-create parent-FK validation.
func jobTemplateInWorkspaceSQL() string {
	return `SELECT EXISTS (
		SELECT 1 FROM ` + entityid.JobTemplate + ` jt
		WHERE jt.id = $1 AND ($2::text = '' OR jt.workspace_id = $2::text))`
}

// jobTemplatePhaseOwnedTemplateIDsSQL returns, from a candidate template-id set
// ($1), those owned by the workspace ($2). Backs the generic-list tenant scope.
func jobTemplatePhaseOwnedTemplateIDsSQL() string {
	return `SELECT id FROM ` + entityid.JobTemplate + `
		WHERE id = ANY($1) AND ($2::text = '' OR workspace_id = $2::text)`
}

// ensurePhaseInWorkspace fails a by-id generic op closed when the phase's parent
// job_template is not owned by the caller's workspace. An empty workspace (a
// service context with no tenant bound) skips the check, matching the raw reads'
// ($N = ” OR ...) posture.
func (r *PostgresJobTemplatePhaseRepository) ensurePhaseInWorkspace(ctx context.Context, phaseID string) error {
	wsID := identity.Must(ctx).WorkspaceID
	if wsID == "" {
		return nil
	}
	var owned bool
	if err := r.db.QueryRowContext(ctx, jobTemplatePhaseWorkspaceOwnedSQL(), phaseID, wsID).Scan(&owned); err != nil {
		return fmt.Errorf("failed to verify job template phase workspace: %w", err)
	}
	if !owned {
		return fmt.Errorf("job template phase with ID '%s' not found", phaseID)
	}
	return nil
}

// ensureTemplateInWorkspace rejects a phase create whose parent job_template FK
// points at another tenant's template.
func (r *PostgresJobTemplatePhaseRepository) ensureTemplateInWorkspace(ctx context.Context, templateID string) error {
	wsID := identity.Must(ctx).WorkspaceID
	if wsID == "" {
		return nil
	}
	var owned bool
	if err := r.db.QueryRowContext(ctx, jobTemplateInWorkspaceSQL(), templateID, wsID).Scan(&owned); err != nil {
		return fmt.Errorf("failed to verify job template workspace: %w", err)
	}
	if !owned {
		return fmt.Errorf("job template with ID '%s' not found", templateID)
	}
	return nil
}

// filterPhasesByWorkspace drops phases whose parent job_template is not owned by
// the caller's workspace. An empty workspace leaves the set unscoped.
func (r *PostgresJobTemplatePhaseRepository) filterPhasesByWorkspace(ctx context.Context, phases []*pb.JobTemplatePhase) ([]*pb.JobTemplatePhase, error) {
	wsID := identity.Must(ctx).WorkspaceID
	if wsID == "" || len(phases) == 0 {
		return phases, nil
	}

	seen := make(map[string]struct{}, len(phases))
	ids := make([]string, 0, len(phases))
	for _, p := range phases {
		tid := p.GetJobTemplateId()
		if tid == "" {
			continue
		}
		if _, ok := seen[tid]; ok {
			continue
		}
		seen[tid] = struct{}{}
		ids = append(ids, tid)
	}

	owned := make(map[string]struct{}, len(ids))
	if len(ids) > 0 {
		rows, err := r.db.QueryContext(ctx, jobTemplatePhaseOwnedTemplateIDsSQL(), sqlStringArray(ids), wsID)
		if err != nil {
			return nil, fmt.Errorf("failed to scope job template phases by workspace: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return nil, fmt.Errorf("failed to scan owned job template id: %w", err)
			}
			owned[id] = struct{}{}
		}
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("error iterating owned job template ids: %w", err)
		}
	}

	out := make([]*pb.JobTemplatePhase, 0, len(phases))
	for _, p := range phases {
		if _, ok := owned[p.GetJobTemplateId()]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}

// ListByJobTemplate retrieves all phases for a given job template, ordered by phase_order
func (r *PostgresJobTemplatePhaseRepository) ListByJobTemplate(
	ctx context.Context,
	req *pb.ListByJobTemplateRequest,
) (*pb.ListByJobTemplateResponse, error) {
	if req == nil || req.JobTemplateId == "" {
		return nil, fmt.Errorf("job template ID is required")
	}

	query := jobTemplatePhaseListByTemplateSQL()

	// Cross-tenant scope through the parent job_template ($2). This method is also
	// consumed by template-materialization, which may run in a service context
	// with no tenant bound; a soft identity read leaves that path unscoped (empty
	// workspace) while an authenticated caller is confined to its own workspace.
	wsID := ""
	if id, ok := identity.FromContext(ctx); ok {
		wsID = id.WorkspaceID
	}

	rows, err := r.executor(ctx).QueryContext(ctx, query, req.JobTemplateId, wsID)
	if err != nil {
		return nil, fmt.Errorf("failed to list job template phases by template: %w", err)
	}
	defer rows.Close()

	var phases []*pb.JobTemplatePhase
	for rows.Next() {
		var (
			id            string
			dateCreated   time.Time
			dateModified  time.Time
			active        bool
			jobTemplateID string
			name          string
			phaseOrder    sql.NullInt32
			scoringScheme sql.NullString
			predecessorID sql.NullString
			code          sql.NullString
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&jobTemplateID,
			&name,
			&phaseOrder,
			&scoringScheme,
			&predecessorID,
			&code,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job template phase row: %w", err)
		}

		phase := &pb.JobTemplatePhase{
			Id:            id,
			Active:        active,
			JobTemplateId: jobTemplateID,
			Name:          name,
			PhaseOrder:    phaseOrder.Int32,
		}
		if scoringScheme.Valid {
			v := scoringScheme.String
			phase.ScoringSchemeId = &v
		}
		if predecessorID.Valid {
			v := predecessorID.String
			phase.PredecessorTemplatePhaseId = &v
		}
		if code.Valid {
			v := code.String
			phase.Code = &v
		}

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
		return nil, fmt.Errorf("error iterating job template phase rows: %w", err)
	}

	return &pb.ListByJobTemplateResponse{
		JobTemplatePhases: phases,
		Success:           true,
	}, nil
}

// NewJobTemplatePhaseRepository creates a new PostgreSQL job_template_phase repository (old-style constructor)
func NewJobTemplatePhaseRepository(db *sql.DB, tableName string) pb.JobTemplatePhaseDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresJobTemplatePhaseRepository(dbOps, tableName)
}

// convertMillisToTime converts a millis-epoch value in a JSON map to time.Time.
// Protobuf int64 fields serialize to JSON strings via protojson (e.g. "1771886746000").
// Postgres timestamp columns need time.Time, not raw millis.
func convertMillisToTime(data map[string]any, jsonKey string) {
	v, ok := data[jsonKey]
	if !ok {
		return
	}
	switch val := v.(type) {
	case string:
		// protojson serializes int64 as string
		var millis int64
		if _, err := fmt.Sscanf(val, "%d", &millis); err == nil && millis > 1e12 {
			data[jsonKey] = time.UnixMilli(millis)
		}
	case float64:
		if val > 1e12 {
			data[jsonKey] = time.UnixMilli(int64(val))
		}
	}
}

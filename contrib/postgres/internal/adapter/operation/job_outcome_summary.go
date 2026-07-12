//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/principalscope"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.JobOutcomeSummary, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres job_outcome_summary repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresJobOutcomeSummaryRepository(dbOps, tableName), nil
	})
}

// PostgresJobOutcomeSummaryRepository implements job_outcome_summary CRUD operations using PostgreSQL
type PostgresJobOutcomeSummaryRepository struct {
	pb.UnimplementedJobOutcomeSummaryDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresJobOutcomeSummaryRepository creates a new PostgreSQL job_outcome_summary repository
func NewPostgresJobOutcomeSummaryRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.JobOutcomeSummaryDomainServiceServer {
	if tableName == "" {
		tableName = "job_outcome_summary"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresJobOutcomeSummaryRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateJobOutcomeSummary creates a new job_outcome_summary record
func (r *PostgresJobOutcomeSummaryRepository) CreateJobOutcomeSummary(ctx context.Context, req *pb.CreateJobOutcomeSummaryRequest) (*pb.CreateJobOutcomeSummaryResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("job outcome summary data is required")
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

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create job outcome summary: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	summary := &pb.JobOutcomeSummary{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, summary); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.CreateJobOutcomeSummaryResponse{
		Success: true,
		Data:    []*pb.JobOutcomeSummary{summary},
	}, nil
}

// ReadJobOutcomeSummary retrieves a job_outcome_summary record by ID
func (r *PostgresJobOutcomeSummaryRepository) ReadJobOutcomeSummary(ctx context.Context, req *pb.ReadJobOutcomeSummaryRequest) (*pb.ReadJobOutcomeSummaryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job outcome summary ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read job outcome summary: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	summary := &pb.JobOutcomeSummary{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, summary); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	// Staff row-scope (Phase 4): the generic dbOps.Read by id has no WHERE seam,
	// so guard post-read — a STAFF principal may only read a summary it issued
	// (issued_by). Fail-closed → not-found on an empty session staff.id or a
	// summary issued by another staff member. Non-staff principals unaffected.
	if staffID, ok := principalscope.StaffRowScope(ctx); ok {
		if staffID == "" || summary.IssuedBy != staffID {
			return nil, fmt.Errorf("job outcome summary with ID '%s' not found", req.Data.Id)
		}
	}

	return &pb.ReadJobOutcomeSummaryResponse{
		Success: true,
		Data:    []*pb.JobOutcomeSummary{summary},
	}, nil
}

// UpdateJobOutcomeSummary updates a job_outcome_summary record
func (r *PostgresJobOutcomeSummaryRepository) UpdateJobOutcomeSummary(ctx context.Context, req *pb.UpdateJobOutcomeSummaryRequest) (*pb.UpdateJobOutcomeSummaryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job outcome summary ID is required")
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

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update job outcome summary: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	summary := &pb.JobOutcomeSummary{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, summary); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.UpdateJobOutcomeSummaryResponse{
		Success: true,
		Data:    []*pb.JobOutcomeSummary{summary},
	}, nil
}

// DeleteJobOutcomeSummary deletes a job_outcome_summary record (soft delete)
func (r *PostgresJobOutcomeSummaryRepository) DeleteJobOutcomeSummary(ctx context.Context, req *pb.DeleteJobOutcomeSummaryRequest) (*pb.DeleteJobOutcomeSummaryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job outcome summary ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete job outcome summary: %w", err)
	}

	return &pb.DeleteJobOutcomeSummaryResponse{
		Success: true,
	}, nil
}

// ListJobOutcomeSummarys lists job_outcome_summary records with optional filters
func (r *PostgresJobOutcomeSummaryRepository) ListJobOutcomeSummarys(ctx context.Context, req *pb.ListJobOutcomeSummarysRequest) (*pb.ListJobOutcomeSummarysResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list job outcome summarys: %w", err)
	}

	var summaries []*pb.JobOutcomeSummary
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal job_outcome_summary row: %v", err)
			continue
		}

		summary := &pb.JobOutcomeSummary{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, summary); err != nil {
			log.Printf("WARN: protojson unmarshal job_outcome_summary: %v", err)
			continue
		}
		summaries = append(summaries, summary)
	}

	return &pb.ListJobOutcomeSummarysResponse{
		Success: true,
		Data:    summaries,
	}, nil
}

var jobOutcomeSummarySortableSQLCols = []string{
	"id", "job_id", "summary_type", "overall_determination", "scoring_method",
	"summary_score", "total_criteria_count", "pass_count", "fail_count",
	"conditional_count", "deferred_count", "na_count", "narrative",
	"issued_by", "issued_date", "valid_until_date", "supersedes_id",
	"attachment_ids", "active", "date_created", "date_modified",
}

// jobOutcomeSummaryProjection is the schema-faithful column list projected by
// every custom-SQL read (the enriched list adds a total; the single-row reads
// project it verbatim). Sourced from a single place so the scanner's dests()
// order stays in lockstep across all three call sites.
const jobOutcomeSummaryProjection = `
			jos.id, jos.job_id, jos.summary_type, jos.overall_determination,
			jos.scoring_method, jos.summary_score, jos.total_criteria_count,
			jos.pass_count, jos.fail_count, jos.conditional_count,
			jos.deferred_count, jos.na_count, jos.narrative,
			jos.issued_by, jos.issued_date, jos.valid_until_date,
			jos.supersedes_id, jos.attachment_ids, jos.active,
			jos.date_created, jos.date_modified,
			jos.scoring_scheme_id, jos.scaled_score, jos.scaled_label,
			jos.workspace_id, jos.client_id`

// sessionWorkspaceID returns the session identity's workspace id, or "" when no
// identity is present. Sourcing from the SESSION (never a request param) is the
// multi-tenancy invariant; the empty fallback is FAIL-CLOSED — a caller with no
// workspace binds jos.workspace_id = '' which matches no real row (every
// job_outcome_summary carries a non-empty workspace_id). Mirrors the
// outcome_matrix_query.go workspace sourcing (FromContext, not identity.Must,
// so a missing identity fails closed rather than panicking).
func sessionWorkspaceID(ctx context.Context) string {
	if id, ok := identity.FromContext(ctx); ok && id != nil {
		return id.WorkspaceID
	}
	return ""
}

// jobOutcomeSummaryListPageDataSQL builds the paginated list query. The
// workspace predicate ($4) and the staff clause ($5, when a STAFF principal is
// active) both live inside the enriched CTE so COUNT(*) OVER () matches the
// scoped set. HAZ-02 close: jos.workspace_id = $4 is always present.
func jobOutcomeSummaryListPageDataSQL(josColumns, staffClause, orderByClause string) string {
	return `
		WITH enriched AS (
			SELECT ` + josColumns + `
			FROM ` + entityid.JobOutcomeSummary + ` jos
			WHERE jos.active = true
			  AND jos.workspace_id = $4
			  AND ($1::text IS NULL OR $1::text = '' OR
			       jos.narrative ILIKE $1)` + staffClause + `
		)
		-- A3 (Q-PAGE-COUNT default tier): COUNT(*) OVER () computes the total in the
		-- same scan as the page rows (the prior counted CTE forced a second scan).
		SELECT
			e.*, COUNT(*) OVER () AS total
		FROM enriched e
		` + orderByClause + `
		LIMIT $2 OFFSET $3;
	`
}

// jobOutcomeSummaryByColumnSQL builds a single-row read filtered by whereExpr
// ($1), always workspace-bound ($2, HAZ-02 close), with an optional staff clause
// ($3) and an optional trailing suffix (e.g. ORDER BY / LIMIT).
func jobOutcomeSummaryByColumnSQL(whereExpr, staffClause, suffix string) string {
	q := `
		SELECT` + jobOutcomeSummaryProjection + `
		FROM ` + entityid.JobOutcomeSummary + ` jos
		WHERE ` + whereExpr + ` AND jos.active = true AND jos.workspace_id = $2` + staffClause
	if suffix != "" {
		q += "\n\t\t" + suffix
	}
	return q + "\n\t"
}

// GetJobOutcomeSummaryListPageData retrieves job outcome summaries with pagination
func (r *PostgresJobOutcomeSummaryRepository) GetJobOutcomeSummaryListPageData(
	ctx context.Context,
	req *pb.GetJobOutcomeSummaryListPageDataRequest,
) (*pb.GetJobOutcomeSummaryListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get job outcome summary list page data request is required")
	}

	searchPattern := ""
	if req.Search != nil && req.Search.Query != "" {
		searchPattern = "%" + req.Search.Query + "%"
	}

	limit := int32(50)
	offset := int32(0)
	page := int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil {
			if offsetPag.Page > 0 {
				page = offsetPag.Page
				offset = (page - 1) * limit
			}
		}
	}

	// Sort — fail-closed against the per-entity whitelist (A2 guard). The outer
	// SELECT projects the enriched columns unprefixed (e.*), so the ORDER BY
	// references unprefixed whitelist columns. An unknown column errors instead
	// of being interpolated verbatim into ORDER BY.
	orderByClause, err := postgresCore.BuildOrderBy(jobOutcomeSummarySortableSQLCols, req.GetSort(), "date_created DESC")
	if err != nil {
		return nil, err
	}

	josColumns := `
		jos.id, jos.job_id, jos.summary_type, jos.overall_determination,
		jos.scoring_method, jos.summary_score, jos.total_criteria_count,
		jos.pass_count, jos.fail_count, jos.conditional_count,
		jos.deferred_count, jos.na_count, jos.narrative,
		jos.issued_by, jos.issued_date, jos.valid_until_date,
		jos.supersedes_id, jos.attachment_ids, jos.active,
		jos.date_created, jos.date_modified,
		jos.scoring_scheme_id, jos.scaled_score, jos.scaled_label,
		jos.workspace_id, jos.client_id
	`

	// HAZ-02 close (Q-SEC-7): bind the workspace ($4, session identity — never a
	// request param). Fail-closed: an empty workspace binds jos.workspace_id = ''
	// which matches no real row. Staff row-scope shifts to $5.
	workspaceID := sessionWorkspaceID(ctx)
	staffClause, staffArgs := principalscope.StaffScopeClause(ctx, "jos.issued_by", 5)
	query := jobOutcomeSummaryListPageDataSQL(josColumns, staffClause, orderByClause)

	rows, err := r.db.QueryContext(ctx, query, append([]any{searchPattern, limit, offset, workspaceID}, staffArgs...)...)
	if err != nil {
		return nil, fmt.Errorf("failed to query job outcome summary list page data: %w", err)
	}
	defer rows.Close()

	var summaries []*pb.JobOutcomeSummary
	var totalCount int64

	for rows.Next() {
		summary, cnt, err := scanJobOutcomeSummaryRowWithTotal(rows)
		if err != nil {
			return nil, err
		}
		totalCount = cnt
		summaries = append(summaries, summary)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating job outcome summary rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &pb.GetJobOutcomeSummaryListPageDataResponse{
		JobOutcomeSummaryList: summaries,
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

// GetJobOutcomeSummaryItemPageData retrieves a single job outcome summary with enriched data
func (r *PostgresJobOutcomeSummaryRepository) GetJobOutcomeSummaryItemPageData(
	ctx context.Context,
	req *pb.GetJobOutcomeSummaryItemPageDataRequest,
) (*pb.GetJobOutcomeSummaryItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get job outcome summary item page data request is required")
	}
	if req.JobOutcomeSummaryId == "" {
		return nil, fmt.Errorf("job outcome summary ID is required")
	}

	// HAZ-02 close (Q-SEC-7): bind the workspace ($2, session identity). Staff
	// row-scope shifts to $3 (a STAFF principal may only read a summary it issued).
	workspaceID := sessionWorkspaceID(ctx)
	staffClause, staffArgs := principalscope.StaffScopeClause(ctx, "jos.issued_by", 3)
	query := jobOutcomeSummaryByColumnSQL("jos.id = $1", staffClause, "")

	row := r.db.QueryRowContext(ctx, query, append([]any{req.JobOutcomeSummaryId, workspaceID}, staffArgs...)...)

	summary, err := scanJobOutcomeSummarySingleRow(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job outcome summary with ID '%s' not found", req.JobOutcomeSummaryId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query job outcome summary item page data: %w", err)
	}

	return &pb.GetJobOutcomeSummaryItemPageDataResponse{
		JobOutcomeSummary: summary,
		Success:           true,
	}, nil
}

// GetByJob retrieves the latest job outcome summary for a given job
func (r *PostgresJobOutcomeSummaryRepository) GetByJob(
	ctx context.Context,
	req *pb.GetJobOutcomeSummaryByJobRequest,
) (*pb.GetJobOutcomeSummaryByJobResponse, error) {
	if req == nil || req.JobId == "" {
		return nil, fmt.Errorf("job ID is required")
	}

	// HAZ-02 close (Q-SEC-7): bind the workspace ($2, session identity). Staff
	// row-scope shifts to $3 (fail-closed → empty result via sql.ErrNoRows).
	workspaceID := sessionWorkspaceID(ctx)
	staffClause, staffArgs := principalscope.StaffScopeClause(ctx, "jos.issued_by", 3)
	query := jobOutcomeSummaryByColumnSQL("jos.job_id = $1", staffClause, "ORDER BY jos.date_created DESC\n\t\tLIMIT 1")

	row := r.db.QueryRowContext(ctx, query, append([]any{req.JobId, workspaceID}, staffArgs...)...)

	summary, err := scanJobOutcomeSummarySingleRow(row)
	if err == sql.ErrNoRows {
		return &pb.GetJobOutcomeSummaryByJobResponse{
			Success: true,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get job outcome summary by job: %w", err)
	}

	return &pb.GetJobOutcomeSummaryByJobResponse{
		JobOutcomeSummary: summary,
		Success:           true,
	}, nil
}

// josFields is the schema-faithful scan destination set for job_outcome_summary.
// Every column except id/active is nullable in the table, and issued_date /
// date_created / date_modified are timestamptz — so the dests must be
// NullString / NullInt32 / NullTime or Scan errors on the FIRST real row (the
// prior plain-string/int32/NullInt64-for-timestamptz dests could never read a
// written row; job_outcome_summary was empty so the bug was latent). Mirrors
// the fixed phase_outcome_summary scanner.
type josFields struct {
	id                   string
	jobID                sql.NullString
	summaryType          sql.NullString
	overallDetermination sql.NullString
	scoringMethod        sql.NullString
	summaryScore         sql.NullFloat64
	totalCriteriaCount   sql.NullInt32
	passCount            sql.NullInt32
	failCount            sql.NullInt32
	conditionalCount     sql.NullInt32
	deferredCount        sql.NullInt32
	naCount              sql.NullInt32
	narrative            sql.NullString
	issuedBy             sql.NullString
	issuedDate           sql.NullTime
	validUntilDate       sql.NullString
	supersedesId         sql.NullString
	attachmentIds        sql.NullString
	active               bool
	dateCreated          sql.NullTime
	dateModified         sql.NullTime
	scoringSchemeId      sql.NullString
	scaledScore          sql.NullFloat64
	scaledLabel          sql.NullString
	workspaceId          sql.NullString
	clientId             sql.NullString
}

// dests returns the scan-target pointers in projection order (see the three
// query column lists). Optionally appends a trailing *int64 total sink.
func (f *josFields) dests(total *int64) []any {
	d := []any{
		&f.id, &f.jobID, &f.summaryType, &f.overallDetermination,
		&f.scoringMethod, &f.summaryScore, &f.totalCriteriaCount,
		&f.passCount, &f.failCount, &f.conditionalCount,
		&f.deferredCount, &f.naCount, &f.narrative,
		&f.issuedBy, &f.issuedDate, &f.validUntilDate,
		&f.supersedesId, &f.attachmentIds, &f.active,
		&f.dateCreated, &f.dateModified,
		&f.scoringSchemeId, &f.scaledScore, &f.scaledLabel,
		&f.workspaceId, &f.clientId,
	}
	if total != nil {
		d = append(d, total)
	}
	return d
}

// scanJobOutcomeSummaryRowWithTotal scans a row with a trailing total count column
func scanJobOutcomeSummaryRowWithTotal(rows *sql.Rows) (*pb.JobOutcomeSummary, int64, error) {
	var f josFields
	var total int64
	if err := rows.Scan(f.dests(&total)...); err != nil {
		return nil, 0, fmt.Errorf("failed to scan job outcome summary row: %w", err)
	}
	return buildJobOutcomeSummary(&f), total, nil
}

// scanJobOutcomeSummarySingleRow scans a single sql.Row into a JobOutcomeSummary proto
func scanJobOutcomeSummarySingleRow(row *sql.Row) (*pb.JobOutcomeSummary, error) {
	var f josFields
	if err := row.Scan(f.dests(nil)...); err != nil {
		return nil, err
	}
	return buildJobOutcomeSummary(&f), nil
}

func buildJobOutcomeSummary(f *josFields) *pb.JobOutcomeSummary {
	summary := &pb.JobOutcomeSummary{
		Id:                 f.id,
		Active:             f.active,
		JobId:              f.jobID.String,
		TotalCriteriaCount: f.totalCriteriaCount.Int32,
		PassCount:          f.passCount.Int32,
		FailCount:          f.failCount.Int32,
		ConditionalCount:   f.conditionalCount.Int32,
		DeferredCount:      f.deferredCount.Int32,
		NaCount:            f.naCount.Int32,
		IssuedBy:           f.issuedBy.String,
	}
	if f.summaryType.Valid {
		summary.SummaryType = enumspb.SummaryType(enumspb.SummaryType_value[f.summaryType.String])
	}
	if f.overallDetermination.Valid {
		summary.OverallDetermination = enumspb.OverallDetermination(enumspb.OverallDetermination_value[f.overallDetermination.String])
	}
	if f.scoringMethod.Valid {
		summary.ScoringMethod = enumspb.ScoringMethod(enumspb.ScoringMethod_value[f.scoringMethod.String])
	}
	if f.summaryScore.Valid {
		summary.SummaryScore = &f.summaryScore.Float64
	}
	if f.narrative.Valid {
		summary.Narrative = &f.narrative.String
	}
	if f.issuedDate.Valid {
		ms := f.issuedDate.Time.UnixMilli()
		summary.IssuedDate = &ms
	}
	if f.validUntilDate.Valid {
		summary.ValidUntilDate = &f.validUntilDate.String
	}
	if f.supersedesId.Valid {
		summary.SupersedesId = &f.supersedesId.String
	}
	if f.attachmentIds.Valid && f.attachmentIds.String != "" {
		var ids []string
		if err := json.Unmarshal([]byte(f.attachmentIds.String), &ids); err == nil {
			summary.AttachmentIds = ids
		}
	}
	if f.dateCreated.Valid {
		ms := f.dateCreated.Time.UnixMilli()
		summary.DateCreated = &ms
		dcStr := f.dateCreated.Time.Format(time.RFC3339)
		summary.DateCreatedString = &dcStr
	}
	if f.dateModified.Valid {
		ms := f.dateModified.Time.UnixMilli()
		summary.DateModified = &ms
		dmStr := f.dateModified.Time.Format(time.RFC3339)
		summary.DateModifiedString = &dmStr
	}
	if f.scoringSchemeId.Valid {
		summary.ScoringSchemeId = &f.scoringSchemeId.String
	}
	if f.scaledScore.Valid {
		summary.ScaledScore = &f.scaledScore.Float64
	}
	if f.scaledLabel.Valid {
		summary.ScaledLabel = &f.scaledLabel.String
	}
	if f.workspaceId.Valid {
		summary.WorkspaceId = f.workspaceId.String
	}
	if f.clientId.Valid {
		summary.ClientId = &f.clientId.String
	}
	return summary
}

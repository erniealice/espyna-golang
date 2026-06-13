//go:build sqlserver

package operation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/erniealice/espyna-golang/shared/identity"
	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	infraports "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
)

// jobSortableSQLCols lists the SQL column names that are safe to sort by in
// GetJobListPageData. Routed through core.BuildOrderBy (A2 guard) — an
// unrecognised column is rejected loudly before query execution.
//
// SQL Server: columns are bracket-quoted automatically by BuildOrderBy; no
// quoting needed here.
var jobSortableSQLCols = []string{
	"j.date_created",
	"j.date_modified",
	"j.name",
	"j.status",
}

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.Job, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver job repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerJobRepository(dbOps, tableName), nil
	})
}

// SQLServerJobRepository implements job CRUD operations using SQL Server.
type SQLServerJobRepository struct {
	pb.UnimplementedJobDomainServiceServer
	dbOps        interfaces.DatabaseOperation
	tableName    string
	auditService infraports.AuditService
}

// NewSQLServerJobRepository creates a new SQL Server job repository.
func NewSQLServerJobRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.JobDomainServiceServer {
	if tableName == "" {
		tableName = "job"
	}
	return &SQLServerJobRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// WithAuditService returns the repository with an audit service attached.
func (r *SQLServerJobRepository) WithAuditService(svc infraports.AuditService) *SQLServerJobRepository {
	r.auditService = svc
	return r
}

// getExec extracts a DBExecutor from the dbOps wrapper.
func (r *SQLServerJobRepository) getExec(ctx context.Context) dbExecutor {
	return r.dbOps.(executorProvider).GetExecutor(ctx)
}

// CreateJob creates a new job record.
func (r *SQLServerJobRepository) CreateJob(ctx context.Context, req *pb.CreateJobRequest) (*pb.CreateJobResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("job data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	job := &pb.Job{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.CreateJobResponse{
		Success: true,
		Data:    []*pb.Job{job},
	}, nil
}

// ReadJob retrieves a job record by ID.
func (r *SQLServerJobRepository) ReadJob(ctx context.Context, req *pb.ReadJobRequest) (*pb.ReadJobResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read job: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	job := &pb.Job{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.ReadJobResponse{
		Success: true,
		Data:    []*pb.Job{job},
	}, nil
}

// UpdateJob updates a job record.
func (r *SQLServerJobRepository) UpdateJob(ctx context.Context, req *pb.UpdateJobRequest) (*pb.UpdateJobResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job ID is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update job: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	job := &pb.Job{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.UpdateJobResponse{
		Success: true,
		Data:    []*pb.Job{job},
	}, nil
}

// DeleteJob deletes a job record (soft delete).
func (r *SQLServerJobRepository) DeleteJob(ctx context.Context, req *pb.DeleteJobRequest) (*pb.DeleteJobResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete job: %w", err)
	}

	return &pb.DeleteJobResponse{Success: true}, nil
}

// ListJobs lists job records with optional filters.
func (r *SQLServerJobRepository) ListJobs(ctx context.Context, req *pb.ListJobsRequest) (*pb.ListJobsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	var jobs []*pb.Job
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal job row: %v", err)
			continue
		}

		job := &pb.Job{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, job); err != nil {
			log.Printf("WARN: protojson unmarshal job: %v", err)
			continue
		}
		jobs = append(jobs, job)
	}

	return &pb.ListJobsResponse{
		Success: true,
		Data:    jobs,
	}, nil
}

// GetJobListPageData retrieves jobs with pagination, filtering, sorting, and search.
// CRITICAL: Always filters by workspace_id for multi-tenancy.
//
// SQL Server differences from the postgres gold standard:
//   - $N → @pN placeholders.
//   - active = true → active = 1  (SQL Server BIT).
//   - ILIKE → LIKE (SQL Server default CI collation).
//   - Pagination: LIMIT n OFFSET m → ORDER BY … OFFSET @pM ROWS FETCH NEXT @pN ROWS ONLY.
//     SQL Server requires ORDER BY before OFFSET/FETCH; BuildOrderBy guarantees a fallback.
//   - COUNT(*) OVER () window function is retained — SQL Server 2017+ supports it.
func (r *SQLServerJobRepository) GetJobListPageData(
	ctx context.Context,
	req *pb.GetJobListPageDataRequest,
) (*pb.GetJobListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get job list page data request is required")
	}

	workspaceID := identity.Must(ctx).WorkspaceID

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

	// A2 sort guard: BuildOrderBy bracket-quotes the column and validates against whitelist.
	orderByClause, err := sqlserverCore.BuildOrderBy(jobSortableSQLCols, req.GetSort(), "j.date_created DESC")
	if err != nil {
		return nil, err
	}

	// Build filter/search WHERE clauses. @p1 = workspaceID, @p2 = search. Start filter params at @p3.
	searchFields := []string{"j.name"}
	filterClauses, filterArgs, nextIdx := sqlserverCore.BuildFilterWhere(req.Filters, req.Search, searchFields, 3)

	whereSQL := "WHERE j.workspace_id = @p1 AND j.active = 1"
	if len(filterClauses) > 0 {
		whereSQL += " AND " + strings.Join(filterClauses, " AND ")
	}

	offsetIdx := nextIdx
	limitIdx := nextIdx + 1
	queryArgs := []any{workspaceID}
	queryArgs = append(queryArgs, filterArgs...)
	queryArgs = append(queryArgs, offset, limit)

	query := fmt.Sprintf(`
		WITH enriched AS (
			SELECT
				j.id,
				j.date_created,
				j.date_modified,
				j.active,
				j.name,
				j.job_template_id,
				j.origin_type,
				j.origin_id,
				j.client_id,
				j.demand_type,
				j.fulfillment_type,
				j.cost_flow_type,
				j.billing_rule_type,
				j.status,
				j.approval_status,
				j.posting_status,
				j.billing_status,
				j.location_id,
				j.created_by,
				j.cycle_index,
				j.cycle_period_start,
				j.cycle_period_end,
				COUNT(*) OVER() AS total_count
			FROM job j
			%s
		)
		SELECT * FROM enriched
		%s OFFSET @p%d ROWS FETCH NEXT @p%d ROWS ONLY;
	`, whereSQL, orderByClause, offsetIdx, limitIdx)

	exec := r.getExec(ctx)
	rows, err := exec.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query job list page data: %w", err)
	}
	defer rows.Close()

	var jobs []*pb.Job
	var totalCount int64

	for rows.Next() {
		var (
			id               string
			dateCreated      time.Time
			dateModified     time.Time
			active           bool
			name             string
			jobTemplateID    sql.NullString
			originType       sql.NullString
			originID         sql.NullString
			clientID         sql.NullString
			demandType       sql.NullString
			fulfillmentType  sql.NullString
			costFlowType     sql.NullString
			billingRuleType  sql.NullString
			status           sql.NullString
			approvalStatus   sql.NullString
			postingStatus    sql.NullString
			billingStatus    sql.NullString
			locationID       sql.NullString
			createdBy        sql.NullString
			cycleIndex       sql.NullInt32
			cyclePeriodStart sql.NullString
			cyclePeriodEnd   sql.NullString
			total            int64
		)

		if err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&name,
			&jobTemplateID,
			&originType,
			&originID,
			&clientID,
			&demandType,
			&fulfillmentType,
			&costFlowType,
			&billingRuleType,
			&status,
			&approvalStatus,
			&postingStatus,
			&billingStatus,
			&locationID,
			&createdBy,
			&cycleIndex,
			&cyclePeriodStart,
			&cyclePeriodEnd,
			&total,
		); err != nil {
			return nil, fmt.Errorf("failed to scan job row: %w", err)
		}

		totalCount = total

		job := &pb.Job{Id: id, Active: active, Name: name}

		if jobTemplateID.Valid {
			job.JobTemplateId = &jobTemplateID.String
		}
		if originID.Valid {
			job.OriginId = &originID.String
		}
		if clientID.Valid {
			job.ClientId = &clientID.String
		}
		if locationID.Valid {
			job.LocationId = &locationID.String
		}
		if createdBy.Valid {
			job.CreatedBy = &createdBy.String
		}
		if cycleIndex.Valid {
			job.CycleIndex = &cycleIndex.Int32
		}
		if cyclePeriodStart.Valid {
			job.CyclePeriodStart = &cyclePeriodStart.String
		}
		if cyclePeriodEnd.Valid {
			job.CyclePeriodEnd = &cyclePeriodEnd.String
		}

		if originType.Valid {
			if v, ok := enumspb.OriginType_value[originType.String]; ok {
				job.OriginType = enumspb.OriginType(v)
			}
		}
		if demandType.Valid {
			if v, ok := enumspb.DemandType_value[demandType.String]; ok {
				job.DemandType = enumspb.DemandType(v)
			}
		}
		if fulfillmentType.Valid {
			if v, ok := enumspb.FulfillmentType_value[fulfillmentType.String]; ok {
				job.FulfillmentType = enumspb.FulfillmentType(v)
			}
		}
		if costFlowType.Valid {
			if v, ok := enumspb.CostFlowType_value[costFlowType.String]; ok {
				job.CostFlowType = enumspb.CostFlowType(v)
			}
		}
		if billingRuleType.Valid {
			if v, ok := enumspb.BillingRuleType_value[billingRuleType.String]; ok {
				job.BillingRuleType = enumspb.BillingRuleType(v)
			}
		}
		if status.Valid {
			if v, ok := enumspb.JobStatus_value[status.String]; ok {
				job.Status = enumspb.JobStatus(v)
			}
		}
		if approvalStatus.Valid {
			if v, ok := enumspb.ApprovalStatus_value[approvalStatus.String]; ok {
				job.ApprovalStatus = enumspb.ApprovalStatus(v)
			}
		}
		if postingStatus.Valid {
			if v, ok := enumspb.PostingStatus_value[postingStatus.String]; ok {
				job.PostingStatus = enumspb.PostingStatus(v)
			}
		}
		if billingStatus.Valid {
			if v, ok := enumspb.BillingStatus_value[billingStatus.String]; ok {
				job.BillingStatus = enumspb.BillingStatus(v)
			}
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			job.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			job.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			job.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			job.DateModifiedString = &dmStr
		}

		jobs = append(jobs, job)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating job rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &pb.GetJobListPageDataResponse{
		JobList: jobs,
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

// GetJobItemPageData retrieves a single job with enriched data.
// CRITICAL: Always filters by workspace_id for multi-tenancy.
//
// SQL Server differences:
//   - $1/$2 → @p1/@p2.
//   - active = true → active = 1.
func (r *SQLServerJobRepository) GetJobItemPageData(
	ctx context.Context,
	req *pb.GetJobItemPageDataRequest,
) (*pb.GetJobItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get job item page data request is required")
	}
	if req.JobId == "" {
		return nil, fmt.Errorf("job ID is required")
	}

	workspaceID := identity.Must(ctx).WorkspaceID

	const query = `
		SELECT
			j.id,
			j.date_created,
			j.date_modified,
			j.active,
			j.name,
			j.job_template_id,
			j.origin_type,
			j.origin_id,
			j.client_id,
			j.demand_type,
			j.fulfillment_type,
			j.cost_flow_type,
			j.billing_rule_type,
			j.status,
			j.approval_status,
			j.posting_status,
			j.billing_status,
			j.location_id,
			j.created_by,
			j.cycle_index,
			j.cycle_period_start,
			j.cycle_period_end
		FROM job j
		WHERE j.id = @p1 AND j.workspace_id = @p2 AND j.active = 1
	`

	exec := r.getExec(ctx)
	row := exec.QueryRowContext(ctx, query, req.JobId, workspaceID)

	var (
		id               string
		dateCreated      time.Time
		dateModified     time.Time
		active           bool
		name             string
		jobTemplateID    sql.NullString
		originType       sql.NullString
		originID         sql.NullString
		clientID         sql.NullString
		demandType       sql.NullString
		fulfillmentType  sql.NullString
		costFlowType     sql.NullString
		billingRuleType  sql.NullString
		status           sql.NullString
		approvalStatus   sql.NullString
		postingStatus    sql.NullString
		billingStatus    sql.NullString
		locationID       sql.NullString
		createdBy        sql.NullString
		cycleIndex       sql.NullInt32
		cyclePeriodStart sql.NullString
		cyclePeriodEnd   sql.NullString
	)

	err := row.Scan(
		&id, &dateCreated, &dateModified, &active, &name,
		&jobTemplateID, &originType, &originID, &clientID,
		&demandType, &fulfillmentType, &costFlowType, &billingRuleType,
		&status, &approvalStatus, &postingStatus, &billingStatus,
		&locationID, &createdBy, &cycleIndex, &cyclePeriodStart, &cyclePeriodEnd,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job with ID '%s' not found", req.JobId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query job item page data: %w", err)
	}

	job := &pb.Job{Id: id, Active: active, Name: name}

	if jobTemplateID.Valid {
		job.JobTemplateId = &jobTemplateID.String
	}
	if originID.Valid {
		job.OriginId = &originID.String
	}
	if clientID.Valid {
		job.ClientId = &clientID.String
	}
	if locationID.Valid {
		job.LocationId = &locationID.String
	}
	if createdBy.Valid {
		job.CreatedBy = &createdBy.String
	}
	if cycleIndex.Valid {
		job.CycleIndex = &cycleIndex.Int32
	}
	if cyclePeriodStart.Valid {
		job.CyclePeriodStart = &cyclePeriodStart.String
	}
	if cyclePeriodEnd.Valid {
		job.CyclePeriodEnd = &cyclePeriodEnd.String
	}

	if originType.Valid {
		if v, ok := enumspb.OriginType_value[originType.String]; ok {
			job.OriginType = enumspb.OriginType(v)
		}
	}
	if demandType.Valid {
		if v, ok := enumspb.DemandType_value[demandType.String]; ok {
			job.DemandType = enumspb.DemandType(v)
		}
	}
	if fulfillmentType.Valid {
		if v, ok := enumspb.FulfillmentType_value[fulfillmentType.String]; ok {
			job.FulfillmentType = enumspb.FulfillmentType(v)
		}
	}
	if costFlowType.Valid {
		if v, ok := enumspb.CostFlowType_value[costFlowType.String]; ok {
			job.CostFlowType = enumspb.CostFlowType(v)
		}
	}
	if billingRuleType.Valid {
		if v, ok := enumspb.BillingRuleType_value[billingRuleType.String]; ok {
			job.BillingRuleType = enumspb.BillingRuleType(v)
		}
	}
	if status.Valid {
		if v, ok := enumspb.JobStatus_value[status.String]; ok {
			job.Status = enumspb.JobStatus(v)
		}
	}
	if approvalStatus.Valid {
		if v, ok := enumspb.ApprovalStatus_value[approvalStatus.String]; ok {
			job.ApprovalStatus = enumspb.ApprovalStatus(v)
		}
	}
	if postingStatus.Valid {
		if v, ok := enumspb.PostingStatus_value[postingStatus.String]; ok {
			job.PostingStatus = enumspb.PostingStatus(v)
		}
	}
	if billingStatus.Valid {
		if v, ok := enumspb.BillingStatus_value[billingStatus.String]; ok {
			job.BillingStatus = enumspb.BillingStatus(v)
		}
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		job.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		job.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		job.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		job.DateModifiedString = &dmStr
	}

	return &pb.GetJobItemPageDataResponse{
		Job:     job,
		Success: true,
	}, nil
}

// GetJobsByClient retrieves all jobs for a given client.
// SQL Server: @p1 placeholder; ORDER BY date_created DESC.
func (r *SQLServerJobRepository) GetJobsByClient(
	ctx context.Context,
	req *pb.GetJobsByClientRequest,
) (*pb.GetJobsByClientResponse, error) {
	if req == nil || req.ClientId == "" {
		return nil, fmt.Errorf("client ID is required")
	}

	query := fmt.Sprintf(`
		SELECT id, date_created, date_modified, active, name,
		       job_template_id, origin_type, origin_id, client_id,
		       demand_type, fulfillment_type, cost_flow_type, billing_rule_type,
		       status, approval_status, posting_status, billing_status,
		       location_id, created_by, parent_job_id,
		       cycle_index, cycle_period_start, cycle_period_end
		FROM %s
		WHERE client_id = @p1 AND active = 1
		ORDER BY date_created DESC
	`, r.tableName)

	exec := r.getExec(ctx)
	rows, err := exec.QueryContext(ctx, query, req.ClientId)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs by client: %w", err)
	}
	defer rows.Close()

	jobs, err := r.scanJobRows(rows)
	if err != nil {
		return nil, err
	}

	return &pb.GetJobsByClientResponse{Jobs: jobs, Success: true}, nil
}

// GetJobsByOrigin retrieves all jobs for a given origin (type + ID).
// SQL Server: NULLS FIRST is not valid; use IS NULL ordering trick or rely on SQL Server's default.
func (r *SQLServerJobRepository) GetJobsByOrigin(
	ctx context.Context,
	req *pb.GetJobsByOriginRequest,
) (*pb.GetJobsByOriginResponse, error) {
	if req == nil || req.OriginId == "" {
		return nil, fmt.Errorf("origin ID is required")
	}

	// SQL Server: parent_job_id IS NULL sorts NULLs first via CASE WHEN.
	query := fmt.Sprintf(`
		SELECT id, date_created, date_modified, active, name,
		       job_template_id, origin_type, origin_id, client_id,
		       demand_type, fulfillment_type, cost_flow_type, billing_rule_type,
		       status, approval_status, posting_status, billing_status,
		       location_id, created_by, parent_job_id,
		       cycle_index, cycle_period_start, cycle_period_end
		FROM %s
		WHERE origin_type = @p1 AND origin_id = @p2 AND active = 1
		ORDER BY CASE WHEN parent_job_id IS NULL THEN 0 ELSE 1 END, date_created ASC
	`, r.tableName)

	exec := r.getExec(ctx)
	rows, err := exec.QueryContext(ctx, query, req.OriginType.String(), req.OriginId)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs by origin: %w", err)
	}
	defer rows.Close()

	jobs, err := r.scanJobRows(rows)
	if err != nil {
		return nil, err
	}

	return &pb.GetJobsByOriginResponse{Jobs: jobs, Success: true}, nil
}

// UpdateJobStatus transitions a job to a new status.
// SQL Server: uses OUTPUT inserted.id instead of RETURNING.
func (r *SQLServerJobRepository) UpdateJobStatus(
	ctx context.Context,
	req *pb.UpdateJobStatusRequest,
) (*pb.UpdateJobStatusResponse, error) {
	if req == nil || req.JobId == "" {
		return nil, fmt.Errorf("job ID is required")
	}

	newStatus := req.Status.String()

	query := fmt.Sprintf(`
		UPDATE %s SET status = @p1, date_modified = GETUTCDATE()
		OUTPUT inserted.id
		WHERE id = @p2 AND active = 1
	`, r.tableName)

	exec := r.getExec(ctx)
	var id string
	err := exec.QueryRowContext(ctx, query, newStatus, req.JobId).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job not found with ID: %s", req.JobId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update job status: %w", err)
	}

	if r.auditService != nil {
		_ = infraports.DiffAndLog(ctx, r.auditService, infraports.DiffAndLogRequest{
			EntityType:     "job",
			EntityID:       id,
			Domain:         "fayna",
			Action:         2,
			PermissionCode: "job:update",
			UseCase:        "UpdateJobStatus",
			MethodName:     "UpdateJobStatus",
			NewData:        map[string]any{"status": newStatus},
		})
	}

	readResp, err := r.ReadJob(ctx, &pb.ReadJobRequest{Data: &pb.Job{Id: id}})
	if err != nil {
		return nil, err
	}

	var job *pb.Job
	if len(readResp.Data) > 0 {
		job = readResp.Data[0]
	}

	return &pb.UpdateJobStatusResponse{Job: job, Success: true}, nil
}

// scanJobRows is a helper to scan multiple job rows from a query result.
func (r *SQLServerJobRepository) scanJobRows(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
}) ([]*pb.Job, error) {
	var jobs []*pb.Job

	for rows.Next() {
		var (
			id               string
			dateCreated      time.Time
			dateModified     time.Time
			active           bool
			name             string
			jobTemplateID    sql.NullString
			originType       sql.NullString
			originID         sql.NullString
			clientID         sql.NullString
			demandType       sql.NullString
			fulfillmentType  sql.NullString
			costFlowType     sql.NullString
			billingRuleType  sql.NullString
			status           sql.NullString
			approvalStatus   sql.NullString
			postingStatus    sql.NullString
			billingStatus    sql.NullString
			locationID       sql.NullString
			createdBy        sql.NullString
			parentJobID      sql.NullString
			cycleIndex       sql.NullInt32
			cyclePeriodStart sql.NullString
			cyclePeriodEnd   sql.NullString
		)

		if err := rows.Scan(
			&id, &dateCreated, &dateModified, &active, &name,
			&jobTemplateID, &originType, &originID, &clientID,
			&demandType, &fulfillmentType, &costFlowType, &billingRuleType,
			&status, &approvalStatus, &postingStatus, &billingStatus,
			&locationID, &createdBy, &parentJobID,
			&cycleIndex, &cyclePeriodStart, &cyclePeriodEnd,
		); err != nil {
			return nil, fmt.Errorf("failed to scan job row: %w", err)
		}

		job := &pb.Job{Id: id, Active: active, Name: name}

		if jobTemplateID.Valid {
			job.JobTemplateId = &jobTemplateID.String
		}
		if originID.Valid {
			job.OriginId = &originID.String
		}
		if clientID.Valid {
			job.ClientId = &clientID.String
		}
		if locationID.Valid {
			job.LocationId = &locationID.String
		}
		if createdBy.Valid {
			job.CreatedBy = &createdBy.String
		}
		if parentJobID.Valid {
			job.ParentJobId = &parentJobID.String
		}
		if cycleIndex.Valid {
			job.CycleIndex = &cycleIndex.Int32
		}
		if cyclePeriodStart.Valid {
			job.CyclePeriodStart = &cyclePeriodStart.String
		}
		if cyclePeriodEnd.Valid {
			job.CyclePeriodEnd = &cyclePeriodEnd.String
		}

		if originType.Valid {
			if v, ok := enumspb.OriginType_value[originType.String]; ok {
				job.OriginType = enumspb.OriginType(v)
			}
		}
		if demandType.Valid {
			if v, ok := enumspb.DemandType_value[demandType.String]; ok {
				job.DemandType = enumspb.DemandType(v)
			}
		}
		if fulfillmentType.Valid {
			if v, ok := enumspb.FulfillmentType_value[fulfillmentType.String]; ok {
				job.FulfillmentType = enumspb.FulfillmentType(v)
			}
		}
		if costFlowType.Valid {
			if v, ok := enumspb.CostFlowType_value[costFlowType.String]; ok {
				job.CostFlowType = enumspb.CostFlowType(v)
			}
		}
		if billingRuleType.Valid {
			if v, ok := enumspb.BillingRuleType_value[billingRuleType.String]; ok {
				job.BillingRuleType = enumspb.BillingRuleType(v)
			}
		}
		if status.Valid {
			if v, ok := enumspb.JobStatus_value[status.String]; ok {
				job.Status = enumspb.JobStatus(v)
			}
		}
		if approvalStatus.Valid {
			if v, ok := enumspb.ApprovalStatus_value[approvalStatus.String]; ok {
				job.ApprovalStatus = enumspb.ApprovalStatus(v)
			}
		}
		if postingStatus.Valid {
			if v, ok := enumspb.PostingStatus_value[postingStatus.String]; ok {
				job.PostingStatus = enumspb.PostingStatus(v)
			}
		}
		if billingStatus.Valid {
			if v, ok := enumspb.BillingStatus_value[billingStatus.String]; ok {
				job.BillingStatus = enumspb.BillingStatus(v)
			}
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			job.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			job.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			job.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			job.DateModifiedString = &dmStr
		}

		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating job rows: %w", err)
	}

	return jobs, nil
}

// NewJobRepository creates a new SQL Server job repository (old-style constructor).
func NewJobRepository(db *sql.DB, tableName string) pb.JobDomainServiceServer {
	dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
	return NewSQLServerJobRepository(dbOps, tableName)
}

//go:build mysql

package operation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"slices"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	infraports "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
)

// jobSortableSQLCols lists the SQL column names that are safe to sort by in
// GetJobListPageData.
var jobSortableSQLCols = []string{
	"j.date_created",
	"j.date_modified",
	"j.name",
	"j.status",
}

// jobViewToSQLColMap translates view-facing sort column keys to SQL column names.
var jobViewToSQLColMap = map[string]string{
	"date_created":  "j.date_created",
	"date_modified": "j.date_modified",
	"name":          "j.name",
	"status":        "j.status",
}

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.Job, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql job repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLJobRepository(dbOps, tableName), nil
	})
}

// MySQLJobRepository implements job CRUD operations using MySQL 8.0+.
type MySQLJobRepository struct {
	pb.UnimplementedJobDomainServiceServer
	dbOps        interfaces.DatabaseOperation
	db           *sql.DB
	tableName    string
	auditService infraports.AuditService
}

// NewMySQLJobRepository creates a new MySQL job repository.
func NewMySQLJobRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.JobDomainServiceServer {
	if tableName == "" {
		tableName = "job"
	}

	var db *sql.DB
	if myOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = myOps.GetDB()
	}

	return &MySQLJobRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// WithAuditService returns the repository with an audit service attached.
func (r *MySQLJobRepository) WithAuditService(svc infraports.AuditService) *MySQLJobRepository {
	r.auditService = svc
	return r
}

// CreateJob creates a new job record.
func (r *MySQLJobRepository) CreateJob(ctx context.Context, req *pb.CreateJobRequest) (*pb.CreateJobResponse, error) {
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

	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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
func (r *MySQLJobRepository) ReadJob(ctx context.Context, req *pb.ReadJobRequest) (*pb.ReadJobResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read job: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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
func (r *MySQLJobRepository) UpdateJob(ctx context.Context, req *pb.UpdateJobRequest) (*pb.UpdateJobResponse, error) {
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

	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update job: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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
func (r *MySQLJobRepository) DeleteJob(ctx context.Context, req *pb.DeleteJobRequest) (*pb.DeleteJobResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete job: %w", err)
	}

	return &pb.DeleteJobResponse{
		Success: true,
	}, nil
}

// ListJobs lists job records with optional filters.
func (r *MySQLJobRepository) ListJobs(ctx context.Context, req *pb.ListJobsRequest) (*pb.ListJobsResponse, error) {
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
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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
//
// Dialect translation from postgres gold standard:
//   - $1, $2, $3 → ? (MySQL positional placeholders, same left-to-right order)
//   - active = true → active = 1 (MySQL TINYINT(1) boolean)
//   - ILIKE → LIKE (MySQL ci collation)
//   - LIMIT $N OFFSET $N → LIMIT ? OFFSET ? (two trailing ? args)
//   - COUNT(*) OVER () stays — MySQL 8.0+ supports window functions
//   - ORDER BY interpolation uses mysqlCore.BuildOrderBy (backtick quoting)
//
// CRITICAL: workspace_id must be enforced at the WorkspaceAwareOperations layer.
func (r *MySQLJobRepository) GetJobListPageData(
	ctx context.Context,
	req *pb.GetJobListPageDataRequest,
) (*pb.GetJobListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get job list page data request is required")
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

	sortField := "j.date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_DESC {
			sortOrder = "DESC"
		}
	}

	if mapped, ok := jobViewToSQLColMap[sortField]; ok {
		sortField = mapped
	}

	// Loud-failure guard: reject any sort column not in the allowlist.
	if sortField != "" && !slices.Contains(jobSortableSQLCols, sortField) {
		return nil, fmt.Errorf("unknown sort column %q for entity %q (allowed: %v)", sortField, "job", jobSortableSQLCols)
	}

	// Dialect: $1::text IS NULL OR ... → ? (MySQL has no ::text cast; NULL check via IS NULL)
	// searchPattern is "" when no search, and LIKE '' matches nothing, so we guard via OR.
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
				j.cycle_period_end
			FROM job j
			WHERE j.active = 1
			  AND (? = '' OR j.name LIKE ?)
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		ORDER BY %s %s
		LIMIT ? OFFSET ?;
	`, sortField, sortOrder)

	rows, err := r.db.QueryContext(ctx, query, searchPattern, searchPattern, limit, offset)
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

		err := rows.Scan(
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
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job row: %w", err)
		}

		totalCount = total

		job := &pb.Job{
			Id:     id,
			Active: active,
			Name:   name,
		}

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
//
// Dialect translation: $1 → ?, active = true → active = 1.
// CRITICAL: workspace_id isolation enforced by WorkspaceAwareOperations on CRUD path.
func (r *MySQLJobRepository) GetJobItemPageData(
	ctx context.Context,
	req *pb.GetJobItemPageDataRequest,
) (*pb.GetJobItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get job item page data request is required")
	}
	if req.JobId == "" {
		return nil, fmt.Errorf("job ID is required")
	}

	// Dialect: $1 → ?, active = true → active = 1
	query := `
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
		WHERE j.id = ? AND j.active = 1
	`

	row := r.db.QueryRowContext(ctx, query, req.JobId)

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
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job with ID '%s' not found", req.JobId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query job item page data: %w", err)
	}

	job := buildJobFromScan(
		id, dateCreated, dateModified, active, name,
		jobTemplateID, originType, originID, clientID,
		demandType, fulfillmentType, costFlowType, billingRuleType,
		status, approvalStatus, postingStatus, billingStatus,
		locationID, createdBy, cycleIndex, cyclePeriodStart, cyclePeriodEnd,
	)

	return &pb.GetJobItemPageDataResponse{
		Job:     job,
		Success: true,
	}, nil
}

// GetJobsByClient retrieves all jobs for a given client.
//
// Dialect: $1 → ?, active = true → active = 1, NULLS FIRST removed (MySQL sorts NULLs first by default).
func (r *MySQLJobRepository) GetJobsByClient(
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
		WHERE client_id = ? AND active = 1
		ORDER BY date_created DESC
	`, r.tableName)

	rows, err := r.db.QueryContext(ctx, query, req.ClientId)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs by client: %w", err)
	}
	defer rows.Close()

	jobs, err := r.scanJobRows(rows)
	if err != nil {
		return nil, err
	}

	return &pb.GetJobsByClientResponse{
		Jobs:    jobs,
		Success: true,
	}, nil
}

// GetJobsByOrigin retrieves all jobs for a given origin (type + ID).
//
// Dialect: $1, $2 → ?, active = true → active = 1.
// Note: NULLS FIRST is omitted — MySQL places NULLs first in ASC order by default.
func (r *MySQLJobRepository) GetJobsByOrigin(
	ctx context.Context,
	req *pb.GetJobsByOriginRequest,
) (*pb.GetJobsByOriginResponse, error) {
	if req == nil || req.OriginId == "" {
		return nil, fmt.Errorf("origin ID is required")
	}

	query := fmt.Sprintf(`
		SELECT id, date_created, date_modified, active, name,
		       job_template_id, origin_type, origin_id, client_id,
		       demand_type, fulfillment_type, cost_flow_type, billing_rule_type,
		       status, approval_status, posting_status, billing_status,
		       location_id, created_by, parent_job_id,
		       cycle_index, cycle_period_start, cycle_period_end
		FROM %s
		WHERE origin_type = ? AND origin_id = ? AND active = 1
		ORDER BY parent_job_id ASC, date_created ASC
	`, r.tableName)

	rows, err := r.db.QueryContext(ctx, query, req.OriginType.String(), req.OriginId)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs by origin: %w", err)
	}
	defer rows.Close()

	jobs, err := r.scanJobRows(rows)
	if err != nil {
		return nil, err
	}

	return &pb.GetJobsByOriginResponse{
		Jobs:    jobs,
		Success: true,
	}, nil
}

// UpdateJobStatus transitions a job to a new status.
//
// Dialect: RETURNING → two-step (UPDATE then re-read via ReadJob).
// active = true → active = 1.
func (r *MySQLJobRepository) UpdateJobStatus(
	ctx context.Context,
	req *pb.UpdateJobStatusRequest,
) (*pb.UpdateJobStatusResponse, error) {
	if req == nil || req.JobId == "" {
		return nil, fmt.Errorf("job ID is required")
	}

	newStatus := req.Status.String()

	// Dialect: RETURNING is not supported — update first, then read back.
	query := fmt.Sprintf(`
		UPDATE %s SET status = ?, date_modified = NOW()
		WHERE id = ? AND active = 1
	`, r.tableName)

	res, err := r.db.ExecContext(ctx, query, newStatus, req.JobId)
	if err != nil {
		return nil, fmt.Errorf("failed to update job status: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check rows affected: %w", err)
	}
	if n == 0 {
		return nil, fmt.Errorf("job not found with ID: %s", req.JobId)
	}

	// Re-read the updated job via the generic CRUD path.
	readResp, err := r.ReadJob(ctx, &pb.ReadJobRequest{Data: &pb.Job{Id: req.JobId}})
	if err != nil {
		return nil, err
	}

	var job *pb.Job
	if len(readResp.Data) > 0 {
		job = readResp.Data[0]
	}

	return &pb.UpdateJobStatusResponse{
		Job:     job,
		Success: true,
	}, nil
}

// scanJobRows is a helper to scan multiple job rows from a query result.
func (r *MySQLJobRepository) scanJobRows(rows *sql.Rows) ([]*pb.Job, error) {
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

		job := buildJobFromScan(
			id, dateCreated, dateModified, active, name,
			jobTemplateID, originType, originID, clientID,
			demandType, fulfillmentType, costFlowType, billingRuleType,
			status, approvalStatus, postingStatus, billingStatus,
			locationID, createdBy, cycleIndex, cyclePeriodStart, cyclePeriodEnd,
		)
		if parentJobID.Valid {
			job.ParentJobId = &parentJobID.String
		}

		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating job rows: %w", err)
	}

	return jobs, nil
}

// buildJobFromScan constructs a Job protobuf from scanned SQL fields.
// Scan column order is preserved exactly so the Go-side response mapping is dialect-agnostic.
func buildJobFromScan(
	id string, dateCreated, dateModified time.Time, active bool, name string,
	jobTemplateID, originType, originID, clientID sql.NullString,
	demandType, fulfillmentType, costFlowType, billingRuleType sql.NullString,
	status, approvalStatus, postingStatus, billingStatus sql.NullString,
	locationID, createdBy sql.NullString,
	cycleIndex sql.NullInt32,
	cyclePeriodStart, cyclePeriodEnd sql.NullString,
) *pb.Job {
	job := &pb.Job{
		Id:     id,
		Active: active,
		Name:   name,
	}

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

	return job
}

// convertMillisToTime converts epoch-millisecond fields in data maps to time.Time
// for MySQL-compatible datetime columns.
func convertMillisToTime(data map[string]any, key string) {
	if v, ok := data[key]; ok {
		switch val := v.(type) {
		case float64:
			ms := int64(val)
			if ms > 0 {
				data[key] = time.UnixMilli(ms).UTC()
			}
		case int64:
			if val > 0 {
				data[key] = time.UnixMilli(val).UTC()
			}
		}
	}
}

// NewJobRepository creates a new MySQL job repository (old-style constructor).
func NewJobRepository(db *sql.DB, tableName string) pb.JobDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLJobRepository(dbOps, tableName)
}

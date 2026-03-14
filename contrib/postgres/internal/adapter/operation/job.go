package operation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.Job, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres job repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresJobRepository(dbOps, tableName), nil
	})
}

// PostgresJobRepository implements job CRUD operations using PostgreSQL
type PostgresJobRepository struct {
	pb.UnimplementedJobDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresJobRepository creates a new PostgreSQL job repository
func NewPostgresJobRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.JobDomainServiceServer {
	if tableName == "" {
		tableName = "job"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresJobRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateJob creates a new job record
func (r *PostgresJobRepository) CreateJob(ctx context.Context, req *pb.CreateJobRequest) (*pb.CreateJobResponse, error) {
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

	resultJSON, err := json.Marshal(result)
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

// ReadJob retrieves a job record by ID
func (r *PostgresJobRepository) ReadJob(ctx context.Context, req *pb.ReadJobRequest) (*pb.ReadJobResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read job: %w", err)
	}

	resultJSON, err := json.Marshal(result)
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

// UpdateJob updates a job record
func (r *PostgresJobRepository) UpdateJob(ctx context.Context, req *pb.UpdateJobRequest) (*pb.UpdateJobResponse, error) {
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

	resultJSON, err := json.Marshal(result)
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

// DeleteJob deletes a job record (soft delete)
func (r *PostgresJobRepository) DeleteJob(ctx context.Context, req *pb.DeleteJobRequest) (*pb.DeleteJobResponse, error) {
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

// ListJobs lists job records with optional filters
func (r *PostgresJobRepository) ListJobs(ctx context.Context, req *pb.ListJobsRequest) (*pb.ListJobsResponse, error) {
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
		resultJSON, err := json.Marshal(result)
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

// GetJobListPageData retrieves jobs with pagination, filtering, sorting, and search
func (r *PostgresJobRepository) GetJobListPageData(
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

	query := `
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
				j.created_by
			FROM job j
			WHERE j.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       j.name ILIKE $1)
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		ORDER BY ` + sortField + ` ` + sortOrder + `
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query job list page data: %w", err)
	}
	defer rows.Close()

	var jobs []*pb.Job
	var totalCount int64

	for rows.Next() {
		var (
			id              string
			dateCreated     time.Time
			dateModified    time.Time
			active          bool
			name            string
			jobTemplateID   sql.NullString
			originType      sql.NullString
			originID        sql.NullString
			clientID        sql.NullString
			demandType      sql.NullString
			fulfillmentType sql.NullString
			costFlowType    sql.NullString
			billingRuleType sql.NullString
			status          sql.NullString
			approvalStatus  sql.NullString
			postingStatus   sql.NullString
			billingStatus   sql.NullString
			locationID      sql.NullString
			createdBy       sql.NullString
			total           int64
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

		// Map enum strings to proto enums
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

// GetJobItemPageData retrieves a single job with enriched data
func (r *PostgresJobRepository) GetJobItemPageData(
	ctx context.Context,
	req *pb.GetJobItemPageDataRequest,
) (*pb.GetJobItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get job item page data request is required")
	}
	if req.JobId == "" {
		return nil, fmt.Errorf("job ID is required")
	}

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
			j.created_by
		FROM job j
		WHERE j.id = $1 AND j.active = true
	`

	row := r.db.QueryRowContext(ctx, query, req.JobId)

	var (
		id              string
		dateCreated     time.Time
		dateModified    time.Time
		active          bool
		name            string
		jobTemplateID   sql.NullString
		originType      sql.NullString
		originID        sql.NullString
		clientID        sql.NullString
		demandType      sql.NullString
		fulfillmentType sql.NullString
		costFlowType    sql.NullString
		billingRuleType sql.NullString
		status          sql.NullString
		approvalStatus  sql.NullString
		postingStatus   sql.NullString
		billingStatus   sql.NullString
		locationID      sql.NullString
		createdBy       sql.NullString
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
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job with ID '%s' not found", req.JobId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query job item page data: %w", err)
	}

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

// GetJobsByClient retrieves all jobs for a given client
func (r *PostgresJobRepository) GetJobsByClient(
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
		       location_id, created_by
		FROM %s
		WHERE client_id = $1 AND active = true
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

// GetJobsByOrigin retrieves all jobs for a given origin (type + ID)
func (r *PostgresJobRepository) GetJobsByOrigin(
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
		       location_id, created_by
		FROM %s
		WHERE origin_type = $1 AND origin_id = $2 AND active = true
		ORDER BY date_created DESC
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

// UpdateJobStatus transitions a job to a new status
func (r *PostgresJobRepository) UpdateJobStatus(
	ctx context.Context,
	req *pb.UpdateJobStatusRequest,
) (*pb.UpdateJobStatusResponse, error) {
	if req == nil || req.JobId == "" {
		return nil, fmt.Errorf("job ID is required")
	}

	newStatus := req.Status.String()

	query := fmt.Sprintf(`
		UPDATE %s SET status = $1, date_modified = NOW()
		WHERE id = $2 AND active = true
		RETURNING id
	`, r.tableName)

	var id string
	err := r.db.QueryRowContext(ctx, query, newStatus, req.JobId).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job not found with ID: %s", req.JobId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update job status: %w", err)
	}

	// Re-read the updated job
	readResp, err := r.ReadJob(ctx, &pb.ReadJobRequest{Data: &pb.Job{Id: id}})
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

// scanJobRows is a helper to scan multiple job rows from a query result
func (r *PostgresJobRepository) scanJobRows(rows *sql.Rows) ([]*pb.Job, error) {
	var jobs []*pb.Job

	for rows.Next() {
		var (
			id              string
			dateCreated     time.Time
			dateModified    time.Time
			active          bool
			name            string
			jobTemplateID   sql.NullString
			originType      sql.NullString
			originID        sql.NullString
			clientID        sql.NullString
			demandType      sql.NullString
			fulfillmentType sql.NullString
			costFlowType    sql.NullString
			billingRuleType sql.NullString
			status          sql.NullString
			approvalStatus  sql.NullString
			postingStatus   sql.NullString
			billingStatus   sql.NullString
			locationID      sql.NullString
			createdBy       sql.NullString
		)

		if err := rows.Scan(
			&id, &dateCreated, &dateModified, &active, &name,
			&jobTemplateID, &originType, &originID, &clientID,
			&demandType, &fulfillmentType, &costFlowType, &billingRuleType,
			&status, &approvalStatus, &postingStatus, &billingStatus,
			&locationID, &createdBy,
		); err != nil {
			return nil, fmt.Errorf("failed to scan job row: %w", err)
		}

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

// NewJobRepository creates a new PostgreSQL job repository (old-style constructor)
func NewJobRepository(db *sql.DB, tableName string) pb.JobDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresJobRepository(dbOps, tableName)
}

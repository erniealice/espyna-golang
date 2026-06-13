//go:build sqlserver

package operation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/erniealice/espyna-golang/shared/identity"
	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	infraports "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	jobpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_activity"
)

// jobActivitySortableSQLCols lists the SQL column names that are safe to sort by in
// GetJobActivityListPageData. Routed through core.BuildOrderBy (A2 guard).
var jobActivitySortableSQLCols = []string{
	"ja.date_created",
	"ja.entry_date",
	"ja.total_cost",
	"ja.entry_type",
	"ja.approval_status",
}

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.JobActivity, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver job_activity repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerJobActivityRepository(dbOps, tableName), nil
	})
}

// SQLServerJobActivityRepository implements job_activity CRUD operations using SQL Server.
type SQLServerJobActivityRepository struct {
	pb.UnimplementedJobActivityDomainServiceServer
	dbOps        interfaces.DatabaseOperation
	tableName    string
	auditService infraports.AuditService
}

// NewSQLServerJobActivityRepository creates a new SQL Server job activity repository.
func NewSQLServerJobActivityRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.JobActivityDomainServiceServer {
	if tableName == "" {
		tableName = "job_activity"
	}
	return &SQLServerJobActivityRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// WithAuditService returns the repository with an audit service attached.
func (r *SQLServerJobActivityRepository) WithAuditService(svc infraports.AuditService) *SQLServerJobActivityRepository {
	r.auditService = svc
	return r
}

// getExec extracts a DBExecutor from the dbOps wrapper.
func (r *SQLServerJobActivityRepository) getExec(ctx context.Context) dbExecutor {
	return r.dbOps.(executorProvider).GetExecutor(ctx)
}

// CreateJobActivity creates a new job activity.
func (r *SQLServerJobActivityRepository) CreateJobActivity(ctx context.Context, req *pb.CreateJobActivityRequest) (*pb.CreateJobActivityResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("job activity data is required")
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
		return nil, fmt.Errorf("failed to create job activity: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	activity := &pb.JobActivity{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, activity); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.CreateJobActivityResponse{Data: []*pb.JobActivity{activity}}, nil
}

// ReadJobActivity retrieves a job activity record by ID.
func (r *SQLServerJobActivityRepository) ReadJobActivity(ctx context.Context, req *pb.ReadJobActivityRequest) (*pb.ReadJobActivityResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job activity ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read job activity: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	activity := &pb.JobActivity{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, activity); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.ReadJobActivityResponse{Data: []*pb.JobActivity{activity}}, nil
}

// UpdateJobActivity updates a job activity record.
func (r *SQLServerJobActivityRepository) UpdateJobActivity(ctx context.Context, req *pb.UpdateJobActivityRequest) (*pb.UpdateJobActivityResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job activity ID is required")
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
		return nil, fmt.Errorf("failed to update job activity: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	activity := &pb.JobActivity{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, activity); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.UpdateJobActivityResponse{Data: []*pb.JobActivity{activity}}, nil
}

// DeleteJobActivity deletes a job activity (soft delete).
func (r *SQLServerJobActivityRepository) DeleteJobActivity(ctx context.Context, req *pb.DeleteJobActivityRequest) (*pb.DeleteJobActivityResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job activity ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete job activity: %w", err)
	}

	return &pb.DeleteJobActivityResponse{Success: true}, nil
}

// ListJobActivities lists job activities with optional filters.
func (r *SQLServerJobActivityRepository) ListJobActivities(ctx context.Context, req *pb.ListJobActivitiesRequest) (*pb.ListJobActivitiesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list job activities: %w", err)
	}

	var activities []*pb.JobActivity
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		activity := &pb.JobActivity{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, activity); err != nil {
			continue
		}
		activities = append(activities, activity)
	}

	return &pb.ListJobActivitiesResponse{Data: activities}, nil
}

// GetJobActivityListPageData retrieves paginated job activity list with job join.
// CRITICAL: Always filters by workspace_id for multi-tenancy.
//
// SQL Server differences from the postgres gold standard:
//   - $N → @pN placeholders.
//   - active = true → active = 1.
//   - Pagination: LIMIT n OFFSET m → ORDER BY … OFFSET @pM ROWS FETCH NEXT @pN ROWS ONLY.
//   - COUNT(*) OVER() window function is retained — SQL Server 2017+ supports it.
func (r *SQLServerJobActivityRepository) GetJobActivityListPageData(ctx context.Context, req *pb.GetJobActivityListPageDataRequest) (*pb.GetJobActivityListPageDataResponse, error) {
	workspaceID := identity.Must(ctx).WorkspaceID

	limit := int32(50)
	offset := int32(0)
	page := int32(1)
	if req != nil && req.Pagination != nil {
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

	var sortReq *commonpb.SortRequest
	if req != nil {
		sortReq = req.GetSort()
	}
	orderByClause, err := sqlserverCore.BuildOrderBy(jobActivitySortableSQLCols, sortReq, "ja.date_created DESC")
	if err != nil {
		return nil, err
	}

	// @p1 = workspaceID; offset/limit are last two params after any filter args.
	filterClauses, filterArgs, nextIdx := sqlserverCore.BuildFilterWhere(
		req.GetFilters(), req.GetSearch(), []string{"j.name", "ja.description"}, 2,
	)

	whereSQL := "WHERE ja.workspace_id = @p1 AND ja.active = 1"
	if len(filterClauses) > 0 {
		whereSQL += " AND " + strings.Join(filterClauses, " AND ")
	}

	offsetIdx := nextIdx
	limitIdx := nextIdx + 1
	queryArgs := append([]any{workspaceID}, filterArgs...)
	queryArgs = append(queryArgs, offset, limit)

	query := fmt.Sprintf(`
		WITH enriched AS (
			SELECT
				ja.id,
				ja.job_id,
				ja.job_task_id,
				ja.entry_type,
				ja.quantity,
				ja.unit_cost,
				ja.total_cost,
				ja.currency,
				ja.entry_date,
				ja.description,
				ja.billable_status,
				ja.approval_status,
				ja.posting_status,
				ja.posted_by,
				ja.date_posted,
				ja.reversal_of_id,
				ja.created_by,
				ja.date_created,
				ja.active,
				j.name AS job_name,
				COUNT(*) OVER() AS total_count
			FROM %s ja
			LEFT JOIN job j ON j.id = ja.job_id
			%s
		)
		SELECT * FROM enriched
		%s OFFSET @p%d ROWS FETCH NEXT @p%d ROWS ONLY;
	`, r.tableName, whereSQL, orderByClause, offsetIdx, limitIdx)

	exec := r.getExec(ctx)
	rows, err := exec.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query job activity list page data: %w", err)
	}
	defer rows.Close()

	var activities []*pb.JobActivity
	var totalCount int64

	for rows.Next() {
		var (
			id             string
			jobId          string
			jobTaskId      sql.NullString
			entryType      string
			quantity       float64
			unitCost       int64
			totalCost      int64
			currency       string
			entryDate      sql.NullTime
			description    sql.NullString
			billableStatus string
			approvalStatus string
			postingStatus  string
			postedBy       sql.NullString
			datePosted     sql.NullTime
			reversalOfId   sql.NullString
			createdBy      sql.NullString
			dateCreated    sql.NullTime
			active         bool
			jobName        sql.NullString
			total          int64
		)

		if err := rows.Scan(
			&id, &jobId, &jobTaskId, &entryType, &quantity, &unitCost, &totalCost,
			&currency, &entryDate, &description, &billableStatus, &approvalStatus,
			&postingStatus, &postedBy, &datePosted, &reversalOfId, &createdBy,
			&dateCreated, &active, &jobName, &total,
		); err != nil {
			return nil, fmt.Errorf("failed to scan job activity row: %w", err)
		}

		totalCount = total

		activity := &pb.JobActivity{
			Id:        id,
			JobId:     jobId,
			Quantity:  quantity,
			UnitCost:  unitCost,
			TotalCost: totalCost,
			Currency:  currency,
			Active:    active,
		}

		if v, ok := pb.EntryType_value[entryType]; ok {
			activity.EntryType = pb.EntryType(v)
		}
		if v, ok := pb.BillableStatus_value[billableStatus]; ok {
			activity.BillableStatus = pb.BillableStatus(v)
		}
		if v, ok := pb.ActivityApprovalStatus_value[approvalStatus]; ok {
			activity.ApprovalStatus = pb.ActivityApprovalStatus(v)
		}
		if v, ok := pb.ActivityPostingStatus_value[postingStatus]; ok {
			activity.PostingStatus = pb.ActivityPostingStatus(v)
		}

		if jobTaskId.Valid {
			activity.JobTaskId = &jobTaskId.String
		}
		if description.Valid {
			activity.Description = &description.String
		}
		if postedBy.Valid {
			activity.PostedBy = &postedBy.String
		}
		if reversalOfId.Valid {
			activity.ReversalOfId = &reversalOfId.String
		}
		if createdBy.Valid {
			activity.CreatedBy = &createdBy.String
		}
		if entryDate.Valid {
			ts := entryDate.Time.Unix()
			activity.EntryDate = &ts
			eds := entryDate.Time.Format("2006-01-02")
			activity.EntryDateString = &eds
		}
		if datePosted.Valid {
			ts := datePosted.Time.Unix()
			activity.DatePosted = &ts
		}
		if dateCreated.Valid {
			ts := dateCreated.Time.Unix()
			activity.DateCreated = &ts
		}
		if jobName.Valid {
			activity.Job = &jobpb.Job{Name: jobName.String}
		}

		activities = append(activities, activity)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating job activity rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &pb.GetJobActivityListPageDataResponse{
		JobActivityList: activities,
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

// GetJobActivityItemPageData retrieves a single job activity with all related data.
func (r *SQLServerJobActivityRepository) GetJobActivityItemPageData(ctx context.Context, req *pb.GetJobActivityItemPageDataRequest) (*pb.GetJobActivityItemPageDataResponse, error) {
	// TODO: Implement CTE-based single item query with job, task joins
	return nil, fmt.Errorf("GetJobActivityItemPageData not yet implemented")
}

// ListByJob lists all activities for a given job.
// SQL Server: @p1 placeholder; active = 1.
func (r *SQLServerJobActivityRepository) ListByJob(ctx context.Context, req *pb.ListJobActivitiesByJobRequest) (*pb.ListJobActivitiesByJobResponse, error) {
	if req.JobId == "" {
		return nil, fmt.Errorf("job ID is required")
	}

	query := fmt.Sprintf(`
		SELECT id, job_id, job_task_id, entry_type, quantity, unit_cost, total_cost,
			   currency, entry_date, description, billable_status, approval_status,
			   posting_status, posted_by, date_posted, reversal_of_id, created_by,
			   date_created, active
		FROM %s
		WHERE job_id = @p1 AND active = 1
		ORDER BY date_created DESC
	`, r.tableName)

	exec := r.getExec(ctx)
	rows, err := exec.QueryContext(ctx, query, req.JobId)
	if err != nil {
		return nil, fmt.Errorf("failed to list job activities by job: %w", err)
	}
	defer rows.Close()

	var activities []*pb.JobActivity
	for rows.Next() {
		var (
			id             string
			jobId          string
			jobTaskId      sql.NullString
			entryType      string
			quantity       float64
			unitCost       int64
			totalCost      int64
			currency       string
			entryDate      sql.NullTime
			description    sql.NullString
			billableStatus string
			approvalStatus string
			postingStatus  string
			postedBy       sql.NullString
			datePosted     sql.NullTime
			reversalOfId   sql.NullString
			createdBy      sql.NullString
			dateCreated    sql.NullTime
			active         bool
		)

		if err := rows.Scan(
			&id, &jobId, &jobTaskId, &entryType, &quantity, &unitCost, &totalCost,
			&currency, &entryDate, &description, &billableStatus, &approvalStatus,
			&postingStatus, &postedBy, &datePosted, &reversalOfId, &createdBy,
			&dateCreated, &active,
		); err != nil {
			return nil, fmt.Errorf("failed to scan job activity row: %w", err)
		}

		activity := &pb.JobActivity{
			Id:        id,
			JobId:     jobId,
			Quantity:  quantity,
			UnitCost:  unitCost,
			TotalCost: totalCost,
			Currency:  currency,
			Active:    active,
		}

		if v, ok := pb.EntryType_value[entryType]; ok {
			activity.EntryType = pb.EntryType(v)
		}
		if v, ok := pb.BillableStatus_value[billableStatus]; ok {
			activity.BillableStatus = pb.BillableStatus(v)
		}
		if v, ok := pb.ActivityApprovalStatus_value[approvalStatus]; ok {
			activity.ApprovalStatus = pb.ActivityApprovalStatus(v)
		}
		if v, ok := pb.ActivityPostingStatus_value[postingStatus]; ok {
			activity.PostingStatus = pb.ActivityPostingStatus(v)
		}

		if jobTaskId.Valid {
			activity.JobTaskId = &jobTaskId.String
		}
		if description.Valid {
			activity.Description = &description.String
		}
		if postedBy.Valid {
			activity.PostedBy = &postedBy.String
		}
		if reversalOfId.Valid {
			activity.ReversalOfId = &reversalOfId.String
		}
		if createdBy.Valid {
			activity.CreatedBy = &createdBy.String
		}
		if entryDate.Valid {
			ts := entryDate.Time.Unix()
			activity.EntryDate = &ts
			eds := entryDate.Time.Format("2006-01-02")
			activity.EntryDateString = &eds
		}
		if datePosted.Valid {
			ts := datePosted.Time.Unix()
			activity.DatePosted = &ts
		}
		if dateCreated.Valid {
			ts := dateCreated.Time.Unix()
			activity.DateCreated = &ts
		}

		activities = append(activities, activity)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating job activity rows: %w", err)
	}

	return &pb.ListJobActivitiesByJobResponse{
		JobActivities: activities,
		Success:       true,
	}, nil
}

// GetJobActivityRollup returns aggregated job activity cost and quantity totals.
func (r *SQLServerJobActivityRepository) GetJobActivityRollup(ctx context.Context, req *pb.GetJobActivityRollupRequest) (*pb.GetJobActivityRollupResponse, error) {
	if req == nil || req.JobId == "" {
		return nil, fmt.Errorf("job ID is required")
	}

	// SQL Server: GROUP BY entry_type to build CostByEntryType rollup.
	const query = `
		SELECT entry_type,
		       SUM(total_cost) AS total_cost,
		       COUNT(*)        AS cnt
		FROM job_activity
		WHERE job_id = @p1 AND active = 1
		GROUP BY entry_type
	`

	exec := r.getExec(ctx)
	rows, err := exec.QueryContext(ctx, query, req.JobId)
	if err != nil {
		return nil, fmt.Errorf("failed to query job activity rollup: %w", err)
	}
	defer rows.Close()

	var (
		rollup     []*pb.CostByEntryType
		grandTotal int64
	)

	for rows.Next() {
		var (
			entryTypeStr string
			totalCost    sql.NullInt64
			cnt          int32
		)
		if err := rows.Scan(&entryTypeStr, &totalCost, &cnt); err != nil {
			return nil, fmt.Errorf("failed to scan job activity rollup row: %w", err)
		}
		cost := totalCost.Int64
		grandTotal += cost
		rollup = append(rollup, &pb.CostByEntryType{
			EntryType: pb.EntryType(pb.EntryType_value[entryTypeStr]),
			TotalCost: cost,
			Count:     cnt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating job activity rollup rows: %w", err)
	}

	return &pb.GetJobActivityRollupResponse{
		Rollup:     rollup,
		GrandTotal: grandTotal,
		Success:    true,
	}, nil
}

// SubmitForApproval transitions activity to PENDING_APPROVAL status.
func (r *SQLServerJobActivityRepository) SubmitForApproval(ctx context.Context, req *pb.SubmitForApprovalRequest) (*pb.SubmitForApprovalResponse, error) {
	if req == nil || req.ActivityId == "" {
		return nil, fmt.Errorf("activity ID is required")
	}

	// SQL Server: OUTPUT inserted.id instead of RETURNING.
	const query = `
		UPDATE job_activity
		SET approval_status = 'ACTIVITY_APPROVAL_STATUS_PENDING_APPROVAL',
		    date_modified = GETUTCDATE()
		OUTPUT inserted.id
		WHERE id = @p1 AND active = 1
	`

	exec := r.getExec(ctx)
	var id string
	if err := exec.QueryRowContext(ctx, query, req.ActivityId).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("activity not found: %s", req.ActivityId)
		}
		return nil, fmt.Errorf("failed to submit activity for approval: %w", err)
	}

	return &pb.SubmitForApprovalResponse{JobActivity: &pb.JobActivity{Id: id}, Success: true}, nil
}

// ApproveActivity approves a job activity.
func (r *SQLServerJobActivityRepository) ApproveActivity(ctx context.Context, req *pb.ApproveJobActivityRequest) (*pb.ApproveJobActivityResponse, error) {
	if req == nil || req.ActivityId == "" {
		return nil, fmt.Errorf("activity ID is required")
	}

	const query = `
		UPDATE job_activity
		SET approval_status = 'ACTIVITY_APPROVAL_STATUS_APPROVED',
		    date_modified = GETUTCDATE()
		OUTPUT inserted.id
		WHERE id = @p1 AND active = 1
	`

	exec := r.getExec(ctx)
	var id string
	if err := exec.QueryRowContext(ctx, query, req.ActivityId).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("activity not found: %s", req.ActivityId)
		}
		return nil, fmt.Errorf("failed to approve activity: %w", err)
	}

	return &pb.ApproveJobActivityResponse{JobActivity: &pb.JobActivity{Id: id}, Success: true}, nil
}

// RejectActivity rejects a job activity.
func (r *SQLServerJobActivityRepository) RejectActivity(ctx context.Context, req *pb.RejectJobActivityRequest) (*pb.RejectJobActivityResponse, error) {
	if req == nil || req.ActivityId == "" {
		return nil, fmt.Errorf("activity ID is required")
	}

	const query = `
		UPDATE job_activity
		SET approval_status = 'ACTIVITY_APPROVAL_STATUS_REJECTED',
		    date_modified = GETUTCDATE()
		OUTPUT inserted.id
		WHERE id = @p1 AND active = 1
	`

	exec := r.getExec(ctx)
	var id string
	if err := exec.QueryRowContext(ctx, query, req.ActivityId).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("activity not found: %s", req.ActivityId)
		}
		return nil, fmt.Errorf("failed to reject activity: %w", err)
	}

	return &pb.RejectJobActivityResponse{JobActivity: &pb.JobActivity{Id: id}, Success: true}, nil
}

// PostActivity marks a job activity as posted.
func (r *SQLServerJobActivityRepository) PostActivity(ctx context.Context, req *pb.PostJobActivityRequest) (*pb.PostJobActivityResponse, error) {
	if req == nil || req.ActivityId == "" {
		return nil, fmt.Errorf("activity ID is required")
	}

	const query = `
		UPDATE job_activity
		SET posting_status = 'ACTIVITY_POSTING_STATUS_POSTED',
		    posted_by = @p2,
		    date_posted = GETUTCDATE(),
		    date_modified = GETUTCDATE()
		OUTPUT inserted.id
		WHERE id = @p1 AND active = 1
	`

	exec := r.getExec(ctx)
	var id string
	if err := exec.QueryRowContext(ctx, query, req.ActivityId, req.PostedBy).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("activity not found: %s", req.ActivityId)
		}
		return nil, fmt.Errorf("failed to post activity: %w", err)
	}

	return &pb.PostJobActivityResponse{JobActivity: &pb.JobActivity{Id: id}, Success: true}, nil
}

// ReverseActivity creates a reversal entry for a posted activity.
func (r *SQLServerJobActivityRepository) ReverseActivity(ctx context.Context, req *pb.ReverseJobActivityRequest) (*pb.ReverseJobActivityResponse, error) {
	if req == nil || req.ActivityId == "" {
		return nil, fmt.Errorf("activity ID is required")
	}

	// Read original activity first.
	original, err := r.ReadJobActivity(ctx, &pb.ReadJobActivityRequest{Data: &pb.JobActivity{Id: req.ActivityId}})
	if err != nil || len(original.Data) == 0 {
		return nil, fmt.Errorf("original activity not found: %s", req.ActivityId)
	}
	orig := original.Data[0]

	// Create reversal with negated amounts — use core CRUD path (INSERT ... OUTPUT).
	reversalData := map[string]any{
		"job_id":         orig.JobId,
		"job_task_id":    orig.JobTaskId,
		"entry_type":     orig.EntryType.String(),
		"quantity":       -orig.Quantity,
		"unit_cost":      orig.UnitCost,
		"total_cost":     -orig.TotalCost,
		"currency":       orig.Currency,
		"description":    fmt.Sprintf("REVERSAL of %s", orig.Id),
		"reversal_of_id": orig.Id,
		"workspace_id":   identity.Must(ctx).WorkspaceID,
		"active":         true,
	}

	reversalResult, err := r.dbOps.Create(ctx, r.tableName, reversalData)
	if err != nil {
		return nil, fmt.Errorf("failed to create reversal activity: %w", err)
	}

	reversalID, _ := reversalResult["id"].(string)

	return &pb.ReverseJobActivityResponse{
		JobActivity: &pb.JobActivity{Id: reversalID},
		Success:     true,
	}, nil
}

// NewJobActivityRepository creates a new SQL Server job activity repository (old-style constructor).
func NewJobActivityRepository(db *sql.DB, tableName string) pb.JobActivityDomainServiceServer {
	dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
	return NewSQLServerJobActivityRepository(dbOps, tableName)
}

// jaGetDB extracts the raw *sql.DB from the dbOps wrapper.
func jaGetDB(dbOps any) *sql.DB {
	if dbOps == nil {
		return nil
	}
	if op, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		return op.GetDB()
	}
	return nil
}

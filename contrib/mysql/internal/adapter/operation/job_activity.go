//go:build mysql

package operation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	jobpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_activity"
)

// MySQLJobActivityRepository implements job_activity CRUD operations using MySQL 8.0+.
type MySQLJobActivityRepository struct {
	pb.UnimplementedJobActivityDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.JobActivity, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql job_activity repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLJobActivityRepository(dbOps, tableName), nil
	})
}

// NewMySQLJobActivityRepository creates a new MySQL job activity repository.
func NewMySQLJobActivityRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.JobActivityDomainServiceServer {
	if tableName == "" {
		tableName = "job_activity"
	}
	return &MySQLJobActivityRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateJobActivity creates a new job activity.
func (r *MySQLJobActivityRepository) CreateJobActivity(ctx context.Context, req *pb.CreateJobActivityRequest) (*pb.CreateJobActivityResponse, error) {
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

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	activity := &pb.JobActivity{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, activity); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.CreateJobActivityResponse{
		Data: []*pb.JobActivity{activity},
	}, nil
}

// ReadJobActivity retrieves a job activity by ID.
func (r *MySQLJobActivityRepository) ReadJobActivity(ctx context.Context, req *pb.ReadJobActivityRequest) (*pb.ReadJobActivityResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job activity ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read job activity: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	activity := &pb.JobActivity{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, activity); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.ReadJobActivityResponse{
		Data: []*pb.JobActivity{activity},
	}, nil
}

// UpdateJobActivity updates a job activity.
func (r *MySQLJobActivityRepository) UpdateJobActivity(ctx context.Context, req *pb.UpdateJobActivityRequest) (*pb.UpdateJobActivityResponse, error) {
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

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	activity := &pb.JobActivity{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, activity); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.UpdateJobActivityResponse{
		Data: []*pb.JobActivity{activity},
	}, nil
}

// DeleteJobActivity deletes a job activity (soft delete).
func (r *MySQLJobActivityRepository) DeleteJobActivity(ctx context.Context, req *pb.DeleteJobActivityRequest) (*pb.DeleteJobActivityResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job activity ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete job activity: %w", err)
	}

	return &pb.DeleteJobActivityResponse{
		Success: true,
	}, nil
}

// ListJobActivities lists job activities with optional filters.
func (r *MySQLJobActivityRepository) ListJobActivities(ctx context.Context, req *pb.ListJobActivitiesRequest) (*pb.ListJobActivitiesResponse, error) {
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
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}

		activity := &pb.JobActivity{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, activity); err != nil {
			continue
		}
		activities = append(activities, activity)
	}

	return &pb.ListJobActivitiesResponse{
		Data: activities,
	}, nil
}

// GetJobActivityListPageData retrieves paginated job activity list with job join.
//
// Dialect translation from postgres gold standard:
//   - No $N params in postgres — query was unparameterized; MySQL version also has no params
//   - active = true → active = 1
//   - COUNT(*) OVER () → two-step CTE with counted (MySQL 8.0 supports window functions but
//     the postgres original used a separate counted CTE; MySQL uses the same pattern)
func (r *MySQLJobActivityRepository) GetJobActivityListPageData(ctx context.Context, req *pb.GetJobActivityListPageDataRequest) (*pb.GetJobActivityListPageDataResponse, error) {
	db, ok := r.dbOps.(interface{ GetDB() *sql.DB })
	if !ok {
		return nil, fmt.Errorf("database operations does not support raw SQL queries")
	}

	// Dialect: active = true → active = 1
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
				j.name AS job_name
			FROM %s ja
			LEFT JOIN job j ON j.id = ja.job_id
			WHERE ja.active = 1
		),
		counted AS (
			SELECT COUNT(*) AS total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		ORDER BY e.date_created DESC
	`, r.tableName)

	rows, err := db.GetDB().QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query job activity list page data: %w", err)
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

		_ = total

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

	return &pb.GetJobActivityListPageDataResponse{
		JobActivityList: activities,
		Success:         true,
	}, nil
}

// GetJobActivityItemPageData retrieves a single job activity with all related data.
func (r *MySQLJobActivityRepository) GetJobActivityItemPageData(ctx context.Context, req *pb.GetJobActivityItemPageDataRequest) (*pb.GetJobActivityItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetJobActivityItemPageData not yet implemented")
}

// ListByJob lists all activities for a given job.
//
// Dialect: $1 → ?, active = true → active = 1.
func (r *MySQLJobActivityRepository) ListByJob(ctx context.Context, req *pb.ListJobActivitiesByJobRequest) (*pb.ListJobActivitiesByJobResponse, error) {
	if req.JobId == "" {
		return nil, fmt.Errorf("job ID is required")
	}

	db, ok := r.dbOps.(interface{ GetDB() *sql.DB })
	if !ok {
		return nil, fmt.Errorf("database operations does not support raw SQL queries")
	}

	// Dialect: $1 → ?, active = true → active = 1
	query := fmt.Sprintf(`
		SELECT id, job_id, job_task_id, entry_type, quantity, unit_cost, total_cost,
			   currency, entry_date, description, billable_status, approval_status,
			   posting_status, posted_by, date_posted, reversal_of_id, created_by,
			   date_created, active
		FROM %s
		WHERE job_id = ? AND active = 1
		ORDER BY date_created DESC
	`, r.tableName)

	rows, err := db.GetDB().QueryContext(ctx, query, req.JobId)
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

// GetJobActivityRollup returns aggregated costs grouped by entry_type for a job.
//
// Dialect: $1 → ?, active = true → active = 1. SUM unchanged.
func (r *MySQLJobActivityRepository) GetJobActivityRollup(ctx context.Context, req *pb.GetJobActivityRollupRequest) (*pb.GetJobActivityRollupResponse, error) {
	if req.JobId == "" {
		return nil, fmt.Errorf("job ID is required")
	}

	db, ok := r.dbOps.(interface{ GetDB() *sql.DB })
	if !ok {
		return nil, fmt.Errorf("database operations does not support raw SQL queries")
	}

	// Dialect: $1 → ?, active = true → active = 1
	query := fmt.Sprintf(`
		SELECT entry_type, SUM(total_cost) as total_cost, COUNT(*) as count
		FROM %s
		WHERE job_id = ? AND active = 1
		GROUP BY entry_type
		ORDER BY entry_type
	`, r.tableName)

	rows, err := db.GetDB().QueryContext(ctx, query, req.JobId)
	if err != nil {
		return nil, fmt.Errorf("failed to get job activity rollup: %w", err)
	}
	defer rows.Close()

	var rollup []*pb.CostByEntryType
	var grandTotal int64

	for rows.Next() {
		var (
			entryType string
			totalCost int64
			count     int32
		)
		if err := rows.Scan(&entryType, &totalCost, &count); err != nil {
			return nil, fmt.Errorf("failed to scan rollup row: %w", err)
		}

		entry := &pb.CostByEntryType{
			TotalCost: totalCost,
			Count:     count,
		}
		if v, ok := pb.EntryType_value[entryType]; ok {
			entry.EntryType = pb.EntryType(v)
		}

		rollup = append(rollup, entry)
		grandTotal += totalCost
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rollup rows: %w", err)
	}

	return &pb.GetJobActivityRollupResponse{
		Rollup:     rollup,
		GrandTotal: grandTotal,
		Success:    true,
	}, nil
}

// SubmitForApproval transitions activity from DRAFT to SUBMITTED.
//
// Dialect: RETURNING → RowsAffected check + re-read via ReadJobActivity.
// active = true → active = 1.
func (r *MySQLJobActivityRepository) SubmitForApproval(ctx context.Context, req *pb.SubmitForApprovalRequest) (*pb.SubmitForApprovalResponse, error) {
	if req.ActivityId == "" {
		return nil, fmt.Errorf("activity ID is required")
	}

	db, ok := r.dbOps.(interface{ GetDB() *sql.DB })
	if !ok {
		return nil, fmt.Errorf("database operations does not support raw SQL queries")
	}

	// Dialect: RETURNING → UPDATE + RowsAffected; active = true → active = 1
	query := fmt.Sprintf(`
		UPDATE %s SET approval_status = 'ACTIVITY_APPROVAL_STATUS_SUBMITTED', date_modified = NOW()
		WHERE id = ? AND active = 1 AND approval_status = 'ACTIVITY_APPROVAL_STATUS_DRAFT'
	`, r.tableName)

	res, err := db.GetDB().ExecContext(ctx, query, req.ActivityId)
	if err != nil {
		return nil, fmt.Errorf("failed to submit activity for approval: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check rows affected: %w", err)
	}
	if n == 0 {
		return nil, fmt.Errorf("activity not found or not in DRAFT status: %s", req.ActivityId)
	}

	readResp, err := r.ReadJobActivity(ctx, &pb.ReadJobActivityRequest{Data: &pb.JobActivity{Id: req.ActivityId}})
	if err != nil {
		return nil, err
	}

	var activity *pb.JobActivity
	if len(readResp.Data) > 0 {
		activity = readResp.Data[0]
	}

	return &pb.SubmitForApprovalResponse{
		JobActivity: activity,
		Success:     true,
	}, nil
}

// ApproveActivity transitions activity from SUBMITTED to APPROVED.
func (r *MySQLJobActivityRepository) ApproveActivity(ctx context.Context, req *pb.ApproveJobActivityRequest) (*pb.ApproveJobActivityResponse, error) {
	if req.ActivityId == "" {
		return nil, fmt.Errorf("activity ID is required")
	}

	db, ok := r.dbOps.(interface{ GetDB() *sql.DB })
	if !ok {
		return nil, fmt.Errorf("database operations does not support raw SQL queries")
	}

	query := fmt.Sprintf(`
		UPDATE %s SET approval_status = 'ACTIVITY_APPROVAL_STATUS_APPROVED', date_modified = NOW()
		WHERE id = ? AND active = 1 AND approval_status = 'ACTIVITY_APPROVAL_STATUS_SUBMITTED'
	`, r.tableName)

	res, err := db.GetDB().ExecContext(ctx, query, req.ActivityId)
	if err != nil {
		return nil, fmt.Errorf("failed to approve activity: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check rows affected: %w", err)
	}
	if n == 0 {
		return nil, fmt.Errorf("activity not found or not in SUBMITTED status: %s", req.ActivityId)
	}

	readResp, err := r.ReadJobActivity(ctx, &pb.ReadJobActivityRequest{Data: &pb.JobActivity{Id: req.ActivityId}})
	if err != nil {
		return nil, err
	}

	var activity *pb.JobActivity
	if len(readResp.Data) > 0 {
		activity = readResp.Data[0]
	}

	return &pb.ApproveJobActivityResponse{
		JobActivity: activity,
		Success:     true,
	}, nil
}

// RejectActivity transitions activity from SUBMITTED to REJECTED.
func (r *MySQLJobActivityRepository) RejectActivity(ctx context.Context, req *pb.RejectJobActivityRequest) (*pb.RejectJobActivityResponse, error) {
	if req.ActivityId == "" {
		return nil, fmt.Errorf("activity ID is required")
	}

	db, ok := r.dbOps.(interface{ GetDB() *sql.DB })
	if !ok {
		return nil, fmt.Errorf("database operations does not support raw SQL queries")
	}

	query := fmt.Sprintf(`
		UPDATE %s SET approval_status = 'ACTIVITY_APPROVAL_STATUS_REJECTED', date_modified = NOW()
		WHERE id = ? AND active = 1 AND approval_status = 'ACTIVITY_APPROVAL_STATUS_SUBMITTED'
	`, r.tableName)

	res, err := db.GetDB().ExecContext(ctx, query, req.ActivityId)
	if err != nil {
		return nil, fmt.Errorf("failed to reject activity: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check rows affected: %w", err)
	}
	if n == 0 {
		return nil, fmt.Errorf("activity not found or not in SUBMITTED status: %s", req.ActivityId)
	}

	readResp, err := r.ReadJobActivity(ctx, &pb.ReadJobActivityRequest{Data: &pb.JobActivity{Id: req.ActivityId}})
	if err != nil {
		return nil, err
	}

	var activity *pb.JobActivity
	if len(readResp.Data) > 0 {
		activity = readResp.Data[0]
	}

	return &pb.RejectJobActivityResponse{
		JobActivity: activity,
		Success:     true,
	}, nil
}

// PostActivity transitions posting_status from UNPOSTED to POSTED.
func (r *MySQLJobActivityRepository) PostActivity(ctx context.Context, req *pb.PostJobActivityRequest) (*pb.PostJobActivityResponse, error) {
	if req.ActivityId == "" {
		return nil, fmt.Errorf("activity ID is required")
	}

	db, ok := r.dbOps.(interface{ GetDB() *sql.DB })
	if !ok {
		return nil, fmt.Errorf("database operations does not support raw SQL queries")
	}

	// Dialect: $1, $2 → ?, active = true → active = 1, no RETURNING
	query := fmt.Sprintf(`
		UPDATE %s SET
			posting_status = 'ACTIVITY_POSTING_STATUS_POSTED',
			posted_by = ?,
			date_posted = NOW(),
			date_modified = NOW()
		WHERE id = ? AND active = 1
			AND approval_status = 'ACTIVITY_APPROVAL_STATUS_APPROVED'
			AND posting_status = 'ACTIVITY_POSTING_STATUS_UNPOSTED'
	`, r.tableName)

	res, err := db.GetDB().ExecContext(ctx, query, req.PostedBy, req.ActivityId)
	if err != nil {
		return nil, fmt.Errorf("failed to post activity: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check rows affected: %w", err)
	}
	if n == 0 {
		return nil, fmt.Errorf("activity not found, not approved, or already posted: %s", req.ActivityId)
	}

	readResp, err := r.ReadJobActivity(ctx, &pb.ReadJobActivityRequest{Data: &pb.JobActivity{Id: req.ActivityId}})
	if err != nil {
		return nil, err
	}

	var activity *pb.JobActivity
	if len(readResp.Data) > 0 {
		activity = readResp.Data[0]
	}

	return &pb.PostJobActivityResponse{
		JobActivity: activity,
		Success:     true,
	}, nil
}

// ReverseActivity marks original activity as REVERSED.
func (r *MySQLJobActivityRepository) ReverseActivity(ctx context.Context, req *pb.ReverseJobActivityRequest) (*pb.ReverseJobActivityResponse, error) {
	if req.ActivityId == "" {
		return nil, fmt.Errorf("activity ID is required")
	}

	db, ok := r.dbOps.(interface{ GetDB() *sql.DB })
	if !ok {
		return nil, fmt.Errorf("database operations does not support raw SQL queries")
	}

	// Dialect: $1 → ?, active = true → active = 1, no RETURNING
	query := fmt.Sprintf(`
		UPDATE %s SET
			posting_status = 'ACTIVITY_POSTING_STATUS_REVERSED',
			date_modified = NOW()
		WHERE id = ? AND active = 1
			AND posting_status = 'ACTIVITY_POSTING_STATUS_POSTED'
	`, r.tableName)

	res, err := db.GetDB().ExecContext(ctx, query, req.ActivityId)
	if err != nil {
		return nil, fmt.Errorf("failed to reverse activity: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check rows affected: %w", err)
	}
	if n == 0 {
		return nil, fmt.Errorf("activity not found or not in POSTED status: %s", req.ActivityId)
	}

	readResp, err := r.ReadJobActivity(ctx, &pb.ReadJobActivityRequest{Data: &pb.JobActivity{Id: req.ActivityId}})
	if err != nil {
		return nil, err
	}

	var activity *pb.JobActivity
	if len(readResp.Data) > 0 {
		activity = readResp.Data[0]
	}

	return &pb.ReverseJobActivityResponse{
		JobActivity: activity,
		Success:     true,
	}, nil
}

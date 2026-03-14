//go:build postgresql

package job_settlement

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"

	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_settlement"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.JobSettlement, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres job_settlement repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresJobSettlementRepository(dbOps, tableName), nil
	})
}

// PostgresJobSettlementRepository implements job_settlement CRUD + custom operations using PostgreSQL
type PostgresJobSettlementRepository struct {
	pb.UnimplementedJobSettlementDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresJobSettlementRepository creates a new PostgreSQL job_settlement repository
func NewPostgresJobSettlementRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.JobSettlementDomainServiceServer {
	if tableName == "" {
		tableName = "job_settlement"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresJobSettlementRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateJobSettlement creates a new job settlement
func (r *PostgresJobSettlementRepository) CreateJobSettlement(ctx context.Context, req *pb.CreateJobSettlementRequest) (*pb.CreateJobSettlementResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("job settlement data is required")
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
		return nil, fmt.Errorf("failed to create job settlement: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	settlement := &pb.JobSettlement{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, settlement); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.CreateJobSettlementResponse{
		Data:    []*pb.JobSettlement{settlement},
		Success: true,
	}, nil
}

// ReadJobSettlement retrieves a job settlement by ID
func (r *PostgresJobSettlementRepository) ReadJobSettlement(ctx context.Context, req *pb.ReadJobSettlementRequest) (*pb.ReadJobSettlementResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job settlement ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read job settlement: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	settlement := &pb.JobSettlement{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, settlement); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.ReadJobSettlementResponse{
		Data:    []*pb.JobSettlement{settlement},
		Success: true,
	}, nil
}

// UpdateJobSettlement updates a job settlement
func (r *PostgresJobSettlementRepository) UpdateJobSettlement(ctx context.Context, req *pb.UpdateJobSettlementRequest) (*pb.UpdateJobSettlementResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job settlement ID is required")
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
		return nil, fmt.Errorf("failed to update job settlement: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	settlement := &pb.JobSettlement{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, settlement); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.UpdateJobSettlementResponse{
		Data:    []*pb.JobSettlement{settlement},
		Success: true,
	}, nil
}

// DeleteJobSettlement soft-deletes a job settlement
func (r *PostgresJobSettlementRepository) DeleteJobSettlement(ctx context.Context, req *pb.DeleteJobSettlementRequest) (*pb.DeleteJobSettlementResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job settlement ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete job settlement: %w", err)
	}

	return &pb.DeleteJobSettlementResponse{
		Success: true,
	}, nil
}

// ListJobSettlements lists all active job settlements
func (r *PostgresJobSettlementRepository) ListJobSettlements(ctx context.Context, req *pb.ListJobSettlementsRequest) (*pb.ListJobSettlementsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}

	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list job settlements: %w", err)
	}

	var settlements []*pb.JobSettlement
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		settlement := &pb.JobSettlement{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, settlement); err != nil {
			continue
		}
		settlements = append(settlements, settlement)
	}

	return &pb.ListJobSettlementsResponse{
		Data:    settlements,
		Success: true,
	}, nil
}

// GetJobSettlementListPageData retrieves paginated, filtered, sorted job settlements with activity JOINs
func (r *PostgresJobSettlementRepository) GetJobSettlementListPageData(ctx context.Context, req *pb.GetJobSettlementListPageDataRequest) (*pb.GetJobSettlementListPageDataResponse, error) {
	limit := int32(20)
	page := int32(1)
	if req.Pagination != nil && req.Pagination.Limit > 0 {
		limit = req.Pagination.Limit
		if limit > 100 {
			limit = 100
		}
		if req.Pagination.GetOffset() != nil {
			page = req.Pagination.GetOffset().Page
			if page < 1 {
				page = 1
			}
		}
	}
	offset := (page - 1) * limit

	searchQuery := ""
	if req.Search != nil && req.Search.Query != "" {
		searchQuery = "%" + req.Search.Query + "%"
	}

	sortField := "date_created"
	sortDirection := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == 1 {
			sortDirection = "DESC"
		} else {
			sortDirection = "ASC"
		}
	}

	query := `
		WITH
		search_filtered AS (
			SELECT js.*
			FROM job_settlement js
			WHERE js.active = true
				AND ($1::text = '' OR js.target_id ILIKE $1)
		),
		enriched AS (
			SELECT
				sf.id,
				sf.job_activity_id,
				sf.target_type,
				sf.target_id,
				sf.allocated_amount,
				sf.allocation_pct,
				sf.settlement_date,
				sf.status,
				sf.reversal_of_id,
				sf.created_by,
				sf.date_created,
				sf.active,
				jsonb_build_object(
					'id', ja.id,
					'jobId', ja.job_id,
					'entryType', ja.entry_type,
					'totalCost', ja.total_cost,
					'active', ja.active
				) as job_activity
			FROM search_filtered sf
			LEFT JOIN job_activity ja ON sf.job_activity_id = ja.id AND ja.active = true
		),
		sorted AS (
			SELECT * FROM enriched
			ORDER BY
				CASE WHEN ($4 = 'date_created' OR $4 = '') AND $5 = 'DESC' THEN date_created END DESC,
				CASE WHEN $4 = 'date_created' AND $5 = 'ASC' THEN date_created END ASC,
				CASE WHEN $4 = 'allocated_amount' AND $5 = 'DESC' THEN allocated_amount END DESC,
				CASE WHEN $4 = 'allocated_amount' AND $5 = 'ASC' THEN allocated_amount END ASC,
				CASE WHEN $4 = 'settlement_date' AND $5 = 'DESC' THEN settlement_date END DESC,
				CASE WHEN $4 = 'settlement_date' AND $5 = 'ASC' THEN settlement_date END ASC
		),
		total_count AS (
			SELECT count(*) as total FROM sorted
		)
		SELECT
			s.id,
			s.job_activity_id,
			s.target_type,
			s.target_id,
			s.allocated_amount,
			s.allocation_pct,
			s.settlement_date,
			s.status,
			s.reversal_of_id,
			s.created_by,
			s.date_created,
			s.active,
			s.job_activity,
			tc.total as _total_count
		FROM sorted s
		CROSS JOIN total_count tc
		LIMIT $2 OFFSET $3
	`

	if r.db == nil {
		return nil, fmt.Errorf("database connection not available for raw SQL queries")
	}

	rows, err := r.db.QueryContext(ctx, query, searchQuery, limit, offset, sortField, sortDirection)
	if err != nil {
		return nil, fmt.Errorf("failed to execute GetJobSettlementListPageData query: %w", err)
	}
	defer rows.Close()

	var settlements []*pb.JobSettlement
	var totalCount int32

	for rows.Next() {
		var (
			id              string
			jobActivityId   string
			targetType      string
			targetId        string
			allocatedAmount float64
			allocationPct   sql.NullFloat64
			settlementDate  sql.NullTime
			status          string
			reversalOfId    sql.NullString
			createdBy       sql.NullString
			dateCreated     sql.NullTime
			active          bool
			jobActivityJSON []byte
			rowTotalCount   int32
		)

		err := rows.Scan(
			&id, &jobActivityId, &targetType, &targetId,
			&allocatedAmount, &allocationPct, &settlementDate,
			&status, &reversalOfId, &createdBy, &dateCreated, &active,
			&jobActivityJSON, &rowTotalCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job settlement row: %w", err)
		}

		totalCount = rowTotalCount

		settlement := &pb.JobSettlement{
			Id:              id,
			JobActivityId:   jobActivityId,
			TargetType:      pb.SettlementTargetType(pb.SettlementTargetType_value[targetType]),
			TargetId:        targetId,
			AllocatedAmount: allocatedAmount,
			Status:          pb.SettlementStatus(pb.SettlementStatus_value[status]),
			Active:          active,
		}

		if allocationPct.Valid {
			settlement.AllocationPct = &allocationPct.Float64
		}
		if settlementDate.Valid {
			ts := settlementDate.Time.UnixMilli()
			settlement.SettlementDate = &ts
		}
		if reversalOfId.Valid {
			settlement.ReversalOfId = &reversalOfId.String
		}
		if createdBy.Valid {
			settlement.CreatedBy = &createdBy.String
		}
		if dateCreated.Valid {
			ts := dateCreated.Time.UnixMilli()
			settlement.DateCreated = &ts
		}

		settlements = append(settlements, settlement)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating job settlement rows: %w", err)
	}

	totalPages := (totalCount + limit - 1) / limit
	hasNext := page < totalPages
	hasPrev := page > 1

	return &pb.GetJobSettlementListPageDataResponse{
		Success:           true,
		JobSettlementList: settlements,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  totalCount,
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     hasNext,
			HasPrev:     hasPrev,
		},
	}, nil
}

// GetJobSettlementItemPageData retrieves a single job settlement with activity data
func (r *PostgresJobSettlementRepository) GetJobSettlementItemPageData(ctx context.Context, req *pb.GetJobSettlementItemPageDataRequest) (*pb.GetJobSettlementItemPageDataResponse, error) {
	if req.JobSettlementId == "" {
		return nil, fmt.Errorf("job settlement ID is required")
	}

	// Use ReadJobSettlement for simple item retrieval
	readResp, err := r.ReadJobSettlement(ctx, &pb.ReadJobSettlementRequest{
		Data: &pb.JobSettlement{Id: req.JobSettlementId},
	})
	if err != nil {
		return nil, err
	}
	if readResp == nil || len(readResp.Data) == 0 {
		return nil, fmt.Errorf("job settlement not found with ID: %s", req.JobSettlementId)
	}

	return &pb.GetJobSettlementItemPageDataResponse{
		Success:       true,
		JobSettlement: readResp.Data[0],
	}, nil
}

// ListByActivity returns all settlements for a specific job_activity_id
func (r *PostgresJobSettlementRepository) ListByActivity(ctx context.Context, req *pb.ListJobSettlementsByActivityRequest) (*pb.ListJobSettlementsByActivityResponse, error) {
	if req.JobActivityId == "" {
		return nil, fmt.Errorf("job activity ID is required")
	}

	query := `
		SELECT id, job_activity_id, target_type, target_id,
			allocated_amount, allocation_pct, settlement_date,
			status, reversal_of_id, created_by, date_created, active
		FROM job_settlement
		WHERE job_activity_id = $1 AND active = true
		ORDER BY date_created DESC
	`

	if r.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	rows, err := r.db.QueryContext(ctx, query, req.JobActivityId)
	if err != nil {
		return nil, fmt.Errorf("failed to list settlements by activity: %w", err)
	}
	defer rows.Close()

	var settlements []*pb.JobSettlement
	for rows.Next() {
		s, err := scanSettlementRow(rows)
		if err != nil {
			return nil, err
		}
		settlements = append(settlements, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating settlement rows: %w", err)
	}

	return &pb.ListJobSettlementsByActivityResponse{
		JobSettlements: settlements,
		Success:        true,
	}, nil
}

// ListByTarget returns all settlements for a specific target_type + target_id
func (r *PostgresJobSettlementRepository) ListByTarget(ctx context.Context, req *pb.ListJobSettlementsByTargetRequest) (*pb.ListJobSettlementsByTargetResponse, error) {
	if req.TargetId == "" {
		return nil, fmt.Errorf("target ID is required")
	}

	query := `
		SELECT id, job_activity_id, target_type, target_id,
			allocated_amount, allocation_pct, settlement_date,
			status, reversal_of_id, created_by, date_created, active
		FROM job_settlement
		WHERE target_type = $1 AND target_id = $2 AND active = true
		ORDER BY date_created DESC
	`

	if r.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	rows, err := r.db.QueryContext(ctx, query, req.TargetType.String(), req.TargetId)
	if err != nil {
		return nil, fmt.Errorf("failed to list settlements by target: %w", err)
	}
	defer rows.Close()

	var settlements []*pb.JobSettlement
	for rows.Next() {
		s, err := scanSettlementRow(rows)
		if err != nil {
			return nil, err
		}
		settlements = append(settlements, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating settlement rows: %w", err)
	}

	return &pb.ListJobSettlementsByTargetResponse{
		JobSettlements: settlements,
		Success:        true,
	}, nil
}

// GetSettlementSummary returns aggregated allocations per target_type for a job
func (r *PostgresJobSettlementRepository) GetSettlementSummary(ctx context.Context, req *pb.GetSettlementSummaryRequest) (*pb.GetSettlementSummaryResponse, error) {
	if req.JobId == "" {
		return nil, fmt.Errorf("job ID is required")
	}

	query := `
		SELECT
			js.target_type,
			SUM(js.allocated_amount) as total_amount,
			COUNT(*) as count
		FROM job_settlement js
		INNER JOIN job_activity ja ON js.job_activity_id = ja.id
		WHERE ja.job_id = $1
			AND js.active = true
			AND js.status != 'SETTLEMENT_STATUS_REVERSED'
		GROUP BY js.target_type
	`

	if r.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	rows, err := r.db.QueryContext(ctx, query, req.JobId)
	if err != nil {
		return nil, fmt.Errorf("failed to get settlement summary: %w", err)
	}
	defer rows.Close()

	var summary []*pb.SettlementByTargetType
	var grandTotal float64

	for rows.Next() {
		var (
			targetType  string
			totalAmount float64
			count       int32
		)
		if err := rows.Scan(&targetType, &totalAmount, &count); err != nil {
			return nil, fmt.Errorf("failed to scan settlement summary row: %w", err)
		}

		summary = append(summary, &pb.SettlementByTargetType{
			TargetType:  pb.SettlementTargetType(pb.SettlementTargetType_value[targetType]),
			TotalAmount: totalAmount,
			Count:       count,
		})
		grandTotal += totalAmount
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating settlement summary rows: %w", err)
	}

	return &pb.GetSettlementSummaryResponse{
		Summary:    summary,
		GrandTotal: grandTotal,
		Success:    true,
	}, nil
}

// scanSettlementRow is a helper to scan a single job_settlement row
func scanSettlementRow(rows *sql.Rows) (*pb.JobSettlement, error) {
	var (
		id              string
		jobActivityId   string
		targetType      string
		targetId        string
		allocatedAmount float64
		allocationPct   sql.NullFloat64
		settlementDate  sql.NullTime
		status          string
		reversalOfId    sql.NullString
		createdBy       sql.NullString
		dateCreated     sql.NullTime
		active          bool
	)

	err := rows.Scan(
		&id, &jobActivityId, &targetType, &targetId,
		&allocatedAmount, &allocationPct, &settlementDate,
		&status, &reversalOfId, &createdBy, &dateCreated, &active,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan settlement row: %w", err)
	}

	settlement := &pb.JobSettlement{
		Id:              id,
		JobActivityId:   jobActivityId,
		TargetType:      pb.SettlementTargetType(pb.SettlementTargetType_value[targetType]),
		TargetId:        targetId,
		AllocatedAmount: allocatedAmount,
		Status:          pb.SettlementStatus(pb.SettlementStatus_value[status]),
		Active:          active,
	}

	if allocationPct.Valid {
		settlement.AllocationPct = &allocationPct.Float64
	}
	if settlementDate.Valid {
		ts := settlementDate.Time.UnixMilli()
		settlement.SettlementDate = &ts
	}
	if reversalOfId.Valid {
		settlement.ReversalOfId = &reversalOfId.String
	}
	if createdBy.Valid {
		settlement.CreatedBy = &createdBy.String
	}
	if dateCreated.Valid {
		ts := dateCreated.Time.UnixMilli()
		settlement.DateCreated = &ts
	}

	return settlement, nil
}

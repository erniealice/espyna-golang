//go:build postgresql

package operation

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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/activity_labor"
)

// PostgresActivityLaborRepository implements activity_labor CRUD operations using PostgreSQL
type PostgresActivityLaborRepository struct {
	pb.UnimplementedActivityLaborDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.ActivityLabor, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres activity_labor repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresActivityLaborRepository(dbOps, tableName), nil
	})
}

// NewPostgresActivityLaborRepository creates a new PostgreSQL activity labor repository
func NewPostgresActivityLaborRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.ActivityLaborDomainServiceServer {
	if tableName == "" {
		tableName = "activity_labor"
	}
	return &PostgresActivityLaborRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateActivityLabor creates a new activity labor record
// activity_id is the PK (1:1 with job_activity)
func (r *PostgresActivityLaborRepository) CreateActivityLabor(ctx context.Context, req *pb.CreateActivityLaborRequest) (*pb.CreateActivityLaborResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("activity labor data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// activity_labor uses activity_id as PK, map it to id for dbOps
	if activityId, ok := data["activityId"]; ok {
		data["id"] = activityId
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create activity labor: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	labor := &pb.ActivityLabor{}
	if err := protojson.Unmarshal(resultJSON, labor); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.CreateActivityLaborResponse{
		Data: []*pb.ActivityLabor{labor},
	}, nil
}

// ReadActivityLabor retrieves an activity labor by activity_id
func (r *PostgresActivityLaborRepository) ReadActivityLabor(ctx context.Context, req *pb.ReadActivityLaborRequest) (*pb.ReadActivityLaborResponse, error) {
	if req.Data == nil || req.Data.ActivityId == "" {
		return nil, fmt.Errorf("activity_id is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.ActivityId)
	if err != nil {
		return nil, fmt.Errorf("failed to read activity labor: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	labor := &pb.ActivityLabor{}
	if err := protojson.Unmarshal(resultJSON, labor); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.ReadActivityLaborResponse{
		Data: []*pb.ActivityLabor{labor},
	}, nil
}

// UpdateActivityLabor updates an activity labor record
func (r *PostgresActivityLaborRepository) UpdateActivityLabor(ctx context.Context, req *pb.UpdateActivityLaborRequest) (*pb.UpdateActivityLaborResponse, error) {
	if req.Data == nil || req.Data.ActivityId == "" {
		return nil, fmt.Errorf("activity_id is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.ActivityId, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update activity labor: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	labor := &pb.ActivityLabor{}
	if err := protojson.Unmarshal(resultJSON, labor); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.UpdateActivityLaborResponse{
		Data: []*pb.ActivityLabor{labor},
	}, nil
}

// DeleteActivityLabor deletes an activity labor record (soft delete)
func (r *PostgresActivityLaborRepository) DeleteActivityLabor(ctx context.Context, req *pb.DeleteActivityLaborRequest) (*pb.DeleteActivityLaborResponse, error) {
	if req.Data == nil || req.Data.ActivityId == "" {
		return nil, fmt.Errorf("activity_id is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.ActivityId)
	if err != nil {
		return nil, fmt.Errorf("failed to delete activity labor: %w", err)
	}

	return &pb.DeleteActivityLaborResponse{
		Success: true,
	}, nil
}

// ListActivityLabors lists activity labor records with optional filters
func (r *PostgresActivityLaborRepository) ListActivityLabors(ctx context.Context, req *pb.ListActivityLaborsRequest) (*pb.ListActivityLaborsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list activity labors: %w", err)
	}

	var labors []*pb.ActivityLabor
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}

		labor := &pb.ActivityLabor{}
		if err := protojson.Unmarshal(resultJSON, labor); err != nil {
			continue
		}
		labors = append(labors, labor)
	}

	return &pb.ListActivityLaborsResponse{
		Data: labors,
	}, nil
}

// GetActivityLaborListPageData retrieves paginated activity labor list
func (r *PostgresActivityLaborRepository) GetActivityLaborListPageData(ctx context.Context, req *pb.GetActivityLaborListPageDataRequest) (*pb.GetActivityLaborListPageDataResponse, error) {
	// TODO: Implement CTE-based paginated query with staff/job_activity joins
	return nil, fmt.Errorf("GetActivityLaborListPageData not yet implemented")
}

// GetActivityLaborItemPageData retrieves a single activity labor with related data
func (r *PostgresActivityLaborRepository) GetActivityLaborItemPageData(ctx context.Context, req *pb.GetActivityLaborItemPageDataRequest) (*pb.GetActivityLaborItemPageDataResponse, error) {
	// TODO: Implement CTE-based single item query with staff, user, job_activity joins
	return nil, fmt.Errorf("GetActivityLaborItemPageData not yet implemented")
}

// ListByStaff lists all labor records for a given staff member
func (r *PostgresActivityLaborRepository) ListByStaff(ctx context.Context, req *pb.ListActivityLaborsByStaffRequest) (*pb.ListActivityLaborsByStaffResponse, error) {
	if req.StaffId == "" {
		return nil, fmt.Errorf("staff ID is required")
	}

	db, ok := r.dbOps.(interface{ GetDB() *sql.DB })
	if !ok {
		return nil, fmt.Errorf("database operations does not support raw SQL queries")
	}

	query := fmt.Sprintf(`
		SELECT al.activity_id, al.staff_id, al.hours, al.rate_type,
			   al.time_start, al.time_end
		FROM %s al
		WHERE al.staff_id = $1
		ORDER BY al.time_start DESC
	`, r.tableName)

	rows, err := db.GetDB().QueryContext(ctx, query, req.StaffId)
	if err != nil {
		return nil, fmt.Errorf("failed to list activity labors by staff: %w", err)
	}
	defer rows.Close()

	var labors []*pb.ActivityLabor
	for rows.Next() {
		var (
			activityId string
			staffId    string
			hours      float64
			rateType   string
			timeStart  sql.NullTime
			timeEnd    sql.NullTime
		)

		if err := rows.Scan(&activityId, &staffId, &hours, &rateType, &timeStart, &timeEnd); err != nil {
			return nil, fmt.Errorf("failed to scan activity labor row: %w", err)
		}

		labor := &pb.ActivityLabor{
			ActivityId: activityId,
			StaffId:    staffId,
			Hours:      hours,
		}

		if v, ok := pb.RateType_value[rateType]; ok {
			labor.RateType = pb.RateType(v)
		}
		if timeStart.Valid {
			ts := timeStart.Time.Unix()
			labor.TimeStart = &ts
		}
		if timeEnd.Valid {
			ts := timeEnd.Time.Unix()
			labor.TimeEnd = &ts
		}

		labors = append(labors, labor)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating activity labor rows: %w", err)
	}

	return &pb.ListActivityLaborsByStaffResponse{
		ActivityLabors: labors,
		Success:        true,
	}, nil
}

// ListByJob lists all labor records for a given job (joins through job_activity)
func (r *PostgresActivityLaborRepository) ListByJob(ctx context.Context, req *pb.ListActivityLaborsByJobRequest) (*pb.ListActivityLaborsByJobResponse, error) {
	if req.JobId == "" {
		return nil, fmt.Errorf("job ID is required")
	}

	db, ok := r.dbOps.(interface{ GetDB() *sql.DB })
	if !ok {
		return nil, fmt.Errorf("database operations does not support raw SQL queries")
	}

	query := fmt.Sprintf(`
		SELECT al.activity_id, al.staff_id, al.hours, al.rate_type,
			   al.time_start, al.time_end
		FROM %s al
		INNER JOIN job_activity ja ON al.activity_id = ja.id AND ja.active = true
		WHERE ja.job_id = $1
		ORDER BY al.time_start DESC
	`, r.tableName)

	rows, err := db.GetDB().QueryContext(ctx, query, req.JobId)
	if err != nil {
		return nil, fmt.Errorf("failed to list activity labors by job: %w", err)
	}
	defer rows.Close()

	var labors []*pb.ActivityLabor
	for rows.Next() {
		var (
			activityId string
			staffId    string
			hours      float64
			rateType   string
			timeStart  sql.NullTime
			timeEnd    sql.NullTime
		)

		if err := rows.Scan(&activityId, &staffId, &hours, &rateType, &timeStart, &timeEnd); err != nil {
			return nil, fmt.Errorf("failed to scan activity labor row: %w", err)
		}

		labor := &pb.ActivityLabor{
			ActivityId: activityId,
			StaffId:    staffId,
			Hours:      hours,
		}

		if v, ok := pb.RateType_value[rateType]; ok {
			labor.RateType = pb.RateType(v)
		}
		if timeStart.Valid {
			ts := timeStart.Time.Unix()
			labor.TimeStart = &ts
		}
		if timeEnd.Valid {
			ts := timeEnd.Time.Unix()
			labor.TimeEnd = &ts
		}

		labors = append(labors, labor)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating activity labor rows: %w", err)
	}

	return &pb.ListActivityLaborsByJobResponse{
		ActivityLabors: labors,
		Success:        true,
	}, nil
}

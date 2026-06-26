//go:build postgresql

package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	activitypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/activity"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.Activity, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres activity repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresActivityRepository(dbOps, tableName), nil
	})
}

// PostgresActivityRepository implements activity CRUD operations using PostgreSQL.
type PostgresActivityRepository struct {
	activitypb.UnimplementedActivityDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresActivityRepository creates a new PostgreSQL activity repository
func NewPostgresActivityRepository(dbOps interfaces.DatabaseOperation, tableName string) activitypb.ActivityDomainServiceServer {
	if tableName == "" {
		tableName = "activity" // default fallback
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresActivityRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateActivity creates a new activity using common PostgreSQL operations
func (r *PostgresActivityRepository) CreateActivity(ctx context.Context, req *activitypb.CreateActivityRequest) (*activitypb.CreateActivityResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("activity data is required")
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
		return nil, fmt.Errorf("failed to create activity: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	activity := &activitypb.Activity{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, activity); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &activitypb.CreateActivityResponse{
		Data:    []*activitypb.Activity{activity},
		Success: true,
	}, nil
}

// ReadActivity retrieves an activity using common PostgreSQL operations
func (r *PostgresActivityRepository) ReadActivity(ctx context.Context, req *activitypb.ReadActivityRequest) (*activitypb.ReadActivityResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("activity ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read activity: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	activity := &activitypb.Activity{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, activity); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &activitypb.ReadActivityResponse{
		Data:    []*activitypb.Activity{activity},
		Success: true,
	}, nil
}

// UpdateActivity updates an activity using common PostgreSQL operations
func (r *PostgresActivityRepository) UpdateActivity(ctx context.Context, req *activitypb.UpdateActivityRequest) (*activitypb.UpdateActivityResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("activity ID is required")
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
		return nil, fmt.Errorf("failed to update activity: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	activity := &activitypb.Activity{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, activity); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &activitypb.UpdateActivityResponse{
		Data:    []*activitypb.Activity{activity},
		Success: true,
	}, nil
}

// DeleteActivity deletes an activity using common PostgreSQL operations (soft delete)
func (r *PostgresActivityRepository) DeleteActivity(ctx context.Context, req *activitypb.DeleteActivityRequest) (*activitypb.DeleteActivityResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("activity ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete activity: %w", err)
	}

	return &activitypb.DeleteActivityResponse{
		Success: true,
	}, nil
}

// ListActivities lists activities using common PostgreSQL operations
func (r *PostgresActivityRepository) ListActivities(ctx context.Context, req *activitypb.ListActivitiesRequest) (*activitypb.ListActivitiesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list activities: %w", err)
	}

	var activities []*activitypb.Activity
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}

		activity := &activitypb.Activity{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, activity); err != nil {
			continue
		}
		activities = append(activities, activity)
	}

	if activities == nil {
		activities = make([]*activitypb.Activity, 0)
	}

	return &activitypb.ListActivitiesResponse{
		Data:    activities,
		Success: true,
	}, nil
}

// GetActivityListPageData retrieves activities with basic pagination via List.
func (r *PostgresActivityRepository) GetActivityListPageData(ctx context.Context, req *activitypb.GetActivityListPageDataRequest) (*activitypb.GetActivityListPageDataResponse, error) {
	listReq := &activitypb.ListActivitiesRequest{}
	listResp, err := r.ListActivities(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity list page data: %w", err)
	}

	return &activitypb.GetActivityListPageDataResponse{
		ActivityList: listResp.Data,
		Success:      true,
	}, nil
}

// GetActivityItemPageData retrieves a single activity via Read.
func (r *PostgresActivityRepository) GetActivityItemPageData(ctx context.Context, req *activitypb.GetActivityItemPageDataRequest) (*activitypb.GetActivityItemPageDataResponse, error) {
	if req.ActivityId == "" {
		return nil, fmt.Errorf("activity ID is required")
	}

	readReq := &activitypb.ReadActivityRequest{
		Data: &activitypb.Activity{Id: req.ActivityId},
	}
	readResp, err := r.ReadActivity(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity item page data: %w", err)
	}

	if len(readResp.Data) == 0 {
		return nil, fmt.Errorf("activity not found")
	}

	return &activitypb.GetActivityItemPageDataResponse{
		Activity: readResp.Data[0],
		Success:  true,
	}, nil
}

// NewActivityRepository creates a new PostgreSQL activity repository (old-style constructor)
func NewActivityRepository(db *sql.DB, tableName string) activitypb.ActivityDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresActivityRepository(dbOps, tableName)
}

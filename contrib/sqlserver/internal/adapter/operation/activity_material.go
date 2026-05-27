//go:build sqlserver

package operation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/activity_material"
)

// SQLServerActivityMaterialRepository implements activity_material CRUD operations using SQL Server.
// activity_material uses activity_id as its PK (1:1 with job_activity).
//
// SQL Server dialect differences vs postgres gold standard:
//   - Placeholders: $1 → @p1
//   - active = true → active = 1
//   - RETURNING → OUTPUT inserted.*
type SQLServerActivityMaterialRepository struct {
	pb.UnimplementedActivityMaterialDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.ActivityMaterial, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver activity_material repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerActivityMaterialRepository(dbOps, tableName), nil
	})
}

// NewSQLServerActivityMaterialRepository creates a new SQL Server activity material repository.
func NewSQLServerActivityMaterialRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.ActivityMaterialDomainServiceServer {
	if tableName == "" {
		tableName = "activity_material"
	}
	return &SQLServerActivityMaterialRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateActivityMaterial creates a new activity material record.
// activity_id is the PK (1:1 with job_activity).
func (r *SQLServerActivityMaterialRepository) CreateActivityMaterial(ctx context.Context, req *pb.CreateActivityMaterialRequest) (*pb.CreateActivityMaterialResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("activity material data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// activity_material uses activity_id as PK; map it to id for dbOps.
	if activityId, ok := data["activityId"]; ok {
		data["id"] = activityId
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create activity material: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	material := &pb.ActivityMaterial{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, material); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.CreateActivityMaterialResponse{
		Data: []*pb.ActivityMaterial{material},
	}, nil
}

// ReadActivityMaterial retrieves an activity material by activity_id.
func (r *SQLServerActivityMaterialRepository) ReadActivityMaterial(ctx context.Context, req *pb.ReadActivityMaterialRequest) (*pb.ReadActivityMaterialResponse, error) {
	if req.Data == nil || req.Data.ActivityId == "" {
		return nil, fmt.Errorf("activity_id is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.ActivityId)
	if err != nil {
		return nil, fmt.Errorf("failed to read activity material: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	material := &pb.ActivityMaterial{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, material); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.ReadActivityMaterialResponse{
		Data: []*pb.ActivityMaterial{material},
	}, nil
}

// UpdateActivityMaterial updates an activity material record.
func (r *SQLServerActivityMaterialRepository) UpdateActivityMaterial(ctx context.Context, req *pb.UpdateActivityMaterialRequest) (*pb.UpdateActivityMaterialResponse, error) {
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
		return nil, fmt.Errorf("failed to update activity material: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	material := &pb.ActivityMaterial{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, material); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.UpdateActivityMaterialResponse{
		Data: []*pb.ActivityMaterial{material},
	}, nil
}

// DeleteActivityMaterial soft-deletes an activity material record.
func (r *SQLServerActivityMaterialRepository) DeleteActivityMaterial(ctx context.Context, req *pb.DeleteActivityMaterialRequest) (*pb.DeleteActivityMaterialResponse, error) {
	if req.Data == nil || req.Data.ActivityId == "" {
		return nil, fmt.Errorf("activity_id is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.ActivityId)
	if err != nil {
		return nil, fmt.Errorf("failed to delete activity material: %w", err)
	}

	return &pb.DeleteActivityMaterialResponse{
		Success: true,
	}, nil
}

// ListActivityMaterials lists activity material records with optional filters.
func (r *SQLServerActivityMaterialRepository) ListActivityMaterials(ctx context.Context, req *pb.ListActivityMaterialsRequest) (*pb.ListActivityMaterialsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list activity materials: %w", err)
	}

	var materials []*pb.ActivityMaterial
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}

		material := &pb.ActivityMaterial{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, material); err != nil {
			continue
		}
		materials = append(materials, material)
	}

	return &pb.ListActivityMaterialsResponse{
		Data: materials,
	}, nil
}

// GetActivityMaterialListPageData retrieves paginated activity material list.
// TODO: Implement CTE-based paginated query with product/location/job_activity joins.
func (r *SQLServerActivityMaterialRepository) GetActivityMaterialListPageData(ctx context.Context, req *pb.GetActivityMaterialListPageDataRequest) (*pb.GetActivityMaterialListPageDataResponse, error) {
	return nil, fmt.Errorf("GetActivityMaterialListPageData not yet implemented")
}

// GetActivityMaterialItemPageData retrieves a single activity material with related data.
// TODO: Implement CTE-based single item query with product, location, job_activity joins.
func (r *SQLServerActivityMaterialRepository) GetActivityMaterialItemPageData(ctx context.Context, req *pb.GetActivityMaterialItemPageDataRequest) (*pb.GetActivityMaterialItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetActivityMaterialItemPageData not yet implemented")
}

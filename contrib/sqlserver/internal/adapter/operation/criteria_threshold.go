//go:build sqlserver

package operation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/criteria_threshold"
	enums "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.CriteriaThreshold, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver criteria_threshold repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerCriteriaThresholdRepository(dbOps, tableName), nil
	})
}

// SQLServerCriteriaThresholdRepository implements criteria_threshold CRUD operations using SQL Server.
type SQLServerCriteriaThresholdRepository struct {
	pb.UnimplementedCriteriaThresholdDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerCriteriaThresholdRepository creates a new SQL Server criteria_threshold repository.
func NewSQLServerCriteriaThresholdRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.CriteriaThresholdDomainServiceServer {
	if tableName == "" {
		tableName = "criteria_threshold"
	}
	return &SQLServerCriteriaThresholdRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

func (r *SQLServerCriteriaThresholdRepository) CreateCriteriaThreshold(ctx context.Context, req *pb.CreateCriteriaThresholdRequest) (*pb.CreateCriteriaThresholdResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("criteria threshold data is required")
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
		return nil, fmt.Errorf("failed to create criteria threshold: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	t := &pb.CriteriaThreshold{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, t); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &pb.CreateCriteriaThresholdResponse{Success: true, Data: []*pb.CriteriaThreshold{t}}, nil
}

func (r *SQLServerCriteriaThresholdRepository) ReadCriteriaThreshold(ctx context.Context, req *pb.ReadCriteriaThresholdRequest) (*pb.ReadCriteriaThresholdResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("criteria threshold ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read criteria threshold: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	t := &pb.CriteriaThreshold{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, t); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &pb.ReadCriteriaThresholdResponse{Success: true, Data: []*pb.CriteriaThreshold{t}}, nil
}

func (r *SQLServerCriteriaThresholdRepository) UpdateCriteriaThreshold(ctx context.Context, req *pb.UpdateCriteriaThresholdRequest) (*pb.UpdateCriteriaThresholdResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("criteria threshold ID is required")
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
		return nil, fmt.Errorf("failed to update criteria threshold: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	t := &pb.CriteriaThreshold{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, t); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &pb.UpdateCriteriaThresholdResponse{Success: true, Data: []*pb.CriteriaThreshold{t}}, nil
}

func (r *SQLServerCriteriaThresholdRepository) DeleteCriteriaThreshold(ctx context.Context, req *pb.DeleteCriteriaThresholdRequest) (*pb.DeleteCriteriaThresholdResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("criteria threshold ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete criteria threshold: %w", err)
	}
	return &pb.DeleteCriteriaThresholdResponse{Success: true}, nil
}

func (r *SQLServerCriteriaThresholdRepository) ListCriteriaThresholds(ctx context.Context, req *pb.ListCriteriaThresholdsRequest) (*pb.ListCriteriaThresholdsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list criteria thresholds: %w", err)
	}
	var items []*pb.CriteriaThreshold
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal criteria_threshold row: %v", err)
			continue
		}
		t := &pb.CriteriaThreshold{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, t); err != nil {
			log.Printf("WARN: protojson unmarshal criteria_threshold: %v", err)
			continue
		}
		items = append(items, t)
	}
	return &pb.ListCriteriaThresholdsResponse{Success: true, Data: items}, nil
}

func (r *SQLServerCriteriaThresholdRepository) GetCriteriaThresholdListPageData(ctx context.Context, req *pb.GetCriteriaThresholdListPageDataRequest) (*pb.GetCriteriaThresholdListPageDataResponse, error) {
	// TODO: Implement CTE-based paginated query with threshold_role enum scan.
	// Use enums.ThresholdRole for row scanning (same as postgres gold standard).
	_ = enums.ThresholdRole_name // ensure compile-time reference
	return nil, fmt.Errorf("GetCriteriaThresholdListPageData not yet implemented")
}

func (r *SQLServerCriteriaThresholdRepository) GetCriteriaThresholdItemPageData(ctx context.Context, req *pb.GetCriteriaThresholdItemPageDataRequest) (*pb.GetCriteriaThresholdItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetCriteriaThresholdItemPageData not yet implemented")
}

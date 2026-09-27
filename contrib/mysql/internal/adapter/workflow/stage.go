//go:build mysql

package workflow

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"fmt"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	stagepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/stage"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.Stage, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql stage repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLStageRepository(dbOps, tableName), nil
	})
}

// MySQLStageRepository implements stage CRUD operations using MySQL.
type MySQLStageRepository struct {
	stagepb.UnimplementedStageDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewMySQLStageRepository creates a new MySQL stage repository
func NewMySQLStageRepository(dbOps interfaces.DatabaseOperation, tableName string) stagepb.StageDomainServiceServer {
	if tableName == "" {
		tableName = "stage" // default fallback
	}

	return &MySQLStageRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateStage creates a new stage using common MySQL operations
func (r *MySQLStageRepository) CreateStage(ctx context.Context, req *stagepb.CreateStageRequest) (*stagepb.CreateStageResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("stage data is required")
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
		return nil, fmt.Errorf("failed to create stage: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	stage := &stagepb.Stage{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, stage); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &stagepb.CreateStageResponse{
		Data:    []*stagepb.Stage{stage},
		Success: true,
	}, nil
}

// ReadStage retrieves a stage using common MySQL operations
func (r *MySQLStageRepository) ReadStage(ctx context.Context, req *stagepb.ReadStageRequest) (*stagepb.ReadStageResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("stage ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read stage: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	stage := &stagepb.Stage{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, stage); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &stagepb.ReadStageResponse{
		Data:    []*stagepb.Stage{stage},
		Success: true,
	}, nil
}

// UpdateStage updates a stage using common MySQL operations
func (r *MySQLStageRepository) UpdateStage(ctx context.Context, req *stagepb.UpdateStageRequest) (*stagepb.UpdateStageResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("stage ID is required")
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
		return nil, fmt.Errorf("failed to update stage: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	stage := &stagepb.Stage{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, stage); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &stagepb.UpdateStageResponse{
		Data:    []*stagepb.Stage{stage},
		Success: true,
	}, nil
}

// DeleteStage deletes a stage using common MySQL operations (soft delete)
func (r *MySQLStageRepository) DeleteStage(ctx context.Context, req *stagepb.DeleteStageRequest) (*stagepb.DeleteStageResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("stage ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete stage: %w", err)
	}

	return &stagepb.DeleteStageResponse{
		Success: true,
	}, nil
}

// ListStages lists stages using common MySQL operations
func (r *MySQLStageRepository) ListStages(ctx context.Context, req *stagepb.ListStagesRequest) (*stagepb.ListStagesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list stages: %w", err)
	}

	var stages []*stagepb.Stage
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}

		stage := &stagepb.Stage{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, stage); err != nil {
			continue
		}
		stages = append(stages, stage)
	}

	if stages == nil {
		stages = make([]*stagepb.Stage, 0)
	}

	return &stagepb.ListStagesResponse{
		Data:    stages,
		Success: true,
	}, nil
}

// GetStageListPageData retrieves stages with basic pagination via List.
func (r *MySQLStageRepository) GetStageListPageData(ctx context.Context, req *stagepb.GetStageListPageDataRequest) (*stagepb.GetStageListPageDataResponse, error) {
	listReq := &stagepb.ListStagesRequest{}
	listResp, err := r.ListStages(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get stage list page data: %w", err)
	}

	return &stagepb.GetStageListPageDataResponse{
		StageList: listResp.Data,
		Success:   true,
	}, nil
}

// GetStageItemPageData retrieves a single stage via Read.
func (r *MySQLStageRepository) GetStageItemPageData(ctx context.Context, req *stagepb.GetStageItemPageDataRequest) (*stagepb.GetStageItemPageDataResponse, error) {
	if req.StageId == "" {
		return nil, fmt.Errorf("stage ID is required")
	}

	readReq := &stagepb.ReadStageRequest{
		Data: &stagepb.Stage{Id: req.StageId},
	}
	readResp, err := r.ReadStage(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get stage item page data: %w", err)
	}

	if len(readResp.Data) == 0 {
		return nil, fmt.Errorf("stage not found")
	}

	return &stagepb.GetStageItemPageDataResponse{
		Stage:   readResp.Data[0],
		Success: true,
	}, nil
}

// NewStageRepository creates a new MySQL stage repository (old-style constructor)
func NewStageRepository(db *sql.DB, tableName string) stagepb.StageDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLStageRepository(dbOps, tableName)
}

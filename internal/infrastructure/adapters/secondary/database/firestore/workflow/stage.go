//go:build firestore

package workflow

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	interfaces "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/interface"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/operations"
	firestoreCore "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/firestore/core"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	stagepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/stage"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "stage", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore stage repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreStageRepository(dbOps, collectionName), nil
	})
}

// FirestoreStageRepository implements stage CRUD operations using Firestore
type FirestoreStageRepository struct {
	stagepb.UnimplementedStageDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreStageRepository creates a new Firestore stage repository
func NewFirestoreStageRepository(dbOps interfaces.DatabaseOperation, collectionName string) stagepb.StageDomainServiceServer {
	if collectionName == "" {
		collectionName = "stage" // default fallback (singular to match database.go)
	}
	return &FirestoreStageRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateStage creates a new stage using common Firestore operations
func (r *FirestoreStageRepository) CreateStage(ctx context.Context, req *stagepb.CreateStageRequest) (*stagepb.CreateStageResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("stage data is required")
	}

	// Convert protobuf to map using ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create stage: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	stage := &stagepb.Stage{}
	convertedStage, err := operations.ConvertMapToProtobuf(result, stage)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &stagepb.CreateStageResponse{
		Data:    []*stagepb.Stage{convertedStage},
		Success: true,
	}, nil
}

// ReadStage retrieves a stage using common Firestore operations
func (r *FirestoreStageRepository) ReadStage(ctx context.Context, req *stagepb.ReadStageRequest) (*stagepb.ReadStageResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("stage ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read stage: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	stage := &stagepb.Stage{}
	convertedStage, err := operations.ConvertMapToProtobuf(result, stage)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &stagepb.ReadStageResponse{
		Data:    []*stagepb.Stage{convertedStage},
		Success: true,
	}, nil
}

// UpdateStage updates a stage using common Firestore operations
func (r *FirestoreStageRepository) UpdateStage(ctx context.Context, req *stagepb.UpdateStageRequest) (*stagepb.UpdateStageResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("stage ID is required")
	}

	// Convert protobuf to map using ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update stage: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	stage := &stagepb.Stage{}
	convertedStage, err := operations.ConvertMapToProtobuf(result, stage)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &stagepb.UpdateStageResponse{
		Data:    []*stagepb.Stage{convertedStage},
		Success: true,
	}, nil
}

// DeleteStage deletes a stage using common Firestore operations
func (r *FirestoreStageRepository) DeleteStage(ctx context.Context, req *stagepb.DeleteStageRequest) (*stagepb.DeleteStageResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("stage ID is required")
	}

	// Delete document using common operations
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete stage: %w", err)
	}

	return &stagepb.DeleteStageResponse{
		Success: true,
	}, nil
}

// ListStages retrieves stages using common Firestore operations
func (r *FirestoreStageRepository) ListStages(ctx context.Context, req *stagepb.ListStagesRequest) (*stagepb.ListStagesResponse, error) {
	// Build ListParams from request - pass filters directly to dbOps.List
	listParams := &interfaces.ListParams{
		Search:     req.Search,
		Filters:    req.Filters,
		Sort:       req.Sort,
		Pagination: req.Pagination,
	}

	// List documents using common operations with proper filter support
	listResult, err := r.dbOps.List(ctx, r.collectionName, listParams)
	if err != nil {
		return nil, fmt.Errorf("failed to list stages: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	stages, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *stagepb.Stage {
		return &stagepb.Stage{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if stages == nil {
		stages = make([]*stagepb.Stage, 0)
	}

	return &stagepb.ListStagesResponse{
		Data:    stages,
		Success: true,
	}, nil
}

// GetStageListPageData retrieves stages with pagination using common Firestore operations
func (r *FirestoreStageRepository) GetStageListPageData(ctx context.Context, req *stagepb.GetStageListPageDataRequest) (*stagepb.GetStageListPageDataResponse, error) {
	// For now, implement basic list functionality
	// TODO: Implement full pagination, filtering, sorting, and search using common operations
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

// GetStageItemPageData retrieves a single stage with enhanced data
func (r *FirestoreStageRepository) GetStageItemPageData(ctx context.Context, req *stagepb.GetStageItemPageDataRequest) (*stagepb.GetStageItemPageDataResponse, error) {
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

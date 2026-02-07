//go:build firestore

package product

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	interfaces "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/interface"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/operations"
	firestoreCore "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/firestore/core"
	"leapfor.xyz/espyna/internal/infrastructure/registry"

	collectionplanpb "leapfor.xyz/esqyma/golang/v1/domain/product/collection_plan"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "collection_plan", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore collection_plan repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreCollectionPlanRepository(dbOps, collectionName), nil
	})
}

// FirestoreCollectionPlanRepository implements collection plan CRUD operations using Firestore
type FirestoreCollectionPlanRepository struct {
	collectionplanpb.UnimplementedCollectionPlanDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreCollectionPlanRepository creates a new Firestore collection plan repository
func NewFirestoreCollectionPlanRepository(dbOps interfaces.DatabaseOperation, collectionName string) collectionplanpb.CollectionPlanDomainServiceServer {
	if collectionName == "" {
		collectionName = "collection_plan" // default fallback
	}
	return &FirestoreCollectionPlanRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateCollectionPlan creates a new collection plan using common Firestore operations
func (r *FirestoreCollectionPlanRepository) CreateCollectionPlan(ctx context.Context, req *collectionplanpb.CreateCollectionPlanRequest) (*collectionplanpb.CreateCollectionPlanResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("collection plan data is required")
	}

	// Convert protobuf to map using ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create collection plan: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	collectionPlan := &collectionplanpb.CollectionPlan{}
	convertedCollectionPlan, err := operations.ConvertMapToProtobuf(result, collectionPlan)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &collectionplanpb.CreateCollectionPlanResponse{
		Data: []*collectionplanpb.CollectionPlan{convertedCollectionPlan},
	}, nil
}

// ReadCollectionPlan retrieves a collection plan using common Firestore operations
func (r *FirestoreCollectionPlanRepository) ReadCollectionPlan(ctx context.Context, req *collectionplanpb.ReadCollectionPlanRequest) (*collectionplanpb.ReadCollectionPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection plan ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read collection plan: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	collectionPlan := &collectionplanpb.CollectionPlan{}
	convertedCollectionPlan, err := operations.ConvertMapToProtobuf(result, collectionPlan)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &collectionplanpb.ReadCollectionPlanResponse{
		Data: []*collectionplanpb.CollectionPlan{convertedCollectionPlan},
	}, nil
}

// UpdateCollectionPlan updates a collection plan using common Firestore operations
func (r *FirestoreCollectionPlanRepository) UpdateCollectionPlan(ctx context.Context, req *collectionplanpb.UpdateCollectionPlanRequest) (*collectionplanpb.UpdateCollectionPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection plan ID is required")
	}

	// Convert protobuf to map using ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update collection plan: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	collectionPlan := &collectionplanpb.CollectionPlan{}
	convertedCollectionPlan, err := operations.ConvertMapToProtobuf(result, collectionPlan)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &collectionplanpb.UpdateCollectionPlanResponse{
		Data: []*collectionplanpb.CollectionPlan{convertedCollectionPlan},
	}, nil
}

// DeleteCollectionPlan deletes a collection plan using common Firestore operations
func (r *FirestoreCollectionPlanRepository) DeleteCollectionPlan(ctx context.Context, req *collectionplanpb.DeleteCollectionPlanRequest) (*collectionplanpb.DeleteCollectionPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection plan ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete collection plan: %w", err)
	}

	return &collectionplanpb.DeleteCollectionPlanResponse{
		Success: true,
	}, nil
}

// ListCollectionPlans lists collection plans using common Firestore operations
func (r *FirestoreCollectionPlanRepository) ListCollectionPlans(ctx context.Context, req *collectionplanpb.ListCollectionPlansRequest) (*collectionplanpb.ListCollectionPlansResponse, error) {
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
		return nil, fmt.Errorf("failed to list collection plans: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	collectionPlans, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *collectionplanpb.CollectionPlan {
		return &collectionplanpb.CollectionPlan{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if collectionPlans == nil {
		collectionPlans = make([]*collectionplanpb.CollectionPlan, 0)
	}

	return &collectionplanpb.ListCollectionPlansResponse{
		Data: collectionPlans,
	}, nil
}

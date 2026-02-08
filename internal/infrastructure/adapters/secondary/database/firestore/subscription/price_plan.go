//go:build firestore

package subscription

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/operations"
	firestoreCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/firestore/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "price_plan", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore price_plan repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestorePricePlanRepository(dbOps, collectionName), nil
	})
}

// FirestorePricePlanRepository implements price plan CRUD operations using Firestore
type FirestorePricePlanRepository struct {
	priceplanpb.UnimplementedPricePlanDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestorePricePlanRepository creates a new Firestore price plan repository
func NewFirestorePricePlanRepository(dbOps interfaces.DatabaseOperation, collectionName string) priceplanpb.PricePlanDomainServiceServer {
	if collectionName == "" {
		collectionName = "price_plan" // default fallback
	}
	return &FirestorePricePlanRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreatePricePlan creates a new price plan using common Firestore operations
func (r *FirestorePricePlanRepository) CreatePricePlan(ctx context.Context, req *priceplanpb.CreatePricePlanRequest) (*priceplanpb.CreatePricePlanResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("price plan data is required")
	}

	// Convert protobuf to map using ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create price plan: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	convertedPricePlan, err := operations.ConvertMapToProtobuf(result, &priceplanpb.PricePlan{})
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &priceplanpb.CreatePricePlanResponse{
		Data: []*priceplanpb.PricePlan{convertedPricePlan},
	}, nil
}

// ReadPricePlan retrieves a price plan using common Firestore operations
func (r *FirestorePricePlanRepository) ReadPricePlan(ctx context.Context, req *priceplanpb.ReadPricePlanRequest) (*priceplanpb.ReadPricePlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price plan ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read price plan: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	convertedPricePlan, err := operations.ConvertMapToProtobuf(result, &priceplanpb.PricePlan{})
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &priceplanpb.ReadPricePlanResponse{
		Data: []*priceplanpb.PricePlan{convertedPricePlan},
	}, nil
}

// UpdatePricePlan updates a price plan using common Firestore operations
func (r *FirestorePricePlanRepository) UpdatePricePlan(ctx context.Context, req *priceplanpb.UpdatePricePlanRequest) (*priceplanpb.UpdatePricePlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price plan ID is required")
	}

	// Convert protobuf to map using ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update price plan: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	convertedPricePlan, err := operations.ConvertMapToProtobuf(result, &priceplanpb.PricePlan{})
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &priceplanpb.UpdatePricePlanResponse{
		Data: []*priceplanpb.PricePlan{convertedPricePlan},
	}, nil
}

// DeletePricePlan deletes a price plan using common Firestore operations
func (r *FirestorePricePlanRepository) DeletePricePlan(ctx context.Context, req *priceplanpb.DeletePricePlanRequest) (*priceplanpb.DeletePricePlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price plan ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete price plan: %w", err)
	}

	return &priceplanpb.DeletePricePlanResponse{
		Success: true,
	}, nil
}

// ListPricePlans lists price plans using common Firestore operations
func (r *FirestorePricePlanRepository) ListPricePlans(ctx context.Context, req *priceplanpb.ListPricePlansRequest) (*priceplanpb.ListPricePlansResponse, error) {
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
		return nil, fmt.Errorf("failed to list price plans: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	pricePlans, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *priceplanpb.PricePlan {
		return &priceplanpb.PricePlan{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if pricePlans == nil {
		pricePlans = make([]*priceplanpb.PricePlan, 0)
	}

	return &priceplanpb.ListPricePlansResponse{
		Data: pricePlans,
	}, nil
}

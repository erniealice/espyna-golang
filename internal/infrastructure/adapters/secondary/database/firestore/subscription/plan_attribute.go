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
	planattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan_attribute"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "plan_attribute", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore plan_attribute repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestorePlanAttributeRepository(dbOps, collectionName), nil
	})
}

// FirestorePlanAttributeRepository implements plan attribute CRUD operations using Firestore
type FirestorePlanAttributeRepository struct {
	planattributepb.UnimplementedPlanAttributeDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestorePlanAttributeRepository creates a new Firestore plan attribute repository
func NewFirestorePlanAttributeRepository(dbOps interfaces.DatabaseOperation, collectionName string) planattributepb.PlanAttributeDomainServiceServer {
	if collectionName == "" {
		collectionName = "plan_attribute" // default fallback
	}
	return &FirestorePlanAttributeRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreatePlanAttribute creates a new plan attribute using common Firestore operations
func (r *FirestorePlanAttributeRepository) CreatePlanAttribute(ctx context.Context, req *planattributepb.CreatePlanAttributeRequest) (*planattributepb.CreatePlanAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("plan attribute data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create plan attribute: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	planAttribute := &planattributepb.PlanAttribute{}
	convertedPlanAttribute, err := operations.ConvertMapToProtobuf(result, planAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &planattributepb.CreatePlanAttributeResponse{
		Data: []*planattributepb.PlanAttribute{convertedPlanAttribute},
	}, nil
}

// ReadPlanAttribute retrieves a plan attribute using common Firestore operations
func (r *FirestorePlanAttributeRepository) ReadPlanAttribute(ctx context.Context, req *planattributepb.ReadPlanAttributeRequest) (*planattributepb.ReadPlanAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("plan attribute ID is required")
	}

	// Read document using common operations with the ID
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan attribute: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	planAttribute := &planattributepb.PlanAttribute{}
	convertedPlanAttribute, err := operations.ConvertMapToProtobuf(result, planAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &planattributepb.ReadPlanAttributeResponse{
		Data: []*planattributepb.PlanAttribute{convertedPlanAttribute},
	}, nil
}

// UpdatePlanAttribute updates a plan attribute using common Firestore operations
func (r *FirestorePlanAttributeRepository) UpdatePlanAttribute(ctx context.Context, req *planattributepb.UpdatePlanAttributeRequest) (*planattributepb.UpdatePlanAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("plan attribute ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update plan attribute: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	planAttribute := &planattributepb.PlanAttribute{}
	convertedPlanAttribute, err := operations.ConvertMapToProtobuf(result, planAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &planattributepb.UpdatePlanAttributeResponse{
		Data: []*planattributepb.PlanAttribute{convertedPlanAttribute},
	}, nil
}

// DeletePlanAttribute deletes a plan attribute using common Firestore operations
func (r *FirestorePlanAttributeRepository) DeletePlanAttribute(ctx context.Context, req *planattributepb.DeletePlanAttributeRequest) (*planattributepb.DeletePlanAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("plan attribute ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete plan attribute: %w", err)
	}

	return &planattributepb.DeletePlanAttributeResponse{
		Success: true,
	}, nil
}

// ListPlanAttributes lists plan attributes using common Firestore operations
func (r *FirestorePlanAttributeRepository) ListPlanAttributes(ctx context.Context, req *planattributepb.ListPlanAttributesRequest) (*planattributepb.ListPlanAttributesResponse, error) {
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
		return nil, fmt.Errorf("failed to list plan attributes: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	planAttributes, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *planattributepb.PlanAttribute {
		return &planattributepb.PlanAttribute{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if planAttributes == nil {
		planAttributes = make([]*planattributepb.PlanAttribute, 0)
	}

	return &planattributepb.ListPlanAttributesResponse{
		Data: planAttributes,
	}, nil
}
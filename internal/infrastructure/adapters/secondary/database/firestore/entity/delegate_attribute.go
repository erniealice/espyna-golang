//go:build firestore

package entity

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	interfaces "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/interface"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/operations"
	firestoreCore "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/firestore/core"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	delegateattributepb "leapfor.xyz/esqyma/golang/v1/domain/entity/delegate_attribute"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "delegate_attribute", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore delegate_attribute repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreDelegateAttributeRepository(dbOps, collectionName), nil
	})
}

// FirestoreDelegateAttributeRepository implements delegate attribute CRUD operations using Firestore
type FirestoreDelegateAttributeRepository struct {
	delegateattributepb.UnimplementedDelegateAttributeDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreDelegateAttributeRepository creates a new Firestore delegate attribute repository
func NewFirestoreDelegateAttributeRepository(dbOps interfaces.DatabaseOperation, collectionName string) delegateattributepb.DelegateAttributeDomainServiceServer {
	if collectionName == "" {
		collectionName = "delegate_attribute" // default fallback
	}
	return &FirestoreDelegateAttributeRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateDelegateAttribute creates a new delegate attribute using common Firestore operations
func (r *FirestoreDelegateAttributeRepository) CreateDelegateAttribute(ctx context.Context, req *delegateattributepb.CreateDelegateAttributeRequest) (*delegateattributepb.CreateDelegateAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("delegate attribute data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create delegate attribute: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	delegateAttribute := &delegateattributepb.DelegateAttribute{}
	convertedDelegateAttribute, err := operations.ConvertMapToProtobuf(result, delegateAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &delegateattributepb.CreateDelegateAttributeResponse{
		Data: []*delegateattributepb.DelegateAttribute{convertedDelegateAttribute},
	}, nil
}

// ReadDelegateAttribute retrieves a delegate attribute using common Firestore operations
func (r *FirestoreDelegateAttributeRepository) ReadDelegateAttribute(ctx context.Context, req *delegateattributepb.ReadDelegateAttributeRequest) (*delegateattributepb.ReadDelegateAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("delegate attribute ID is required")
	}

	// Read document using common operations with the ID
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read delegate attribute: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	delegateAttribute := &delegateattributepb.DelegateAttribute{}
	convertedDelegateAttribute, err := operations.ConvertMapToProtobuf(result, delegateAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &delegateattributepb.ReadDelegateAttributeResponse{
		Data: []*delegateattributepb.DelegateAttribute{convertedDelegateAttribute},
	}, nil
}

// UpdateDelegateAttribute updates a delegate attribute using common Firestore operations
func (r *FirestoreDelegateAttributeRepository) UpdateDelegateAttribute(ctx context.Context, req *delegateattributepb.UpdateDelegateAttributeRequest) (*delegateattributepb.UpdateDelegateAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("delegate attribute ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update delegate attribute: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	delegateAttribute := &delegateattributepb.DelegateAttribute{}
	convertedDelegateAttribute, err := operations.ConvertMapToProtobuf(result, delegateAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &delegateattributepb.UpdateDelegateAttributeResponse{
		Data: []*delegateattributepb.DelegateAttribute{convertedDelegateAttribute},
	}, nil
}

// DeleteDelegateAttribute deletes a delegate attribute using common Firestore operations
func (r *FirestoreDelegateAttributeRepository) DeleteDelegateAttribute(ctx context.Context, req *delegateattributepb.DeleteDelegateAttributeRequest) (*delegateattributepb.DeleteDelegateAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("delegate attribute ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete delegate attribute: %w", err)
	}

	return &delegateattributepb.DeleteDelegateAttributeResponse{
		Success: true,
	}, nil
}

// ListDelegateAttributes lists delegate attributes using common Firestore operations
func (r *FirestoreDelegateAttributeRepository) ListDelegateAttributes(ctx context.Context, req *delegateattributepb.ListDelegateAttributesRequest) (*delegateattributepb.ListDelegateAttributesResponse, error) {
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
		return nil, fmt.Errorf("failed to list delegate attributes: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	delegateAttributes, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *delegateattributepb.DelegateAttribute {
		return &delegateattributepb.DelegateAttribute{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if delegateAttributes == nil {
		delegateAttributes = make([]*delegateattributepb.DelegateAttribute, 0)
	}

	return &delegateattributepb.ListDelegateAttributesResponse{
		Data: delegateAttributes,
	}, nil
}
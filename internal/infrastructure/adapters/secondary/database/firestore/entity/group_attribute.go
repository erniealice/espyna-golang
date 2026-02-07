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
	groupattributepb "leapfor.xyz/esqyma/golang/v1/domain/entity/group_attribute"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "group_attribute", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore group_attribute repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreGroupAttributeRepository(dbOps, collectionName), nil
	})
}

// FirestoreGroupAttributeRepository implements group attribute CRUD operations using Firestore
type FirestoreGroupAttributeRepository struct {
	groupattributepb.UnimplementedGroupAttributeDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreGroupAttributeRepository creates a new Firestore group attribute repository
func NewFirestoreGroupAttributeRepository(dbOps interfaces.DatabaseOperation, collectionName string) groupattributepb.GroupAttributeDomainServiceServer {
	if collectionName == "" {
		collectionName = "group_attribute" // default fallback
	}
	return &FirestoreGroupAttributeRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateGroupAttribute creates a new group attribute using common Firestore operations
func (r *FirestoreGroupAttributeRepository) CreateGroupAttribute(ctx context.Context, req *groupattributepb.CreateGroupAttributeRequest) (*groupattributepb.CreateGroupAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("group attribute data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create group attribute: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	groupAttribute := &groupattributepb.GroupAttribute{}
	convertedGroupAttribute, err := operations.ConvertMapToProtobuf(result, groupAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &groupattributepb.CreateGroupAttributeResponse{
		Data: []*groupattributepb.GroupAttribute{convertedGroupAttribute},
	}, nil
}

// ReadGroupAttribute retrieves a group attribute using common Firestore operations
func (r *FirestoreGroupAttributeRepository) ReadGroupAttribute(ctx context.Context, req *groupattributepb.ReadGroupAttributeRequest) (*groupattributepb.ReadGroupAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("group attribute ID is required")
	}

	// Read document using common operations with the ID
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read group attribute: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	groupAttribute := &groupattributepb.GroupAttribute{}
	convertedGroupAttribute, err := operations.ConvertMapToProtobuf(result, groupAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &groupattributepb.ReadGroupAttributeResponse{
		Data: []*groupattributepb.GroupAttribute{convertedGroupAttribute},
	}, nil
}

// UpdateGroupAttribute updates a group attribute using common Firestore operations
func (r *FirestoreGroupAttributeRepository) UpdateGroupAttribute(ctx context.Context, req *groupattributepb.UpdateGroupAttributeRequest) (*groupattributepb.UpdateGroupAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("group attribute ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update group attribute: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	groupAttribute := &groupattributepb.GroupAttribute{}
	convertedGroupAttribute, err := operations.ConvertMapToProtobuf(result, groupAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &groupattributepb.UpdateGroupAttributeResponse{
		Data: []*groupattributepb.GroupAttribute{convertedGroupAttribute},
	}, nil
}

// DeleteGroupAttribute deletes a group attribute using common Firestore operations
func (r *FirestoreGroupAttributeRepository) DeleteGroupAttribute(ctx context.Context, req *groupattributepb.DeleteGroupAttributeRequest) (*groupattributepb.DeleteGroupAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("group attribute ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete group attribute: %w", err)
	}

	return &groupattributepb.DeleteGroupAttributeResponse{
		Success: true,
	}, nil
}

// ListGroupAttributes lists group attributes using common Firestore operations
func (r *FirestoreGroupAttributeRepository) ListGroupAttributes(ctx context.Context, req *groupattributepb.ListGroupAttributesRequest) (*groupattributepb.ListGroupAttributesResponse, error) {
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
		return nil, fmt.Errorf("failed to list group attributes: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	groupAttributes, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *groupattributepb.GroupAttribute {
		return &groupattributepb.GroupAttribute{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if groupAttributes == nil {
		groupAttributes = make([]*groupattributepb.GroupAttribute, 0)
	}

	return &groupattributepb.ListGroupAttributesResponse{
		Data: groupAttributes,
	}, nil
}
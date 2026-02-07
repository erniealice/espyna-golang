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
	grouppb "leapfor.xyz/esqyma/golang/v1/domain/entity/group"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "group", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore group repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreGroupRepository(dbOps, collectionName), nil
	})
}

// FirestoreGroupRepository implements group CRUD operations using Firestore
type FirestoreGroupRepository struct {
	grouppb.UnimplementedGroupDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreGroupRepository creates a new Firestore group repository
func NewFirestoreGroupRepository(dbOps interfaces.DatabaseOperation, collectionName string) grouppb.GroupDomainServiceServer {
	if collectionName == "" {
		collectionName = "group" // default fallback
	}
	return &FirestoreGroupRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateGroup creates a new group using common Firestore operations
func (r *FirestoreGroupRepository) CreateGroup(ctx context.Context, req *grouppb.CreateGroupRequest) (*grouppb.CreateGroupResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("group data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	group := &grouppb.Group{}
	convertedGroup, err := operations.ConvertMapToProtobuf(result, group)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &grouppb.CreateGroupResponse{
		Data: []*grouppb.Group{convertedGroup},
	}, nil
}

// ReadGroup retrieves a group using common Firestore operations
func (r *FirestoreGroupRepository) ReadGroup(ctx context.Context, req *grouppb.ReadGroupRequest) (*grouppb.ReadGroupResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("group ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read group: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	group := &grouppb.Group{}
	convertedGroup, err := operations.ConvertMapToProtobuf(result, group)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &grouppb.ReadGroupResponse{
		Data: []*grouppb.Group{convertedGroup},
	}, nil
}

// UpdateGroup updates a group using common Firestore operations
func (r *FirestoreGroupRepository) UpdateGroup(ctx context.Context, req *grouppb.UpdateGroupRequest) (*grouppb.UpdateGroupResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("group ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update group: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	group := &grouppb.Group{}
	convertedGroup, err := operations.ConvertMapToProtobuf(result, group)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &grouppb.UpdateGroupResponse{
		Data: []*grouppb.Group{convertedGroup},
	}, nil
}

// DeleteGroup deletes a group using common Firestore operations
func (r *FirestoreGroupRepository) DeleteGroup(ctx context.Context, req *grouppb.DeleteGroupRequest) (*grouppb.DeleteGroupResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("group ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete group: %w", err)
	}

	return &grouppb.DeleteGroupResponse{
		Success: true,
	}, nil
}

// ListGroups lists groups using common Firestore operations
func (r *FirestoreGroupRepository) ListGroups(ctx context.Context, req *grouppb.ListGroupsRequest) (*grouppb.ListGroupsResponse, error) {
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
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	groups, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *grouppb.Group {
		return &grouppb.Group{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if groups == nil {
		groups = make([]*grouppb.Group, 0)
	}

	return &grouppb.ListGroupsResponse{
		Data: groups,
	}, nil
}

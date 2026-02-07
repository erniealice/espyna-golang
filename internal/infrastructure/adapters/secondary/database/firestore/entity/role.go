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
	rolepb "leapfor.xyz/esqyma/golang/v1/domain/entity/role"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "role", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore role repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreRoleRepository(dbOps, collectionName), nil
	})
}

// FirestoreRoleRepository implements role CRUD operations using Firestore
type FirestoreRoleRepository struct {
	rolepb.UnimplementedRoleDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreRoleRepository creates a new Firestore role repository
func NewFirestoreRoleRepository(dbOps interfaces.DatabaseOperation, collectionName string) rolepb.RoleDomainServiceServer {
	if collectionName == "" {
		collectionName = "role" // default fallback
	}
	return &FirestoreRoleRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateRole creates a new role using common Firestore operations
func (r *FirestoreRoleRepository) CreateRole(ctx context.Context, req *rolepb.CreateRoleRequest) (*rolepb.CreateRoleResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("role data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create role: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	role := &rolepb.Role{}
	convertedRole, err := operations.ConvertMapToProtobuf(result, role)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &rolepb.CreateRoleResponse{
		Data: []*rolepb.Role{convertedRole},
	}, nil
}

// ReadRole retrieves a role using common Firestore operations
func (r *FirestoreRoleRepository) ReadRole(ctx context.Context, req *rolepb.ReadRoleRequest) (*rolepb.ReadRoleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("role ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read role: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	role := &rolepb.Role{}
	convertedRole, err := operations.ConvertMapToProtobuf(result, role)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &rolepb.ReadRoleResponse{
		Data: []*rolepb.Role{convertedRole},
	}, nil
}

// UpdateRole updates a role using common Firestore operations
func (r *FirestoreRoleRepository) UpdateRole(ctx context.Context, req *rolepb.UpdateRoleRequest) (*rolepb.UpdateRoleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("role ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update role: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	role := &rolepb.Role{}
	convertedRole, err := operations.ConvertMapToProtobuf(result, role)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &rolepb.UpdateRoleResponse{
		Data: []*rolepb.Role{convertedRole},
	}, nil
}

// DeleteRole deletes a role using common Firestore operations
func (r *FirestoreRoleRepository) DeleteRole(ctx context.Context, req *rolepb.DeleteRoleRequest) (*rolepb.DeleteRoleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("role ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete role: %w", err)
	}

	return &rolepb.DeleteRoleResponse{
		Success: true,
	}, nil
}

// ListRoles lists roles using common Firestore operations
func (r *FirestoreRoleRepository) ListRoles(ctx context.Context, req *rolepb.ListRolesRequest) (*rolepb.ListRolesResponse, error) {
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
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	roles, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *rolepb.Role {
		return &rolepb.Role{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if roles == nil {
		roles = make([]*rolepb.Role, 0)
	}

	return &rolepb.ListRolesResponse{
		Data: roles,
	}, nil
}

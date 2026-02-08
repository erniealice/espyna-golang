//go:build firestore

package entity

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/operations"
	firestoreCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/firestore/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	rolepermissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role_permission"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "role_permission", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore role_permission repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreRolePermissionRepository(dbOps, collectionName), nil
	})
}

// FirestoreRolePermissionRepository implements role permission CRUD operations using Firestore
type FirestoreRolePermissionRepository struct {
	rolepermissionpb.UnimplementedRolePermissionDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreRolePermissionRepository creates a new Firestore role permission repository
func NewFirestoreRolePermissionRepository(dbOps interfaces.DatabaseOperation, collectionName string) rolepermissionpb.RolePermissionDomainServiceServer {
	if collectionName == "" {
		collectionName = "role_permissions" // default fallback
	}
	return &FirestoreRolePermissionRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateRolePermission creates a new role permission using common Firestore operations
func (r *FirestoreRolePermissionRepository) CreateRolePermission(ctx context.Context, req *rolepermissionpb.CreateRolePermissionRequest) (*rolepermissionpb.CreateRolePermissionResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("role permission data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create role permission: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	convertedRolePermission, err := operations.ConvertMapToProtobuf(result, &rolepermissionpb.RolePermission{})
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &rolepermissionpb.CreateRolePermissionResponse{
		Data: []*rolepermissionpb.RolePermission{convertedRolePermission},
	}, nil
}

// ReadRolePermission retrieves a role permission using common Firestore operations
func (r *FirestoreRolePermissionRepository) ReadRolePermission(ctx context.Context, req *rolepermissionpb.ReadRolePermissionRequest) (*rolepermissionpb.ReadRolePermissionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("role permission ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read role permission: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	convertedRolePermission, err := operations.ConvertMapToProtobuf(result, &rolepermissionpb.RolePermission{})
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &rolepermissionpb.ReadRolePermissionResponse{
		Data: []*rolepermissionpb.RolePermission{convertedRolePermission},
	}, nil
}

// UpdateRolePermission updates a role permission using common Firestore operations
func (r *FirestoreRolePermissionRepository) UpdateRolePermission(ctx context.Context, req *rolepermissionpb.UpdateRolePermissionRequest) (*rolepermissionpb.UpdateRolePermissionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("role permission ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update role permission: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	convertedRolePermission, err := operations.ConvertMapToProtobuf(result, &rolepermissionpb.RolePermission{})
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &rolepermissionpb.UpdateRolePermissionResponse{
		Data: []*rolepermissionpb.RolePermission{convertedRolePermission},
	}, nil
}

// DeleteRolePermission deletes a role permission using common Firestore operations
func (r *FirestoreRolePermissionRepository) DeleteRolePermission(ctx context.Context, req *rolepermissionpb.DeleteRolePermissionRequest) (*rolepermissionpb.DeleteRolePermissionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("role permission ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete role permission: %w", err)
	}

	return &rolepermissionpb.DeleteRolePermissionResponse{
		Success: true,
	}, nil
}

// ListRolePermissions lists role permissions using common Firestore operations
func (r *FirestoreRolePermissionRepository) ListRolePermissions(ctx context.Context, req *rolepermissionpb.ListRolePermissionsRequest) (*rolepermissionpb.ListRolePermissionsResponse, error) {
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
		return nil, fmt.Errorf("failed to list role permissions: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	rolePermissions, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *rolepermissionpb.RolePermission {
		return &rolepermissionpb.RolePermission{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if rolePermissions == nil {
		rolePermissions = make([]*rolepermissionpb.RolePermission, 0)
	}

	return &rolepermissionpb.ListRolePermissionsResponse{
		Data: rolePermissions,
	}, nil
}

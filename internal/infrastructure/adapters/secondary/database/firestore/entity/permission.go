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
	permissionpb "leapfor.xyz/esqyma/golang/v1/domain/entity/permission"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "permission", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore permission repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestorePermissionRepository(dbOps, collectionName), nil
	})
}

// FirestorePermissionRepository implements permission CRUD operations using Firestore
type FirestorePermissionRepository struct {
	permissionpb.UnimplementedPermissionDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestorePermissionRepository creates a new Firestore permission repository
func NewFirestorePermissionRepository(dbOps interfaces.DatabaseOperation, collectionName string) permissionpb.PermissionDomainServiceServer {
	if collectionName == "" {
		collectionName = "permission" // default fallback
	}
	return &FirestorePermissionRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreatePermission creates a new permission using common Firestore operations
func (r *FirestorePermissionRepository) CreatePermission(ctx context.Context, req *permissionpb.CreatePermissionRequest) (*permissionpb.CreatePermissionResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("permission data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create permission: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	convertedPermission, err := operations.ConvertMapToProtobuf(result, &permissionpb.Permission{})
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &permissionpb.CreatePermissionResponse{
		Data: []*permissionpb.Permission{convertedPermission},
	}, nil
}

// ReadPermission retrieves a permission using common Firestore operations
func (r *FirestorePermissionRepository) ReadPermission(ctx context.Context, req *permissionpb.ReadPermissionRequest) (*permissionpb.ReadPermissionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("permission ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read permission: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	convertedPermission, err := operations.ConvertMapToProtobuf(result, &permissionpb.Permission{})
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &permissionpb.ReadPermissionResponse{
		Data: []*permissionpb.Permission{convertedPermission},
	}, nil
}

// UpdatePermission updates a permission using common Firestore operations
func (r *FirestorePermissionRepository) UpdatePermission(ctx context.Context, req *permissionpb.UpdatePermissionRequest) (*permissionpb.UpdatePermissionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("permission ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update permission: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	convertedPermission, err := operations.ConvertMapToProtobuf(result, &permissionpb.Permission{})
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &permissionpb.UpdatePermissionResponse{
		Data: []*permissionpb.Permission{convertedPermission},
	}, nil
}

// DeletePermission deletes a permission using common Firestore operations
func (r *FirestorePermissionRepository) DeletePermission(ctx context.Context, req *permissionpb.DeletePermissionRequest) (*permissionpb.DeletePermissionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("permission ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete permission: %w", err)
	}

	return &permissionpb.DeletePermissionResponse{
		Success: true,
	}, nil
}

// ListPermissions lists permissions using common Firestore operations
func (r *FirestorePermissionRepository) ListPermissions(ctx context.Context, req *permissionpb.ListPermissionsRequest) (*permissionpb.ListPermissionsResponse, error) {
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
		return nil, fmt.Errorf("failed to list permissions: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	permissions, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *permissionpb.Permission {
		return &permissionpb.Permission{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if permissions == nil {
		permissions = make([]*permissionpb.Permission, 0)
	}

	return &permissionpb.ListPermissionsResponse{
		Data: permissions,
	}, nil
}

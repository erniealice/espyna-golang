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
	workspaceuserrolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user_role"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "workspace_user_role", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore workspace_user_role repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreWorkspaceUserRoleRepository(dbOps, collectionName), nil
	})
}

// FirestoreWorkspaceUserRoleRepository implements workspace user role CRUD operations using Firestore
type FirestoreWorkspaceUserRoleRepository struct {
	workspaceuserrolepb.UnimplementedWorkspaceUserRoleDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreWorkspaceUserRoleRepository creates a new Firestore workspace user role repository
func NewFirestoreWorkspaceUserRoleRepository(dbOps interfaces.DatabaseOperation, collectionName string) workspaceuserrolepb.WorkspaceUserRoleDomainServiceServer {
	if collectionName == "" {
		collectionName = "workspace_user_role" // default fallback
	}
	return &FirestoreWorkspaceUserRoleRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateWorkspaceUserRole creates a new workspace user role using common Firestore operations
func (r *FirestoreWorkspaceUserRoleRepository) CreateWorkspaceUserRole(ctx context.Context, req *workspaceuserrolepb.CreateWorkspaceUserRoleRequest) (*workspaceuserrolepb.CreateWorkspaceUserRoleResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("workspace user role data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace user role: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	convertedWorkspaceUserRole, err := operations.ConvertMapToProtobuf(result, &workspaceuserrolepb.WorkspaceUserRole{})
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &workspaceuserrolepb.CreateWorkspaceUserRoleResponse{
		Data: []*workspaceuserrolepb.WorkspaceUserRole{convertedWorkspaceUserRole},
	}, nil
}

// ReadWorkspaceUserRole retrieves a workspace user role using common Firestore operations
func (r *FirestoreWorkspaceUserRoleRepository) ReadWorkspaceUserRole(ctx context.Context, req *workspaceuserrolepb.ReadWorkspaceUserRoleRequest) (*workspaceuserrolepb.ReadWorkspaceUserRoleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace user role ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read workspace user role: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	convertedWorkspaceUserRole, err := operations.ConvertMapToProtobuf(result, &workspaceuserrolepb.WorkspaceUserRole{})
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &workspaceuserrolepb.ReadWorkspaceUserRoleResponse{
		Data: []*workspaceuserrolepb.WorkspaceUserRole{convertedWorkspaceUserRole},
	}, nil
}

// UpdateWorkspaceUserRole updates a workspace user role using common Firestore operations
func (r *FirestoreWorkspaceUserRoleRepository) UpdateWorkspaceUserRole(ctx context.Context, req *workspaceuserrolepb.UpdateWorkspaceUserRoleRequest) (*workspaceuserrolepb.UpdateWorkspaceUserRoleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace user role ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update workspace user role: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	convertedWorkspaceUserRole, err := operations.ConvertMapToProtobuf(result, &workspaceuserrolepb.WorkspaceUserRole{})
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &workspaceuserrolepb.UpdateWorkspaceUserRoleResponse{
		Data: []*workspaceuserrolepb.WorkspaceUserRole{convertedWorkspaceUserRole},
	}, nil
}

// DeleteWorkspaceUserRole deletes a workspace user role using common Firestore operations
func (r *FirestoreWorkspaceUserRoleRepository) DeleteWorkspaceUserRole(ctx context.Context, req *workspaceuserrolepb.DeleteWorkspaceUserRoleRequest) (*workspaceuserrolepb.DeleteWorkspaceUserRoleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace user role ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete workspace user role: %w", err)
	}

	return &workspaceuserrolepb.DeleteWorkspaceUserRoleResponse{
		Success: true,
	}, nil
}

// ListWorkspaceUserRoles lists workspace user roles using common Firestore operations
func (r *FirestoreWorkspaceUserRoleRepository) ListWorkspaceUserRoles(ctx context.Context, req *workspaceuserrolepb.ListWorkspaceUserRolesRequest) (*workspaceuserrolepb.ListWorkspaceUserRolesResponse, error) {
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
		return nil, fmt.Errorf("failed to list workspace user roles: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	workspaceUserRoles, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *workspaceuserrolepb.WorkspaceUserRole {
		return &workspaceuserrolepb.WorkspaceUserRole{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if workspaceUserRoles == nil {
		workspaceUserRoles = make([]*workspaceuserrolepb.WorkspaceUserRole, 0)
	}

	return &workspaceuserrolepb.ListWorkspaceUserRolesResponse{
		Data: workspaceUserRoles,
	}, nil
}

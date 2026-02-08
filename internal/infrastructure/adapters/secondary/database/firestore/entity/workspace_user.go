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
	workspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "workspace_user", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore workspace_user repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreWorkspaceUserRepository(dbOps, collectionName), nil
	})
}

// FirestoreWorkspaceUserRepository implements workspace user CRUD operations using Firestore
type FirestoreWorkspaceUserRepository struct {
	workspaceuserpb.UnimplementedWorkspaceUserDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreWorkspaceUserRepository creates a new Firestore workspace user repository
func NewFirestoreWorkspaceUserRepository(dbOps interfaces.DatabaseOperation, collectionName string) workspaceuserpb.WorkspaceUserDomainServiceServer {
	if collectionName == "" {
		collectionName = "workspace_user" // default fallback
	}
	return &FirestoreWorkspaceUserRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateWorkspaceUser creates a new workspace user using common Firestore operations
func (r *FirestoreWorkspaceUserRepository) CreateWorkspaceUser(ctx context.Context, req *workspaceuserpb.CreateWorkspaceUserRequest) (*workspaceuserpb.CreateWorkspaceUserResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("workspace user data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Use transaction-aware database operations
	txAwareDbOps := r.dbOps

	// Create document using common operations (automatically transaction-aware)
	result, err := txAwareDbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace user: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	convertedWorkspaceUser, err := operations.ConvertMapToProtobuf(result, &workspaceuserpb.WorkspaceUser{})
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &workspaceuserpb.CreateWorkspaceUserResponse{
		Data: []*workspaceuserpb.WorkspaceUser{convertedWorkspaceUser},
	}, nil
}

// ReadWorkspaceUser retrieves a workspace user using common Firestore operations
func (r *FirestoreWorkspaceUserRepository) ReadWorkspaceUser(ctx context.Context, req *workspaceuserpb.ReadWorkspaceUserRequest) (*workspaceuserpb.ReadWorkspaceUserResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace user ID is required")
	}

	// Use transaction-aware database operations
	txAwareDbOps := r.dbOps

	// Read document using common operations (automatically transaction-aware)
	result, err := txAwareDbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read workspace user: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	convertedWorkspaceUser, err := operations.ConvertMapToProtobuf(result, &workspaceuserpb.WorkspaceUser{})
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &workspaceuserpb.ReadWorkspaceUserResponse{
		Data: []*workspaceuserpb.WorkspaceUser{convertedWorkspaceUser},
	}, nil
}

// UpdateWorkspaceUser updates a workspace user using common Firestore operations
func (r *FirestoreWorkspaceUserRepository) UpdateWorkspaceUser(ctx context.Context, req *workspaceuserpb.UpdateWorkspaceUserRequest) (*workspaceuserpb.UpdateWorkspaceUserResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace user ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Use transaction-aware database operations
	txAwareDbOps := r.dbOps

	// Update document using common operations (automatically transaction-aware)
	result, err := txAwareDbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update workspace user: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	convertedWorkspaceUser, err := operations.ConvertMapToProtobuf(result, &workspaceuserpb.WorkspaceUser{})
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &workspaceuserpb.UpdateWorkspaceUserResponse{
		Data: []*workspaceuserpb.WorkspaceUser{convertedWorkspaceUser},
	}, nil
}

// DeleteWorkspaceUser deletes a workspace user using common Firestore operations
func (r *FirestoreWorkspaceUserRepository) DeleteWorkspaceUser(ctx context.Context, req *workspaceuserpb.DeleteWorkspaceUserRequest) (*workspaceuserpb.DeleteWorkspaceUserResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace user ID is required")
	}

	// Use transaction-aware database operations
	txAwareDbOps := r.dbOps

	// Delete document using common operations (soft delete, automatically transaction-aware)
	err := txAwareDbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete workspace user: %w", err)
	}

	return &workspaceuserpb.DeleteWorkspaceUserResponse{
		Success: true,
	}, nil
}

// ListWorkspaceUsers lists workspace users using common Firestore operations
func (r *FirestoreWorkspaceUserRepository) ListWorkspaceUsers(ctx context.Context, req *workspaceuserpb.ListWorkspaceUsersRequest) (*workspaceuserpb.ListWorkspaceUsersResponse, error) {
	// Use transaction-aware database operations
	txAwareDbOps := r.dbOps

	// Build ListParams from request - pass filters directly to dbOps.List
	listParams := &interfaces.ListParams{
		Search:     req.Search,
		Filters:    req.Filters,
		Sort:       req.Sort,
		Pagination: req.Pagination,
	}

	// List documents using common operations with proper filter support
	listResult, err := txAwareDbOps.List(ctx, r.collectionName, listParams)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspace users: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	workspaceUsers, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *workspaceuserpb.WorkspaceUser {
		return &workspaceuserpb.WorkspaceUser{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if workspaceUsers == nil {
		workspaceUsers = make([]*workspaceuserpb.WorkspaceUser, 0)
	}

	return &workspaceuserpb.ListWorkspaceUsersResponse{
		Data: workspaceUsers,
	}, nil
}

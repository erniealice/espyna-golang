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
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "workspace", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore workspace repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreWorkspaceRepository(dbOps, collectionName), nil
	})
}

// FirestoreWorkspaceRepository implements workspace CRUD operations using Firestore
type FirestoreWorkspaceRepository struct {
	workspacepb.UnimplementedWorkspaceDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreWorkspaceRepository creates a new Firestore workspace repository
func NewFirestoreWorkspaceRepository(dbOps interfaces.DatabaseOperation, collectionName string) workspacepb.WorkspaceDomainServiceServer {
	if collectionName == "" {
		collectionName = "workspace" // default fallback
	}
	return &FirestoreWorkspaceRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateWorkspace creates a new workspace using common Firestore operations
func (r *FirestoreWorkspaceRepository) CreateWorkspace(ctx context.Context, req *workspacepb.CreateWorkspaceRequest) (*workspacepb.CreateWorkspaceResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("workspace data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	convertedWorkspace, err := operations.ConvertMapToProtobuf(result, &workspacepb.Workspace{})
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &workspacepb.CreateWorkspaceResponse{
		Data: []*workspacepb.Workspace{convertedWorkspace},
	}, nil
}

// ReadWorkspace retrieves a workspace using common Firestore operations
func (r *FirestoreWorkspaceRepository) ReadWorkspace(ctx context.Context, req *workspacepb.ReadWorkspaceRequest) (*workspacepb.ReadWorkspaceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read workspace: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	convertedWorkspace, err := operations.ConvertMapToProtobuf(result, &workspacepb.Workspace{})
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &workspacepb.ReadWorkspaceResponse{
		Data: []*workspacepb.Workspace{convertedWorkspace},
	}, nil
}

// UpdateWorkspace updates a workspace using common Firestore operations
func (r *FirestoreWorkspaceRepository) UpdateWorkspace(ctx context.Context, req *workspacepb.UpdateWorkspaceRequest) (*workspacepb.UpdateWorkspaceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update workspace: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	convertedWorkspace, err := operations.ConvertMapToProtobuf(result, &workspacepb.Workspace{})
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &workspacepb.UpdateWorkspaceResponse{
		Data: []*workspacepb.Workspace{convertedWorkspace},
	}, nil
}

// DeleteWorkspace deletes a workspace using common Firestore operations
func (r *FirestoreWorkspaceRepository) DeleteWorkspace(ctx context.Context, req *workspacepb.DeleteWorkspaceRequest) (*workspacepb.DeleteWorkspaceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete workspace: %w", err)
	}

	return &workspacepb.DeleteWorkspaceResponse{
		Success: true,
	}, nil
}

// ListWorkspaces lists workspaces using common Firestore operations
func (r *FirestoreWorkspaceRepository) ListWorkspaces(ctx context.Context, req *workspacepb.ListWorkspacesRequest) (*workspacepb.ListWorkspacesResponse, error) {
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
		return nil, fmt.Errorf("failed to list workspaces: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	workspaces, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *workspacepb.Workspace {
		return &workspacepb.Workspace{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if workspaces == nil {
		workspaces = make([]*workspacepb.Workspace, 0)
	}

	return &workspacepb.ListWorkspacesResponse{
		Data: workspaces,
	}, nil
}

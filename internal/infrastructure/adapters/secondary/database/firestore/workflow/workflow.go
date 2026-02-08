//go:build firestore

package workflow

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/operations"
	firestoreCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/firestore/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	workflowpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/workflow"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "workflow", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore workflow repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreWorkflowRepository(dbOps, collectionName), nil
	})
}

// FirestoreWorkflowRepository implements workflow CRUD operations using Firestore
type FirestoreWorkflowRepository struct {
	workflowpb.UnimplementedWorkflowDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreWorkflowRepository creates a new Firestore workflow repository
func NewFirestoreWorkflowRepository(dbOps interfaces.DatabaseOperation, collectionName string) workflowpb.WorkflowDomainServiceServer {
	if collectionName == "" {
		collectionName = "workflow" // default fallback
	}
	return &FirestoreWorkflowRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateWorkflow creates a new workflow using common Firestore operations
func (r *FirestoreWorkflowRepository) CreateWorkflow(ctx context.Context, req *workflowpb.CreateWorkflowRequest) (*workflowpb.CreateWorkflowResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("workflow data is required")
	}

	// Convert protobuf to map using ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create workflow: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	workflow := &workflowpb.Workflow{}
	convertedWorkflow, err := operations.ConvertMapToProtobuf(result, workflow)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &workflowpb.CreateWorkflowResponse{
		Data:    []*workflowpb.Workflow{convertedWorkflow},
		Success: true,
	}, nil
}

// ReadWorkflow retrieves a workflow using common Firestore operations
func (r *FirestoreWorkflowRepository) ReadWorkflow(ctx context.Context, req *workflowpb.ReadWorkflowRequest) (*workflowpb.ReadWorkflowResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workflow ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	workflow := &workflowpb.Workflow{}
	convertedWorkflow, err := operations.ConvertMapToProtobuf(result, workflow)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &workflowpb.ReadWorkflowResponse{
		Data:    []*workflowpb.Workflow{convertedWorkflow},
		Success: true,
	}, nil
}

// UpdateWorkflow updates a workflow using common Firestore operations
func (r *FirestoreWorkflowRepository) UpdateWorkflow(ctx context.Context, req *workflowpb.UpdateWorkflowRequest) (*workflowpb.UpdateWorkflowResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workflow ID is required")
	}

	// Convert protobuf to map using ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update workflow: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	workflow := &workflowpb.Workflow{}
	convertedWorkflow, err := operations.ConvertMapToProtobuf(result, workflow)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &workflowpb.UpdateWorkflowResponse{
		Data:    []*workflowpb.Workflow{convertedWorkflow},
		Success: true,
	}, nil
}

// DeleteWorkflow deletes a workflow using common Firestore operations
func (r *FirestoreWorkflowRepository) DeleteWorkflow(ctx context.Context, req *workflowpb.DeleteWorkflowRequest) (*workflowpb.DeleteWorkflowResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workflow ID is required")
	}

	// Delete document using common operations
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete workflow: %w", err)
	}

	return &workflowpb.DeleteWorkflowResponse{
		Success: true,
	}, nil
}

// ListWorkflows retrieves workflows using common Firestore operations
func (r *FirestoreWorkflowRepository) ListWorkflows(ctx context.Context, req *workflowpb.ListWorkflowsRequest) (*workflowpb.ListWorkflowsResponse, error) {
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
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	workflows, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *workflowpb.Workflow {
		return &workflowpb.Workflow{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if workflows == nil {
		workflows = make([]*workflowpb.Workflow, 0)
	}

	return &workflowpb.ListWorkflowsResponse{
		Data:    workflows,
		Success: true,
	}, nil
}

// GetWorkflowListPageData retrieves workflows with pagination using common Firestore operations
func (r *FirestoreWorkflowRepository) GetWorkflowListPageData(ctx context.Context, req *workflowpb.GetWorkflowListPageDataRequest) (*workflowpb.GetWorkflowListPageDataResponse, error) {
	// For now, implement basic list functionality
	// TODO: Implement full pagination, filtering, sorting, and search using common operations
	listReq := &workflowpb.ListWorkflowsRequest{}
	listResp, err := r.ListWorkflows(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow list page data: %w", err)
	}

	return &workflowpb.GetWorkflowListPageDataResponse{
		WorkflowList: listResp.Data,
		Success:      true,
	}, nil
}

// GetWorkflowItemPageData retrieves a single workflow with enhanced data
func (r *FirestoreWorkflowRepository) GetWorkflowItemPageData(ctx context.Context, req *workflowpb.GetWorkflowItemPageDataRequest) (*workflowpb.GetWorkflowItemPageDataResponse, error) {
	if req.WorkflowId == "" {
		return nil, fmt.Errorf("workflow ID is required")
	}

	readReq := &workflowpb.ReadWorkflowRequest{
		Data: &workflowpb.Workflow{Id: req.WorkflowId},
	}
	readResp, err := r.ReadWorkflow(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow item page data: %w", err)
	}

	if len(readResp.Data) == 0 {
		return nil, fmt.Errorf("workflow not found")
	}

	return &workflowpb.GetWorkflowItemPageDataResponse{
		Workflow: readResp.Data[0],
		Success:  true,
	}, nil
}

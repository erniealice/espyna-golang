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
	workflowtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/workflow_template"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "workflow_template", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore workflow_template repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreWorkflowTemplateRepository(dbOps, collectionName), nil
	})
}

// FirestoreWorkflowTemplateRepository implements workflow_template CRUD operations using Firestore
type FirestoreWorkflowTemplateRepository struct {
	workflowtemplatepb.UnimplementedWorkflowTemplateDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreWorkflowTemplateRepository creates a new Firestore workflow_template repository
func NewFirestoreWorkflowTemplateRepository(dbOps interfaces.DatabaseOperation, collectionName string) workflowtemplatepb.WorkflowTemplateDomainServiceServer {
	if collectionName == "" {
		collectionName = "workflow_template" // default fallback (singular to match database.go)
	}
	return &FirestoreWorkflowTemplateRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateWorkflowTemplate creates a new workflow_template using common Firestore operations
func (r *FirestoreWorkflowTemplateRepository) CreateWorkflowTemplate(ctx context.Context, req *workflowtemplatepb.CreateWorkflowTemplateRequest) (*workflowtemplatepb.CreateWorkflowTemplateResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("workflow_template data is required")
	}

	// Convert protobuf to map using ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create workflow_template: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	workflowTemplate := &workflowtemplatepb.WorkflowTemplate{}
	convertedWorkflowTemplate, err := operations.ConvertMapToProtobuf(result, workflowTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &workflowtemplatepb.CreateWorkflowTemplateResponse{
		Data: []*workflowtemplatepb.WorkflowTemplate{convertedWorkflowTemplate},
	}, nil
}

// ReadWorkflowTemplate retrieves a workflow_template using common Firestore operations
func (r *FirestoreWorkflowTemplateRepository) ReadWorkflowTemplate(ctx context.Context, req *workflowtemplatepb.ReadWorkflowTemplateRequest) (*workflowtemplatepb.ReadWorkflowTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workflow_template ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow_template: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	workflowTemplate := &workflowtemplatepb.WorkflowTemplate{}
	convertedWorkflowTemplate, err := operations.ConvertMapToProtobuf(result, workflowTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &workflowtemplatepb.ReadWorkflowTemplateResponse{
		Data: []*workflowtemplatepb.WorkflowTemplate{convertedWorkflowTemplate},
	}, nil
}

// UpdateWorkflowTemplate updates a workflow_template using common Firestore operations
func (r *FirestoreWorkflowTemplateRepository) UpdateWorkflowTemplate(ctx context.Context, req *workflowtemplatepb.UpdateWorkflowTemplateRequest) (*workflowtemplatepb.UpdateWorkflowTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workflow_template ID is required")
	}

	// Convert protobuf to map using ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update workflow_template: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	workflowTemplate := &workflowtemplatepb.WorkflowTemplate{}
	convertedWorkflowTemplate, err := operations.ConvertMapToProtobuf(result, workflowTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &workflowtemplatepb.UpdateWorkflowTemplateResponse{
		Data: []*workflowtemplatepb.WorkflowTemplate{convertedWorkflowTemplate},
	}, nil
}

// DeleteWorkflowTemplate deletes a workflow_template using common Firestore operations
func (r *FirestoreWorkflowTemplateRepository) DeleteWorkflowTemplate(ctx context.Context, req *workflowtemplatepb.DeleteWorkflowTemplateRequest) (*workflowtemplatepb.DeleteWorkflowTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workflow_template ID is required")
	}

	// Delete document using common operations
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete workflow_template: %w", err)
	}

	return &workflowtemplatepb.DeleteWorkflowTemplateResponse{
		Success: true,
	}, nil
}

// ListWorkflowTemplates retrieves workflow_templates using common Firestore operations
func (r *FirestoreWorkflowTemplateRepository) ListWorkflowTemplates(ctx context.Context, req *workflowtemplatepb.ListWorkflowTemplatesRequest) (*workflowtemplatepb.ListWorkflowTemplatesResponse, error) {
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
		return nil, fmt.Errorf("failed to list workflow_templates: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	workflowTemplates, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *workflowtemplatepb.WorkflowTemplate {
		return &workflowtemplatepb.WorkflowTemplate{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if workflowTemplates == nil {
		workflowTemplates = make([]*workflowtemplatepb.WorkflowTemplate, 0)
	}

	return &workflowtemplatepb.ListWorkflowTemplatesResponse{
		Data: workflowTemplates,
	}, nil
}

// GetWorkflowTemplateListPageData retrieves workflow_templates with pagination using common Firestore operations
func (r *FirestoreWorkflowTemplateRepository) GetWorkflowTemplateListPageData(ctx context.Context, req *workflowtemplatepb.GetWorkflowTemplateListPageDataRequest) (*workflowtemplatepb.GetWorkflowTemplateListPageDataResponse, error) {
	// For now, implement basic list functionality
	// TODO: Implement full pagination, filtering, sorting, and search using common operations
	listReq := &workflowtemplatepb.ListWorkflowTemplatesRequest{}
	listResp, err := r.ListWorkflowTemplates(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow_template list page data: %w", err)
	}

	return &workflowtemplatepb.GetWorkflowTemplateListPageDataResponse{
		WorkflowTemplateList: listResp.Data,
		Success:              true,
	}, nil
}

// GetWorkflowTemplateItemPageData retrieves a single workflow_template with enhanced data
func (r *FirestoreWorkflowTemplateRepository) GetWorkflowTemplateItemPageData(ctx context.Context, req *workflowtemplatepb.GetWorkflowTemplateItemPageDataRequest) (*workflowtemplatepb.GetWorkflowTemplateItemPageDataResponse, error) {
	if req.WorkflowTemplateId == "" {
		return nil, fmt.Errorf("workflow_template ID is required")
	}

	readReq := &workflowtemplatepb.ReadWorkflowTemplateRequest{
		Data: &workflowtemplatepb.WorkflowTemplate{Id: req.WorkflowTemplateId},
	}
	readResp, err := r.ReadWorkflowTemplate(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow_template item page data: %w", err)
	}

	if len(readResp.Data) == 0 {
		return nil, fmt.Errorf("workflow_template not found")
	}

	return &workflowtemplatepb.GetWorkflowTemplateItemPageDataResponse{
		WorkflowTemplate: readResp.Data[0],
		Success:          true,
	}, nil
}

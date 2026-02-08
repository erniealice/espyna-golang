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
	stagetemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/stage_template"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "stage_template", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore stage_template repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreStageTemplateRepository(dbOps, collectionName), nil
	})
}

// FirestoreStageTemplateRepository implements stage_template CRUD operations using Firestore
type FirestoreStageTemplateRepository struct {
	stagetemplatepb.UnimplementedStageTemplateDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreStageTemplateRepository creates a new Firestore stage_template repository
func NewFirestoreStageTemplateRepository(dbOps interfaces.DatabaseOperation, collectionName string) stagetemplatepb.StageTemplateDomainServiceServer {
	if collectionName == "" {
		collectionName = "stage_template" // default fallback (singular to match database.go)
	}
	return &FirestoreStageTemplateRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateStageTemplate creates a new stage_template using common Firestore operations
func (r *FirestoreStageTemplateRepository) CreateStageTemplate(ctx context.Context, req *stagetemplatepb.CreateStageTemplateRequest) (*stagetemplatepb.CreateStageTemplateResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("stage_template data is required")
	}

	// Convert protobuf to map using ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create stage_template: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	stageTemplate := &stagetemplatepb.StageTemplate{}
	convertedStageTemplate, err := operations.ConvertMapToProtobuf(result, stageTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &stagetemplatepb.CreateStageTemplateResponse{
		Data:    []*stagetemplatepb.StageTemplate{convertedStageTemplate},
		Success: true,
	}, nil
}

// ReadStageTemplate retrieves a stage_template using common Firestore operations
func (r *FirestoreStageTemplateRepository) ReadStageTemplate(ctx context.Context, req *stagetemplatepb.ReadStageTemplateRequest) (*stagetemplatepb.ReadStageTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("stage_template ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read stage_template: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	stageTemplate := &stagetemplatepb.StageTemplate{}
	convertedStageTemplate, err := operations.ConvertMapToProtobuf(result, stageTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &stagetemplatepb.ReadStageTemplateResponse{
		Data:    []*stagetemplatepb.StageTemplate{convertedStageTemplate},
		Success: true,
	}, nil
}

// UpdateStageTemplate updates a stage_template using common Firestore operations
func (r *FirestoreStageTemplateRepository) UpdateStageTemplate(ctx context.Context, req *stagetemplatepb.UpdateStageTemplateRequest) (*stagetemplatepb.UpdateStageTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("stage_template ID is required")
	}

	// Convert protobuf to map using ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update stage_template: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	stageTemplate := &stagetemplatepb.StageTemplate{}
	convertedStageTemplate, err := operations.ConvertMapToProtobuf(result, stageTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &stagetemplatepb.UpdateStageTemplateResponse{
		Data:    []*stagetemplatepb.StageTemplate{convertedStageTemplate},
		Success: true,
	}, nil
}

// DeleteStageTemplate deletes a stage_template using common Firestore operations
func (r *FirestoreStageTemplateRepository) DeleteStageTemplate(ctx context.Context, req *stagetemplatepb.DeleteStageTemplateRequest) (*stagetemplatepb.DeleteStageTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("stage_template ID is required")
	}

	// Delete document using common operations
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete stage_template: %w", err)
	}

	return &stagetemplatepb.DeleteStageTemplateResponse{
		Success: true,
	}, nil
}

// ListStageTemplates retrieves stage_templates using common Firestore operations
func (r *FirestoreStageTemplateRepository) ListStageTemplates(ctx context.Context, req *stagetemplatepb.ListStageTemplatesRequest) (*stagetemplatepb.ListStageTemplatesResponse, error) {
	fmt.Printf("[ListStageTemplates] START - collectionName=%s\n", r.collectionName)
	fmt.Printf("[ListStageTemplates] Request filters: %+v\n", req.Filters)
	fmt.Printf("[ListStageTemplates] Request search: %s\n", req.Search)
	fmt.Printf("[ListStageTemplates] Request sort: %+v\n", req.Sort)
	fmt.Printf("[ListStageTemplates] Request pagination: %+v\n", req.Pagination)

	// Build ListParams from request - pass filters directly to dbOps.List
	listParams := &interfaces.ListParams{
		Search:     req.Search,
		Filters:    req.Filters,
		Sort:       req.Sort,
		Pagination: req.Pagination,
	}

	fmt.Printf("[ListStageTemplates] Calling dbOps.List with collectionName=%s\n", r.collectionName)
	fmt.Printf("[ListStageTemplates] listParams.Filters: %+v\n", listParams.Filters)

	// List documents using common operations with proper filter support
	listResult, err := r.dbOps.List(ctx, r.collectionName, listParams)
	if err != nil {
		fmt.Printf("[ListStageTemplates] ERROR from dbOps.List: %v\n", err)
		return nil, fmt.Errorf("failed to list stage_templates: %w", err)
	}

	fmt.Printf("[ListStageTemplates] dbOps.List succeeded\n")
	fmt.Printf("[ListStageTemplates] listResult.Data length: %d\n", len(listResult.Data))
	fmt.Printf("[ListStageTemplates] listResult.Data content: %+v\n", listResult.Data)

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	stageTemplates, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *stagetemplatepb.StageTemplate {
		return &stagetemplatepb.StageTemplate{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	fmt.Printf("[ListStageTemplates] Converted stageTemplates length: %d\n", len(stageTemplates))

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if stageTemplates == nil {
		stageTemplates = make([]*stagetemplatepb.StageTemplate, 0)
	}

	fmt.Printf("[ListStageTemplates] END - returning %d stageTemplates, Success=true\n", len(stageTemplates))

	return &stagetemplatepb.ListStageTemplatesResponse{
		Data:    stageTemplates,
		Success: true,
	}, nil
}

// GetStageTemplateListPageData retrieves stage_templates with pagination using common Firestore operations
func (r *FirestoreStageTemplateRepository) GetStageTemplateListPageData(ctx context.Context, req *stagetemplatepb.GetStageTemplateListPageDataRequest) (*stagetemplatepb.GetStageTemplateListPageDataResponse, error) {
	// For now, implement basic list functionality
	// TODO: Implement full pagination, filtering, sorting, and search using common operations
	listReq := &stagetemplatepb.ListStageTemplatesRequest{}
	listResp, err := r.ListStageTemplates(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get stage_template list page data: %w", err)
	}

	return &stagetemplatepb.GetStageTemplateListPageDataResponse{
		StageTemplateList: listResp.Data,
		Success:           true,
	}, nil
}

// GetStageTemplateItemPageData retrieves a single stage_template with enhanced data
func (r *FirestoreStageTemplateRepository) GetStageTemplateItemPageData(ctx context.Context, req *stagetemplatepb.GetStageTemplateItemPageDataRequest) (*stagetemplatepb.GetStageTemplateItemPageDataResponse, error) {
	if req.StageTemplateId == "" {
		return nil, fmt.Errorf("stage_template ID is required")
	}

	readReq := &stagetemplatepb.ReadStageTemplateRequest{
		Data: &stagetemplatepb.StageTemplate{Id: req.StageTemplateId},
	}
	readResp, err := r.ReadStageTemplate(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get stage_template item page data: %w", err)
	}

	if len(readResp.Data) == 0 {
		return nil, fmt.Errorf("stage_template not found")
	}

	return &stagetemplatepb.GetStageTemplateItemPageDataResponse{
		StageTemplate: readResp.Data[0],
		Success:       true,
	}, nil
}

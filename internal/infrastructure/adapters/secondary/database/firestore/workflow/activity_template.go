//go:build firestore

package workflow

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	interfaces "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/interface"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/operations"
	firestoreCore "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/firestore/core"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	activitytemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/activity_template"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "activity_template", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore activity_template repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreActivityTemplateRepository(dbOps, collectionName), nil
	})
}

// FirestoreActivityTemplateRepository implements activity_template CRUD operations using Firestore
type FirestoreActivityTemplateRepository struct {
	activitytemplatepb.UnimplementedActivityTemplateDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreActivityTemplateRepository creates a new Firestore activity_template repository
func NewFirestoreActivityTemplateRepository(dbOps interfaces.DatabaseOperation, collectionName string) activitytemplatepb.ActivityTemplateDomainServiceServer {
	if collectionName == "" {
		collectionName = "activity_template" // default fallback (singular to match database.go)
	}
	return &FirestoreActivityTemplateRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateActivityTemplate creates a new activity_template using common Firestore operations
func (r *FirestoreActivityTemplateRepository) CreateActivityTemplate(ctx context.Context, req *activitytemplatepb.CreateActivityTemplateRequest) (*activitytemplatepb.CreateActivityTemplateResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("activity_template data is required")
	}

	// Convert protobuf to map using ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create activity_template: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	activityTemplate := &activitytemplatepb.ActivityTemplate{}
	convertedActivityTemplate, err := operations.ConvertMapToProtobuf(result, activityTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &activitytemplatepb.CreateActivityTemplateResponse{
		Data:    []*activitytemplatepb.ActivityTemplate{convertedActivityTemplate},
		Success: true,
	}, nil
}

// ReadActivityTemplate retrieves an activity_template using common Firestore operations
func (r *FirestoreActivityTemplateRepository) ReadActivityTemplate(ctx context.Context, req *activitytemplatepb.ReadActivityTemplateRequest) (*activitytemplatepb.ReadActivityTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("activity_template ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read activity_template: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	activityTemplate := &activitytemplatepb.ActivityTemplate{}
	convertedActivityTemplate, err := operations.ConvertMapToProtobuf(result, activityTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &activitytemplatepb.ReadActivityTemplateResponse{
		Data:    []*activitytemplatepb.ActivityTemplate{convertedActivityTemplate},
		Success: true,
	}, nil
}

// UpdateActivityTemplate updates an activity_template using common Firestore operations
func (r *FirestoreActivityTemplateRepository) UpdateActivityTemplate(ctx context.Context, req *activitytemplatepb.UpdateActivityTemplateRequest) (*activitytemplatepb.UpdateActivityTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("activity_template ID is required")
	}

	// Convert protobuf to map using ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update activity_template: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	activityTemplate := &activitytemplatepb.ActivityTemplate{}
	convertedActivityTemplate, err := operations.ConvertMapToProtobuf(result, activityTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &activitytemplatepb.UpdateActivityTemplateResponse{
		Data:    []*activitytemplatepb.ActivityTemplate{convertedActivityTemplate},
		Success: true,
	}, nil
}

// DeleteActivityTemplate deletes an activity_template using common Firestore operations
func (r *FirestoreActivityTemplateRepository) DeleteActivityTemplate(ctx context.Context, req *activitytemplatepb.DeleteActivityTemplateRequest) (*activitytemplatepb.DeleteActivityTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("activity_template ID is required")
	}

	// Delete document using common operations
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete activity_template: %w", err)
	}

	return &activitytemplatepb.DeleteActivityTemplateResponse{
		Success: true,
	}, nil
}

// ListActivityTemplates retrieves activity_templates using common Firestore operations
func (r *FirestoreActivityTemplateRepository) ListActivityTemplates(ctx context.Context, req *activitytemplatepb.ListActivityTemplatesRequest) (*activitytemplatepb.ListActivityTemplatesResponse, error) {
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
		return nil, fmt.Errorf("failed to list activity_templates: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	activityTemplates, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *activitytemplatepb.ActivityTemplate {
		return &activitytemplatepb.ActivityTemplate{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if activityTemplates == nil {
		activityTemplates = make([]*activitytemplatepb.ActivityTemplate, 0)
	}

	return &activitytemplatepb.ListActivityTemplatesResponse{
		Data:    activityTemplates,
		Success: true,
	}, nil
}

// GetActivityTemplateListPageData retrieves activity_templates with pagination using common Firestore operations
func (r *FirestoreActivityTemplateRepository) GetActivityTemplateListPageData(ctx context.Context, req *activitytemplatepb.GetActivityTemplateListPageDataRequest) (*activitytemplatepb.GetActivityTemplateListPageDataResponse, error) {
	// For now, implement basic list functionality
	// TODO: Implement full pagination, filtering, sorting, and search using common operations
	listReq := &activitytemplatepb.ListActivityTemplatesRequest{}
	listResp, err := r.ListActivityTemplates(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity_template list page data: %w", err)
	}

	return &activitytemplatepb.GetActivityTemplateListPageDataResponse{
		ActivityTemplateList: listResp.Data,
		Success:              true,
	}, nil
}

// GetActivityTemplateItemPageData retrieves a single activity_template with enhanced data
func (r *FirestoreActivityTemplateRepository) GetActivityTemplateItemPageData(ctx context.Context, req *activitytemplatepb.GetActivityTemplateItemPageDataRequest) (*activitytemplatepb.GetActivityTemplateItemPageDataResponse, error) {
	if req.ActivityTemplateId == "" {
		return nil, fmt.Errorf("activity_template ID is required")
	}

	readReq := &activitytemplatepb.ReadActivityTemplateRequest{
		Data: &activitytemplatepb.ActivityTemplate{Id: req.ActivityTemplateId},
	}
	readResp, err := r.ReadActivityTemplate(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity_template item page data: %w", err)
	}

	if len(readResp.Data) == 0 {
		return nil, fmt.Errorf("activity_template not found")
	}

	return &activitytemplatepb.GetActivityTemplateItemPageDataResponse{
		ActivityTemplate: readResp.Data[0],
		Success:          true,
	}, nil
}

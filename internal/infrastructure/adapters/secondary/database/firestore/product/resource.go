//go:build firestore

package product

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	interfaces "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/interface"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/operations"
	firestoreCore "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/firestore/core"
	"leapfor.xyz/espyna/internal/infrastructure/registry"

	resourcepb "leapfor.xyz/esqyma/golang/v1/domain/product/resource"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "resource", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore resource repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreResourceRepository(dbOps, collectionName), nil
	})
}

// FirestoreResourceRepository implements resource CRUD operations using Firestore
type FirestoreResourceRepository struct {
	resourcepb.UnimplementedResourceDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreResourceRepository creates a new Firestore resource repository
func NewFirestoreResourceRepository(dbOps interfaces.DatabaseOperation, collectionName string) resourcepb.ResourceDomainServiceServer {
	if collectionName == "" {
		collectionName = "resource" // default fallback
	}
	return &FirestoreResourceRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateResource creates a new resource using common Firestore operations
func (r *FirestoreResourceRepository) CreateResource(ctx context.Context, req *resourcepb.CreateResourceRequest) (*resourcepb.CreateResourceResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("resource data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	resource := &resourcepb.Resource{}
	convertedResource, err := operations.ConvertMapToProtobuf(result, resource)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &resourcepb.CreateResourceResponse{
		Data: []*resourcepb.Resource{convertedResource},
	}, nil
}

// ReadResource retrieves a resource using common Firestore operations
func (r *FirestoreResourceRepository) ReadResource(ctx context.Context, req *resourcepb.ReadResourceRequest) (*resourcepb.ReadResourceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("resource ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read resource: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	resource := &resourcepb.Resource{}
	convertedResource, err := operations.ConvertMapToProtobuf(result, resource)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &resourcepb.ReadResourceResponse{
		Data: []*resourcepb.Resource{convertedResource},
	}, nil
}

// UpdateResource updates a resource using common Firestore operations
func (r *FirestoreResourceRepository) UpdateResource(ctx context.Context, req *resourcepb.UpdateResourceRequest) (*resourcepb.UpdateResourceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("resource ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update resource: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	resource := &resourcepb.Resource{}
	convertedResource, err := operations.ConvertMapToProtobuf(result, resource)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &resourcepb.UpdateResourceResponse{
		Data: []*resourcepb.Resource{convertedResource},
	}, nil
}

// DeleteResource deletes a resource using common Firestore operations
func (r *FirestoreResourceRepository) DeleteResource(ctx context.Context, req *resourcepb.DeleteResourceRequest) (*resourcepb.DeleteResourceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("resource ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete resource: %w", err)
	}

	return &resourcepb.DeleteResourceResponse{
		Success: true,
	}, nil
}

// ListResources lists resources using common Firestore operations
func (r *FirestoreResourceRepository) ListResources(ctx context.Context, req *resourcepb.ListResourcesRequest) (*resourcepb.ListResourcesResponse, error) {
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
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	resources, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *resourcepb.Resource {
		return &resourcepb.Resource{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if resources == nil {
		resources = make([]*resourcepb.Resource, 0)
	}

	return &resourcepb.ListResourcesResponse{
		Data: resources,
	}, nil
}

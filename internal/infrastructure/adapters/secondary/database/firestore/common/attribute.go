//go:build firestore

package common

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/operations"
	firestoreCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/firestore/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "attribute", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore attribute repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreAttributeRepository(dbOps, collectionName), nil
	})
}

// FirestoreAttributeRepository implements attribute CRUD operations using Firestore
type FirestoreAttributeRepository struct {
	commonpb.UnimplementedAttributeDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreAttributeRepository creates a new Firestore attribute repository
func NewFirestoreAttributeRepository(dbOps interfaces.DatabaseOperation, collectionName string) commonpb.AttributeDomainServiceServer {
	if collectionName == "" {
		collectionName = "attribute" // default fallback
	}
	return &FirestoreAttributeRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateAttribute creates a new attribute using common Firestore operations
func (r *FirestoreAttributeRepository) CreateAttribute(ctx context.Context, req *commonpb.CreateAttributeRequest) (*commonpb.CreateAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("attribute data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create attribute: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	attribute := &commonpb.Attribute{}
	convertedAttribute, err := operations.ConvertMapToProtobuf(result, attribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &commonpb.CreateAttributeResponse{
		Data:    []*commonpb.Attribute{convertedAttribute},
		Success: true,
	}, nil
}

// ReadAttribute retrieves an attribute using common Firestore operations
func (r *FirestoreAttributeRepository) ReadAttribute(ctx context.Context, req *commonpb.ReadAttributeRequest) (*commonpb.ReadAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("attribute ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read attribute: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	attribute := &commonpb.Attribute{}
	convertedAttribute, err := operations.ConvertMapToProtobuf(result, attribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &commonpb.ReadAttributeResponse{
		Data:    []*commonpb.Attribute{convertedAttribute},
		Success: true,
	}, nil
}

// UpdateAttribute updates an attribute using common Firestore operations
func (r *FirestoreAttributeRepository) UpdateAttribute(ctx context.Context, req *commonpb.UpdateAttributeRequest) (*commonpb.UpdateAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("attribute ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update attribute: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	attribute := &commonpb.Attribute{}
	convertedAttribute, err := operations.ConvertMapToProtobuf(result, attribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &commonpb.UpdateAttributeResponse{
		Data:    []*commonpb.Attribute{convertedAttribute},
		Success: true,
	}, nil
}

// DeleteAttribute deletes an attribute using common Firestore operations
func (r *FirestoreAttributeRepository) DeleteAttribute(ctx context.Context, req *commonpb.DeleteAttributeRequest) (*commonpb.DeleteAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("attribute ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete attribute: %w", err)
	}

	return &commonpb.DeleteAttributeResponse{
		Success: true,
	}, nil
}

// ListAttributes lists attributes using common Firestore operations with filter support
func (r *FirestoreAttributeRepository) ListAttributes(ctx context.Context, req *commonpb.ListAttributesRequest) (*commonpb.ListAttributesResponse, error) {
	// Log the collection name being queried
	fmt.Printf("📋 ListAttributes: Querying Firestore collection '%s'\n", r.collectionName)

	// Build ListParams from request - pass filters directly to dbOps.List
	listParams := &interfaces.ListParams{
		Filters:    req.Filters,
		Pagination: req.Pagination,
	}

	fmt.Printf("📋 ListAttributes: Filters applied: %+v\n", req.Filters)

	// List documents using common operations with proper filter support
	listResult, err := r.dbOps.List(ctx, r.collectionName, listParams)
	if err != nil {
		fmt.Printf("❌ ListAttributes: Failed to query collection '%s': %v\n", r.collectionName, err)
		return nil, fmt.Errorf("failed to list attributes: %w", err)
	}

	// Use listResult.Data instead of results
	results := listResult.Data

	fmt.Printf("✅ ListAttributes: Retrieved %d documents from collection '%s'\n", len(results), r.collectionName)

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	attributes, conversionErrs := operations.ConvertSliceToProtobuf(results, func() *commonpb.Attribute {
		return &commonpb.Attribute{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// CRITICAL FIX: Ensure we always return a non-nil slice for proper JSON marshaling
	// This guarantees the "data" field is always included in the JSON response
	if attributes == nil {
		attributes = make([]*commonpb.Attribute, 0)
	}

	return &commonpb.ListAttributesResponse{
		Data:    attributes,
		Success: true,
	}, nil
}

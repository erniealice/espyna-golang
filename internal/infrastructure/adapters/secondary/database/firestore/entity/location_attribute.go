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
	locationattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location_attribute"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "location_attribute", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore location_attribute repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreLocationAttributeRepository(dbOps, collectionName), nil
	})
}

// FirestoreLocationAttributeRepository implements location attribute CRUD operations using Firestore
type FirestoreLocationAttributeRepository struct {
	locationattributepb.UnimplementedLocationAttributeDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreLocationAttributeRepository creates a new Firestore location attribute repository
func NewFirestoreLocationAttributeRepository(dbOps interfaces.DatabaseOperation, collectionName string) locationattributepb.LocationAttributeDomainServiceServer {
	if collectionName == "" {
		collectionName = "location_attribute" // default fallback
	}
	return &FirestoreLocationAttributeRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateLocationAttribute creates a new location attribute using common Firestore operations
func (r *FirestoreLocationAttributeRepository) CreateLocationAttribute(ctx context.Context, req *locationattributepb.CreateLocationAttributeRequest) (*locationattributepb.CreateLocationAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("location attribute data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create location attribute: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	locationAttribute := &locationattributepb.LocationAttribute{}
	convertedLocationAttribute, err := operations.ConvertMapToProtobuf(result, locationAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &locationattributepb.CreateLocationAttributeResponse{
		Data: []*locationattributepb.LocationAttribute{convertedLocationAttribute},
	}, nil
}

// ReadLocationAttribute retrieves a location attribute using common Firestore operations
func (r *FirestoreLocationAttributeRepository) ReadLocationAttribute(ctx context.Context, req *locationattributepb.ReadLocationAttributeRequest) (*locationattributepb.ReadLocationAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("location attribute ID is required")
	}

	// Use the proper primary ID from the protobuf model
	// This follows Firestore best practices using document IDs
	docID := req.Data.Id

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, docID)
	if err != nil {
		return nil, fmt.Errorf("failed to read location attribute: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	locationAttribute := &locationattributepb.LocationAttribute{}
	convertedLocationAttribute, err := operations.ConvertMapToProtobuf(result, locationAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &locationattributepb.ReadLocationAttributeResponse{
		Data: []*locationattributepb.LocationAttribute{convertedLocationAttribute},
	}, nil
}

// UpdateLocationAttribute updates a location attribute using common Firestore operations
func (r *FirestoreLocationAttributeRepository) UpdateLocationAttribute(ctx context.Context, req *locationattributepb.UpdateLocationAttributeRequest) (*locationattributepb.UpdateLocationAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("location attribute ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Use the proper primary ID from the protobuf model
	docID := req.Data.Id

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, docID, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update location attribute: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	locationAttribute := &locationattributepb.LocationAttribute{}
	convertedLocationAttribute, err := operations.ConvertMapToProtobuf(result, locationAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &locationattributepb.UpdateLocationAttributeResponse{
		Data: []*locationattributepb.LocationAttribute{convertedLocationAttribute},
	}, nil
}

// DeleteLocationAttribute deletes a location attribute using common Firestore operations
func (r *FirestoreLocationAttributeRepository) DeleteLocationAttribute(ctx context.Context, req *locationattributepb.DeleteLocationAttributeRequest) (*locationattributepb.DeleteLocationAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("location attribute ID is required")
	}

	// Use the proper primary ID from the protobuf model
	docID := req.Data.Id

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, docID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete location attribute: %w", err)
	}

	return &locationattributepb.DeleteLocationAttributeResponse{
		Success: true,
	}, nil
}

// ListLocationAttributes lists location attributes using common Firestore operations
func (r *FirestoreLocationAttributeRepository) ListLocationAttributes(ctx context.Context, req *locationattributepb.ListLocationAttributesRequest) (*locationattributepb.ListLocationAttributesResponse, error) {
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
		return nil, fmt.Errorf("failed to list location attributes: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	locationAttributes, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *locationattributepb.LocationAttribute {
		return &locationattributepb.LocationAttribute{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if locationAttributes == nil {
		locationAttributes = make([]*locationattributepb.LocationAttribute, 0)
	}

	return &locationattributepb.ListLocationAttributesResponse{
		Data: locationAttributes,
	}, nil
}

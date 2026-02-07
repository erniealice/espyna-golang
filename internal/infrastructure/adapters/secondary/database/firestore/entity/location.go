//go:build firestore

package entity

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	interfaces "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/interface"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/operations"
	firestoreCore "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/firestore/core"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	locationpb "leapfor.xyz/esqyma/golang/v1/domain/entity/location"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "location", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore location repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreLocationRepository(dbOps, collectionName), nil
	})
}

// FirestoreLocationRepository implements location CRUD operations using Firestore
type FirestoreLocationRepository struct {
	locationpb.UnimplementedLocationDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreLocationRepository creates a new Firestore location repository
func NewFirestoreLocationRepository(dbOps interfaces.DatabaseOperation, collectionName string) locationpb.LocationDomainServiceServer {
	if collectionName == "" {
		collectionName = "location" // default fallback
	}
	return &FirestoreLocationRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateLocation creates a new location using common Firestore operations
func (r *FirestoreLocationRepository) CreateLocation(ctx context.Context, req *locationpb.CreateLocationRequest) (*locationpb.CreateLocationResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("location data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create location: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	location := &locationpb.Location{}
	convertedLocation, err := operations.ConvertMapToProtobuf(result, location)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &locationpb.CreateLocationResponse{
		Data: []*locationpb.Location{convertedLocation},
	}, nil
}

// ReadLocation retrieves a location using common Firestore operations
func (r *FirestoreLocationRepository) ReadLocation(ctx context.Context, req *locationpb.ReadLocationRequest) (*locationpb.ReadLocationResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("location ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read location: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	location := &locationpb.Location{}
	convertedLocation, err := operations.ConvertMapToProtobuf(result, location)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &locationpb.ReadLocationResponse{
		Data: []*locationpb.Location{convertedLocation},
	}, nil
}

// UpdateLocation updates a location using common Firestore operations
func (r *FirestoreLocationRepository) UpdateLocation(ctx context.Context, req *locationpb.UpdateLocationRequest) (*locationpb.UpdateLocationResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("location ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update location: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	location := &locationpb.Location{}
	convertedLocation, err := operations.ConvertMapToProtobuf(result, location)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &locationpb.UpdateLocationResponse{
		Data: []*locationpb.Location{convertedLocation},
	}, nil
}

// DeleteLocation deletes a location using common Firestore operations
func (r *FirestoreLocationRepository) DeleteLocation(ctx context.Context, req *locationpb.DeleteLocationRequest) (*locationpb.DeleteLocationResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("location ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete location: %w", err)
	}

	return &locationpb.DeleteLocationResponse{
		Success: true,
	}, nil
}

// ListLocations lists locations using common Firestore operations
func (r *FirestoreLocationRepository) ListLocations(ctx context.Context, req *locationpb.ListLocationsRequest) (*locationpb.ListLocationsResponse, error) {
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
		return nil, fmt.Errorf("failed to list locations: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	locations, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *locationpb.Location {
		return &locationpb.Location{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if locations == nil {
		locations = make([]*locationpb.Location, 0)
	}

	return &locationpb.ListLocationsResponse{
		Data: locations,
	}, nil
}

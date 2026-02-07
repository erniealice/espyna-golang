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
	staffattributepb "leapfor.xyz/esqyma/golang/v1/domain/entity/staff_attribute"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "staff_attribute", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore staff_attribute repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreStaffAttributeRepository(dbOps, collectionName), nil
	})
}

// FirestoreStaffAttributeRepository implements staff attribute CRUD operations using Firestore
type FirestoreStaffAttributeRepository struct {
	staffattributepb.UnimplementedStaffAttributeDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreStaffAttributeRepository creates a new Firestore staff attribute repository
func NewFirestoreStaffAttributeRepository(dbOps interfaces.DatabaseOperation, collectionName string) staffattributepb.StaffAttributeDomainServiceServer {
	if collectionName == "" {
		collectionName = "staff_attribute" // default fallback
	}
	return &FirestoreStaffAttributeRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateStaffAttribute creates a new staff attribute using common Firestore operations
func (r *FirestoreStaffAttributeRepository) CreateStaffAttribute(ctx context.Context, req *staffattributepb.CreateStaffAttributeRequest) (*staffattributepb.CreateStaffAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("staff attribute data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create staff attribute: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	staffAttribute := &staffattributepb.StaffAttribute{}
	convertedStaffAttribute, err := operations.ConvertMapToProtobuf(result, staffAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &staffattributepb.CreateStaffAttributeResponse{
		Data: []*staffattributepb.StaffAttribute{convertedStaffAttribute},
	}, nil
}

// ReadStaffAttribute retrieves a staff attribute using common Firestore operations
func (r *FirestoreStaffAttributeRepository) ReadStaffAttribute(ctx context.Context, req *staffattributepb.ReadStaffAttributeRequest) (*staffattributepb.ReadStaffAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("staff attribute ID is required")
	}

	// Read document using common operations with the ID
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read staff attribute: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	staffAttribute := &staffattributepb.StaffAttribute{}
	convertedStaffAttribute, err := operations.ConvertMapToProtobuf(result, staffAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &staffattributepb.ReadStaffAttributeResponse{
		Data: []*staffattributepb.StaffAttribute{convertedStaffAttribute},
	}, nil
}

// UpdateStaffAttribute updates a staff attribute using common Firestore operations
func (r *FirestoreStaffAttributeRepository) UpdateStaffAttribute(ctx context.Context, req *staffattributepb.UpdateStaffAttributeRequest) (*staffattributepb.UpdateStaffAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("staff attribute ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update staff attribute: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	staffAttribute := &staffattributepb.StaffAttribute{}
	convertedStaffAttribute, err := operations.ConvertMapToProtobuf(result, staffAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &staffattributepb.UpdateStaffAttributeResponse{
		Data: []*staffattributepb.StaffAttribute{convertedStaffAttribute},
	}, nil
}

// DeleteStaffAttribute deletes a staff attribute using common Firestore operations
func (r *FirestoreStaffAttributeRepository) DeleteStaffAttribute(ctx context.Context, req *staffattributepb.DeleteStaffAttributeRequest) (*staffattributepb.DeleteStaffAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("staff attribute ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete staff attribute: %w", err)
	}

	return &staffattributepb.DeleteStaffAttributeResponse{
		Success: true,
	}, nil
}

// ListStaffAttributes lists staff attributes using common Firestore operations
func (r *FirestoreStaffAttributeRepository) ListStaffAttributes(ctx context.Context, req *staffattributepb.ListStaffAttributesRequest) (*staffattributepb.ListStaffAttributesResponse, error) {
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
		return nil, fmt.Errorf("failed to list staff attributes: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	staffAttributes, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *staffattributepb.StaffAttribute {
		return &staffattributepb.StaffAttribute{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if staffAttributes == nil {
		staffAttributes = make([]*staffattributepb.StaffAttribute, 0)
	}

	return &staffattributepb.ListStaffAttributesResponse{
		Data: staffAttributes,
	}, nil
}
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
	staffpb "leapfor.xyz/esqyma/golang/v1/domain/entity/staff"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "staff", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore staff repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreStaffRepository(dbOps, collectionName), nil
	})
}

// FirestoreStaffRepository implements staff CRUD operations using Firestore
type FirestoreStaffRepository struct {
	staffpb.UnimplementedStaffDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreStaffRepository creates a new Firestore staff repository
func NewFirestoreStaffRepository(dbOps interfaces.DatabaseOperation, collectionName string) staffpb.StaffDomainServiceServer {
	if collectionName == "" {
		collectionName = "staff" // default fallback
	}
	return &FirestoreStaffRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateStaff creates a new staff using common Firestore operations
func (r *FirestoreStaffRepository) CreateStaff(ctx context.Context, req *staffpb.CreateStaffRequest) (*staffpb.CreateStaffResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("staff data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create staff: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	convertedStaff, err := operations.ConvertMapToProtobuf(result, &staffpb.Staff{})
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &staffpb.CreateStaffResponse{
		Data: []*staffpb.Staff{convertedStaff},
	}, nil
}

// ReadStaff retrieves a staff using common Firestore operations
func (r *FirestoreStaffRepository) ReadStaff(ctx context.Context, req *staffpb.ReadStaffRequest) (*staffpb.ReadStaffResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("staff ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read staff: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	convertedStaff, err := operations.ConvertMapToProtobuf(result, &staffpb.Staff{})
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &staffpb.ReadStaffResponse{
		Data: []*staffpb.Staff{convertedStaff},
	}, nil
}

// UpdateStaff updates a staff using common Firestore operations
func (r *FirestoreStaffRepository) UpdateStaff(ctx context.Context, req *staffpb.UpdateStaffRequest) (*staffpb.UpdateStaffResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("staff ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update staff: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	convertedStaff, err := operations.ConvertMapToProtobuf(result, &staffpb.Staff{})
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &staffpb.UpdateStaffResponse{
		Data: []*staffpb.Staff{convertedStaff},
	}, nil
}

// DeleteStaff deletes a staff using common Firestore operations
func (r *FirestoreStaffRepository) DeleteStaff(ctx context.Context, req *staffpb.DeleteStaffRequest) (*staffpb.DeleteStaffResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("staff ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete staff: %w", err)
	}

	return &staffpb.DeleteStaffResponse{
		Success: true,
	}, nil
}

// ListStaffs lists staffs using common Firestore operations
func (r *FirestoreStaffRepository) ListStaffs(ctx context.Context, req *staffpb.ListStaffsRequest) (*staffpb.ListStaffsResponse, error) {
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
		return nil, fmt.Errorf("failed to list staffs: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	staffs, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *staffpb.Staff {
		return &staffpb.Staff{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if staffs == nil {
		staffs = make([]*staffpb.Staff, 0)
	}

	return &staffpb.ListStaffsResponse{
		Data: staffs,
	}, nil
}

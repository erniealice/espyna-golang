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
	adminpb "leapfor.xyz/esqyma/golang/v1/domain/entity/admin"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "admin", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore admin repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreAdminRepository(dbOps, collectionName), nil
	})
}

// FirestoreAdminRepository implements admin CRUD operations using Firestore
type FirestoreAdminRepository struct {
	adminpb.UnimplementedAdminDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreAdminRepository creates a new Firestore admin repository
func NewFirestoreAdminRepository(dbOps interfaces.DatabaseOperation, collectionName string) adminpb.AdminDomainServiceServer {
	if collectionName == "" {
		collectionName = "admin" // default fallback
	}
	return &FirestoreAdminRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateAdmin creates a new admin using common Firestore operations
func (r *FirestoreAdminRepository) CreateAdmin(ctx context.Context, req *adminpb.CreateAdminRequest) (*adminpb.CreateAdminResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("admin data is required")
	}

	// Convert protobuf to map using ProtobufMapper (more efficient)
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create admin: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	admin := &adminpb.Admin{}
	convertedAdmin, err := operations.ConvertMapToProtobuf(result, admin)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &adminpb.CreateAdminResponse{
		Data: []*adminpb.Admin{convertedAdmin},
	}, nil
}

// ReadAdmin retrieves an admin using common Firestore operations
func (r *FirestoreAdminRepository) ReadAdmin(ctx context.Context, req *adminpb.ReadAdminRequest) (*adminpb.ReadAdminResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("admin ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read admin: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	admin := &adminpb.Admin{}
	convertedAdmin, err := operations.ConvertMapToProtobuf(result, admin)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &adminpb.ReadAdminResponse{
		Data: []*adminpb.Admin{convertedAdmin},
	}, nil
}

// UpdateAdmin updates an admin using common Firestore operations
func (r *FirestoreAdminRepository) UpdateAdmin(ctx context.Context, req *adminpb.UpdateAdminRequest) (*adminpb.UpdateAdminResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("admin ID is required")
	}

	// Convert protobuf to map using ProtobufMapper (more efficient)
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update admin: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	admin := &adminpb.Admin{}
	convertedAdmin, err := operations.ConvertMapToProtobuf(result, admin)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &adminpb.UpdateAdminResponse{
		Data: []*adminpb.Admin{convertedAdmin},
	}, nil
}

// DeleteAdmin deletes an admin using common Firestore operations
func (r *FirestoreAdminRepository) DeleteAdmin(ctx context.Context, req *adminpb.DeleteAdminRequest) (*adminpb.DeleteAdminResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("admin ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete admin: %w", err)
	}

	return &adminpb.DeleteAdminResponse{
		Success: true,
	}, nil
}

// ListAdmins lists admins using common Firestore operations
func (r *FirestoreAdminRepository) ListAdmins(ctx context.Context, req *adminpb.ListAdminsRequest) (*adminpb.ListAdminsResponse, error) {
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
		return nil, fmt.Errorf("failed to list admins: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	admins, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *adminpb.Admin {
		return &adminpb.Admin{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if admins == nil {
		admins = make([]*adminpb.Admin, 0)
	}

	return &adminpb.ListAdminsResponse{
		Data: admins,
	}, nil
}

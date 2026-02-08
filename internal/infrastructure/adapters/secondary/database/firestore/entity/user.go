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
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "user", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore user repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreUserRepository(dbOps, collectionName), nil
	})
}

// FirestoreUserRepository implements user CRUD operations using Firestore
type FirestoreUserRepository struct {
	userpb.UnimplementedUserDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreUserRepository creates a new Firestore user repository
func NewFirestoreUserRepository(dbOps interfaces.DatabaseOperation, collectionName string) userpb.UserDomainServiceServer {
	if collectionName == "" {
		collectionName = "user" // default fallback
	}
	return &FirestoreUserRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateUser creates a new user using common Firestore operations
func (r *FirestoreUserRepository) CreateUser(ctx context.Context, req *userpb.CreateUserRequest) (*userpb.CreateUserResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("user data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Use transaction-aware database operations
	txAwareDbOps := r.dbOps

	// Create document using common operations (automatically transaction-aware)
	result, err := txAwareDbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	convertedUser, err := operations.ConvertMapToProtobuf(result, &userpb.User{})
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &userpb.CreateUserResponse{
		Data: []*userpb.User{convertedUser},
	}, nil
}

// ReadUser retrieves a user using common Firestore operations
func (r *FirestoreUserRepository) ReadUser(ctx context.Context, req *userpb.ReadUserRequest) (*userpb.ReadUserResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	// Use transaction-aware database operations
	txAwareDbOps := r.dbOps

	// Read document using common operations (automatically transaction-aware)
	result, err := txAwareDbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read user: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	convertedUser, err := operations.ConvertMapToProtobuf(result, &userpb.User{})
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &userpb.ReadUserResponse{
		Data: []*userpb.User{convertedUser},
	}, nil
}

// UpdateUser updates a user using common Firestore operations
func (r *FirestoreUserRepository) UpdateUser(ctx context.Context, req *userpb.UpdateUserRequest) (*userpb.UpdateUserResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Use transaction-aware database operations
	txAwareDbOps := r.dbOps

	// Update document using common operations (automatically transaction-aware)
	result, err := txAwareDbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	convertedUser, err := operations.ConvertMapToProtobuf(result, &userpb.User{})
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &userpb.UpdateUserResponse{
		Data: []*userpb.User{convertedUser},
	}, nil
}

// DeleteUser deletes a user using common Firestore operations
func (r *FirestoreUserRepository) DeleteUser(ctx context.Context, req *userpb.DeleteUserRequest) (*userpb.DeleteUserResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	// Use transaction-aware database operations
	txAwareDbOps := r.dbOps

	// Delete document using common operations (soft delete, automatically transaction-aware)
	err := txAwareDbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete user: %w", err)
	}

	return &userpb.DeleteUserResponse{
		Success: true,
	}, nil
}

// ListUsers lists users using common Firestore operations
func (r *FirestoreUserRepository) ListUsers(ctx context.Context, req *userpb.ListUsersRequest) (*userpb.ListUsersResponse, error) {
	// Use transaction-aware database operations
	txAwareDbOps := r.dbOps

	// Build ListParams from request - pass filters directly to dbOps.List
	listParams := &interfaces.ListParams{
		Search:     req.Search,
		Filters:    req.Filters,
		Sort:       req.Sort,
		Pagination: req.Pagination,
	}

	// List documents using common operations with proper filter support
	listResult, err := txAwareDbOps.List(ctx, r.collectionName, listParams)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	users, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *userpb.User {
		return &userpb.User{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// CRITICAL FIX: Ensure we always return a non-nil slice for proper JSON marshaling
	// This guarantees the "data" field is always included in the JSON response
	if users == nil {
		users = make([]*userpb.User, 0)
	}

	return &userpb.ListUsersResponse{
		Data: users,
	}, nil
}

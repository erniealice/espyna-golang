//go:build firestore

package subscription

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	interfaces "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/interface"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/operations"
	firestoreCore "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/firestore/core"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	balanceattributepb "leapfor.xyz/esqyma/golang/v1/domain/subscription/balance_attribute"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "balance_attribute", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore balance_attribute repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreBalanceAttributeRepository(dbOps, collectionName), nil
	})
}

// FirestoreBalanceAttributeRepository implements balance attribute CRUD operations using Firestore
type FirestoreBalanceAttributeRepository struct {
	balanceattributepb.UnimplementedBalanceAttributeDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreBalanceAttributeRepository creates a new Firestore balance attribute repository
func NewFirestoreBalanceAttributeRepository(dbOps interfaces.DatabaseOperation, collectionName string) balanceattributepb.BalanceAttributeDomainServiceServer {
	if collectionName == "" {
		collectionName = "balance_attribute" // default fallback
	}
	return &FirestoreBalanceAttributeRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateBalanceAttribute creates a new balance attribute using common Firestore operations
func (r *FirestoreBalanceAttributeRepository) CreateBalanceAttribute(ctx context.Context, req *balanceattributepb.CreateBalanceAttributeRequest) (*balanceattributepb.CreateBalanceAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("balance attribute data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create balance attribute: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	balanceAttribute := &balanceattributepb.BalanceAttribute{}
	convertedBalanceAttribute, err := operations.ConvertMapToProtobuf(result, balanceAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &balanceattributepb.CreateBalanceAttributeResponse{
		Data: []*balanceattributepb.BalanceAttribute{convertedBalanceAttribute},
	}, nil
}

// ReadBalanceAttribute retrieves a balance attribute using common Firestore operations
func (r *FirestoreBalanceAttributeRepository) ReadBalanceAttribute(ctx context.Context, req *balanceattributepb.ReadBalanceAttributeRequest) (*balanceattributepb.ReadBalanceAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("balance attribute ID is required")
	}

	// Read document using common operations with the ID
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read balance attribute: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	balanceAttribute := &balanceattributepb.BalanceAttribute{}
	convertedBalanceAttribute, err := operations.ConvertMapToProtobuf(result, balanceAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &balanceattributepb.ReadBalanceAttributeResponse{
		Data: []*balanceattributepb.BalanceAttribute{convertedBalanceAttribute},
	}, nil
}

// UpdateBalanceAttribute updates a balance attribute using common Firestore operations
func (r *FirestoreBalanceAttributeRepository) UpdateBalanceAttribute(ctx context.Context, req *balanceattributepb.UpdateBalanceAttributeRequest) (*balanceattributepb.UpdateBalanceAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("balance attribute ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update balance attribute: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	balanceAttribute := &balanceattributepb.BalanceAttribute{}
	convertedBalanceAttribute, err := operations.ConvertMapToProtobuf(result, balanceAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &balanceattributepb.UpdateBalanceAttributeResponse{
		Data: []*balanceattributepb.BalanceAttribute{convertedBalanceAttribute},
	}, nil
}

// DeleteBalanceAttribute deletes a balance attribute using common Firestore operations
func (r *FirestoreBalanceAttributeRepository) DeleteBalanceAttribute(ctx context.Context, req *balanceattributepb.DeleteBalanceAttributeRequest) (*balanceattributepb.DeleteBalanceAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("balance attribute ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete balance attribute: %w", err)
	}

	return &balanceattributepb.DeleteBalanceAttributeResponse{
		Success: true,
	}, nil
}

// ListBalanceAttributes lists balance attributes using common Firestore operations
func (r *FirestoreBalanceAttributeRepository) ListBalanceAttributes(ctx context.Context, req *balanceattributepb.ListBalanceAttributesRequest) (*balanceattributepb.ListBalanceAttributesResponse, error) {
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
		return nil, fmt.Errorf("failed to list balance attributes: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	balanceAttributes, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *balanceattributepb.BalanceAttribute {
		return &balanceattributepb.BalanceAttribute{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if balanceAttributes == nil {
		balanceAttributes = make([]*balanceattributepb.BalanceAttribute, 0)
	}

	return &balanceattributepb.ListBalanceAttributesResponse{
		Data: balanceAttributes,
	}, nil
}
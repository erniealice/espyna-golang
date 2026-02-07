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
	subscriptionattributepb "leapfor.xyz/esqyma/golang/v1/domain/subscription/subscription_attribute"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "subscription_attribute", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore subscription_attribute repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreSubscriptionAttributeRepository(dbOps, collectionName), nil
	})
}

// FirestoreSubscriptionAttributeRepository implements subscription attribute CRUD operations using Firestore
type FirestoreSubscriptionAttributeRepository struct {
	subscriptionattributepb.UnimplementedSubscriptionAttributeDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreSubscriptionAttributeRepository creates a new Firestore subscription attribute repository
func NewFirestoreSubscriptionAttributeRepository(dbOps interfaces.DatabaseOperation, collectionName string) subscriptionattributepb.SubscriptionAttributeDomainServiceServer {
	if collectionName == "" {
		collectionName = "subscription_attribute" // default fallback
	}
	return &FirestoreSubscriptionAttributeRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateSubscriptionAttribute creates a new subscription attribute using common Firestore operations
func (r *FirestoreSubscriptionAttributeRepository) CreateSubscriptionAttribute(ctx context.Context, req *subscriptionattributepb.CreateSubscriptionAttributeRequest) (*subscriptionattributepb.CreateSubscriptionAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("subscription attribute data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription attribute: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	subscriptionAttribute := &subscriptionattributepb.SubscriptionAttribute{}
	convertedSubscriptionAttribute, err := operations.ConvertMapToProtobuf(result, subscriptionAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &subscriptionattributepb.CreateSubscriptionAttributeResponse{
		Data: []*subscriptionattributepb.SubscriptionAttribute{convertedSubscriptionAttribute},
	}, nil
}

// ReadSubscriptionAttribute retrieves a subscription attribute using common Firestore operations
func (r *FirestoreSubscriptionAttributeRepository) ReadSubscriptionAttribute(ctx context.Context, req *subscriptionattributepb.ReadSubscriptionAttributeRequest) (*subscriptionattributepb.ReadSubscriptionAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription attribute ID is required")
	}

	// Read document using common operations with the ID
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read subscription attribute: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	subscriptionAttribute := &subscriptionattributepb.SubscriptionAttribute{}
	convertedSubscriptionAttribute, err := operations.ConvertMapToProtobuf(result, subscriptionAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &subscriptionattributepb.ReadSubscriptionAttributeResponse{
		Data: []*subscriptionattributepb.SubscriptionAttribute{convertedSubscriptionAttribute},
	}, nil
}

// UpdateSubscriptionAttribute updates a subscription attribute using common Firestore operations
func (r *FirestoreSubscriptionAttributeRepository) UpdateSubscriptionAttribute(ctx context.Context, req *subscriptionattributepb.UpdateSubscriptionAttributeRequest) (*subscriptionattributepb.UpdateSubscriptionAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription attribute ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update subscription attribute: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	subscriptionAttribute := &subscriptionattributepb.SubscriptionAttribute{}
	convertedSubscriptionAttribute, err := operations.ConvertMapToProtobuf(result, subscriptionAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &subscriptionattributepb.UpdateSubscriptionAttributeResponse{
		Data: []*subscriptionattributepb.SubscriptionAttribute{convertedSubscriptionAttribute},
	}, nil
}

// DeleteSubscriptionAttribute deletes a subscription attribute using common Firestore operations
func (r *FirestoreSubscriptionAttributeRepository) DeleteSubscriptionAttribute(ctx context.Context, req *subscriptionattributepb.DeleteSubscriptionAttributeRequest) (*subscriptionattributepb.DeleteSubscriptionAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription attribute ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete subscription attribute: %w", err)
	}

	return &subscriptionattributepb.DeleteSubscriptionAttributeResponse{
		Success: true,
	}, nil
}

// ListSubscriptionAttributes lists subscription attributes using common Firestore operations
func (r *FirestoreSubscriptionAttributeRepository) ListSubscriptionAttributes(ctx context.Context, req *subscriptionattributepb.ListSubscriptionAttributesRequest) (*subscriptionattributepb.ListSubscriptionAttributesResponse, error) {
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
		return nil, fmt.Errorf("failed to list subscription attributes: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	subscriptionAttributes, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *subscriptionattributepb.SubscriptionAttribute {
		return &subscriptionattributepb.SubscriptionAttribute{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if subscriptionAttributes == nil {
		subscriptionAttributes = make([]*subscriptionattributepb.SubscriptionAttribute, 0)
	}

	return &subscriptionattributepb.ListSubscriptionAttributesResponse{
		Data: subscriptionAttributes,
	}, nil
}
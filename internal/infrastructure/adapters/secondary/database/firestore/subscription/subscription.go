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
	subscriptionpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/subscription"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "subscription", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore subscription repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreSubscriptionRepository(dbOps, collectionName), nil
	})
}

// FirestoreSubscriptionRepository implements subscription CRUD operations using Firestore
type FirestoreSubscriptionRepository struct {
	subscriptionpb.UnimplementedSubscriptionDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreSubscriptionRepository creates a new Firestore subscription repository
func NewFirestoreSubscriptionRepository(dbOps interfaces.DatabaseOperation, collectionName string) subscriptionpb.SubscriptionDomainServiceServer {
	if collectionName == "" {
		collectionName = "subscription" // default fallback
	}
	return &FirestoreSubscriptionRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateSubscription creates a new subscription using common Firestore operations
func (r *FirestoreSubscriptionRepository) CreateSubscription(ctx context.Context, req *subscriptionpb.CreateSubscriptionRequest) (*subscriptionpb.CreateSubscriptionResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("subscription data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	subscription := &subscriptionpb.Subscription{}
	convertedSubscription, err := operations.ConvertMapToProtobuf(result, subscription)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &subscriptionpb.CreateSubscriptionResponse{
		Data: []*subscriptionpb.Subscription{convertedSubscription},
	}, nil
}

// ReadSubscription retrieves a subscription using common Firestore operations
func (r *FirestoreSubscriptionRepository) ReadSubscription(ctx context.Context, req *subscriptionpb.ReadSubscriptionRequest) (*subscriptionpb.ReadSubscriptionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read subscription: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	subscription := &subscriptionpb.Subscription{}
	convertedSubscription, err := operations.ConvertMapToProtobuf(result, subscription)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &subscriptionpb.ReadSubscriptionResponse{
		Data: []*subscriptionpb.Subscription{convertedSubscription},
	}, nil
}

// UpdateSubscription updates a subscription using common Firestore operations
func (r *FirestoreSubscriptionRepository) UpdateSubscription(ctx context.Context, req *subscriptionpb.UpdateSubscriptionRequest) (*subscriptionpb.UpdateSubscriptionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update subscription: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	subscription := &subscriptionpb.Subscription{}
	convertedSubscription, err := operations.ConvertMapToProtobuf(result, subscription)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &subscriptionpb.UpdateSubscriptionResponse{
		Data: []*subscriptionpb.Subscription{convertedSubscription},
	}, nil
}

// DeleteSubscription deletes a subscription using common Firestore operations
func (r *FirestoreSubscriptionRepository) DeleteSubscription(ctx context.Context, req *subscriptionpb.DeleteSubscriptionRequest) (*subscriptionpb.DeleteSubscriptionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete subscription: %w", err)
	}

	return &subscriptionpb.DeleteSubscriptionResponse{
		Success: true,
	}, nil
}

// ListSubscriptions lists subscriptions using common Firestore operations
func (r *FirestoreSubscriptionRepository) ListSubscriptions(ctx context.Context, req *subscriptionpb.ListSubscriptionsRequest) (*subscriptionpb.ListSubscriptionsResponse, error) {
	// Log the collection name being queried
	fmt.Printf("📋 ListSubscriptions: Querying Firestore collection '%s'\n", r.collectionName)

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
		fmt.Printf("❌ ListSubscriptions: Failed to query collection '%s': %v\n", r.collectionName, err)
		return nil, fmt.Errorf("failed to list subscriptions: %w", err)
	}

	fmt.Printf("✅ ListSubscriptions: Retrieved %d documents from collection '%s'\n", len(listResult.Data), r.collectionName)

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	subscriptions, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *subscriptionpb.Subscription {
		return &subscriptionpb.Subscription{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if subscriptions == nil {
		subscriptions = make([]*subscriptionpb.Subscription, 0)
	}

	return &subscriptionpb.ListSubscriptionsResponse{
		Data:    subscriptions,
		Success: true,
	}, nil
}


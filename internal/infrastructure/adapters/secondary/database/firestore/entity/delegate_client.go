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
	delegateclientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate_client"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "delegate_client", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore delegate_client repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreDelegateClientRepository(dbOps, collectionName), nil
	})
}

// FirestoreDelegateClientRepository implements delegate client CRUD operations using Firestore
type FirestoreDelegateClientRepository struct {
	delegateclientpb.UnimplementedDelegateClientDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreDelegateClientRepository creates a new Firestore delegate client repository
func NewFirestoreDelegateClientRepository(dbOps interfaces.DatabaseOperation, collectionName string) delegateclientpb.DelegateClientDomainServiceServer {
	if collectionName == "" {
		collectionName = "delegate_client" // default fallback
	}
	return &FirestoreDelegateClientRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateDelegateClient creates a new delegate client using common Firestore operations
func (r *FirestoreDelegateClientRepository) CreateDelegateClient(ctx context.Context, req *delegateclientpb.CreateDelegateClientRequest) (*delegateclientpb.CreateDelegateClientResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("delegate client data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create delegate client: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	delegateClient := &delegateclientpb.DelegateClient{}
	convertedDelegateClient, err := operations.ConvertMapToProtobuf(result, delegateClient)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &delegateclientpb.CreateDelegateClientResponse{
		Data: []*delegateclientpb.DelegateClient{convertedDelegateClient},
	}, nil
}

// ReadDelegateClient retrieves a delegate client using common Firestore operations
func (r *FirestoreDelegateClientRepository) ReadDelegateClient(ctx context.Context, req *delegateclientpb.ReadDelegateClientRequest) (*delegateclientpb.ReadDelegateClientResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("delegate client ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read delegate client: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	delegateClient := &delegateclientpb.DelegateClient{}
	convertedDelegateClient, err := operations.ConvertMapToProtobuf(result, delegateClient)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &delegateclientpb.ReadDelegateClientResponse{
		Data: []*delegateclientpb.DelegateClient{convertedDelegateClient},
	}, nil
}

// UpdateDelegateClient updates a delegate client using common Firestore operations
func (r *FirestoreDelegateClientRepository) UpdateDelegateClient(ctx context.Context, req *delegateclientpb.UpdateDelegateClientRequest) (*delegateclientpb.UpdateDelegateClientResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("delegate client ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update delegate client: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	delegateClient := &delegateclientpb.DelegateClient{}
	convertedDelegateClient, err := operations.ConvertMapToProtobuf(result, delegateClient)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &delegateclientpb.UpdateDelegateClientResponse{
		Data: []*delegateclientpb.DelegateClient{convertedDelegateClient},
	}, nil
}

// DeleteDelegateClient deletes a delegate client using common Firestore operations
func (r *FirestoreDelegateClientRepository) DeleteDelegateClient(ctx context.Context, req *delegateclientpb.DeleteDelegateClientRequest) (*delegateclientpb.DeleteDelegateClientResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("delegate client ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete delegate client: %w", err)
	}

	return &delegateclientpb.DeleteDelegateClientResponse{
		Success: true,
	}, nil
}

// ListDelegateClients lists delegate clients using common Firestore operations
func (r *FirestoreDelegateClientRepository) ListDelegateClients(ctx context.Context, req *delegateclientpb.ListDelegateClientsRequest) (*delegateclientpb.ListDelegateClientsResponse, error) {
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
		return nil, fmt.Errorf("failed to list delegate clients: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	delegateClients, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *delegateclientpb.DelegateClient {
		return &delegateclientpb.DelegateClient{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if delegateClients == nil {
		delegateClients = make([]*delegateclientpb.DelegateClient, 0)
	}

	return &delegateclientpb.ListDelegateClientsResponse{
		Data: delegateClients,
	}, nil
}

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
	clientattributepb "leapfor.xyz/esqyma/golang/v1/domain/entity/client_attribute"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "client_attribute", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore client_attribute repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreClientAttributeRepository(dbOps, collectionName), nil
	})
}

// FirestoreClientAttributeRepository implements client attribute CRUD operations using Firestore
type FirestoreClientAttributeRepository struct {
	clientattributepb.UnimplementedClientAttributeDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreClientAttributeRepository creates a new Firestore client attribute repository
func NewFirestoreClientAttributeRepository(dbOps interfaces.DatabaseOperation, collectionName string) clientattributepb.ClientAttributeDomainServiceServer {
	if collectionName == "" {
		collectionName = "client_attribute" // default fallback
	}
	return &FirestoreClientAttributeRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateClientAttribute creates a new client attribute using common Firestore operations
func (r *FirestoreClientAttributeRepository) CreateClientAttribute(ctx context.Context, req *clientattributepb.CreateClientAttributeRequest) (*clientattributepb.CreateClientAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("client attribute data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create client attribute: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	clientAttribute := &clientattributepb.ClientAttribute{}
	convertedClientAttribute, err := operations.ConvertMapToProtobuf(result, clientAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &clientattributepb.CreateClientAttributeResponse{
		Data: []*clientattributepb.ClientAttribute{convertedClientAttribute},
	}, nil
}

// ReadClientAttribute retrieves a client attribute using common Firestore operations
func (r *FirestoreClientAttributeRepository) ReadClientAttribute(ctx context.Context, req *clientattributepb.ReadClientAttributeRequest) (*clientattributepb.ReadClientAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client attribute ID is required")
	}

	// Read document using common operations with the ID
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read client attribute: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	clientAttribute := &clientattributepb.ClientAttribute{}
	convertedClientAttribute, err := operations.ConvertMapToProtobuf(result, clientAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &clientattributepb.ReadClientAttributeResponse{
		Data: []*clientattributepb.ClientAttribute{convertedClientAttribute},
	}, nil
}

// UpdateClientAttribute updates a client attribute using common Firestore operations
func (r *FirestoreClientAttributeRepository) UpdateClientAttribute(ctx context.Context, req *clientattributepb.UpdateClientAttributeRequest) (*clientattributepb.UpdateClientAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client attribute ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update client attribute: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	clientAttribute := &clientattributepb.ClientAttribute{}
	convertedClientAttribute, err := operations.ConvertMapToProtobuf(result, clientAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &clientattributepb.UpdateClientAttributeResponse{
		Data: []*clientattributepb.ClientAttribute{convertedClientAttribute},
	}, nil
}

// DeleteClientAttribute deletes a client attribute using common Firestore operations
func (r *FirestoreClientAttributeRepository) DeleteClientAttribute(ctx context.Context, req *clientattributepb.DeleteClientAttributeRequest) (*clientattributepb.DeleteClientAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client attribute ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete client attribute: %w", err)
	}

	return &clientattributepb.DeleteClientAttributeResponse{
		Success: true,
	}, nil
}

// ListClientAttributes lists client attributes using common Firestore operations
func (r *FirestoreClientAttributeRepository) ListClientAttributes(ctx context.Context, req *clientattributepb.ListClientAttributesRequest) (*clientattributepb.ListClientAttributesResponse, error) {
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
		return nil, fmt.Errorf("failed to list client attributes: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	clientAttributes, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *clientattributepb.ClientAttribute {
		return &clientattributepb.ClientAttribute{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if clientAttributes == nil {
		clientAttributes = make([]*clientattributepb.ClientAttribute, 0)
	}

	return &clientattributepb.ListClientAttributesResponse{
		Data: clientAttributes,
	}, nil
}

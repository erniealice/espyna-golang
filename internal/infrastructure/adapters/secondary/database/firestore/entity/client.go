//go:build firestore

package entity

import (
	"context"
	"fmt"
	"os"

	"cloud.google.com/go/firestore"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/operations"
	firestoreCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/firestore/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// =============================================================================
// Self-Registration - Repository registers itself with the factory
// =============================================================================

func init() {
	registry.RegisterRepositoryFactory("firestore", "client", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore client repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreClientRepository(dbOps, collectionName), nil
	})
}

// FirestoreClientRepository implements client CRUD operations using Firestore
type FirestoreClientRepository struct {
	clientpb.UnimplementedClientDomainServiceServer
	dbOps              interfaces.DatabaseOperation
	collectionName     string
	userCollectionName string // For enriching Client with User data
	mapper             *operations.ProtobufMapper
}

// NewFirestoreClientRepository creates a new Firestore client repository
func NewFirestoreClientRepository(dbOps interfaces.DatabaseOperation, collectionName string) clientpb.ClientDomainServiceServer {
	if collectionName == "" {
		collectionName = "client" // default fallback
	}

	// Get user collection name from environment for enrichment
	// Falls back to "user" if not set
	userCollectionName := os.Getenv("LEAPFOR_DATABASE_FIRESTORE_COLLECTION_USER")
	if userCollectionName == "" {
		userCollectionName = "user"
	}

	return &FirestoreClientRepository{
		dbOps:              dbOps,
		collectionName:     collectionName,
		userCollectionName: userCollectionName,
		mapper:             operations.NewProtobufMapper(),
	}
}

// CreateClient creates a new client using common Firestore operations
func (r *FirestoreClientRepository) CreateClient(ctx context.Context, req *clientpb.CreateClientRequest) (*clientpb.CreateClientResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("client data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Remove nested objects before storing to Firestore - only store references
	// The 'user' object is for READ operations, we only need to store user_id
	delete(data, "user")

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	client := &clientpb.Client{}
	convertedClient, err := operations.ConvertMapToProtobuf(result, client)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &clientpb.CreateClientResponse{
		Data: []*clientpb.Client{convertedClient},
	}, nil
}

// ReadClient retrieves a client using common Firestore operations
// Also enriches the Client with User data if user_id is present
func (r *FirestoreClientRepository) ReadClient(ctx context.Context, req *clientpb.ReadClientRequest) (*clientpb.ReadClientResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read client: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	client := &clientpb.Client{}
	convertedClient, err := operations.ConvertMapToProtobuf(result, client)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	// Enrich: Fetch and populate User if user_id exists but user is nil
	if convertedClient.User == nil && convertedClient.UserId != "" {
		userResult, userErr := r.dbOps.Read(ctx, r.userCollectionName, convertedClient.UserId)
		if userErr == nil && userResult != nil {
			user := &userpb.User{}
			if convertedUser, convErr := operations.ConvertMapToProtobuf(userResult, user); convErr == nil {
				convertedClient.User = convertedUser
			}
		}
		// Note: We silently ignore user fetch errors - client data is still valid
		// This prevents workflow failures when user record doesn't exist
	}

	return &clientpb.ReadClientResponse{
		Data:    []*clientpb.Client{convertedClient},
		Success: true,
	}, nil
}

// UpdateClient updates a client using common Firestore operations
func (r *FirestoreClientRepository) UpdateClient(ctx context.Context, req *clientpb.UpdateClientRequest) (*clientpb.UpdateClientResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update client: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	client := &clientpb.Client{}
	convertedClient, err := operations.ConvertMapToProtobuf(result, client)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &clientpb.UpdateClientResponse{
		Data: []*clientpb.Client{convertedClient},
	}, nil
}

// DeleteClient deletes a client using common Firestore operations
func (r *FirestoreClientRepository) DeleteClient(ctx context.Context, req *clientpb.DeleteClientRequest) (*clientpb.DeleteClientResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete client: %w", err)
	}

	return &clientpb.DeleteClientResponse{
		Success: true,
	}, nil
}

// ListClients lists clients using common Firestore operations
func (r *FirestoreClientRepository) ListClients(ctx context.Context, req *clientpb.ListClientsRequest) (*clientpb.ListClientsResponse, error) {
	// Log the collection name being queried
	fmt.Printf("📋 ListClients: Querying Firestore collection '%s'\n", r.collectionName)

	// Build ListParams from request - pass filters directly to dbOps.List
	listParams := &interfaces.ListParams{
		Search:     req.Search,
		Filters:    req.Filters,
		Sort:       req.Sort,
		Pagination: req.Pagination,
	}

	fmt.Printf("📋 ListClients: Filters applied: %+v\n", req.Filters)

	// List documents using common operations with proper filter support
	listResult, err := r.dbOps.List(ctx, r.collectionName, listParams)
	if err != nil {
		fmt.Printf("❌ ListClients: Failed to query collection '%s': %v\n", r.collectionName, err)
		return nil, fmt.Errorf("failed to list clients: %w", err)
	}

	fmt.Printf("✅ ListClients: Retrieved %d documents from collection '%s'\n", len(listResult.Data), r.collectionName)

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	clients, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *clientpb.Client {
		return &clientpb.Client{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// CRITICAL FIX: Ensure we always return a non-nil slice for proper JSON marshaling
	// This guarantees the "data" field is always included in the JSON response
	if clients == nil {
		clients = make([]*clientpb.Client, 0)
	}

	return &clientpb.ListClientsResponse{
		Data:    clients,
		Success: true,
	}, nil
}

package entity

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	firestoreCore "github.com/erniealice/espyna-golang/contrib/google/internal/database/firestore/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/shared/database/operations"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	sessionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/session"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", entityid.Session, func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore session repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreSessionRepository(dbOps, collectionName), nil
	})
}

// FirestoreSessionRepository implements session CRUD operations using Firestore
type FirestoreSessionRepository struct {
	sessionpb.UnimplementedSessionDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreSessionRepository creates a new Firestore session repository
func NewFirestoreSessionRepository(dbOps interfaces.DatabaseOperation, collectionName string) sessionpb.SessionDomainServiceServer {
	if collectionName == "" {
		collectionName = "session" // default fallback
	}
	return &FirestoreSessionRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateSession creates a new session using common Firestore operations
func (r *FirestoreSessionRepository) CreateSession(ctx context.Context, req *sessionpb.CreateSessionRequest) (*sessionpb.CreateSessionResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("session data is required")
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
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	convertedSession, err := operations.ConvertMapToProtobuf(result, &sessionpb.Session{})
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &sessionpb.CreateSessionResponse{
		Data: []*sessionpb.Session{convertedSession},
	}, nil
}

// ReadSession retrieves a session using common Firestore operations
func (r *FirestoreSessionRepository) ReadSession(ctx context.Context, req *sessionpb.ReadSessionRequest) (*sessionpb.ReadSessionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("session ID is required")
	}

	// Use transaction-aware database operations
	txAwareDbOps := r.dbOps

	// Read document using common operations (automatically transaction-aware)
	result, err := txAwareDbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read session: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	convertedSession, err := operations.ConvertMapToProtobuf(result, &sessionpb.Session{})
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &sessionpb.ReadSessionResponse{
		Data: []*sessionpb.Session{convertedSession},
	}, nil
}

// UpdateSession updates a session using common Firestore operations
func (r *FirestoreSessionRepository) UpdateSession(ctx context.Context, req *sessionpb.UpdateSessionRequest) (*sessionpb.UpdateSessionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("session ID is required")
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
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	convertedSession, err := operations.ConvertMapToProtobuf(result, &sessionpb.Session{})
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &sessionpb.UpdateSessionResponse{
		Data: []*sessionpb.Session{convertedSession},
	}, nil
}

// DeleteSession deletes a session using common Firestore operations
func (r *FirestoreSessionRepository) DeleteSession(ctx context.Context, req *sessionpb.DeleteSessionRequest) (*sessionpb.DeleteSessionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("session ID is required")
	}

	// Use transaction-aware database operations
	txAwareDbOps := r.dbOps

	// Delete document using common operations (soft delete, automatically transaction-aware)
	err := txAwareDbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete session: %w", err)
	}

	return &sessionpb.DeleteSessionResponse{
		Success: true,
	}, nil
}

// ListSessions lists sessions using common Firestore operations
func (r *FirestoreSessionRepository) ListSessions(ctx context.Context, req *sessionpb.ListSessionsRequest) (*sessionpb.ListSessionsResponse, error) {
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
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	sessions, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *sessionpb.Session {
		return &sessionpb.Session{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if sessions == nil {
		sessions = make([]*sessionpb.Session, 0)
	}

	return &sessionpb.ListSessionsResponse{
		Data: sessions,
	}, nil
}

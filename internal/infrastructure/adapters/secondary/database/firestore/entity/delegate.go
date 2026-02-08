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
	delegatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "delegate", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore delegate repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreDelegateRepository(dbOps, collectionName), nil
	})
}

// FirestoreDelegateRepository implements delegate CRUD operations using Firestore
type FirestoreDelegateRepository struct {
	delegatepb.UnimplementedDelegateDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreDelegateRepository creates a new Firestore delegate repository
func NewFirestoreDelegateRepository(dbOps interfaces.DatabaseOperation, collectionName string) delegatepb.DelegateDomainServiceServer {
	if collectionName == "" {
		collectionName = "delegate" // default fallback
	}
	return &FirestoreDelegateRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateDelegate creates a new delegate using common Firestore operations
func (r *FirestoreDelegateRepository) CreateDelegate(ctx context.Context, req *delegatepb.CreateDelegateRequest) (*delegatepb.CreateDelegateResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("delegate data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create delegate: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	delegate := &delegatepb.Delegate{}
	convertedDelegate, err := operations.ConvertMapToProtobuf(result, delegate)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &delegatepb.CreateDelegateResponse{
		Data: []*delegatepb.Delegate{convertedDelegate},
	}, nil
}

// ReadDelegate retrieves a delegate using common Firestore operations
func (r *FirestoreDelegateRepository) ReadDelegate(ctx context.Context, req *delegatepb.ReadDelegateRequest) (*delegatepb.ReadDelegateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("delegate ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read delegate: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	delegate := &delegatepb.Delegate{}
	convertedDelegate, err := operations.ConvertMapToProtobuf(result, delegate)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &delegatepb.ReadDelegateResponse{
		Data: []*delegatepb.Delegate{convertedDelegate},
	}, nil
}

// UpdateDelegate updates a delegate using common Firestore operations
func (r *FirestoreDelegateRepository) UpdateDelegate(ctx context.Context, req *delegatepb.UpdateDelegateRequest) (*delegatepb.UpdateDelegateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("delegate ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update delegate: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	delegate := &delegatepb.Delegate{}
	convertedDelegate, err := operations.ConvertMapToProtobuf(result, delegate)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &delegatepb.UpdateDelegateResponse{
		Data: []*delegatepb.Delegate{convertedDelegate},
	}, nil
}

// DeleteDelegate deletes a delegate using common Firestore operations
func (r *FirestoreDelegateRepository) DeleteDelegate(ctx context.Context, req *delegatepb.DeleteDelegateRequest) (*delegatepb.DeleteDelegateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("delegate ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete delegate: %w", err)
	}

	return &delegatepb.DeleteDelegateResponse{
		Success: true,
	}, nil
}

// ListDelegates lists delegates using common Firestore operations
func (r *FirestoreDelegateRepository) ListDelegates(ctx context.Context, req *delegatepb.ListDelegatesRequest) (*delegatepb.ListDelegatesResponse, error) {
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
		return nil, fmt.Errorf("failed to list delegates: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	delegates, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *delegatepb.Delegate {
		return &delegatepb.Delegate{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if delegates == nil {
		delegates = make([]*delegatepb.Delegate, 0)
	}

	return &delegatepb.ListDelegatesResponse{
		Data: delegates,
	}, nil
}

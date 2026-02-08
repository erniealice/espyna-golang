//go:build firestore

package subscription

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/operations"
	firestoreCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/firestore/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	balancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "balance", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore balance repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreBalanceRepository(dbOps, collectionName), nil
	})
}

// FirestoreBalanceRepository implements balance CRUD operations using Firestore
type FirestoreBalanceRepository struct {
	balancepb.UnimplementedBalanceDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreBalanceRepository creates a new Firestore balance repository
func NewFirestoreBalanceRepository(dbOps interfaces.DatabaseOperation, collectionName string) balancepb.BalanceDomainServiceServer {
	if collectionName == "" {
		collectionName = "balance" // default fallback
	}
	return &FirestoreBalanceRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateBalance creates a new balance using common Firestore operations
func (r *FirestoreBalanceRepository) CreateBalance(ctx context.Context, req *balancepb.CreateBalanceRequest) (*balancepb.CreateBalanceResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("balance data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create balance: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	balance := &balancepb.Balance{}
	convertedBalance, err := operations.ConvertMapToProtobuf(result, balance)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &balancepb.CreateBalanceResponse{
		Data: []*balancepb.Balance{convertedBalance},
	}, nil
}

// ReadBalance retrieves a balance using common Firestore operations
func (r *FirestoreBalanceRepository) ReadBalance(ctx context.Context, req *balancepb.ReadBalanceRequest) (*balancepb.ReadBalanceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("balance ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read balance: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	balance := &balancepb.Balance{}
	convertedBalance, err := operations.ConvertMapToProtobuf(result, balance)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &balancepb.ReadBalanceResponse{
		Data: []*balancepb.Balance{convertedBalance},
	}, nil
}

// UpdateBalance updates a balance using common Firestore operations
func (r *FirestoreBalanceRepository) UpdateBalance(ctx context.Context, req *balancepb.UpdateBalanceRequest) (*balancepb.UpdateBalanceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("balance ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update balance: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	balance := &balancepb.Balance{}
	convertedBalance, err := operations.ConvertMapToProtobuf(result, balance)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &balancepb.UpdateBalanceResponse{
		Data: []*balancepb.Balance{convertedBalance},
	}, nil
}

// DeleteBalance deletes a balance using common Firestore operations
func (r *FirestoreBalanceRepository) DeleteBalance(ctx context.Context, req *balancepb.DeleteBalanceRequest) (*balancepb.DeleteBalanceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("balance ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete balance: %w", err)
	}

	return &balancepb.DeleteBalanceResponse{
		Success: true,
	}, nil
}

// ListBalances lists balances using common Firestore operations
func (r *FirestoreBalanceRepository) ListBalances(ctx context.Context, req *balancepb.ListBalancesRequest) (*balancepb.ListBalancesResponse, error) {
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
		return nil, fmt.Errorf("failed to list balances: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	balances, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *balancepb.Balance {
		return &balancepb.Balance{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if balances == nil {
		balances = make([]*balancepb.Balance, 0)
	}

	return &balancepb.ListBalancesResponse{
		Data: balances,
	}, nil
}

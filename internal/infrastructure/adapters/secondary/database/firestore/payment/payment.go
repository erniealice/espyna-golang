//go:build firestore

package payment

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/operations"
	firestoreCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/firestore/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"

	paymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "payment", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore payment repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestorePaymentRepository(dbOps, collectionName), nil
	})
}

// FirestorePaymentRepository implements payment CRUD operations using Firestore
type FirestorePaymentRepository struct {
	paymentpb.UnimplementedPaymentDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestorePaymentRepository creates a new Firestore payment repository
func NewFirestorePaymentRepository(dbOps interfaces.DatabaseOperation, collectionName string) paymentpb.PaymentDomainServiceServer {
	if collectionName == "" {
		collectionName = "payment" // default fallback
	}
	return &FirestorePaymentRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreatePayment creates a new payment using common Firestore operations
func (r *FirestorePaymentRepository) CreatePayment(ctx context.Context, req *paymentpb.CreatePaymentRequest) (*paymentpb.CreatePaymentResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("payment data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	payment := &paymentpb.Payment{}
	convertedPayment, err := operations.ConvertMapToProtobuf(result, payment)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &paymentpb.CreatePaymentResponse{
		Data: []*paymentpb.Payment{convertedPayment},
	}, nil
}

// ReadPayment retrieves a payment using common Firestore operations
func (r *FirestorePaymentRepository) ReadPayment(ctx context.Context, req *paymentpb.ReadPaymentRequest) (*paymentpb.ReadPaymentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payment ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read payment: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	payment := &paymentpb.Payment{}
	convertedPayment, err := operations.ConvertMapToProtobuf(result, payment)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &paymentpb.ReadPaymentResponse{
		Data: []*paymentpb.Payment{convertedPayment},
	}, nil
}

// UpdatePayment updates a payment using common Firestore operations
func (r *FirestorePaymentRepository) UpdatePayment(ctx context.Context, req *paymentpb.UpdatePaymentRequest) (*paymentpb.UpdatePaymentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payment ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update payment: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	payment := &paymentpb.Payment{}
	convertedPayment, err := operations.ConvertMapToProtobuf(result, payment)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &paymentpb.UpdatePaymentResponse{
		Data: []*paymentpb.Payment{convertedPayment},
	}, nil
}

// DeletePayment deletes a payment using common Firestore operations
func (r *FirestorePaymentRepository) DeletePayment(ctx context.Context, req *paymentpb.DeletePaymentRequest) (*paymentpb.DeletePaymentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payment ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete payment: %w", err)
	}

	return &paymentpb.DeletePaymentResponse{
		Success: true,
	}, nil
}

// ListPayments lists payments using common Firestore operations
func (r *FirestorePaymentRepository) ListPayments(ctx context.Context, req *paymentpb.ListPaymentsRequest) (*paymentpb.ListPaymentsResponse, error) {
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
		return nil, fmt.Errorf("failed to list payments: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	payments, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *paymentpb.Payment {
		return &paymentpb.Payment{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if payments == nil {
		payments = make([]*paymentpb.Payment, 0)
	}

	return &paymentpb.ListPaymentsResponse{
		Data: payments,
	}, nil
}

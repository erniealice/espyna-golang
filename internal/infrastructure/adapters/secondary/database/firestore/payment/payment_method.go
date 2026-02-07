//go:build firestore

package payment

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	interfaces "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/interface"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/operations"
	firestoreCore "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/firestore/core"
	"leapfor.xyz/espyna/internal/infrastructure/registry"

	paymentmethodpb "leapfor.xyz/esqyma/golang/v1/domain/payment/payment_method"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "payment_method", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore payment_method repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestorePaymentMethodRepository(dbOps, collectionName), nil
	})
}

// FirestorePaymentMethodRepository implements payment method CRUD operations using Firestore
type FirestorePaymentMethodRepository struct {
	paymentmethodpb.UnimplementedPaymentMethodDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestorePaymentMethodRepository creates a new Firestore payment method repository
func NewFirestorePaymentMethodRepository(dbOps interfaces.DatabaseOperation, collectionName string) paymentmethodpb.PaymentMethodDomainServiceServer {
	if collectionName == "" {
		collectionName = "payment_method" // default fallback
	}
	return &FirestorePaymentMethodRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreatePaymentMethod creates a new payment method using common Firestore operations
func (r *FirestorePaymentMethodRepository) CreatePaymentMethod(ctx context.Context, req *paymentmethodpb.CreatePaymentMethodRequest) (*paymentmethodpb.CreatePaymentMethodResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("payment method data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment method: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	paymentMethod := &paymentmethodpb.PaymentMethod{}
	convertedPaymentMethod, err := operations.ConvertMapToProtobuf(result, paymentMethod)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &paymentmethodpb.CreatePaymentMethodResponse{
		Data: []*paymentmethodpb.PaymentMethod{convertedPaymentMethod},
	}, nil
}

// ReadPaymentMethod retrieves a payment method using common Firestore operations
func (r *FirestorePaymentMethodRepository) ReadPaymentMethod(ctx context.Context, req *paymentmethodpb.ReadPaymentMethodRequest) (*paymentmethodpb.ReadPaymentMethodResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payment method ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read payment method: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	paymentMethod := &paymentmethodpb.PaymentMethod{}
	convertedPaymentMethod, err := operations.ConvertMapToProtobuf(result, paymentMethod)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &paymentmethodpb.ReadPaymentMethodResponse{
		Data: []*paymentmethodpb.PaymentMethod{convertedPaymentMethod},
	}, nil
}

// UpdatePaymentMethod updates a payment method using common Firestore operations
func (r *FirestorePaymentMethodRepository) UpdatePaymentMethod(ctx context.Context, req *paymentmethodpb.UpdatePaymentMethodRequest) (*paymentmethodpb.UpdatePaymentMethodResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payment method ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update payment method: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	paymentMethod := &paymentmethodpb.PaymentMethod{}
	convertedPaymentMethod, err := operations.ConvertMapToProtobuf(result, paymentMethod)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &paymentmethodpb.UpdatePaymentMethodResponse{
		Data: []*paymentmethodpb.PaymentMethod{convertedPaymentMethod},
	}, nil
}

// DeletePaymentMethod deletes a payment method using common Firestore operations
func (r *FirestorePaymentMethodRepository) DeletePaymentMethod(ctx context.Context, req *paymentmethodpb.DeletePaymentMethodRequest) (*paymentmethodpb.DeletePaymentMethodResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payment method ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete payment method: %w", err)
	}

	return &paymentmethodpb.DeletePaymentMethodResponse{
		Success: true,
	}, nil
}

// ListPaymentMethods lists payment methods using common Firestore operations
func (r *FirestorePaymentMethodRepository) ListPaymentMethods(ctx context.Context, req *paymentmethodpb.ListPaymentMethodsRequest) (*paymentmethodpb.ListPaymentMethodsResponse, error) {
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
		return nil, fmt.Errorf("failed to list payment methods: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	paymentMethods, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *paymentmethodpb.PaymentMethod {
		return &paymentmethodpb.PaymentMethod{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if paymentMethods == nil {
		paymentMethods = make([]*paymentmethodpb.PaymentMethod, 0)
	}

	return &paymentmethodpb.ListPaymentMethodsResponse{
		Data: paymentMethods,
	}, nil
}

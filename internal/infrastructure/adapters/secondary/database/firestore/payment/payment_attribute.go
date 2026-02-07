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
	paymentattributepb "leapfor.xyz/esqyma/golang/v1/domain/payment/payment_attribute"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "payment_attribute", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore payment_attribute repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestorePaymentAttributeRepository(dbOps, collectionName), nil
	})
}

// FirestorePaymentAttributeRepository implements payment attribute CRUD operations using Firestore
type FirestorePaymentAttributeRepository struct {
	paymentattributepb.UnimplementedPaymentAttributeDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestorePaymentAttributeRepository creates a new Firestore payment attribute repository
func NewFirestorePaymentAttributeRepository(dbOps interfaces.DatabaseOperation, collectionName string) paymentattributepb.PaymentAttributeDomainServiceServer {
	if collectionName == "" {
		collectionName = "payment_attribute" // default fallback
	}
	return &FirestorePaymentAttributeRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreatePaymentAttribute creates a new payment attribute using common Firestore operations
func (r *FirestorePaymentAttributeRepository) CreatePaymentAttribute(ctx context.Context, req *paymentattributepb.CreatePaymentAttributeRequest) (*paymentattributepb.CreatePaymentAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("payment attribute data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment attribute: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	paymentAttribute := &paymentattributepb.PaymentAttribute{}
	convertedPaymentAttribute, err := operations.ConvertMapToProtobuf(result, paymentAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &paymentattributepb.CreatePaymentAttributeResponse{
		Data: []*paymentattributepb.PaymentAttribute{convertedPaymentAttribute},
	}, nil
}

// ReadPaymentAttribute retrieves a payment attribute using common Firestore operations
func (r *FirestorePaymentAttributeRepository) ReadPaymentAttribute(ctx context.Context, req *paymentattributepb.ReadPaymentAttributeRequest) (*paymentattributepb.ReadPaymentAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payment attribute ID is required")
	}

	// Read document using common operations with the ID
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read payment attribute: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	paymentAttribute := &paymentattributepb.PaymentAttribute{}
	convertedPaymentAttribute, err := operations.ConvertMapToProtobuf(result, paymentAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &paymentattributepb.ReadPaymentAttributeResponse{
		Data: []*paymentattributepb.PaymentAttribute{convertedPaymentAttribute},
	}, nil
}

// UpdatePaymentAttribute updates a payment attribute using common Firestore operations
func (r *FirestorePaymentAttributeRepository) UpdatePaymentAttribute(ctx context.Context, req *paymentattributepb.UpdatePaymentAttributeRequest) (*paymentattributepb.UpdatePaymentAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payment attribute ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update payment attribute: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	paymentAttribute := &paymentattributepb.PaymentAttribute{}
	convertedPaymentAttribute, err := operations.ConvertMapToProtobuf(result, paymentAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &paymentattributepb.UpdatePaymentAttributeResponse{
		Data: []*paymentattributepb.PaymentAttribute{convertedPaymentAttribute},
	}, nil
}

// DeletePaymentAttribute deletes a payment attribute using common Firestore operations
func (r *FirestorePaymentAttributeRepository) DeletePaymentAttribute(ctx context.Context, req *paymentattributepb.DeletePaymentAttributeRequest) (*paymentattributepb.DeletePaymentAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payment attribute ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete payment attribute: %w", err)
	}

	return &paymentattributepb.DeletePaymentAttributeResponse{
		Success: true,
	}, nil
}

// ListPaymentAttributes lists payment attributes using common Firestore operations
func (r *FirestorePaymentAttributeRepository) ListPaymentAttributes(ctx context.Context, req *paymentattributepb.ListPaymentAttributesRequest) (*paymentattributepb.ListPaymentAttributesResponse, error) {
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
		return nil, fmt.Errorf("failed to list payment attributes: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	paymentAttributes, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *paymentattributepb.PaymentAttribute {
		return &paymentattributepb.PaymentAttribute{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if paymentAttributes == nil {
		paymentAttributes = make([]*paymentattributepb.PaymentAttribute, 0)
	}

	return &paymentattributepb.ListPaymentAttributesResponse{
		Data: paymentAttributes,
	}, nil
}
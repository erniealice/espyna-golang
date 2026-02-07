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

	paymentprofilepb "leapfor.xyz/esqyma/golang/v1/domain/payment/payment_profile"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "payment_profile", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore payment_profile repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestorePaymentProfileRepository(dbOps, collectionName), nil
	})
}

// FirestorePaymentProfileRepository implements payment profile CRUD operations using Firestore
type FirestorePaymentProfileRepository struct {
	paymentprofilepb.UnimplementedPaymentProfileDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestorePaymentProfileRepository creates a new Firestore payment profile repository
func NewFirestorePaymentProfileRepository(dbOps interfaces.DatabaseOperation, collectionName string) paymentprofilepb.PaymentProfileDomainServiceServer {
	if collectionName == "" {
		collectionName = "payment_profile" // default fallback
	}
	return &FirestorePaymentProfileRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreatePaymentProfile creates a new payment profile using common Firestore operations
func (r *FirestorePaymentProfileRepository) CreatePaymentProfile(ctx context.Context, req *paymentprofilepb.CreatePaymentProfileRequest) (*paymentprofilepb.CreatePaymentProfileResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("payment profile data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment profile: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	paymentProfile := &paymentprofilepb.PaymentProfile{}
	convertedPaymentProfile, err := operations.ConvertMapToProtobuf(result, paymentProfile)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &paymentprofilepb.CreatePaymentProfileResponse{
		Data: []*paymentprofilepb.PaymentProfile{convertedPaymentProfile},
	}, nil
}

// ReadPaymentProfile retrieves a payment profile using common Firestore operations
func (r *FirestorePaymentProfileRepository) ReadPaymentProfile(ctx context.Context, req *paymentprofilepb.ReadPaymentProfileRequest) (*paymentprofilepb.ReadPaymentProfileResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payment profile ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read payment profile: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	paymentProfile := &paymentprofilepb.PaymentProfile{}
	convertedPaymentProfile, err := operations.ConvertMapToProtobuf(result, paymentProfile)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &paymentprofilepb.ReadPaymentProfileResponse{
		Data: []*paymentprofilepb.PaymentProfile{convertedPaymentProfile},
	}, nil
}

// UpdatePaymentProfile updates a payment profile using common Firestore operations
func (r *FirestorePaymentProfileRepository) UpdatePaymentProfile(ctx context.Context, req *paymentprofilepb.UpdatePaymentProfileRequest) (*paymentprofilepb.UpdatePaymentProfileResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payment profile ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update payment profile: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	paymentProfile := &paymentprofilepb.PaymentProfile{}
	convertedPaymentProfile, err := operations.ConvertMapToProtobuf(result, paymentProfile)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &paymentprofilepb.UpdatePaymentProfileResponse{
		Data: []*paymentprofilepb.PaymentProfile{convertedPaymentProfile},
	}, nil
}

// DeletePaymentProfile deletes a payment profile using common Firestore operations
func (r *FirestorePaymentProfileRepository) DeletePaymentProfile(ctx context.Context, req *paymentprofilepb.DeletePaymentProfileRequest) (*paymentprofilepb.DeletePaymentProfileResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payment profile ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete payment profile: %w", err)
	}

	return &paymentprofilepb.DeletePaymentProfileResponse{
		Success: true,
	}, nil
}

// ListPaymentProfiles lists payment profiles using common Firestore operations
func (r *FirestorePaymentProfileRepository) ListPaymentProfiles(ctx context.Context, req *paymentprofilepb.ListPaymentProfilesRequest) (*paymentprofilepb.ListPaymentProfilesResponse, error) {
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
		return nil, fmt.Errorf("failed to list payment profiles: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	paymentProfiles, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *paymentprofilepb.PaymentProfile {
		return &paymentprofilepb.PaymentProfile{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if paymentProfiles == nil {
		paymentProfiles = make([]*paymentprofilepb.PaymentProfile, 0)
	}

	return &paymentprofilepb.ListPaymentProfilesResponse{
		Data: paymentProfiles,
	}, nil
}

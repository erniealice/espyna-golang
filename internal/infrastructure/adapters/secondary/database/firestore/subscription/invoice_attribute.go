//go:build firestore

package subscription

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	interfaces "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/interface"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/operations"
	firestoreCore "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/firestore/core"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	invoiceattributepb "leapfor.xyz/esqyma/golang/v1/domain/subscription/invoice_attribute"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "invoice_attribute", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore invoice_attribute repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreInvoiceAttributeRepository(dbOps, collectionName), nil
	})
}

// FirestoreInvoiceAttributeRepository implements invoice attribute CRUD operations using Firestore
type FirestoreInvoiceAttributeRepository struct {
	invoiceattributepb.UnimplementedInvoiceAttributeDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreInvoiceAttributeRepository creates a new Firestore invoice attribute repository
func NewFirestoreInvoiceAttributeRepository(dbOps interfaces.DatabaseOperation, collectionName string) invoiceattributepb.InvoiceAttributeDomainServiceServer {
	if collectionName == "" {
		collectionName = "invoice_attribute" // default fallback
	}
	return &FirestoreInvoiceAttributeRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateInvoiceAttribute creates a new invoice attribute using common Firestore operations
func (r *FirestoreInvoiceAttributeRepository) CreateInvoiceAttribute(ctx context.Context, req *invoiceattributepb.CreateInvoiceAttributeRequest) (*invoiceattributepb.CreateInvoiceAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("invoice attribute data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create invoice attribute: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	invoiceAttribute := &invoiceattributepb.InvoiceAttribute{}
	convertedInvoiceAttribute, err := operations.ConvertMapToProtobuf(result, invoiceAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &invoiceattributepb.CreateInvoiceAttributeResponse{
		Data: []*invoiceattributepb.InvoiceAttribute{convertedInvoiceAttribute},
	}, nil
}

// ReadInvoiceAttribute retrieves a invoice attribute using common Firestore operations
func (r *FirestoreInvoiceAttributeRepository) ReadInvoiceAttribute(ctx context.Context, req *invoiceattributepb.ReadInvoiceAttributeRequest) (*invoiceattributepb.ReadInvoiceAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("invoice attribute ID is required")
	}

	// Read document using common operations with the ID
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read invoice attribute: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	invoiceAttribute := &invoiceattributepb.InvoiceAttribute{}
	convertedInvoiceAttribute, err := operations.ConvertMapToProtobuf(result, invoiceAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &invoiceattributepb.ReadInvoiceAttributeResponse{
		Data: []*invoiceattributepb.InvoiceAttribute{convertedInvoiceAttribute},
	}, nil
}

// UpdateInvoiceAttribute updates a invoice attribute using common Firestore operations
func (r *FirestoreInvoiceAttributeRepository) UpdateInvoiceAttribute(ctx context.Context, req *invoiceattributepb.UpdateInvoiceAttributeRequest) (*invoiceattributepb.UpdateInvoiceAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("invoice attribute ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update invoice attribute: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	invoiceAttribute := &invoiceattributepb.InvoiceAttribute{}
	convertedInvoiceAttribute, err := operations.ConvertMapToProtobuf(result, invoiceAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &invoiceattributepb.UpdateInvoiceAttributeResponse{
		Data: []*invoiceattributepb.InvoiceAttribute{convertedInvoiceAttribute},
	}, nil
}

// DeleteInvoiceAttribute deletes a invoice attribute using common Firestore operations
func (r *FirestoreInvoiceAttributeRepository) DeleteInvoiceAttribute(ctx context.Context, req *invoiceattributepb.DeleteInvoiceAttributeRequest) (*invoiceattributepb.DeleteInvoiceAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("invoice attribute ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete invoice attribute: %w", err)
	}

	return &invoiceattributepb.DeleteInvoiceAttributeResponse{
		Success: true,
	}, nil
}

// ListInvoiceAttributes lists invoice attributes using common Firestore operations
func (r *FirestoreInvoiceAttributeRepository) ListInvoiceAttributes(ctx context.Context, req *invoiceattributepb.ListInvoiceAttributesRequest) (*invoiceattributepb.ListInvoiceAttributesResponse, error) {
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
		return nil, fmt.Errorf("failed to list invoice attributes: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	invoiceAttributes, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *invoiceattributepb.InvoiceAttribute {
		return &invoiceattributepb.InvoiceAttribute{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if invoiceAttributes == nil {
		invoiceAttributes = make([]*invoiceattributepb.InvoiceAttribute, 0)
	}

	return &invoiceattributepb.ListInvoiceAttributesResponse{
		Data: invoiceAttributes,
	}, nil
}
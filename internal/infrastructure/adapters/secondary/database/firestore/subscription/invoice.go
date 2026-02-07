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
	invoicepb "leapfor.xyz/esqyma/golang/v1/domain/subscription/invoice"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "invoice", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore invoice repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreInvoiceRepository(dbOps, collectionName), nil
	})
}

// FirestoreInvoiceRepository implements invoice CRUD operations using Firestore
type FirestoreInvoiceRepository struct {
	invoicepb.UnimplementedInvoiceDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreInvoiceRepository creates a new Firestore invoice repository
func NewFirestoreInvoiceRepository(dbOps interfaces.DatabaseOperation, collectionName string) invoicepb.InvoiceDomainServiceServer {
	if collectionName == "" {
		collectionName = "invoice" // default fallback
	}
	return &FirestoreInvoiceRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateInvoice creates a new invoice using common Firestore operations
func (r *FirestoreInvoiceRepository) CreateInvoice(ctx context.Context, req *invoicepb.CreateInvoiceRequest) (*invoicepb.CreateInvoiceResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("invoice data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	invoice := &invoicepb.Invoice{}
	convertedInvoice, err := operations.ConvertMapToProtobuf(result, invoice)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &invoicepb.CreateInvoiceResponse{
		Data: []*invoicepb.Invoice{convertedInvoice},
	}, nil
}

// ReadInvoice retrieves an invoice using common Firestore operations
func (r *FirestoreInvoiceRepository) ReadInvoice(ctx context.Context, req *invoicepb.ReadInvoiceRequest) (*invoicepb.ReadInvoiceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("invoice ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read invoice: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	invoice := &invoicepb.Invoice{}
	convertedInvoice, err := operations.ConvertMapToProtobuf(result, invoice)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &invoicepb.ReadInvoiceResponse{
		Data: []*invoicepb.Invoice{convertedInvoice},
	}, nil
}

// UpdateInvoice updates an invoice using common Firestore operations
func (r *FirestoreInvoiceRepository) UpdateInvoice(ctx context.Context, req *invoicepb.UpdateInvoiceRequest) (*invoicepb.UpdateInvoiceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("invoice ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update invoice: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	invoice := &invoicepb.Invoice{}
	convertedInvoice, err := operations.ConvertMapToProtobuf(result, invoice)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &invoicepb.UpdateInvoiceResponse{
		Data: []*invoicepb.Invoice{convertedInvoice},
	}, nil
}

// DeleteInvoice deletes an invoice using common Firestore operations
func (r *FirestoreInvoiceRepository) DeleteInvoice(ctx context.Context, req *invoicepb.DeleteInvoiceRequest) (*invoicepb.DeleteInvoiceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("invoice ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete invoice: %w", err)
	}

	return &invoicepb.DeleteInvoiceResponse{
		Success: true,
	}, nil
}

// ListInvoices lists invoices using common Firestore operations
func (r *FirestoreInvoiceRepository) ListInvoices(ctx context.Context, req *invoicepb.ListInvoicesRequest) (*invoicepb.ListInvoicesResponse, error) {
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
		return nil, fmt.Errorf("failed to list invoices: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	invoices, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *invoicepb.Invoice {
		return &invoicepb.Invoice{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if invoices == nil {
		invoices = make([]*invoicepb.Invoice, 0)
	}

	return &invoicepb.ListInvoicesResponse{
		Data: invoices,
	}, nil
}

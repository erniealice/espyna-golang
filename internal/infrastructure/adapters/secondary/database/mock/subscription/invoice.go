//go:build mock_db

package subscription

import (
	"context"
	"fmt"
	"sync"
	"time"

	datamock "leapfor.xyz/copya/golang"
	"leapfor.xyz/espyna/internal/application/shared/listdata"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	invoicepb "leapfor.xyz/esqyma/golang/v1/domain/subscription/invoice"
)

func init() {
	registry.RegisterRepositoryFactory("mock", "invoice", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockInvoiceRepository(businessType), nil
	})
}

// MockInvoiceRepository implements subscription.InvoiceRepository using stateful mock data
type MockInvoiceRepository struct {
	invoicepb.UnimplementedInvoiceDomainServiceServer
	businessType string
	invoices     map[string]*invoicepb.Invoice // Persistent in-memory store
	mutex        sync.RWMutex                  // Thread-safe concurrent access
	initialized  bool                          // Prevent double initialization
	processor    *listdata.ListDataProcessor   // List data processing utilities
}

// InvoiceRepositoryOption allows configuration of repository behavior
type InvoiceRepositoryOption func(*MockInvoiceRepository)

// WithInvoiceTestOptimizations enables test-specific optimizations
func WithInvoiceTestOptimizations(enabled bool) InvoiceRepositoryOption {
	return func(r *MockInvoiceRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockInvoiceRepository creates a new mock invoice repository
func NewMockInvoiceRepository(businessType string, options ...InvoiceRepositoryOption) invoicepb.InvoiceDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockInvoiceRepository{
		businessType: businessType,
		invoices:     make(map[string]*invoicepb.Invoice),
		processor:    listdata.NewListDataProcessor(),
	}

	// Apply optional configurations
	for _, option := range options {
		option(repo)
	}

	// Initialize with mock data once
	if err := repo.loadInitialData(); err != nil {
		// Log error but don't fail - allows graceful degradation
		fmt.Printf("Warning: Failed to load initial mock data: %v\n", err)
	}

	return repo
}

// loadInitialData loads mock data from the copya package
func (r *MockInvoiceRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawInvoices, err := datamock.LoadBusinessTypeModule(r.businessType, "invoice")
	if err != nil {
		return fmt.Errorf("failed to load initial invoices: %w", err)
	}

	// Convert and store each invoice
	for _, rawInvoice := range rawInvoices {
		if invoice, err := r.mapToProtobufInvoice(rawInvoice); err == nil {
			r.invoices[invoice.Id] = invoice
		}
	}

	r.initialized = true
	return nil
}

// CreateInvoice creates a new invoice with stateful storage
func (r *MockInvoiceRepository) CreateInvoice(ctx context.Context, req *invoicepb.CreateInvoiceRequest) (*invoicepb.CreateInvoiceResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create invoice request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("invoice data is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate unique ID with timestamp
	now := time.Now()
	invoiceID := fmt.Sprintf("invoice-%d-%d", now.UnixNano(), len(r.invoices))
	invoiceNumber := fmt.Sprintf("INV-%d", now.Unix())

	// Create new invoice with proper timestamps and defaults
	newInvoice := &invoicepb.Invoice{
		Id:                 invoiceID,
		InvoiceNumber:      invoiceNumber,
		Amount:             req.Data.Amount,
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             true, // Default to active
	}

	// Store in persistent map
	r.invoices[invoiceID] = newInvoice

	return &invoicepb.CreateInvoiceResponse{
		Data:    []*invoicepb.Invoice{newInvoice},
		Success: true,
	}, nil
}

// ReadInvoice retrieves an invoice by ID from stateful storage
func (r *MockInvoiceRepository) ReadInvoice(ctx context.Context, req *invoicepb.ReadInvoiceRequest) (*invoicepb.ReadInvoiceResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read invoice request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("invoice ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store first (includes created/updated invoices)
	if invoice, exists := r.invoices[req.Data.Id]; exists {
		return &invoicepb.ReadInvoiceResponse{
			Data:    []*invoicepb.Invoice{invoice},
			Success: true,
		}, nil
	}

	return nil, fmt.Errorf("invoice with ID '%s' not found", req.Data.Id)
}

// UpdateInvoice updates an existing invoice in stateful storage
func (r *MockInvoiceRepository) UpdateInvoice(ctx context.Context, req *invoicepb.UpdateInvoiceRequest) (*invoicepb.UpdateInvoiceResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update invoice request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("invoice ID is required for update")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify invoice exists
	existingInvoice, exists := r.invoices[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("invoice with ID '%s' not found", req.Data.Id)
	}

	// Update only specified fields (preserve others)
	now := time.Now()
	updatedInvoice := &invoicepb.Invoice{
		Id:                 req.Data.Id,
		InvoiceNumber:      req.Data.InvoiceNumber,
		Amount:             req.Data.Amount,
		DateCreated:        existingInvoice.DateCreated,       // Preserve original
		DateCreatedString:  existingInvoice.DateCreatedString, // Preserve original
		DateModified:       &[]int64{now.UnixMilli()}[0],           // Update modification time
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             req.Data.Active,
	}

	// Update in persistent store
	r.invoices[req.Data.Id] = updatedInvoice

	return &invoicepb.UpdateInvoiceResponse{
		Data:    []*invoicepb.Invoice{updatedInvoice},
		Success: true,
	}, nil
}

// DeleteInvoice deletes an invoice from stateful storage
func (r *MockInvoiceRepository) DeleteInvoice(ctx context.Context, req *invoicepb.DeleteInvoiceRequest) (*invoicepb.DeleteInvoiceResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete invoice request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("invoice ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify invoice exists before deletion
	if _, exists := r.invoices[req.Data.Id]; !exists {
		return nil, fmt.Errorf("invoice with ID '%s' not found", req.Data.Id)
	}

	// Perform actual deletion from persistent store
	delete(r.invoices, req.Data.Id)

	return &invoicepb.DeleteInvoiceResponse{
		Success: true,
	}, nil
}

// ListInvoices retrieves all invoices from stateful storage
func (r *MockInvoiceRepository) ListInvoices(ctx context.Context, req *invoicepb.ListInvoicesRequest) (*invoicepb.ListInvoicesResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of invoices
	invoices := make([]*invoicepb.Invoice, 0, len(r.invoices))
	for _, invoice := range r.invoices {
		invoices = append(invoices, invoice)
	}

	return &invoicepb.ListInvoicesResponse{
		Data:    invoices,
		Success: true,
	}, nil
}

// GetInvoiceListPageData retrieves invoices with advanced filtering, sorting, searching, and pagination
func (r *MockInvoiceRepository) GetInvoiceListPageData(
	ctx context.Context,
	req *invoicepb.GetInvoiceListPageDataRequest,
) (*invoicepb.GetInvoiceListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get invoice list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of invoices
	invoices := make([]*invoicepb.Invoice, 0, len(r.invoices))
	for _, invoice := range r.invoices {
		invoices = append(invoices, invoice)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		invoices,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process invoice list data: %w", err)
	}

	// Convert processed items back to invoice protobuf format
	processedInvoices := make([]*invoicepb.Invoice, len(result.Items))
	for i, item := range result.Items {
		if invoice, ok := item.(*invoicepb.Invoice); ok {
			processedInvoices[i] = invoice
		} else {
			return nil, fmt.Errorf("failed to convert item to invoice type")
		}
	}

	// Convert search results to protobuf format
	searchResults := make([]*commonpb.SearchResult, len(result.SearchResults))
	for i, searchResult := range result.SearchResults {
		searchResults[i] = &commonpb.SearchResult{
			Score:      searchResult.Score,
			Highlights: searchResult.Highlights,
		}
	}

	return &invoicepb.GetInvoiceListPageDataResponse{
		InvoiceList:   processedInvoices,
		Pagination:    result.PaginationResponse,
		SearchResults: searchResults,
		Success:       true,
	}, nil
}

// GetInvoiceItemPageData retrieves a single invoice with enhanced item page data
func (r *MockInvoiceRepository) GetInvoiceItemPageData(
	ctx context.Context,
	req *invoicepb.GetInvoiceItemPageDataRequest,
) (*invoicepb.GetInvoiceItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get invoice item page data request is required")
	}
	if req.InvoiceId == "" {
		return nil, fmt.Errorf("invoice ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	invoice, exists := r.invoices[req.InvoiceId]
	if !exists {
		return nil, fmt.Errorf("invoice with ID '%s' not found", req.InvoiceId)
	}

	// In a real implementation, you might:
	// 1. Load related data (client details, subscription details)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &invoicepb.GetInvoiceItemPageDataResponse{
		Invoice: invoice,
		Success: true,
	}, nil
}

// mapToProtobufInvoice converts raw mock data to protobuf Invoice
func (r *MockInvoiceRepository) mapToProtobufInvoice(rawInvoice map[string]any) (*invoicepb.Invoice, error) {
	invoice := &invoicepb.Invoice{}

	// Map required fields
	if id, ok := rawInvoice["id"].(string); ok {
		invoice.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	// Map optional fields - add basic field mapping as needed
	if amount, ok := rawInvoice["amount"].(float64); ok {
		invoice.Amount = amount
	}

	// Note: Description field may not exist in the protobuf definition
	// Remove or comment out if not needed

	// Set default active status
	invoice.Active = true

	return invoice, nil
}

// NewInvoiceRepository creates a new mock invoice repository (registry constructor)
func NewInvoiceRepository(data map[string]*invoicepb.Invoice) invoicepb.InvoiceDomainServiceServer {
	repo := &MockInvoiceRepository{
		businessType: "education", // Default business type
		invoices:     data,
		mutex:        sync.RWMutex{},
		processor:    listdata.NewListDataProcessor(),
	}
	if data == nil {
		repo.invoices = make(map[string]*invoicepb.Invoice)
		// Initialize with mock data
		if err := repo.loadInitialData(); err != nil {
			fmt.Printf("Warning: Failed to load initial mock data: %v\n", err)
		}
	}
	return repo
}

//go:build mock_db

package payment

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	datamock "leapfor.xyz/copya/golang"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	paymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment"
	"github.com/erniealice/espyna-golang/internal/application/shared/listdata"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// MockPaymentRepository implements payment.PaymentRepository using stateful mock data
type MockPaymentRepository struct {
	paymentpb.UnimplementedPaymentDomainServiceServer
	businessType string
	payments     map[string]*paymentpb.Payment // Persistent in-memory store
	mutex        sync.RWMutex                  // Thread-safe concurrent access
	initialized  bool                          // Prevent double initialization
	processor    *listdata.ListDataProcessor   // List data processor for filtering, sorting, searching, and pagination
}

// PaymentRepositoryOption allows configuration of payment repository behavior
type PaymentRepositoryOption func(*MockPaymentRepository)

// WithPaymentTestOptimizations enables test-specific optimizations
func WithPaymentTestOptimizations(enabled bool) PaymentRepositoryOption {
	return func(r *MockPaymentRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockPaymentRepository creates a new mock payment repository
func NewMockPaymentRepository(businessType string, options ...PaymentRepositoryOption) paymentpb.PaymentDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockPaymentRepository{
		businessType: businessType,
		payments:     make(map[string]*paymentpb.Payment),
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
func (r *MockPaymentRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawPayments, err := datamock.LoadPayments(r.businessType)
	if err != nil {
		return fmt.Errorf("failed to load initial payments: %w", err)
	}

	// Convert and store each payment
	for _, rawPayment := range rawPayments {
		if payment, err := r.mapToProtobufPayment(rawPayment); err == nil {
			r.payments[payment.Id] = payment
		}
	}

	r.initialized = true
	return nil
}

// CreatePayment creates a new payment with stateful storage
func (r *MockPaymentRepository) CreatePayment(ctx context.Context, req *paymentpb.CreatePaymentRequest) (*paymentpb.CreatePaymentResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create payment request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("payment data is required")
	}
	if req.Data.Name == "" {
		return nil, fmt.Errorf("payment name is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate unique ID with timestamp
	now := time.Now()
	paymentID := fmt.Sprintf("payment-%d-%d", now.UnixNano(), len(r.payments))

	// Create new payment with proper timestamps and defaults
	newPayment := &paymentpb.Payment{
		Id:                 paymentID,
		Name:               req.Data.Name,
		SubscriptionId:     req.Data.SubscriptionId,
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             true, // Default to active
	}

	// Store in persistent map
	r.payments[paymentID] = newPayment

	return &paymentpb.CreatePaymentResponse{
		Data:    []*paymentpb.Payment{newPayment},
		Success: true,
	}, nil
}

// ReadPayment retrieves a payment by ID from stateful storage
func (r *MockPaymentRepository) ReadPayment(ctx context.Context, req *paymentpb.ReadPaymentRequest) (*paymentpb.ReadPaymentResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read payment request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payment ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store first (includes created/updated payments)
	if payment, exists := r.payments[req.Data.Id]; exists {
		return &paymentpb.ReadPaymentResponse{
			Data:    []*paymentpb.Payment{payment},
			Success: true,
		}, nil
	}

	return nil, fmt.Errorf("payment with ID '%s' not found", req.Data.Id)
}

// UpdatePayment updates an existing payment in stateful storage
func (r *MockPaymentRepository) UpdatePayment(ctx context.Context, req *paymentpb.UpdatePaymentRequest) (*paymentpb.UpdatePaymentResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update payment request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payment ID is required for update")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify payment exists
	existingPayment, exists := r.payments[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("payment with ID '%s' not found", req.Data.Id)
	}

	// Update only specified fields (preserve others)
	now := time.Now()
	updatedPayment := &paymentpb.Payment{
		Id:                 req.Data.Id,
		Name:               req.Data.Name,
		SubscriptionId:     req.Data.SubscriptionId,
		DateCreated:        existingPayment.DateCreated,       // Preserve original
		DateCreatedString:  existingPayment.DateCreatedString, // Preserve original
		DateModified:       &[]int64{now.UnixMilli()}[0],           // Update modification time
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             req.Data.Active,
	}

	// Update in persistent store
	r.payments[req.Data.Id] = updatedPayment

	return &paymentpb.UpdatePaymentResponse{
		Data:    []*paymentpb.Payment{updatedPayment},
		Success: true,
	}, nil
}

// DeletePayment deletes a payment from stateful storage
func (r *MockPaymentRepository) DeletePayment(ctx context.Context, req *paymentpb.DeletePaymentRequest) (*paymentpb.DeletePaymentResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete payment request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payment ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify payment exists before deletion
	if _, exists := r.payments[req.Data.Id]; !exists {
		return nil, fmt.Errorf("payment with ID '%s' not found", req.Data.Id)
	}

	// Perform actual deletion from persistent store
	delete(r.payments, req.Data.Id)

	return &paymentpb.DeletePaymentResponse{
		Success: true,
	}, nil
}

// ListPayments retrieves all payments from stateful storage
func (r *MockPaymentRepository) ListPayments(ctx context.Context, req *paymentpb.ListPaymentsRequest) (*paymentpb.ListPaymentsResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of payments
	payments := make([]*paymentpb.Payment, 0, len(r.payments))
	for _, payment := range r.payments {
		payments = append(payments, payment)
	}

	return &paymentpb.ListPaymentsResponse{
		Data:    payments,
		Success: true,
	}, nil
}

// mapToProtobufPayment converts raw mock data to protobuf Payment
func (r *MockPaymentRepository) mapToProtobufPayment(rawPayment map[string]any) (*paymentpb.Payment, error) {
	payment := &paymentpb.Payment{}

	if id, ok := rawPayment["id"].(string); ok {
		payment.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if name, ok := rawPayment["name"].(string); ok {
		payment.Name = name
	}

	if subscriptionId, ok := rawPayment["subscriptionId"].(string); ok {
		payment.SubscriptionId = subscriptionId
	}

	if dateCreated, ok := rawPayment["dateCreated"].(string); ok {
		payment.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			payment.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawPayment["dateModified"].(string); ok {
		payment.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			payment.DateModified = &timestamp
		}
	}

	if active, ok := rawPayment["active"].(bool); ok {
		payment.Active = active
	}

	return payment, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockPaymentRepository) parseTimestamp(timestampStr string) (int64, error) {
	if timestamp, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
		return timestamp, nil
	}

	if t, err := time.Parse(time.RFC3339, timestampStr); err == nil {
		return t.UnixMilli(), nil
	}

	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05.000Z",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timestampStr); err == nil {
			return t.UnixMilli(), nil
		}
	}

	return 0, fmt.Errorf("unable to parse timestamp: %s", timestampStr)
}

// GetPaymentListPageData retrieves payments with advanced filtering, sorting, searching, and pagination
func (r *MockPaymentRepository) GetPaymentListPageData(
	ctx context.Context,
	req *paymentpb.GetPaymentListPageDataRequest,
) (*paymentpb.GetPaymentListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get payment list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of payments
	payments := make([]*paymentpb.Payment, 0, len(r.payments))
	for _, payment := range r.payments {
		payments = append(payments, payment)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		payments,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process payment list data: %w", err)
	}

	// Convert processed items back to payment protobuf format
	processedPayments := make([]*paymentpb.Payment, len(result.Items))
	for i, item := range result.Items {
		if payment, ok := item.(*paymentpb.Payment); ok {
			processedPayments[i] = payment
		} else {
			return nil, fmt.Errorf("failed to convert item to payment type")
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

	return &paymentpb.GetPaymentListPageDataResponse{
		PaymentList:   processedPayments,
		Pagination:    result.PaginationResponse,
		SearchResults: searchResults,
		Success:       true,
	}, nil
}

// GetPaymentItemPageData retrieves a single payment with enhanced item page data
func (r *MockPaymentRepository) GetPaymentItemPageData(
	ctx context.Context,
	req *paymentpb.GetPaymentItemPageDataRequest,
) (*paymentpb.GetPaymentItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get payment item page data request is required")
	}
	if req.PaymentId == "" {
		return nil, fmt.Errorf("payment ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	payment, exists := r.payments[req.PaymentId]
	if !exists {
		return nil, fmt.Errorf("payment with ID '%s' not found", req.PaymentId)
	}

	// In a real implementation, you might:
	// 1. Load related data (subscription details, transaction details)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &paymentpb.GetPaymentItemPageDataResponse{
		Payment: payment,
		Success: true,
	}, nil
}

// NewPaymentRepository creates a new payment repository - Provider interface compatibility
func NewPaymentRepository(businessType string) paymentpb.PaymentDomainServiceServer {
	return NewMockPaymentRepository(businessType)
}

func init() {
	registry.RegisterRepositoryFactory("mock", "payment", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockPaymentRepository(businessType), nil
	})
}

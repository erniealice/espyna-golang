//go:build mock_db

package payment

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	datamock "leapfor.xyz/copya/golang"
	"github.com/erniealice/espyna-golang/internal/application/shared/listdata"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	paymentmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment_method"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// MockPaymentMethodRepository implements payment.PaymentMethodRepository using stateful mock data
type MockPaymentMethodRepository struct {
	paymentmethodpb.UnimplementedPaymentMethodDomainServiceServer
	businessType   string
	paymentMethods map[string]*paymentmethodpb.PaymentMethod // Persistent in-memory store
	mutex          sync.RWMutex                              // Thread-safe concurrent access
	initialized    bool                                      // Prevent double initialization
	processor      *listdata.ListDataProcessor               // List data processing
}

// PaymentMethodRepositoryOption allows configuration of payment method repository behavior
type PaymentMethodRepositoryOption func(*MockPaymentMethodRepository)

// WithPaymentMethodTestOptimizations enables test-specific optimizations
func WithPaymentMethodTestOptimizations(enabled bool) PaymentMethodRepositoryOption {
	return func(r *MockPaymentMethodRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockPaymentMethodRepository creates a new mock payment method repository
func NewMockPaymentMethodRepository(businessType string, options ...PaymentMethodRepositoryOption) paymentmethodpb.PaymentMethodDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockPaymentMethodRepository{
		businessType:   businessType,
		paymentMethods: make(map[string]*paymentmethodpb.PaymentMethod),
		processor:      listdata.NewListDataProcessor(),
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
func (r *MockPaymentMethodRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawPaymentMethods, err := datamock.LoadPaymentMethods(r.businessType)
	if err != nil {
		return fmt.Errorf("failed to load initial payment methods: %w", err)
	}

	// Convert and store each payment method
	for _, rawPaymentMethod := range rawPaymentMethods {
		if paymentMethod, err := r.mapToProtobufPaymentMethod(rawPaymentMethod); err == nil {
			r.paymentMethods[paymentMethod.Id] = paymentMethod
		}
	}

	r.initialized = true
	return nil
}

// CreatePaymentMethod creates a new payment method with stateful storage
func (r *MockPaymentMethodRepository) CreatePaymentMethod(ctx context.Context, req *paymentmethodpb.CreatePaymentMethodRequest) (*paymentmethodpb.CreatePaymentMethodResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create payment method request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("payment method data is required")
	}
	if req.Data.Name == "" {
		return nil, fmt.Errorf("payment method name is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate unique ID with timestamp
	now := time.Now()
	paymentMethodID := fmt.Sprintf("payment-method-%d-%d", now.UnixNano(), len(r.paymentMethods))

	// Create new payment method with proper timestamps and defaults
	newPaymentMethod := &paymentmethodpb.PaymentMethod{
		Id:                 paymentMethodID,
		Name:               req.Data.Name,
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             true, // Default to active
	}

	// Store in persistent map
	r.paymentMethods[paymentMethodID] = newPaymentMethod

	return &paymentmethodpb.CreatePaymentMethodResponse{
		Data:    []*paymentmethodpb.PaymentMethod{newPaymentMethod},
		Success: true,
	}, nil
}

// ReadPaymentMethod retrieves a payment method by ID from stateful storage
func (r *MockPaymentMethodRepository) ReadPaymentMethod(ctx context.Context, req *paymentmethodpb.ReadPaymentMethodRequest) (*paymentmethodpb.ReadPaymentMethodResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read payment method request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payment method ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store first (includes created/updated payment methods)
	if paymentMethod, exists := r.paymentMethods[req.Data.Id]; exists {
		return &paymentmethodpb.ReadPaymentMethodResponse{
			Data:    []*paymentmethodpb.PaymentMethod{paymentMethod},
			Success: true,
		}, nil
	}

	return nil, fmt.Errorf("payment method with ID '%s' not found", req.Data.Id)
}

// UpdatePaymentMethod updates an existing payment method in stateful storage
func (r *MockPaymentMethodRepository) UpdatePaymentMethod(ctx context.Context, req *paymentmethodpb.UpdatePaymentMethodRequest) (*paymentmethodpb.UpdatePaymentMethodResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update payment method request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payment method ID is required for update")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify payment method exists
	existingPaymentMethod, exists := r.paymentMethods[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("payment method with ID '%s' not found", req.Data.Id)
	}

	// Update only specified fields (preserve others)
	now := time.Now()
	updatedPaymentMethod := &paymentmethodpb.PaymentMethod{
		Id:                 req.Data.Id,
		Name:               req.Data.Name,
		DateCreated:        existingPaymentMethod.DateCreated,       // Preserve original
		DateCreatedString:  existingPaymentMethod.DateCreatedString, // Preserve original
		DateModified:       &[]int64{now.UnixMilli()}[0],                 // Update modification time
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             req.Data.Active,
	}

	// Update in persistent store
	r.paymentMethods[req.Data.Id] = updatedPaymentMethod

	return &paymentmethodpb.UpdatePaymentMethodResponse{
		Data:    []*paymentmethodpb.PaymentMethod{updatedPaymentMethod},
		Success: true,
	}, nil
}

// DeletePaymentMethod deletes a payment method from stateful storage
func (r *MockPaymentMethodRepository) DeletePaymentMethod(ctx context.Context, req *paymentmethodpb.DeletePaymentMethodRequest) (*paymentmethodpb.DeletePaymentMethodResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete payment method request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payment method ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify payment method exists before deletion
	if _, exists := r.paymentMethods[req.Data.Id]; !exists {
		return nil, fmt.Errorf("payment method with ID '%s' not found", req.Data.Id)
	}

	// Perform actual deletion from persistent store
	delete(r.paymentMethods, req.Data.Id)

	return &paymentmethodpb.DeletePaymentMethodResponse{
		Success: true,
	}, nil
}

// ListPaymentMethods retrieves all payment methods from stateful storage
func (r *MockPaymentMethodRepository) ListPaymentMethods(ctx context.Context, req *paymentmethodpb.ListPaymentMethodsRequest) (*paymentmethodpb.ListPaymentMethodsResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of payment methods
	paymentMethods := make([]*paymentmethodpb.PaymentMethod, 0, len(r.paymentMethods))
	for _, paymentMethod := range r.paymentMethods {
		paymentMethods = append(paymentMethods, paymentMethod)
	}

	return &paymentmethodpb.ListPaymentMethodsResponse{
		Data:    paymentMethods,
		Success: true,
	}, nil
}

// GetPaymentMethodListPageData retrieves payment methods with advanced filtering, sorting, searching, and pagination
func (r *MockPaymentMethodRepository) GetPaymentMethodListPageData(
	ctx context.Context,
	req *paymentmethodpb.GetPaymentMethodListPageDataRequest,
) (*paymentmethodpb.GetPaymentMethodListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get payment method list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of payment methods
	paymentMethods := make([]*paymentmethodpb.PaymentMethod, 0, len(r.paymentMethods))
	for _, paymentMethod := range r.paymentMethods {
		paymentMethods = append(paymentMethods, paymentMethod)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		paymentMethods,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process payment method list data: %w", err)
	}

	// Convert processed items back to payment method protobuf format
	processedPaymentMethods := make([]*paymentmethodpb.PaymentMethod, len(result.Items))
	for i, item := range result.Items {
		if paymentMethod, ok := item.(*paymentmethodpb.PaymentMethod); ok {
			processedPaymentMethods[i] = paymentMethod
		} else {
			return nil, fmt.Errorf("failed to convert item to payment method type")
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

	return &paymentmethodpb.GetPaymentMethodListPageDataResponse{
		PaymentMethodList: processedPaymentMethods,
		Pagination:        result.PaginationResponse,
		SearchResults:     searchResults,
		Success:           true,
	}, nil
}

// GetPaymentMethodItemPageData retrieves a single payment method with enhanced item page data
func (r *MockPaymentMethodRepository) GetPaymentMethodItemPageData(
	ctx context.Context,
	req *paymentmethodpb.GetPaymentMethodItemPageDataRequest,
) (*paymentmethodpb.GetPaymentMethodItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get payment method item page data request is required")
	}
	if req.PaymentMethodId == "" {
		return nil, fmt.Errorf("payment method ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	paymentMethod, exists := r.paymentMethods[req.PaymentMethodId]
	if !exists {
		return nil, fmt.Errorf("payment method with ID '%s' not found", req.PaymentMethodId)
	}

	// In a real implementation, you might:
	// 1. Load related data (payment profile details, transaction history)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &paymentmethodpb.GetPaymentMethodItemPageDataResponse{
		PaymentMethod: paymentMethod,
		Success:       true,
	}, nil
}

// mapToProtobufPaymentMethod converts raw mock data to protobuf PaymentMethod
func (r *MockPaymentMethodRepository) mapToProtobufPaymentMethod(rawPaymentMethod map[string]any) (*paymentmethodpb.PaymentMethod, error) {
	paymentMethod := &paymentmethodpb.PaymentMethod{}

	if id, ok := rawPaymentMethod["id"].(string); ok {
		paymentMethod.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	// Map name field - try different field names for compatibility
	if name, ok := rawPaymentMethod["name"].(string); ok {
		paymentMethod.Name = name
	} else if cardholderName, ok := rawPaymentMethod["cardholderName"].(string); ok {
		// Fallback to cardholderName if name not present
		paymentMethod.Name = cardholderName + " Payment Method"
	} else {
		// Default name based on method type
		if methodType, ok := rawPaymentMethod["methodType"].(string); ok {
			paymentMethod.Name = methodType
		} else {
			paymentMethod.Name = "Payment Method"
		}
	}

	if dateCreated, ok := rawPaymentMethod["dateCreated"].(string); ok {
		paymentMethod.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			paymentMethod.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawPaymentMethod["dateModified"].(string); ok {
		paymentMethod.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			paymentMethod.DateModified = &timestamp
		}
	}

	if active, ok := rawPaymentMethod["active"].(bool); ok {
		paymentMethod.Active = active
	} else {
		// Default to active if not specified
		paymentMethod.Active = true
	}

	// Map method details - this is the crucial fix for the oneof field
	if err := r.mapMethodDetails(rawPaymentMethod, paymentMethod); err != nil {
		return nil, fmt.Errorf("failed to map method details: %w", err)
	}

	return paymentMethod, nil
}

// mapMethodDetails maps raw payment method data to CardDetails or BankAccountDetails
func (r *MockPaymentMethodRepository) mapMethodDetails(rawPaymentMethod map[string]any, paymentMethod *paymentmethodpb.PaymentMethod) error {
	// Check method type to determine if it's a card or bank account
	methodType, hasMethodType := rawPaymentMethod["methodType"].(string)
	lastFourDigits, hasLastFour := rawPaymentMethod["lastFourDigits"].(string)
	accountLastFour, hasAccountLastFour := rawPaymentMethod["accountLastFour"].(string)

	if !hasLastFour && !hasAccountLastFour {
		return fmt.Errorf("lastFourDigits or accountLastFour field is required for payment methods")
	}

	if !hasLastFour {
		lastFourDigits = accountLastFour
	}

	// Determine if this is a card or bank account based on available fields
	if hasMethodType && (methodType == "Credit Card" || methodType == "Debit Card") {
		// Map as CardDetails
		cardDetails := &paymentmethodpb.CardDetails{
			LastFourDigits: lastFourDigits,
		}

		// Map card type
		if methodType == "Credit Card" {
			cardDetails.CardType = "Visa" // Default for mock data
		} else {
			cardDetails.CardType = methodType
		}

		// Map expiry information
		if expiryMonth, ok := rawPaymentMethod["expiryMonth"].(string); ok {
			if month, err := strconv.Atoi(expiryMonth); err == nil {
				cardDetails.ExpiryMonth = int32(month)
			}
		}

		if expiryYear, ok := rawPaymentMethod["expiryYear"].(string); ok {
			if year, err := strconv.Atoi(expiryYear); err == nil {
				cardDetails.ExpiryYear = int32(year)
			}
		}

		// Set the oneof field
		paymentMethod.MethodDetails = &paymentmethodpb.PaymentMethod_Card{
			Card: cardDetails,
		}

	} else {
		// Map as BankAccountDetails (fallback)
		bankDetails := &paymentmethodpb.BankAccountDetails{
			LastFourDigits: lastFourDigits,
			BankName:       "Mock Bank", // Default for mock data
		}

		// Try to get bank name if available
		if bankName, ok := rawPaymentMethod["bankName"].(string); ok {
			bankDetails.BankName = bankName
		}

		// Set the oneof field
		paymentMethod.MethodDetails = &paymentmethodpb.PaymentMethod_BankAccount{
			BankAccount: bankDetails,
		}
	}

	return nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockPaymentMethodRepository) parseTimestamp(timestampStr string) (int64, error) {
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

// NewPaymentMethodRepository creates a new payment method repository - Provider interface compatibility
func NewPaymentMethodRepository(businessType string) paymentmethodpb.PaymentMethodDomainServiceServer {
	return NewMockPaymentMethodRepository(businessType)
}

func init() {
	registry.RegisterRepositoryFactory("mock", "payment_method", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockPaymentMethodRepository(businessType), nil
	})
}

//go:build mock_db && mock_auth && google && uuidv7

// Package product provides comprehensive tests for the product reading use case.
//
// The tests cover various scenarios, including success, validation errors,
// nil requests, nil data, empty IDs, and domain-specific functionality.
// Each test function has a specific test code for tracking and logging.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestReadProductUseCase_Execute_Success
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-PRODUCT-READ-SUCCESS-v1.0: Basic successful reading
//   - ESPYNA-TEST-PRODUCT-PRODUCT-READ-NOT-FOUND-v1.0: Non-existent ID handling
//   - ESPYNA-TEST-PRODUCT-PRODUCT-READ-NIL-REQUEST-v1.0: Nil request validation
//   - ESPYNA-TEST-PRODUCT-PRODUCT-READ-NIL-DATA-v1.0: Nil data validation
//   - ESPYNA-TEST-PRODUCT-PRODUCT-READ-VALIDATION-EMPTY-ID-v1.0: Empty ID validation
//   - ESPYNA-TEST-PRODUCT-PRODUCT-READ-VALIDATION-INPUT-v1.0: Input validation scenarios
//   - ESPYNA-TEST-PRODUCT-PRODUCT-READ-INTEGRATION-EDUCATION-v1.0: Domain-specific functionality

package product

import (
	"context"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	mockAuth "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/auth/mock"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock"
	mockProduct "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/product"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/id/uuidv7"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/translation"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
)

// MockTransactionServiceAdapter adapts infrastructure MockTransactionManager to ports.TransactionService
// This eliminates duplication by reusing the sophisticated infrastructure mock instead of a custom implementation
type MockTransactionServiceAdapterForRead struct {
	mockTxManager        *mock.MockTransactionManager
	supportsTransactions bool
}

// NewMockTransactionServiceForRead creates transaction service using infrastructure mock (replaces old custom mock)
func NewMockTransactionServiceForRead(supportsTransactions bool) ports.TransactionService {
	if !supportsTransactions {
		return ports.NewNoOpTransactionService()
	}

	// Create infrastructure mock and cast to access setter methods
	txManager := mock.NewMockTransactionManager().(*mock.MockTransactionManager)

	return &MockTransactionServiceAdapterForRead{
		mockTxManager:        txManager,
		supportsTransactions: true,
	}
}

// NewFailingMockTransactionServiceForRead creates a transaction service that will fail RunInTransaction
func NewFailingMockTransactionServiceForRead() ports.TransactionService {
	txManager := mock.NewMockTransactionManager().(*mock.MockTransactionManager)

	// Configure to fail at RunInTransaction level using new setter method
	txManager.SetShouldFailRunInTx(true)

	return &MockTransactionServiceAdapterForRead{
		mockTxManager:        txManager,
		supportsTransactions: true,
	}
}

func (m *MockTransactionServiceAdapterForRead) SupportsTransactions() bool {
	return m.supportsTransactions
}

func (m *MockTransactionServiceAdapterForRead) IsTransactionActive(ctx context.Context) bool {
	if !m.supportsTransactions {
		return false
	}
	_, inTx := m.mockTxManager.GetTransaction(ctx)
	return inTx
}

func (m *MockTransactionServiceAdapterForRead) ExecuteInTransaction(ctx context.Context, fn func(context.Context) error) error {
	if !m.supportsTransactions {
		return fn(ctx)
	}
	return m.mockTxManager.RunInTransaction(ctx, fn)
}

// GetMockTransactionManager returns the underlying infrastructure mock for advanced configuration
func (m *MockTransactionServiceAdapterForRead) GetMockTransactionManager() *mock.MockTransactionManager {
	return m.mockTxManager
}

// TestServiceFactoryForRead creates services for testing (uses real services where appropriate)
type TestServiceFactoryForRead struct {
	realIDService          ports.IDService
	realTranslationService ports.TranslationService
}

func NewTestServiceFactoryForRead() *TestServiceFactoryForRead {
	// Use real Google UUID service
	realIDService := uuidv7.NewGoogleUUIDv7Service()

	// Use the same Go-idiomatic translation service as the production container
	// This automatically resolves workspace paths and provides compile-time safety
	realTranslationService := translation.NewLynguaTranslationService()

	return &TestServiceFactoryForRead{
		realIDService:          realIDService,
		realTranslationService: realTranslationService,
	}
}

func (f *TestServiceFactoryForRead) CreateServices(supportsTransaction bool, shouldAuthorize bool) ReadProductServices {
	var authService ports.AuthorizationService
	if shouldAuthorize {
		authService = mockAuth.NewAllowAllAuth() // Allow all for simple positive tests
	} else {
		authService = mockAuth.NewDenyAllAuth() // Real mock that denies all
	}

	return ReadProductServices{
		AuthorizationService: authService,
		TransactionService:   NewMockTransactionServiceForRead(supportsTransaction),
		TranslationService:   f.realTranslationService, // Real translation service
	}
}

func (f *TestServiceFactoryForRead) CreateServicesWithFailingTransaction(shouldAuthorize bool) ReadProductServices {
	// Use new infrastructure mock-based failing transaction service
	failingTxService := NewFailingMockTransactionServiceForRead()

	var authService ports.AuthorizationService
	if shouldAuthorize {
		authService = mockAuth.NewAllowAllAuth() // Allow all for simple positive tests
	} else {
		authService = mockAuth.NewDenyAllAuth() // Real mock that denies all
	}

	return ReadProductServices{
		AuthorizationService: authService,
		TransactionService:   failingTxService,
		TranslationService:   f.realTranslationService, // Real translation service
	}
}

// Global test service factory - reused across all tests
var testServiceFactoryForRead = NewTestServiceFactoryForRead()

// Test helper to create use case with real services where appropriate
func createReadTestUseCase(businessType string, supportsTransaction bool) *ReadProductUseCase {
	return createReadTestUseCaseWithAuth(businessType, supportsTransaction, true)
}

func createReadTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *ReadProductUseCase {
	mockRepo := mockProduct.NewMockProductRepository(businessType)

	repositories := ReadProductRepositories{
		Product: mockRepo,
	}

	services := testServiceFactoryForRead.CreateServices(supportsTransaction, shouldAuthorize)

	return NewReadProductUseCase(repositories, services)
}

func TestReadProductUseCase_Execute_Success(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-READ-SUCCESS-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "Success", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createReadTestUseCase(businessType, false)

	// Test reading with valid product IDs from education domain
	testCases := []struct {
		name      string
		productID string
		expected  string
	}{
		{
			name:      "Mathematics Subject",
			productID: "subject-math",
			expected:  "Mathematics",
		},
		{
			name:      "Science Subject",
			productID: "subject-science",
			expected:  "Science",
		},
		{
			name:      "English Subject",
			productID: "subject-english",
			expected:  "English",
		},
		{
			name:      "Music Subject",
			productID: "subject-music",
			expected:  "Music",
		},
		{
			name:      "Physical Education Subject",
			productID: "subject-pe",
			expected:  "Physical Education",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &productpb.ReadProductRequest{
				Data: &productpb.Product{
					Id: tc.productID,
				},
			}

			response, err := useCase.Execute(ctx, req)

			testutil.AssertNoError(t, err)
			testutil.AssertNotNil(t, response, "response")
			testutil.AssertTrue(t, response.Success, "success")
			testutil.AssertEqual(t, 1, len(response.Data), "response data length")

			foundProduct := response.Data[0]
			testutil.AssertStringEqual(t, tc.expected, foundProduct.Name, "product name")
			testutil.AssertStringEqual(t, tc.productID, foundProduct.Id, "product ID")
			testutil.AssertTrue(t, foundProduct.Active, "product active status")

			// Verify audit fields are present
			testutil.AssertFieldSet(t, foundProduct.DateCreated, "DateCreated")
			testutil.AssertFieldSet(t, foundProduct.DateCreatedString, "DateCreatedString")
		})
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "Success", true, nil)
}

func TestReadProductUseCase_Execute_NotFound(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-READ-NOT-FOUND-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "NotFound", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createReadTestUseCase(businessType, false)

	req := &productpb.ReadProductRequest{
		Data: &productpb.Product{
			Id: "non-existent-product-id",
		},
	}

	_, err := useCase.Execute(ctx, req)

	testutil.AssertError(t, err)

	// The implementation manually replaces {courseId} with the actual ID
	// So we expect the error to contain the translated message with the ID substituted
	if !strings.Contains(err.Error(), "Course with ID \"non-existent-product-id\" not found") {
		t.Errorf("Expected error to contain product not found message with ID, got '%s'", err.Error())
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "NotFound", false, err)
}

func TestReadProductUseCase_Execute_NilRequest(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-READ-NIL-REQUEST-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "NilRequest", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createReadTestUseCase(businessType, false)

	_, err := useCase.Execute(ctx, nil)

	testutil.AssertErrorForNilRequest(t, err)
	testutil.AssertTranslatedError(t, err, "product.validation.request_required", useCase.services.TranslationService, ctx)

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "NilRequest", false, err)
}

func TestReadProductUseCase_Execute_NilData(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-READ-NIL-DATA-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "NilData", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createReadTestUseCase(businessType, false)

	req := &productpb.ReadProductRequest{
		Data: nil,
	}

	// Create context with business type to simulate HTTP middleware behavior
	_, err := useCase.Execute(ctx, req)

	testutil.AssertErrorForNilData(t, err)
	testutil.AssertTranslatedError(t, err, "product.validation.data_required", useCase.services.TranslationService, ctx)

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "NilData", false, err)
}

func TestReadProductUseCase_Execute_EmptyID(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-READ-VALIDATION-EMPTY-ID-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "EmptyID", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createReadTestUseCase(businessType, false)

	req := &productpb.ReadProductRequest{
		Data: &productpb.Product{
			Id: "", // Empty ID
		},
	}

	_, err := useCase.Execute(ctx, req)

	testutil.AssertValidationError(t, err, "empty ID")
	testutil.AssertTranslatedError(t, err, "product.validation.id_required", useCase.services.TranslationService, ctx)

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "EmptyID", false, err)
}

func TestReadProductUseCase_Execute_ValidateInput(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-READ-VALIDATION-INPUT-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "ValidateInput", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createReadTestUseCase(businessType, false)

	// Test comprehensive input validation scenarios
	testCases := []struct {
		name string
		req  *productpb.ReadProductRequest
	}{
		{
			name: "Valid Request",
			req: &productpb.ReadProductRequest{
				Data: &productpb.Product{
					Id: "subject-math",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			response, err := useCase.Execute(ctx, tc.req)

			testutil.AssertNoError(t, err)
			testutil.AssertNotNil(t, response, "response")
			testutil.AssertTrue(t, response.Success, "success")
		})
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "ValidateInput", true, nil)
}

func TestReadProductUseCase_Execute_EducationDomainSpecific(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-READ-INTEGRATION-EDUCATION-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "EducationDomainSpecific", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createReadTestUseCase(businessType, false)

	// Test reading education-specific products by their actual IDs
	educationProducts := []struct {
		id           string
		expectedName string
		description  string
	}{
		{
			id:           "subject-math",
			expectedName: "Mathematics",
			description:  "The study of numbers, quantity, and space.",
		},
		{
			id:           "subject-science",
			expectedName: "Science",
			description:  "The study of the natural and physical world through observation and experimentation.",
		},
		{
			id:           "subject-english",
			expectedName: "English",
			description:  "The study of English language and literature.",
		},
		{
			id:           "subject-music",
			expectedName: "Music",
			description:  "The study of the art of sound.",
		},
		{
			id:           "subject-pe",
			expectedName: "Physical Education",
			description:  "Instruction in physical exercise and games.",
		},
	}

	for _, prod := range educationProducts {
		t.Run(prod.expectedName, func(t *testing.T) {
			req := &productpb.ReadProductRequest{
				Data: &productpb.Product{
					Id: prod.id,
				},
			}

			response, err := useCase.Execute(ctx, req)

			testutil.AssertNoError(t, err)
			foundProduct := response.Data[0]

			// Verify education-specific product data
			testutil.AssertStringEqual(t, prod.expectedName, foundProduct.Name, "education product name")
			testutil.AssertStringEqual(t, prod.id, foundProduct.Id, "education product ID")

			// Education products should have descriptions
			testutil.AssertNotNil(t, foundProduct.Description, "education product description")
			testutil.AssertStringEqual(t, prod.description, *foundProduct.Description, "description content")

			// Education products should be active
			testutil.AssertTrue(t, foundProduct.Active, "education product active status")

			// Verify audit fields for education products
			testutil.AssertFieldSet(t, foundProduct.DateCreated, "DateCreated")
			testutil.AssertFieldSet(t, foundProduct.DateCreatedString, "DateCreatedString")
		})
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "EducationDomainSpecific", true, nil)
}

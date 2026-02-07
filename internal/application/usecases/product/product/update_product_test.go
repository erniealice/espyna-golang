//go:build mock_db && mock_auth && google && uuidv7

// Package product provides comprehensive tests for the product update use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, data enrichment, and boundary conditions.
// Each test function has a specific test code for tracking and logging.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestUpdateProductUseCase_Execute_Success
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-PRODUCT-UPDATE-SUCCESS-v1.0: Basic successful update
//   - ESPYNA-TEST-PRODUCT-PRODUCT-UPDATE-TRANSACTION-v1.0: Update with transaction support
//   - ESPYNA-TEST-PRODUCT-PRODUCT-UPDATE-NIL-REQUEST-v1.0: Nil request validation
//   - ESPYNA-TEST-PRODUCT-PRODUCT-UPDATE-NIL-DATA-v1.0: Nil data validation
//   - ESPYNA-TEST-PRODUCT-PRODUCT-UPDATE-VALIDATION-EMPTY-ID-v1.0: Empty ID validation
//   - ESPYNA-TEST-PRODUCT-PRODUCT-UPDATE-VALIDATION-EMPTY-NAME-v1.0: Empty name validation
//   - ESPYNA-TEST-PRODUCT-PRODUCT-UPDATE-VALIDATION-NAME-TOO-SHORT-v1.0: Name too short validation
//   - ESPYNA-TEST-PRODUCT-PRODUCT-UPDATE-VALIDATION-NAME-TOO-LONG-v1.0: Name too long validation
//   - ESPYNA-TEST-PRODUCT-PRODUCT-UPDATE-VALIDATION-DESCRIPTION-TOO-LONG-v1.0: Description too long validation
//   - ESPYNA-TEST-PRODUCT-PRODUCT-UPDATE-BUSINESS-RULE-v1.0: Business rule name normalization
//   - ESPYNA-TEST-PRODUCT-PRODUCT-UPDATE-ENRICHMENT-v1.0: Auto-generated fields verification
//   - ESPYNA-TEST-PRODUCT-PRODUCT-UPDATE-NOT-FOUND-v1.0: Non-existent ID handling
//   - ESPYNA-TEST-PRODUCT-PRODUCT-UPDATE-BOUNDARY-MINIMAL-v1.0: Boundary testing (minimum valid)
//   - ESPYNA-TEST-PRODUCT-PRODUCT-UPDATE-BOUNDARY-MAXIMAL-v1.0: Boundary testing (maximum valid)
//   - ESPYNA-TEST-PRODUCT-PRODUCT-UPDATE-TRANSACTION-FAILURE-v1.0: Transaction error handling
//   - ESPYNA-TEST-PRODUCT-PRODUCT-UPDATE-INTEGRATION-v1.0: Domain-specific functionality
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/product.json
//   - Mock data: packages/copya/data/{businessType}/product.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/product.json

package product

import (
	"context"
	"strings"
	"testing"
	"time"

	"leapfor.xyz/espyna/internal/application/ports"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockAuth "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/auth/mock"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock"
	mockProduct "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/product"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/id/uuidv7"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/translation"
	productpb "leapfor.xyz/esqyma/golang/v1/domain/product/product"
)

// MockTransactionServiceAdapter adapts infrastructure MockTransactionManager to ports.TransactionService
// This eliminates duplication by reusing the sophisticated infrastructure mock instead of a custom implementation
type MockTransactionServiceAdapterForUpdate struct {
	mockTxManager        *mock.MockTransactionManager
	supportsTransactions bool
}

// NewMockTransactionServiceForUpdate creates transaction service using infrastructure mock (replaces old custom mock)
func NewMockTransactionServiceForUpdate(supportsTransactions bool) ports.TransactionService {
	if !supportsTransactions {
		return ports.NewNoOpTransactionService()
	}

	// Create infrastructure mock and cast to access setter methods
	txManager := mock.NewMockTransactionManager().(*mock.MockTransactionManager)

	return &MockTransactionServiceAdapterForUpdate{
		mockTxManager:        txManager,
		supportsTransactions: true,
	}
}

// NewFailingMockTransactionServiceForUpdate creates a transaction service that will fail RunInTransaction
func NewFailingMockTransactionServiceForUpdate() ports.TransactionService {
	txManager := mock.NewMockTransactionManager().(*mock.MockTransactionManager)

	// Configure to fail at RunInTransaction level using new setter method
	txManager.SetShouldFailRunInTx(true)

	return &MockTransactionServiceAdapterForUpdate{
		mockTxManager:        txManager,
		supportsTransactions: true,
	}
}

func (m *MockTransactionServiceAdapterForUpdate) SupportsTransactions() bool {
	return m.supportsTransactions
}

func (m *MockTransactionServiceAdapterForUpdate) IsTransactionActive(ctx context.Context) bool {
	if !m.supportsTransactions {
		return false
	}
	_, inTx := m.mockTxManager.GetTransaction(ctx)
	return inTx
}

func (m *MockTransactionServiceAdapterForUpdate) ExecuteInTransaction(ctx context.Context, fn func(context.Context) error) error {
	if !m.supportsTransactions {
		return fn(ctx)
	}
	return m.mockTxManager.RunInTransaction(ctx, fn)
}

// GetMockTransactionManager returns the underlying infrastructure mock for advanced configuration
func (m *MockTransactionServiceAdapterForUpdate) GetMockTransactionManager() *mock.MockTransactionManager {
	return m.mockTxManager
}

// TestServiceFactoryForUpdate creates services for testing (uses real services where appropriate)
type TestServiceFactoryForUpdate struct {
	realIDService          ports.IDService
	realTranslationService ports.TranslationService
}

func NewTestServiceFactoryForUpdate() *TestServiceFactoryForUpdate {
	// Use real Google UUID service
	realIDService := uuidv7.NewGoogleUUIDv7Service()

	// Use the same Go-idiomatic translation service as the production container
	// This automatically resolves workspace paths and provides compile-time safety
	realTranslationService := translation.NewLynguaTranslationService()

	return &TestServiceFactoryForUpdate{
		realIDService:          realIDService,
		realTranslationService: realTranslationService,
	}
}

func (f *TestServiceFactoryForUpdate) CreateServices(supportsTransaction bool, shouldAuthorize bool) UpdateProductServices {
	var authService ports.AuthorizationService
	if shouldAuthorize {
		authService = mockAuth.NewAllowAllAuth() // Allow all for simple positive tests
	} else {
		authService = mockAuth.NewDenyAllAuth() // Real mock that denies all
	}

	return UpdateProductServices{
		AuthorizationService: authService,
		TransactionService:   NewMockTransactionServiceForUpdate(supportsTransaction),
		TranslationService:   f.realTranslationService, // Real translation service
	}
}

func (f *TestServiceFactoryForUpdate) CreateServicesWithFailingTransaction(shouldAuthorize bool) UpdateProductServices {
	// Use new infrastructure mock-based failing transaction service
	failingTxService := NewFailingMockTransactionServiceForUpdate()

	var authService ports.AuthorizationService
	if shouldAuthorize {
		authService = mockAuth.NewAllowAllAuth() // Allow all for simple positive tests
	} else {
		authService = mockAuth.NewDenyAllAuth() // Real mock that denies all
	}

	return UpdateProductServices{
		AuthorizationService: authService,
		TransactionService:   failingTxService,
		TranslationService:   f.realTranslationService, // Real translation service
	}
}

// Global test service factory - reused across all tests
var testServiceFactoryForUpdate = NewTestServiceFactoryForUpdate()

// Test helper to create use case with real services where appropriate
func createUpdateTestUseCase(businessType string, supportsTransaction bool) *UpdateProductUseCase {
	return createUpdateTestUseCaseWithAuth(businessType, supportsTransaction, true)
}

func createUpdateTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *UpdateProductUseCase {
	mockRepo := mockProduct.NewMockProductRepository(businessType)

	repositories := UpdateProductRepositories{
		Product: mockRepo,
	}

	services := testServiceFactoryForUpdate.CreateServices(supportsTransaction, shouldAuthorize)

	return NewUpdateProductUseCase(repositories, services)
}

func TestUpdateProductUseCase_Execute_Success(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-UPDATE-SUCCESS-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "Success", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createUpdateTestUseCase(businessType, false)

	req := &productpb.UpdateProductRequest{
		Data: &productpb.Product{
			Id:          "subject-math", // Existing product ID
			Name:        "Advanced Mathematics Course",
			Description: &[]string{"Updated comprehensive advanced mathematics course covering calculus, algebra, and geometry"}[0],
		},
	}

	response, err := useCase.Execute(ctx, req)

	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, response, "response")
	testutil.AssertTrue(t, response.Success, "success")
	testutil.AssertDataLength(t, 1, len(response.Data), "response data")

	updatedProduct := response.Data[0]
	testutil.AssertStringEqual(t, "Advanced Mathematics Course", updatedProduct.Name, "updated name")

	expectedDescription := "Updated comprehensive advanced mathematics course covering calculus, algebra, and geometry"
	testutil.AssertDescriptionMatch(t, expectedDescription, updatedProduct.Description, "updated description")

	testutil.AssertStringEqual(t, "subject-math", updatedProduct.Id, "product ID")
	testutil.AssertTrue(t, updatedProduct.Active, "product active status")
	testutil.AssertFieldSet(t, updatedProduct.DateModified, "DateModified")
	testutil.AssertFieldSet(t, updatedProduct.DateModifiedString, "DateModifiedString")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "Success", true, nil)
}

func TestUpdateProductUseCase_Execute_WithTransaction(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-UPDATE-TRANSACTION-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "WithTransaction", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createUpdateTestUseCase(businessType, true)

	req := &productpb.UpdateProductRequest{
		Data: &productpb.Product{
			Id:          "subject-science", // Existing product ID
			Name:        "Elementary Science Course",
			Description: &[]string{"Updated basic science concepts for elementary students with transaction support"}[0],
		},
	}

	response, err := useCase.Execute(ctx, req)

	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, response, "response")
	testutil.AssertTrue(t, response.Success, "success")

	updatedProduct := response.Data[0]
	testutil.AssertStringEqual(t, "Elementary Science Course", updatedProduct.Name, "updated name")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "WithTransaction", true, nil)
}

func TestUpdateProductUseCase_Execute_NilRequest(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-UPDATE-NIL-REQUEST-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "NilRequest", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createUpdateTestUseCase(businessType, false)

	_, err := useCase.Execute(ctx, nil)

	testutil.AssertErrorForNilRequest(t, err)
	testutil.AssertTranslatedError(t, err, "product.validation.request_required", useCase.services.TranslationService, ctx)

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "NilRequest", false, err)
}

func TestUpdateProductUseCase_Execute_NilData(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-UPDATE-NIL-DATA-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "NilData", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createUpdateTestUseCase(businessType, false)

	req := &productpb.UpdateProductRequest{
		Data: nil,
	}

	// Create context with business type to simulate HTTP middleware behavior
	_, err := useCase.Execute(ctx, req)

	testutil.AssertErrorForNilData(t, err)
	testutil.AssertTranslatedError(t, err, "product.validation.data_required", useCase.services.TranslationService, ctx)

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "NilData", false, err)
}

func TestUpdateProductUseCase_Execute_EmptyID(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-UPDATE-VALIDATION-EMPTY-ID-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "EmptyID", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createUpdateTestUseCase(businessType, false)

	req := &productpb.UpdateProductRequest{
		Data: &productpb.Product{
			Id:   "", // Empty ID
			Name: "Valid Product Name",
		},
	}

	_, err := useCase.Execute(ctx, req)

	testutil.AssertValidationError(t, err, "empty ID")
	testutil.AssertTranslatedError(t, err, "product.validation.id_required", useCase.services.TranslationService, ctx)

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "EmptyID", false, err)
}

func TestUpdateProductUseCase_Execute_EmptyName(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-UPDATE-VALIDATION-EMPTY-NAME-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "EmptyName", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createUpdateTestUseCase(businessType, false)

	req := &productpb.UpdateProductRequest{
		Data: &productpb.Product{
			Id:   "subject-math",
			Name: "", // Empty name
		},
	}

	_, err := useCase.Execute(ctx, req)

	testutil.AssertValidationError(t, err, "empty name")
	testutil.AssertTranslatedError(t, err, "product.validation.name_required", useCase.services.TranslationService, ctx)

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "EmptyName", false, err)
}

func TestUpdateProductUseCase_Execute_NameTooShort(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-UPDATE-VALIDATION-NAME-TOO-SHORT-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "NameTooShort", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createUpdateTestUseCase(businessType, false)

	req := &productpb.UpdateProductRequest{
		Data: &productpb.Product{
			Id:   "subject-math",
			Name: "A", // Too short (less than 2 characters)
		},
	}

	_, err := useCase.Execute(ctx, req)

	testutil.AssertValidationError(t, err, "name too short")
	testutil.AssertTranslatedError(t, err, "product.validation.name_min_length", useCase.services.TranslationService, ctx)

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "NameTooShort", false, err)
}

func TestUpdateProductUseCase_Execute_NameTooLong(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-UPDATE-VALIDATION-NAME-TOO-LONG-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "NameTooLong", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createUpdateTestUseCase(businessType, false)

	// Create a string longer than 100 characters (actual limit from implementation)
	longName := testutil.GenerateDefaultLongString(101)

	req := &productpb.UpdateProductRequest{
		Data: &productpb.Product{
			Id:   "subject-math",
			Name: longName,
		},
	}

	_, err := useCase.Execute(ctx, req)

	testutil.AssertValidationError(t, err, "name too long")
	testutil.AssertTranslatedError(t, err, "product.validation.name_max_length", useCase.services.TranslationService, ctx)

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "NameTooLong", false, err)
}

func TestUpdateProductUseCase_Execute_DescriptionTooLong(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-UPDATE-VALIDATION-DESCRIPTION-TOO-LONG-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "DescriptionTooLong", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createUpdateTestUseCase(businessType, false)

	// Create a description longer than 1000 characters
	longDescription := testutil.GenerateDefaultLongString(1001)

	req := &productpb.UpdateProductRequest{
		Data: &productpb.Product{
			Id:          "subject-math",
			Name:        "Valid Product Name",
			Description: &longDescription,
		},
	}

	_, err := useCase.Execute(ctx, req)

	testutil.AssertValidationError(t, err, "description too long")
	testutil.AssertTranslatedError(t, err, "product.validation.description_max_length", useCase.services.TranslationService, ctx)

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "DescriptionTooLong", false, err)
}

func TestUpdateProductUseCase_Execute_BusinessRuleNameNormalization(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-UPDATE-BUSINESS-RULE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "BusinessRuleNameNormalization", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createUpdateTestUseCase(businessType, false)

	req := &productpb.UpdateProductRequest{
		Data: &productpb.Product{
			Id:   "subject-math",
			Name: "  advanced MATHEMATICS  ", // Name with spaces and mixed case
		},
	}

	response, err := useCase.Execute(ctx, req)

	testutil.AssertNoError(t, err)

	updatedProduct := response.Data[0]
	// Verify name normalization with strings.Title behavior
	expectedName := "Advanced Mathematics"
	testutil.AssertStringEqual(t, expectedName, updatedProduct.Name, "normalized name")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "BusinessRuleNameNormalization", true, nil)
}

func TestUpdateProductUseCase_Execute_DataEnrichment(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-UPDATE-ENRICHMENT-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "DataEnrichment", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createUpdateTestUseCase(businessType, false)

	req := &productpb.UpdateProductRequest{
		Data: &productpb.Product{
			Id:   "subject-math",
			Name: "Test Product Update Data Enrichment",
		},
	}

	response, err := useCase.Execute(ctx, req)

	testutil.AssertNoError(t, err)

	updatedProduct := response.Data[0]

	// Verify ID was preserved
	testutil.AssertStringEqual(t, "subject-math", updatedProduct.Id, "product ID")

	// Verify DateModified was updated
	testutil.AssertFieldSet(t, updatedProduct.DateModified, "DateModified")
	testutil.AssertFieldSet(t, updatedProduct.DateModifiedString, "DateModifiedString")

	// Verify DateModified is recent (within last 5 seconds)
	now := time.Now().UnixMilli()
	if *updatedProduct.DateModified < now-5000 || *updatedProduct.DateModified > now+5000 {
		t.Errorf("Expected DateModified to be recent, got %d (now: %d)", *updatedProduct.DateModified, now)
	}

	// Note: DateCreated should NOT be updated for updates, only DateModified

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "DataEnrichment", true, nil)
}

func TestUpdateProductUseCase_Execute_NotFound(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-UPDATE-NOT-FOUND-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "NotFound", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createUpdateTestUseCase(businessType, false)

	req := &productpb.UpdateProductRequest{
		Data: &productpb.Product{
			Id:   "non-existent-product-id",
			Name: "Valid Product Name",
		},
	}

	_, err := useCase.Execute(ctx, req)

	testutil.AssertError(t, err)

	// The error message should indicate the product was not found
	// This will be handled by the repository layer
	testutil.AssertNonEmptyString(t, err.Error(), "error message")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "NotFound", false, err)
}

func TestUpdateProductUseCase_Execute_MinimalValidData(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-UPDATE-BOUNDARY-MINIMAL-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "MinimalValidData", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createUpdateTestUseCase(businessType, false)

	req := &productpb.UpdateProductRequest{
		Data: &productpb.Product{
			Id:   "subject-math",
			Name: "AB", // Minimal valid name (2 characters)
		},
	}

	response, err := useCase.Execute(ctx, req)

	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, response, "response")
	testutil.AssertTrue(t, response.Success, "success")

	updatedProduct := response.Data[0]
	testutil.AssertStringEqual(t, "Ab", updatedProduct.Name, "normalized name") // strings.Title applied

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "MinimalValidData", true, nil)
}

func TestUpdateProductUseCase_Execute_MaxValidData(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-UPDATE-BOUNDARY-MAXIMAL-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "MaxValidData", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createUpdateTestUseCase(businessType, false)

	// Create exactly 100 characters for name (actual limit from implementation)
	maxName := testutil.GenerateDefaultLongString(100)

	// Create exactly 1000 characters for description
	maxDescription := strings.Repeat("B", 1000)

	req := &productpb.UpdateProductRequest{
		Data: &productpb.Product{
			Id:          "subject-math",
			Name:        maxName,
			Description: &maxDescription,
		},
	}

	response, err := useCase.Execute(ctx, req)

	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, response, "response")
	testutil.AssertTrue(t, response.Success, "success")

	updatedProduct := response.Data[0]
	testutil.AssertFieldLength(t, 100, len(updatedProduct.Name), "name")

	testutil.AssertFieldSet(t, updatedProduct.Description, "description")
	if updatedProduct.Description != nil {
		testutil.AssertFieldLength(t, 1000, len(*updatedProduct.Description), "description")
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "MaxValidData", true, nil)
}

func TestUpdateProductUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-UPDATE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service using factory
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockRepo := mockProduct.NewMockProductRepository(businessType)

	repositories := UpdateProductRepositories{
		Product: mockRepo,
	}

	services := testServiceFactoryForUpdate.CreateServicesWithFailingTransaction(true)

	useCase := NewUpdateProductUseCase(repositories, services)

	req := &productpb.UpdateProductRequest{
		Data: &productpb.Product{
			Id:   "subject-math",
			Name: "Test transaction failure",
		},
	}

	_, err := useCase.Execute(ctx, req)

	testutil.AssertTransactionError(t, err)

	// Check for infrastructure mock transaction error message (with error wrapping context)
	expectedError := "transaction error [TRANSACTION_GENERAL] during run_in_transaction: mock run in transaction failed"
	testutil.AssertStringEqual(t, expectedError, err.Error(), "transaction error message")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}

func TestUpdateProductUseCase_Execute_EducationDomainSpecific(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-UPDATE-INTEGRATION-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "EducationDomainSpecific", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createUpdateTestUseCase(businessType, false)

	// Test updating education-specific products using real product IDs
	testCases := []struct {
		name        string
		id          string
		updatedName string
		description string
	}{
		{
			name:        "Update Mathematics Subject",
			id:          "subject-math",
			updatedName: "Advanced Mathematics",
			description: "Updated advanced mathematics curriculum covering algebra, geometry, and calculus",
		},
		{
			name:        "Update Science Subject",
			id:          "subject-science",
			updatedName: "Elementary Science",
			description: "Updated comprehensive science program covering physics, chemistry, and biology",
		},
		{
			name:        "Update English Subject",
			id:          "subject-english",
			updatedName: "Advanced English Literature",
			description: "Updated English language arts including literature, writing, and communication skills",
		},
		{
			name:        "Update Music Subject",
			id:          "subject-music",
			updatedName: "Music Theory And Practice",
			description: "Updated music education covering theory, performance, and composition",
		},
		{
			name:        "Update Physical Education Subject",
			id:          "subject-pe",
			updatedName: "Physical Education And Wellness",
			description: "Updated physical education focusing on fitness, sports, and health education",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &productpb.UpdateProductRequest{
				Data: &productpb.Product{
					Id:          tc.id,
					Name:        tc.updatedName,
					Description: &tc.description,
				},
			}

			response, err := useCase.Execute(ctx, req)

			testutil.AssertOperationSuccess(t, err, "update "+tc.name)

			updatedProduct := response.Data[0]

			// Verify education-specific product data was updated
			testutil.AssertStringEqual(t, tc.id, updatedProduct.Id, "product ID for "+tc.name)

			// Name should be updated with proper capitalization
			testutil.AssertNonEmptyString(t, updatedProduct.Name, "updated education-specific product name for "+tc.name)

			testutil.AssertFieldSet(t, updatedProduct.Description, "updated description for educational product")
			if updatedProduct.Description != nil {
				testutil.AssertGreaterThan(t, len(*updatedProduct.Description), 29, "comprehensive description length")
			}

			// Verify the product remains active after update
			testutil.AssertTrue(t, updatedProduct.Active, "educational product active status after update")

			// Verify audit fields were updated
			testutil.AssertFieldSet(t, updatedProduct.DateModified, "DateModified for educational product")
			testutil.AssertFieldSet(t, updatedProduct.DateModifiedString, "DateModifiedString for educational product")
		})
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "EducationDomainSpecific", true, nil)
}

//go:build mock_db && mock_auth

// Package collection provides table-driven tests for the collection deletion use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, and error handling.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestDeleteCollectionUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-COLLECTION-DELETE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PRODUCT-COLLECTION-DELETE-NOT-FOUND-v1.0: NonExistentCollection
//   - ESPYNA-TEST-PRODUCT-COLLECTION-DELETE-VALIDATION-EMPTY-ID-v1.0: EmptyCollectionId
//   - ESPYNA-TEST-PRODUCT-COLLECTION-DELETE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PRODUCT-COLLECTION-DELETE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PRODUCT-COLLECTION-DELETE-TRANSACTION-SUCCESS-v1.0: WithTransactionSuccess
//   - ESPYNA-TEST-PRODUCT-COLLECTION-DELETE-AUTHORIZATION-DENIED-v1.0: AuthorizationDenied
//   - ESPYNA-TEST-PRODUCT-COLLECTION-DELETE-TRANSACTION-FAILURE-v1.0: WithTransactionFailure
//   - ESPYNA-TEST-PRODUCT-COLLECTION-DELETE-INTEGRATION-v1.0: MultipleValidCollections
//   - ESPYNA-TEST-PRODUCT-COLLECTION-DELETE-BUSINESS-LOGIC-v1.0: BusinessLogicValidation
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/product_collection.json
//   - Mock data: packages/copya/data/{businessType}/product_collection.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/product_collection.json
package collection

import (
	"context"
	"fmt"
	"strings"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockDb "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/product"
	collectionpb "leapfor.xyz/esqyma/golang/v1/domain/product/collection"
)

// Type alias for delete collection test cases
type DeleteCollectionTestCase = testutil.GenericTestCase[*collectionpb.DeleteCollectionRequest, *collectionpb.DeleteCollectionResponse]

func createDeleteTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *DeleteCollectionUseCase {
	mockCollectionRepo := product.NewMockCollectionRepository(businessType)

	repositories := DeleteCollectionRepositories{
		Collection: mockCollectionRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := DeleteCollectionServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewDeleteCollectionUseCase(repositories, services)
}

func createDeleteTestUseCaseWithFailingTransaction(businessType string, shouldAuthorize bool) *DeleteCollectionUseCase {
	mockCollectionRepo := product.NewMockCollectionRepository(businessType)

	repositories := DeleteCollectionRepositories{
		Collection: mockCollectionRepo,
	}

	standardServices := testutil.CreateStandardServices(false, shouldAuthorize)
	// Override with failing transaction service
	standardServices.TransactionService = mockDb.NewFailingMockTransactionService()
	services := DeleteCollectionServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewDeleteCollectionUseCase(repositories, services)
}

func TestDeleteCollectionUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	commonDataResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection", "Collection_CommonData")
	testutil.AssertTestCaseLoad(t, err, "Collection_CommonData")

	testCases := []DeleteCollectionTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-DELETE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.DeleteCollectionRequest {
				return &collectionpb.DeleteCollectionRequest{
					Data: &collectionpb.Collection{
						Id: commonDataResolver.MustGetString("primaryCollectionId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *collectionpb.DeleteCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "successful deletion")
			},
		},
		{
			Name:     "NonExistentCollection",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-DELETE-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.DeleteCollectionRequest {
				return &collectionpb.DeleteCollectionRequest{
					Data: &collectionpb.Collection{
						Id: commonDataResolver.MustGetString("nonExistentId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection.errors.not_found",
			ErrorTags:      map[string]any{"collectionId": commonDataResolver.MustGetString("nonExistentId")},
			Assertions: func(t *testing.T, response *collectionpb.DeleteCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
				testutil.AssertNil(t, response, "response for non-existent collection")
			},
		},
		{
			Name:     "EmptyCollectionId",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-DELETE-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.DeleteCollectionRequest {
				return &collectionpb.DeleteCollectionRequest{
					Data: &collectionpb.Collection{
						Id: "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection.validation.id_required",
			Assertions: func(t *testing.T, response *collectionpb.DeleteCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty collection ID")
				testutil.AssertNil(t, response, "response for invalid input")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-DELETE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.DeleteCollectionRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection.validation.request_required",
			Assertions: func(t *testing.T, response *collectionpb.DeleteCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
				testutil.AssertNil(t, response, "response for nil request")
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-DELETE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.DeleteCollectionRequest {
				return &collectionpb.DeleteCollectionRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection.validation.request_required",
			Assertions: func(t *testing.T, response *collectionpb.DeleteCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
				testutil.AssertNil(t, response, "response for nil data")
			},
		},
		{
			Name:     "WithTransactionSuccess",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-DELETE-TRANSACTION-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.DeleteCollectionRequest {
				return &collectionpb.DeleteCollectionRequest{
					Data: &collectionpb.Collection{
						Id: commonDataResolver.MustGetString("secondaryProductCollectionId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *collectionpb.DeleteCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "successful deletion with transaction")
			},
		},
		{
			Name:     "AuthorizationDenied",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-DELETE-AUTHORIZATION-DENIED-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.DeleteCollectionRequest {
				return &collectionpb.DeleteCollectionRequest{
					Data: &collectionpb.Collection{
						Id: commonDataResolver.MustGetString("businessRulesProductCollectionId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "collection.errors.authorization_failed",
			Assertions: func(t *testing.T, response *collectionpb.DeleteCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			// Set test code and log execution start
			testutil.SetTestCode(t, tc.TestCode)
			testutil.LogTestExecution(t, tc.TestCode, tc.Name, tc.ExpectSuccess)

			ctx := testutil.CreateTestContext()
			businessType := testutil.GetTestBusinessType()
			useCase := createDeleteTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

			req := tc.SetupRequest(t, businessType)
			response, err := useCase.Execute(ctx, req)

			// Determine actual success/failure
			actualSuccess := err == nil && tc.ExpectSuccess

			if tc.ExpectSuccess {
				testutil.AssertNoError(t, err)
				testutil.AssertNotNil(t, response, "response")
			} else {
				testutil.AssertError(t, err)
				if tc.ExpectedError != "" {
					if tc.ErrorTags != nil {
						testutil.AssertTranslatedErrorWithTags(t, err, tc.ExpectedError, tc.ErrorTags, useCase.services.TranslationService, ctx)
					} else {
						testutil.AssertTranslatedError(t, err, tc.ExpectedError, useCase.services.TranslationService, ctx)
					}
				}
			}

			if tc.Assertions != nil {
				tc.Assertions(t, response, err, useCase, ctx)
			}

			// Log test completion with result
			testutil.LogTestResult(t, tc.TestCode, tc.Name, actualSuccess, err)
		})
	}
}

func TestDeleteCollectionUseCase_Execute_WithTransaction_Failure(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-COLLECTION-DELETE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "WithTransactionFailure", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createDeleteTestUseCaseWithFailingTransaction(businessType, true)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection", "Collection_CommonData")
	testutil.AssertTestCaseLoad(t, err, "Collection_CommonData")

	req := &collectionpb.DeleteCollectionRequest{
		Data: &collectionpb.Collection{
			Id: resolver.MustGetString("thirdProductCollectionId"),
		},
	}

	_, err2 := useCase.Execute(ctx, req)

	testutil.AssertTransactionError(t, err2)

	expectedError := "Transaction execution failed: transaction error [TRANSACTION_GENERAL] during run_in_transaction: mock run in transaction failed"
	testutil.AssertStringEqual(t, expectedError, err2.Error(), "error message")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "WithTransactionFailure", false, err2)
}

func TestDeleteCollectionUseCase_Execute_MultipleValidCollections(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-COLLECTION-DELETE-INTEGRATION-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "MultipleValidCollections", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection", "Collection_CommonData")
	testutil.AssertTestCaseLoad(t, err, "Collection_CommonData")

	collectionIds := []string{
		resolver.MustGetString("primaryProductCollectionId"),
		resolver.MustGetString("secondaryProductCollectionId"),
		resolver.MustGetString("thirdProductCollectionId"),
	}

	// Create test cases dynamically based on available collection IDs
	testCases := make([]struct {
		name         string
		collectionId string
		expectError  bool
	}, len(collectionIds))

	for i, collectionId := range collectionIds {
		testCases[i] = struct {
			name         string
			collectionId string
			expectError  bool
		}{
			name:         fmt.Sprintf("Delete collection %d (%s)", i+1, collectionId),
			collectionId: collectionId,
			expectError:  false,
		}
	}

	useCase := createDeleteTestUseCaseWithAuth(businessType, false, true)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			req := &collectionpb.DeleteCollectionRequest{
				Data: &collectionpb.Collection{
					Id: tc.collectionId,
				},
			}

			response, err := useCase.Execute(ctx, req)

			if tc.expectError {
				testutil.AssertError(t, err)
			} else {
				testutil.AssertNoError(t, err)

				testutil.AssertNotNil(t, response, "response")

				testutil.AssertTrue(t, response.Success, "successful deletion")
			}
		})
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "MultipleValidCollections", true, nil)
}

func TestDeleteCollectionUseCase_Execute_BusinessLogicValidation(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-COLLECTION-DELETE-BUSINESS-LOGIC-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "BusinessLogicValidation", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createDeleteTestUseCaseWithAuth(businessType, false, true)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection", "Collection_CommonData")
	testutil.AssertTestCaseLoad(t, err, "Collection_CommonData")

	tests := []struct {
		name           string
		collectionData *collectionpb.Collection
		expectError    bool
		expectedError  string
	}{
		{
			name: "Valid collection ID",
			collectionData: &collectionpb.Collection{
				Id: resolver.MustGetString("primaryProductCollectionId"),
			},
			expectError: false,
		},
		{
			name: "Invalid collection ID format",
			collectionData: &collectionpb.Collection{
				Id: "invalid-id-format",
			},
			expectError: true,
		},
		{
			name: "Extremely long collection ID",
			collectionData: &collectionpb.Collection{
				Id: fmt.Sprintf("collection-%s", strings.Repeat("A", 300)), // Create long string for validation test
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			req := &collectionpb.DeleteCollectionRequest{
				Data: tt.collectionData,
			}

			response, err := useCase.Execute(ctx, req)

			if tt.expectError {
				testutil.AssertError(t, err)
				if tt.expectedError != "" {
					testutil.AssertStringEqual(t, tt.expectedError, err.Error(), "error message")
				}
				testutil.AssertNil(t, response, "response")
			} else {
				testutil.AssertNoError(t, err)
				testutil.AssertNotNil(t, response, "response")
			}
		})
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "BusinessLogicValidation", true, nil)
}

//go:build mock_db && mock_auth

// Package product_line provides table-driven tests for the product line read use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, not found, and boundary conditions.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestReadProductLineUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-READ-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-READ-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-READ-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-READ-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-READ-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-READ-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-READ-NOT-FOUND-v1.0: NotFound
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-READ-TRANSACTION-FAILURE-v1.0: TransactionFailure
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/product_line.json
//   - Mock data: packages/copya/data/{businessType}/product_line.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/product_line.json
package product_line

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	mockDb "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock"
	mockProduct "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/product"
	productlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_line"
	copyatestutil "leapfor.xyz/copya/golang/testutil"
)

// Type alias for read product line test cases
type ReadProductLineTestCase = testutil.GenericTestCase[*productlinepb.ReadProductLineRequest, *productlinepb.ReadProductLineResponse]

func createReadTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *ReadProductLineUseCase {
	mockProductLineRepo := mockProduct.NewMockProductLineRepository(businessType)

	repositories := ReadProductLineRepositories{
		ProductLine: mockProductLineRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := ReadProductLineServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewReadProductLineUseCase(repositories, services)
}

func TestReadProductLineUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	readNotFoundResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_line", "ReadProductLine_NotFound")
	testutil.AssertTestCaseLoad(t, err, "ReadProductLine_NotFound")

	testCases := []ReadProductLineTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-READ-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productlinepb.ReadProductLineRequest {
				return &productlinepb.ReadProductLineRequest{
					Data: &productlinepb.ProductLine{
						Id: "product-line-001", // Pre-loaded in mock data
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productlinepb.ReadProductLineResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				readLine := response.Data[0]
				testutil.AssertStringEqual(t, "product-line-001", readLine.Id, "product line ID")
				testutil.AssertStringEqual(t, "subject-math", readLine.ProductId, "product ID")
				testutil.AssertStringEqual(t, "line-g1-seahorse", readLine.LineId, "line ID")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-READ-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productlinepb.ReadProductLineRequest {
				return &productlinepb.ReadProductLineRequest{
					Data: &productlinepb.ProductLine{
						Id: "product-line-001",
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productlinepb.ReadProductLineResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				readLine := response.Data[0]
				testutil.AssertStringEqual(t, "product-line-001", readLine.Id, "product line ID")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-READ-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productlinepb.ReadProductLineRequest {
				return &productlinepb.ReadProductLineRequest{
					Data: &productlinepb.ProductLine{
						Id: "product-line-001",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "product_line.errors.authorization_failed",
			Assertions: func(t *testing.T, response *productlinepb.ReadProductLineResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-READ-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productlinepb.ReadProductLineRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_line.validation.request_required",
			Assertions: func(t *testing.T, response *productlinepb.ReadProductLineResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-READ-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productlinepb.ReadProductLineRequest {
				return &productlinepb.ReadProductLineRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_line.validation.data_required",
			Assertions: func(t *testing.T, response *productlinepb.ReadProductLineResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-READ-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productlinepb.ReadProductLineRequest {
				return &productlinepb.ReadProductLineRequest{
					Data: &productlinepb.ProductLine{
						Id: "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_line.validation.id_required",
			Assertions: func(t *testing.T, response *productlinepb.ReadProductLineResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "NotFound",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-READ-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productlinepb.ReadProductLineRequest {
				return &productlinepb.ReadProductLineRequest{
					Data: &productlinepb.ProductLine{
						Id: readNotFoundResolver.MustGetString("nonExistentProductLineId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_line.errors.not_found",
			ErrorTags:      map[string]any{"productLineId": readNotFoundResolver.MustGetString("nonExistentProductLineId")},
			Assertions: func(t *testing.T, response *productlinepb.ReadProductLineResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "not found")
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
			useCase := createReadTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

func TestReadProductLineUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-READ-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockProductLineRepo := mockProduct.NewMockProductLineRepository(businessType)

	repositories := ReadProductLineRepositories{
		ProductLine: mockProductLineRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := ReadProductLineServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewReadProductLineUseCase(repositories, services)

	req := &productlinepb.ReadProductLineRequest{
		Data: &productlinepb.ProductLine{
			Id: "product-line-001",
		},
	}

	_, err := useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}

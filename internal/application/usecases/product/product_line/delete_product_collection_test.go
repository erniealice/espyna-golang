//go:build mock_db && mock_auth

// Package product_line provides table-driven tests for the product line deletion use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, not found, and transaction failures.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestDeleteProductLineUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-DELETE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-DELETE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-DELETE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-DELETE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-DELETE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-DELETE-NOT-FOUND-v1.0: NotFound
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-DELETE-TRANSACTION-FAILURE-v1.0: TransactionFailure
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

// Type alias for delete product line test cases
type DeleteProductLineTestCase = testutil.GenericTestCase[*productlinepb.DeleteProductLineRequest, *productlinepb.DeleteProductLineResponse]

func createDeleteTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *DeleteProductLineUseCase {
	mockProductLineRepo := mockProduct.NewMockProductLineRepository(businessType)

	repositories := DeleteProductLineRepositories{
		ProductLine: mockProductLineRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := DeleteProductLineServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewDeleteProductLineUseCase(repositories, services)
}

func TestDeleteProductLineUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	deleteSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_line", "DeleteProductLine_Success")
	testutil.AssertTestCaseLoad(t, err, "DeleteProductLine_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_line", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	deleteNotFoundResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_line", "DeleteProductLine_NotFound")
	testutil.AssertTestCaseLoad(t, err, "DeleteProductLine_NotFound")

	testCases := []DeleteProductLineTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-DELETE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productlinepb.DeleteProductLineRequest {
				return &productlinepb.DeleteProductLineRequest{
					Data: &productlinepb.ProductLine{
						Id:           deleteSuccessResolver.MustGetString("validProductLineId"),
						ProductId:    deleteSuccessResolver.MustGetString("validProductId"),
						LineId: deleteSuccessResolver.MustGetString("validLineId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productlinepb.DeleteProductLineResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-DELETE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productlinepb.DeleteProductLineRequest {
				return &productlinepb.DeleteProductLineRequest{
					Data: &productlinepb.ProductLine{
						Id:           deleteSuccessResolver.MustGetString("validProductLineId"),
						ProductId:    deleteSuccessResolver.MustGetString("validProductId"),
						LineId: deleteSuccessResolver.MustGetString("validLineId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productlinepb.DeleteProductLineResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-DELETE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productlinepb.DeleteProductLineRequest {
				return &productlinepb.DeleteProductLineRequest{
					Data: &productlinepb.ProductLine{
						Id: authorizationUnauthorizedResolver.MustGetString("unauthorizedProductLineId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "product_line.errors.authorization_failed",
			Assertions: func(t *testing.T, response *productlinepb.DeleteProductLineResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-DELETE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productlinepb.DeleteProductLineRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_line.validation.request_required",
			Assertions: func(t *testing.T, response *productlinepb.DeleteProductLineResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-DELETE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productlinepb.DeleteProductLineRequest {
				return &productlinepb.DeleteProductLineRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_line.validation.data_required",
			Assertions: func(t *testing.T, response *productlinepb.DeleteProductLineResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "NotFound",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-DELETE-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productlinepb.DeleteProductLineRequest {
				return &productlinepb.DeleteProductLineRequest{
					Data: &productlinepb.ProductLine{
						Id:           deleteNotFoundResolver.MustGetString("nonExistentProductLineId"),
						ProductId:    deleteNotFoundResolver.MustGetString("validProductId"),
						LineId: deleteNotFoundResolver.MustGetString("validLineId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_line.errors.not_found",
			ErrorTags:      map[string]any{"productLineId": deleteNotFoundResolver.MustGetString("nonExistentProductLineId")},
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

func TestDeleteProductLineUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-DELETE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockProductLineRepo := mockProduct.NewMockProductLineRepository(businessType)

	repositories := DeleteProductLineRepositories{
		ProductLine: mockProductLineRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := DeleteProductLineServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewDeleteProductLineUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_line", "DeleteProductLine_Success")
	testutil.AssertTestCaseLoad(t, err, "DeleteProductLine_Success")

	req := &productlinepb.DeleteProductLineRequest{
		Data: &productlinepb.ProductLine{
			Id:           resolver.MustGetString("validProductLineId"),
			ProductId:    resolver.MustGetString("validProductId"),
			LineId: resolver.MustGetString("validLineId"),
		},
	}

	_, err = useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}

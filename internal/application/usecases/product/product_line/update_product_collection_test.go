//go:build mock_db && mock_auth

// Package product_line provides table-driven tests for the product line update use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, not found, business rule validation, and boundary conditions.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestUpdateProductLineUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-VALIDATION-EMPTY-PRODUCT-ID-v1.0: EmptyProductId
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-VALIDATION-EMPTY-COLLECTION-ID-v1.0: EmptyLineId
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-VALIDATION-PRODUCT-ID-TOO-SHORT-v1.0: ProductIdTooShort
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-VALIDATION-COLLECTION-ID-TOO-SHORT-v1.0: LineIdTooShort
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-NOT-FOUND-v1.0: NotFound
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-TRANSACTION-FAILURE-v1.0: TransactionFailure
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

// Type alias for update product line test cases
type UpdateProductLineTestCase = testutil.GenericTestCase[*productlinepb.UpdateProductLineRequest, *productlinepb.UpdateProductLineResponse]

func createUpdateTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *UpdateProductLineUseCase {
	mockProductLineRepo := mockProduct.NewMockProductLineRepository(businessType)
	mockProductRepo := mockProduct.NewMockProductRepository(businessType)
	mockLineRepo := mockProduct.NewMockLineRepository(businessType)

	repositories := UpdateProductLineRepositories{
		ProductLine: mockProductLineRepo,
		Product:           mockProductRepo,
		Line:        mockLineRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := UpdateProductLineServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewUpdateProductLineUseCase(repositories, services)
}

func TestUpdateProductLineUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	updateSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_line", "UpdateProductLine_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdateProductLine_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_line", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	updateNotFoundResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_line", "UpdateProductLine_NotFound")
	testutil.AssertTestCaseLoad(t, err, "UpdateProductLine_NotFound")

	testCases := []UpdateProductLineTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productlinepb.UpdateProductLineRequest {
				return &productlinepb.UpdateProductLineRequest{
					Data: &productlinepb.ProductLine{
						Id:           updateSuccessResolver.MustGetString("validProductLineId"),
						ProductId:    updateSuccessResolver.MustGetString("validProductId"),
						LineId: updateSuccessResolver.MustGetString("validLineId"),
						SortOrder:    int32(updateSuccessResolver.MustGetInt("updatedSortOrder")),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productlinepb.UpdateProductLineResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				updatedLine := response.Data[0]
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("validProductId"), updatedLine.ProductId, "product ID")
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("validLineId"), updatedLine.LineId, "line ID")
				testutil.AssertFieldSet(t, updatedLine.DateModified, "DateModified")
				testutil.AssertFieldSet(t, updatedLine.DateModifiedString, "DateModifiedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productlinepb.UpdateProductLineRequest {
				return &productlinepb.UpdateProductLineRequest{
					Data: &productlinepb.ProductLine{
						Id:           updateSuccessResolver.MustGetString("validProductLineId"),
						ProductId:    updateSuccessResolver.MustGetString("validProductId"),
						LineId: updateSuccessResolver.MustGetString("validLineId"),
						SortOrder:    int32(updateSuccessResolver.MustGetInt("updatedSortOrder")),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productlinepb.UpdateProductLineResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				updatedLine := response.Data[0]
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("validProductId"), updatedLine.ProductId, "product ID")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productlinepb.UpdateProductLineRequest {
				return &productlinepb.UpdateProductLineRequest{
					Data: &productlinepb.ProductLine{
						Id:           authorizationUnauthorizedResolver.MustGetString("unauthorizedProductLineId"),
						ProductId:    authorizationUnauthorizedResolver.MustGetString("unauthorizedProductId"),
						LineId: authorizationUnauthorizedResolver.MustGetString("unauthorizedLineId"),
						SortOrder:    int32(authorizationUnauthorizedResolver.MustGetInt("unauthorizedSortOrder")),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "product_line.errors.authorization_failed",
			Assertions: func(t *testing.T, response *productlinepb.UpdateProductLineResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productlinepb.UpdateProductLineRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_line.validation.request_required",
			Assertions: func(t *testing.T, response *productlinepb.UpdateProductLineResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productlinepb.UpdateProductLineRequest {
				return &productlinepb.UpdateProductLineRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_line.validation.data_required",
			Assertions: func(t *testing.T, response *productlinepb.UpdateProductLineResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productlinepb.UpdateProductLineRequest {
				return &productlinepb.UpdateProductLineRequest{
					Data: &productlinepb.ProductLine{
						Id:           "",
						ProductId:    updateSuccessResolver.MustGetString("validProductId"),
						LineId: updateSuccessResolver.MustGetString("validLineId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_line.validation.id_required",
			Assertions: func(t *testing.T, response *productlinepb.UpdateProductLineResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "EmptyProductId",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-VALIDATION-EMPTY-PRODUCT-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productlinepb.UpdateProductLineRequest {
				return &productlinepb.UpdateProductLineRequest{
					Data: &productlinepb.ProductLine{
						Id:           "test-id",
						ProductId:    "",
						LineId: "line-g1-seahorse",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_line.validation.product_id_required",
			Assertions: func(t *testing.T, response *productlinepb.UpdateProductLineResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty product ID")
			},
		},
		{
			Name:     "EmptyLineId",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-VALIDATION-EMPTY-COLLECTION-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productlinepb.UpdateProductLineRequest {
				return &productlinepb.UpdateProductLineRequest{
					Data: &productlinepb.ProductLine{
						Id:           "test-id",
						ProductId:    "subject-math",
						LineId: "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_line.validation.line_id_required",
			Assertions: func(t *testing.T, response *productlinepb.UpdateProductLineResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty line ID")
			},
		},
		{
			Name:     "ProductIdTooShort",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-VALIDATION-PRODUCT-ID-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productlinepb.UpdateProductLineRequest {
				return &productlinepb.UpdateProductLineRequest{
					Data: &productlinepb.ProductLine{
						Id:           "product-line-003",
						ProductId:    "abc", // Less than 5 characters
						LineId: "line-g1-seahorse",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_line.validation.product_id_min_length",
			Assertions: func(t *testing.T, response *productlinepb.UpdateProductLineResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "product ID too short")
			},
		},
		{
			Name:     "LineIdTooShort",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-VALIDATION-COLLECTION-ID-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productlinepb.UpdateProductLineRequest {
				return &productlinepb.UpdateProductLineRequest{
					Data: &productlinepb.ProductLine{
						Id:           updateSuccessResolver.MustGetString("validProductLineId"),
						ProductId:    updateSuccessResolver.MustGetString("validProductId"),
						LineId: "a", // Less than 2 characters
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_line.validation.line_id_min_length",
			Assertions: func(t *testing.T, response *productlinepb.UpdateProductLineResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "line ID too short")
			},
		},
		{
			Name:     "NotFound",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productlinepb.UpdateProductLineRequest {
				return &productlinepb.UpdateProductLineRequest{
					Data: &productlinepb.ProductLine{
						Id:           updateNotFoundResolver.MustGetString("invalidProductLineId"),
						ProductId:    updateNotFoundResolver.MustGetString("validProductId"),
						LineId: updateNotFoundResolver.MustGetString("validLineId"),
						SortOrder:    int32(updateNotFoundResolver.MustGetInt("updatedSortOrder")),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_line.errors.not_found",
			ErrorTags:      map[string]any{"productLineId": updateNotFoundResolver.MustGetString("invalidProductLineId")},
			Assertions: func(t *testing.T, response *productlinepb.UpdateProductLineResponse, err error, useCase interface{}, ctx context.Context) {
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
			useCase := createUpdateTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

func TestUpdateProductLineUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockProductLineRepo := mockProduct.NewMockProductLineRepository(businessType)
	mockProductRepo := mockProduct.NewMockProductRepository(businessType)
	mockLineRepo := mockProduct.NewMockLineRepository(businessType)

	repositories := UpdateProductLineRepositories{
		ProductLine: mockProductLineRepo,
		Product:           mockProductRepo,
		Line:        mockLineRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := UpdateProductLineServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewUpdateProductLineUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_line", "UpdateProductLine_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdateProductLine_Success")

	req := &productlinepb.UpdateProductLineRequest{
		Data: &productlinepb.ProductLine{
			Id:           resolver.MustGetString("validProductLineId"),
			ProductId:    resolver.MustGetString("validProductId"),
			LineId: resolver.MustGetString("validLineId"),
			SortOrder:    int32(resolver.MustGetInt("updatedSortOrder")),
		},
	}

	_, err = useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}

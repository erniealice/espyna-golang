//go:build mock_db && mock_auth

// Package product_attribute provides table-driven tests for the product attribute reading use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, not found errors, and boundary conditions.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestReadProductAttributeUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-READ-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-READ-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-READ-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-READ-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-READ-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-READ-NOT-FOUND-v1.0: NotFound
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-READ-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-READ-VALIDATION-v1.0: ValidationError
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-READ-TRANSACTION-FAILURE-v1.0: TransactionFailure
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/product_attribute.json
//   - Mock data: packages/copya/data/{businessType}/product_attribute.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/product_attribute.json
package product_attribute

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockDb "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/product"
	productattributepb "leapfor.xyz/esqyma/golang/v1/domain/product/product_attribute"
)

// Type alias for read product attribute test cases
type ReadProductAttributeTestCase = testutil.GenericTestCase[*productattributepb.ReadProductAttributeRequest, *productattributepb.ReadProductAttributeResponse]

func createReadTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *ReadProductAttributeUseCase {
	mockProductAttributeRepo := product.NewMockProductAttributeRepository(businessType)

	repositories := ReadProductAttributeRepositories{
		ProductAttribute: mockProductAttributeRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := ReadProductAttributeServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewReadProductAttributeUseCase(repositories, services)
}

func TestReadProductAttributeUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	readNotFoundResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_attribute", "ReadProductAttribute_NotFound")
	testutil.AssertTestCaseLoad(t, err, "ReadProductAttribute_NotFound")

	testCases := []ReadProductAttributeTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-READ-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.ReadProductAttributeRequest {
				return &productattributepb.ReadProductAttributeRequest{
					Data: &productattributepb.ProductAttribute{
						Id: "product-attr-001",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productattributepb.ReadProductAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")

				found := response.Data[0]
				testutil.AssertStringEqual(t, "product-attr-001", found.Id, "product attribute ID")
				testutil.AssertNonEmptyString(t, found.ProductId, "product ID")
				testutil.AssertNonEmptyString(t, found.AttributeId, "attribute ID")
				testutil.AssertNonEmptyString(t, found.Value, "value")
				testutil.AssertFieldSet(t, found.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, found.DateCreatedString, "DateCreatedString")

				// Verify expected pre-loaded values from mock data
				testutil.AssertStringEqual(t, "subject-math", found.ProductId, "expected product ID")
				testutil.AssertStringEqual(t, "difficulty-level", found.AttributeId, "expected attribute ID")
				testutil.AssertStringEqual(t, "Intermediate", found.Value, "expected value")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-READ-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.ReadProductAttributeRequest {
				return &productattributepb.ReadProductAttributeRequest{
					Data: &productattributepb.ProductAttribute{
						Id: "product-attr-001",
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productattributepb.ReadProductAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				found := response.Data[0]
				testutil.AssertStringEqual(t, "product-attr-001", found.Id, "product attribute ID")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-READ-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.ReadProductAttributeRequest {
				return &productattributepb.ReadProductAttributeRequest{
					Data: &productattributepb.ProductAttribute{
						Id: "product-attr-001",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "product_attribute.errors.authorization_failed",
			Assertions: func(t *testing.T, response *productattributepb.ReadProductAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-READ-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.ReadProductAttributeRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_attribute.validation.request_required",
			Assertions: func(t *testing.T, response *productattributepb.ReadProductAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-READ-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.ReadProductAttributeRequest {
				return &productattributepb.ReadProductAttributeRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_attribute.validation.data_required",
			Assertions: func(t *testing.T, response *productattributepb.ReadProductAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "NotFound",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-READ-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.ReadProductAttributeRequest {
				return &productattributepb.ReadProductAttributeRequest{
					Data: &productattributepb.ProductAttribute{
						Id: readNotFoundResolver.MustGetString("nonExistentProductAttributeId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_attribute.errors.not_found",
			ErrorTags:      map[string]any{"productAttributeId": readNotFoundResolver.MustGetString("nonExistentProductAttributeId")},
			Assertions: func(t *testing.T, response *productattributepb.ReadProductAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "not found")
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-READ-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.ReadProductAttributeRequest {
				return &productattributepb.ReadProductAttributeRequest{
					Data: &productattributepb.ProductAttribute{
						Id: "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_attribute.validation.id_required",
			Assertions: func(t *testing.T, response *productattributepb.ReadProductAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "ValidationError",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-READ-VALIDATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.ReadProductAttributeRequest {
				return &productattributepb.ReadProductAttributeRequest{
					Data: &productattributepb.ProductAttribute{
						Id: readNotFoundResolver.MustGetString("invalidIdFormat"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_attribute.errors.not_found",
			ErrorTags:      map[string]any{"productAttributeId": readNotFoundResolver.MustGetString("invalidIdFormat")},
			Assertions: func(t *testing.T, response *productattributepb.ReadProductAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "validation error")
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

func TestReadProductAttributeUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-READ-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockProductAttributeRepo := product.NewMockProductAttributeRepository(businessType)

	repositories := ReadProductAttributeRepositories{
		ProductAttribute: mockProductAttributeRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := ReadProductAttributeServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewReadProductAttributeUseCase(repositories, services)

	req := &productattributepb.ReadProductAttributeRequest{
		Data: &productattributepb.ProductAttribute{
			Id: "product-attr-001",
		},
	}

	_, err := useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}

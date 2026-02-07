//go:build mock_db && mock_auth

// Package product_plan provides table-driven tests for the product plan update use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestUpdateProductPlanUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-UPDATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-UPDATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-UPDATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-UPDATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-UPDATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-UPDATE-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-UPDATE-VALIDATION-EMPTY-NAME-v1.0: EmptyName
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-UPDATE-VALIDATION-EMPTY-PRODUCT-ID-v1.0: EmptyProductId
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-UPDATE-VALIDATION-NAME-TOO-SHORT-v1.0: NameTooShort
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-UPDATE-VALIDATION-NAME-TOO-LONG-v1.0: NameTooLong
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-UPDATE-VALIDATION-PRODUCT-ID-TOO-SHORT-v1.0: ProductIdTooShort
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-UPDATE-NOT-FOUND-v1.0: NotFound
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-UPDATE-VALIDATION-INVALID-PRODUCT-ID-v1.0: InvalidProductId
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/product_plan.json
//   - Mock data: packages/copya/data/{businessType}/product_plan.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/product_plan.json
package product_plan

import (
	"context"
	"strings"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockProduct "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/product"
	productplanpb "leapfor.xyz/esqyma/golang/v1/domain/product/product_plan"
)

// Type alias for update product plan test cases
type UpdateProductPlanTestCase = testutil.GenericTestCase[*productplanpb.UpdateProductPlanRequest, *productplanpb.UpdateProductPlanResponse]

func createUpdateTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *UpdateProductPlanUseCase {
	mockProductPlanRepo := mockProduct.NewMockProductPlanRepository(businessType)
	mockProductRepo := mockProduct.NewMockProductRepository(businessType)

	repositories := UpdateProductPlanRepositories{
		ProductPlan: mockProductPlanRepo,
		Product:     mockProductRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := UpdateProductPlanServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewUpdateProductPlanUseCase(repositories, services)
}

func TestUpdateProductPlanUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	updateSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_plan", "UpdateProductPlan_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdateProductPlan_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_plan", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	validationErrorEmptyNameResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_plan", "ValidationError_EmptyName")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyName")

	validationErrorEmptyProductIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_plan", "ValidationError_EmptyProductId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyProductId")

	validationErrorNameTooShortResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_plan", "ValidationError_NameTooShort")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_NameTooShort")

	validationErrorNameTooLongResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_plan", "ValidationError_NameTooLongGenerated")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_NameTooLongGenerated")

	testCases := []UpdateProductPlanTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-UPDATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.UpdateProductPlanRequest {
				return &productplanpb.UpdateProductPlanRequest{
					Data: &productplanpb.ProductPlan{
						Id:          updateSuccessResolver.MustGetString("validProductPlanId"),
						Name:        updateSuccessResolver.MustGetString("updatedPlanName"),
						Description: &[]string{updateSuccessResolver.MustGetString("updatedPlanDescription")}[0],
						ProductId:   updateSuccessResolver.MustGetString("validProductId"),
						Price:       updateSuccessResolver.MustGetFloat64("updatedPrice"),
						Currency:    updateSuccessResolver.MustGetString("validCurrency"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productplanpb.UpdateProductPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				updatedPlan := response.Data[0]
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("validProductPlanId"), updatedPlan.Id, "product plan ID")
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("updatedPlanName"), updatedPlan.Name, "updated product plan name")
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("validProductId"), updatedPlan.ProductId, "product ID")
				testutil.AssertFieldSet(t, updatedPlan.DateModified, "DateModified")
				testutil.AssertFieldSet(t, updatedPlan.DateModifiedString, "DateModifiedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-UPDATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.UpdateProductPlanRequest {
				return &productplanpb.UpdateProductPlanRequest{
					Data: &productplanpb.ProductPlan{
						Id:          updateSuccessResolver.MustGetString("validProductPlanId"),
						Name:        updateSuccessResolver.MustGetString("updatedPlanName"),
						Description: &[]string{updateSuccessResolver.MustGetString("updatedPlanDescription")}[0],
						ProductId:   updateSuccessResolver.MustGetString("validProductId"),
						Price:       updateSuccessResolver.MustGetFloat64("updatedPrice"),
						Currency:    updateSuccessResolver.MustGetString("validCurrency"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productplanpb.UpdateProductPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				updatedPlan := response.Data[0]
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("updatedPlanName"), updatedPlan.Name, "updated product plan name")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-UPDATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.UpdateProductPlanRequest {
				return &productplanpb.UpdateProductPlanRequest{
					Data: &productplanpb.ProductPlan{
						Id:        authorizationUnauthorizedResolver.MustGetString("unauthorizedProductPlanId"),
						Name:      "Unauthorized Update",
						ProductId: "subject-math",
						Price:     199.99,
						Currency:  "USD",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "product_plan.errors.authorization_failed",
			Assertions: func(t *testing.T, response *productplanpb.UpdateProductPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-UPDATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.UpdateProductPlanRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_plan.validation.request_required",
			Assertions: func(t *testing.T, response *productplanpb.UpdateProductPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-UPDATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.UpdateProductPlanRequest {
				return &productplanpb.UpdateProductPlanRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_plan.validation.data_required",
			Assertions: func(t *testing.T, response *productplanpb.UpdateProductPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-UPDATE-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.UpdateProductPlanRequest {
				return &productplanpb.UpdateProductPlanRequest{
					Data: &productplanpb.ProductPlan{
						Id:        "",
						Name:      "Valid Plan Name",
						ProductId: "subject-math",
						Price:     199.99,
						Currency:  "USD",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_plan.validation.id_required",
			Assertions: func(t *testing.T, response *productplanpb.UpdateProductPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "EmptyName",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-UPDATE-VALIDATION-EMPTY-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.UpdateProductPlanRequest {
				return &productplanpb.UpdateProductPlanRequest{
					Data: &productplanpb.ProductPlan{
						Id:        "test-plan-id",
						Name:      validationErrorEmptyNameResolver.MustGetString("emptyPlanName"),
						ProductId: validationErrorEmptyNameResolver.MustGetString("validProductId"),
						Price:     validationErrorEmptyNameResolver.MustGetFloat64("validPrice"),
						Currency:  validationErrorEmptyNameResolver.MustGetString("validCurrency"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_plan.validation.name_required",
			Assertions: func(t *testing.T, response *productplanpb.UpdateProductPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty name")
			},
		},
		{
			Name:     "EmptyProductId",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-UPDATE-VALIDATION-EMPTY-PRODUCT-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.UpdateProductPlanRequest {
				return &productplanpb.UpdateProductPlanRequest{
					Data: &productplanpb.ProductPlan{
						Id:        "test-plan-id",
						Name:      validationErrorEmptyProductIdResolver.MustGetString("validPlanName"),
						ProductId: validationErrorEmptyProductIdResolver.MustGetString("emptyProductId"),
						Price:     validationErrorEmptyProductIdResolver.MustGetFloat64("validPrice"),
						Currency:  validationErrorEmptyProductIdResolver.MustGetString("validCurrency"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_plan.validation.product_id_required",
			Assertions: func(t *testing.T, response *productplanpb.UpdateProductPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty product ID")
			},
		},
		{
			Name:     "NameTooShort",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-UPDATE-VALIDATION-NAME-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.UpdateProductPlanRequest {
				return &productplanpb.UpdateProductPlanRequest{
					Data: &productplanpb.ProductPlan{
						Id:        "test-plan-id",
						Name:      validationErrorNameTooShortResolver.MustGetString("tooShortName"),
						ProductId: "subject-math",
						Price:     199.99,
						Currency:  "USD",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_plan.validation.name_too_short",
			Assertions: func(t *testing.T, response *productplanpb.UpdateProductPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too short")
			},
		},
		{
			Name:     "NameTooLong",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-UPDATE-VALIDATION-NAME-TOO-LONG-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.UpdateProductPlanRequest {
				return &productplanpb.UpdateProductPlanRequest{
					Data: &productplanpb.ProductPlan{
						Id:        "test-plan-id",
						Name:      validationErrorNameTooLongResolver.MustGetString("tooLongNameGenerated"),
						ProductId: "subject-math",
						Price:     199.99,
						Currency:  "USD",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_plan.validation.name_too_long",
			Assertions: func(t *testing.T, response *productplanpb.UpdateProductPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too long")
			},
		},
		{
			Name:     "ProductIdTooShort",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-UPDATE-VALIDATION-PRODUCT-ID-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.UpdateProductPlanRequest {
				return &productplanpb.UpdateProductPlanRequest{
					Data: &productplanpb.ProductPlan{
						Id:        "test-plan-id",
						Name:      "Valid Plan Name",
						ProductId: "abc", // Less than minimum length
						Price:     199.99,
						Currency:  "USD",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_plan.validation.product_id_too_short",
			Assertions: func(t *testing.T, response *productplanpb.UpdateProductPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "product ID too short")
			},
		},
		{
			Name:     "NotFound",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-UPDATE-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.UpdateProductPlanRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_plan", "UpdateProductPlan_NotFound")
				testutil.AssertTestCaseLoad(t, err, "UpdateProductPlan_NotFound")
				return &productplanpb.UpdateProductPlanRequest{
					Data: &productplanpb.ProductPlan{
						Id:        resolver.MustGetString("invalidProductPlanId"),
						Name:      resolver.MustGetString("updatedPlanName"),
						ProductId: resolver.MustGetString("validProductId"),
						Price:     resolver.MustGetFloat64("updatedPrice"),
						Currency:  resolver.MustGetString("validCurrency"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_plan.errors.not_found",
			ErrorTags:      map[string]any{"productPlanId": "non-existent-plan-999"},
			Assertions: func(t *testing.T, response *productplanpb.UpdateProductPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
			},
		},
		{
			Name:     "InvalidProductId",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-UPDATE-VALIDATION-INVALID-PRODUCT-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.UpdateProductPlanRequest {
				return &productplanpb.UpdateProductPlanRequest{
					Data: &productplanpb.ProductPlan{
						Id:        "test-plan-id",
						Name:      "Valid Plan Name",
						ProductId: "non-existent-product",
						Price:     199.99,
						Currency:  "USD",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "",
			Assertions: func(t *testing.T, response *productplanpb.UpdateProductPlanResponse, err error, useCase interface{}, ctx context.Context) {
				// This may fail with validation error or not found error depending on implementation
				testutil.AssertError(t, err)
				// Check for either validation error or reference error
				errorMsg := strings.ToLower(err.Error())
				validError := strings.Contains(errorMsg, "not found") ||
					strings.Contains(errorMsg, "product") ||
					strings.Contains(errorMsg, "invalid") ||
					strings.Contains(errorMsg, "reference")
				testutil.AssertTrue(t, validError, "error should be related to invalid product reference")
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

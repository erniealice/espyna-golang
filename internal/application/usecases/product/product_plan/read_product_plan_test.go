//go:build mock_db && mock_auth

// Package product_plan provides table-driven tests for the product plan read use case.
//
// The tests cover various scenarios, including success, authorization,
// nil requests, validation errors, not found, and boundary conditions.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestReadProductPlanUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-READ-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-READ-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-READ-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-READ-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-READ-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-READ-NOT-FOUND-v1.0: NotFound
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-READ-WITH-EXISTING-DATA-v1.0: WithExistingData
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/product_plan.json
//   - Mock data: packages/copya/data/{businessType}/product_plan.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/product_plan.json
package product_plan

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	mockProduct "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/product"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
)

// Type alias for read product plan test cases
type ReadProductPlanTestCase = testutil.GenericTestCase[*productplanpb.ReadProductPlanRequest, *productplanpb.ReadProductPlanResponse]

func createReadTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *ReadProductPlanUseCase {
	mockProductPlanRepo := mockProduct.NewMockProductPlanRepository(businessType)

	repositories := ReadProductPlanRepositories{
		ProductPlan: mockProductPlanRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := ReadProductPlanServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewReadProductPlanUseCase(repositories, services)
}

func TestReadProductPlanUseCase_Execute_TableDriven(t *testing.T) {

	testCases := []ReadProductPlanTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-READ-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.ReadProductPlanRequest {
				return &productplanpb.ReadProductPlanRequest{
					Data: &productplanpb.ProductPlan{
						Id: "product-plan-002", // Exists in product_plan.json
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productplanpb.ReadProductPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				productPlan := response.Data[0]
				testutil.AssertStringEqual(t, "product-plan-002", productPlan.Id, "product plan ID")
				testutil.AssertNonEmptyString(t, productPlan.Name, "product plan name")
				testutil.AssertNonEmptyString(t, productPlan.ProductId, "product ID")
				testutil.AssertTrue(t, productPlan.Active, "product plan active status")
				testutil.AssertFieldSet(t, productPlan.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, productPlan.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-READ-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.ReadProductPlanRequest {
				return &productplanpb.ReadProductPlanRequest{
					Data: &productplanpb.ProductPlan{
						Id: "product-plan-001",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "product_plan.errors.authorization_failed",
			Assertions: func(t *testing.T, response *productplanpb.ReadProductPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-READ-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.ReadProductPlanRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_plan.validation.request_required",
			Assertions: func(t *testing.T, response *productplanpb.ReadProductPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-READ-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.ReadProductPlanRequest {
				return &productplanpb.ReadProductPlanRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_plan.validation.data_required",
			Assertions: func(t *testing.T, response *productplanpb.ReadProductPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-READ-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.ReadProductPlanRequest {
				return &productplanpb.ReadProductPlanRequest{
					Data: &productplanpb.ProductPlan{Id: ""},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_plan.validation.id_required",
			Assertions: func(t *testing.T, response *productplanpb.ReadProductPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "NotFound",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-READ-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.ReadProductPlanRequest {
				return &productplanpb.ReadProductPlanRequest{
					Data: &productplanpb.ProductPlan{
						Id: "non-existent-product-plan-id",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_plan.errors.not_found",
			ErrorTags:      map[string]any{"productPlanId": "non-existent-product-plan-id"},
			Assertions: func(t *testing.T, response *productplanpb.ReadProductPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
			},
		},
		{
			Name:     "WithExistingData",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-READ-WITH-EXISTING-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.ReadProductPlanRequest {
				return &productplanpb.ReadProductPlanRequest{
					Data: &productplanpb.ProductPlan{
						Id: "product-plan-001", // Exists in product-plan.json
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productplanpb.ReadProductPlanResponse, err error, useCase interface{}, ctx context.Context) {
				// This test may succeed or fail depending on mock data availability
				if err != nil {
					// If the mock doesn't have pre-populated data, log and return
					t.Logf("Read of mock data item returned error (mock may not have pre-populated data): %v", err)
					return
				}

				if !response.Success {
					t.Log("Read of mock data item was unsuccessful (mock may not have pre-populated data)")
					return
				}

				if len(response.Data) == 0 {
					t.Log("No mock data found (mock may not have pre-populated data)")
					return
				}

				// If we do get data, verify its structure
				productPlan := response.Data[0]
				testutil.AssertNonEmptyString(t, productPlan.Id, "product plan ID should not be empty")
				testutil.AssertNonEmptyString(t, productPlan.Name, "product plan name should not be empty")
				testutil.AssertNonEmptyString(t, productPlan.ProductId, "product ID should not be empty")
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

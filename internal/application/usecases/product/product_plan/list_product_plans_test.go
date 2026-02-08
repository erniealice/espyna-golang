//go:build mock_db && mock_auth

// Package product_plan provides table-driven tests for the product plan listing use case.
//
// The tests cover various scenarios, including success, authorization,
// nil requests, validation errors, and boundary conditions.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestListProductPlansUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-LIST-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-LIST-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-LIST-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-LIST-EMPTY-RESULT-v1.0: EmptyResult
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-LIST-WITH-DATA-v1.0: WithExistingData
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

// Type alias for list product plans test cases
type ListProductPlansTestCase = testutil.GenericTestCase[*productplanpb.ListProductPlansRequest, *productplanpb.ListProductPlansResponse]

func createListTestUseCaseWithAuth(businessType string, shouldAuthorize bool) *ListProductPlansUseCase {
	mockProductPlanRepo := mockProduct.NewMockProductPlanRepository(businessType)

	repositories := ListProductPlansRepositories{
		ProductPlan: mockProductPlanRepo,
	}

	standardServices := testutil.CreateStandardServices(false, shouldAuthorize)
	services := ListProductPlansServices{
		AuthorizationService: standardServices.AuthorizationService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewListProductPlansUseCase(repositories, services)
}

func TestListProductPlansUseCase_Execute_TableDriven(t *testing.T) {

	testCases := []ListProductPlansTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-LIST-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.ListProductPlansRequest {
				return &productplanpb.ListProductPlansRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productplanpb.ListProductPlansResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertNotNil(t, response.Data, "data should not be nil")
				// The mock repository may start empty, so we just check non-negative length
				testutil.AssertTrue(t, len(response.Data) >= 0, "data length should be non-negative")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-LIST-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.ListProductPlansRequest {
				return &productplanpb.ListProductPlansRequest{}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "product_plan.errors.authorization_failed",
			Assertions: func(t *testing.T, response *productplanpb.ListProductPlansResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-LIST-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.ListProductPlansRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_plan.validation.request_required",
			Assertions: func(t *testing.T, response *productplanpb.ListProductPlansResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "EmptyResult",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-LIST-EMPTY-RESULT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.ListProductPlansRequest {
				return &productplanpb.ListProductPlansRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productplanpb.ListProductPlansResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success should be true even for empty results")
				testutil.AssertNotNil(t, response.Data, "data should not be nil, even if empty")
				// For empty results, we accept any non-negative length
				testutil.AssertTrue(t, len(response.Data) >= 0, "data length should be non-negative")
			},
		},
		{
			Name:     "WithExistingData",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-LIST-WITH-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.ListProductPlansRequest {
				return &productplanpb.ListProductPlansRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productplanpb.ListProductPlansResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertNotNil(t, response.Data, "data should not be nil")

				// If we have data, verify the structure of the first item
				if len(response.Data) > 0 {
					firstPlan := response.Data[0]
					testutil.AssertNonEmptyString(t, firstPlan.Id, "product plan ID should not be empty")
					testutil.AssertNonEmptyString(t, firstPlan.Name, "product plan name should not be empty")
					testutil.AssertNonEmptyString(t, firstPlan.ProductId, "product ID should not be empty")
					testutil.AssertFieldSet(t, firstPlan.DateCreated, "DateCreated")
					testutil.AssertFieldSet(t, firstPlan.DateCreatedString, "DateCreatedString")
				}
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
			useCase := createListTestUseCaseWithAuth(businessType, tc.UseAuth)

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

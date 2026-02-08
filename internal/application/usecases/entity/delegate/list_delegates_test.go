//go:build mock_db && mock_auth

// Package delegate provides table-driven tests for the delegate listing use case.
//
// The tests cover various scenarios, including success, authorization, nil requests,
// validation errors, and business logic validation. Each test case is defined in a table
// with a specific test code, request setup, and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestListDelegatesUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-ENTITY-DELEGATE-LIST-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-ENTITY-DELEGATE-LIST-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-ENTITY-DELEGATE-LIST-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-ENTITY-DELEGATE-LIST-VERIFY-DETAILS-v1.0: VerifyDetails
//   - ESPYNA-TEST-ENTITY-DELEGATE-LIST-BUSINESS-LOGIC-v1.0: BusinessLogic
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/delegate.json
//   - Mock data: packages/copya/data/{businessType}/delegate.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/delegate.json
package delegate

import (
	"context"
	"fmt"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	delegatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate"
)

// Type alias for list delegates test cases
type ListDelegatesTestCase = testutil.GenericTestCase[*delegatepb.ListDelegatesRequest, *delegatepb.ListDelegatesResponse]

func createTestListUseCaseWithAuth(businessType string, shouldAuthorize bool) *ListDelegatesUseCase {
	mockDelegateRepo := entity.NewMockDelegateRepository(businessType)

	repositories := ListDelegatesRepositories{
		Delegate: mockDelegateRepo,
	}

	standardServices := testutil.CreateStandardServices(false, shouldAuthorize)
	services := ListDelegatesServices{
		AuthorizationService: standardServices.AuthorizationService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewListDelegatesUseCase(repositories, services)
}

func TestListDelegatesUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	listSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "delegate", "ListDelegates_Success")
	testutil.AssertTestCaseLoad(t, err, "ListDelegates_Success")

	// Note: Test resolvers simplified for now - using basic validation instead of complex data structures

	testCases := []ListDelegatesTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-LIST-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.ListDelegatesRequest {
				return &delegatepb.ListDelegatesRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *delegatepb.ListDelegatesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertGreaterThanOrEqual(t, len(response.Data), listSuccessResolver.MustGetInt("expectedDelegateCount"), "minimum delegate count")

				// Verify expected delegate IDs are present
				expectedIds := listSuccessResolver.MustGetStringArray("expectedDelegateIds")
				foundIds := make(map[string]bool)
				for _, delegate := range response.Data {
					foundIds[delegate.Id] = true
				}
				for _, expectedId := range expectedIds {
					testutil.AssertTrue(t, foundIds[expectedId], "expected delegate ID "+expectedId+" found")
				}
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-LIST-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.ListDelegatesRequest {
				return &delegatepb.ListDelegatesRequest{}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "delegate.errors.authorization_failed",
			Assertions: func(t *testing.T, response *delegatepb.ListDelegatesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-LIST-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.ListDelegatesRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "delegate.validation.request_required",
			Assertions: func(t *testing.T, response *delegatepb.ListDelegatesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "VerifyDetails",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-LIST-VERIFY-DETAILS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.ListDelegatesRequest {
				return &delegatepb.ListDelegatesRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *delegatepb.ListDelegatesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")

				// Verify that we have at least some delegates with expected data structure
				testutil.AssertGreaterThanOrEqual(t, len(response.Data), 1, "should have at least one delegate")

				// Verify first delegate has expected fields
				if len(response.Data) > 0 {
					firstDelegate := response.Data[0]
					testutil.AssertNonEmptyString(t, firstDelegate.Id, "delegate ID")
					testutil.AssertNotNil(t, firstDelegate.User, "delegate user")
					testutil.AssertNonEmptyString(t, firstDelegate.User.FirstName, "user first name")
					testutil.AssertNonEmptyString(t, firstDelegate.User.LastName, "user last name")
					testutil.AssertNonEmptyString(t, firstDelegate.User.EmailAddress, "user email address")
				}
			},
		},
		{
			Name:     "BusinessLogic",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-LIST-BUSINESS-LOGIC-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.ListDelegatesRequest {
				return &delegatepb.ListDelegatesRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *delegatepb.ListDelegatesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")

				// Test basic business logic - should return delegates
				testutil.AssertGreaterThanOrEqual(t, len(response.Data), 1, "should return at least one delegate")

				// Verify all delegates have required fields
				for i, delegate := range response.Data {
					testutil.AssertNonEmptyString(t, delegate.Id, fmt.Sprintf("delegate[%d].Id", i))
					testutil.AssertNotNil(t, delegate.User, fmt.Sprintf("delegate[%d].User", i))
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
			useCase := createTestListUseCaseWithAuth(businessType, tc.UseAuth)

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

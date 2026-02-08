//go:build mock_db && mock_auth

// Package delegate provides table-driven tests for the delegate reading use case.
//
// The tests cover various scenarios, including success, authorization, nil requests,
// validation errors, and not found conditions. Each test case is defined in a table
// with a specific test code, request setup, and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestReadDelegateUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-ENTITY-DELEGATE-READ-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-ENTITY-DELEGATE-READ-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-ENTITY-DELEGATE-READ-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-ENTITY-DELEGATE-READ-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-ENTITY-DELEGATE-READ-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-ENTITY-DELEGATE-READ-NOT-FOUND-v1.0: NotFound
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/delegate.json
//   - Mock data: packages/copya/data/{businessType}/delegate.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/delegate.json
package delegate

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	delegatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate"
)

// Type alias for read delegate test cases
type ReadDelegateTestCase = testutil.GenericTestCase[*delegatepb.ReadDelegateRequest, *delegatepb.ReadDelegateResponse]

func createTestReadUseCaseWithAuth(businessType string, shouldAuthorize bool) *ReadDelegateUseCase {
	mockDelegateRepo := entity.NewMockDelegateRepository(businessType)

	repositories := ReadDelegateRepositories{
		Delegate: mockDelegateRepo,
	}

	standardServices := testutil.CreateStandardServices(false, shouldAuthorize)
	services := ReadDelegateServices{
		AuthorizationService: standardServices.AuthorizationService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewReadDelegateUseCase(repositories, services)
}

func TestReadDelegateUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	readSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "delegate", "ReadDelegate_Success")
	testutil.AssertTestCaseLoad(t, err, "ReadDelegate_Success")

	readNotFoundResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "delegate", "ReadDelegate_NotFound")
	testutil.AssertTestCaseLoad(t, err, "ReadDelegate_NotFound")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "delegate", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	testCases := []ReadDelegateTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-READ-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.ReadDelegateRequest {
				return &delegatepb.ReadDelegateRequest{
					Data: &delegatepb.Delegate{
						Id: readSuccessResolver.MustGetString("targetDelegateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *delegatepb.ReadDelegateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				readDelegate := response.Data[0]
				testutil.AssertStringEqual(t, readSuccessResolver.MustGetString("targetDelegateId"), readDelegate.Id, "delegate ID")
				testutil.AssertStringEqual(t, readSuccessResolver.MustGetString("expectedFirstName"), readDelegate.User.FirstName, "first name")
				testutil.AssertStringEqual(t, readSuccessResolver.MustGetString("expectedLastName"), readDelegate.User.LastName, "last name")
				testutil.AssertStringEqual(t, readSuccessResolver.MustGetString("expectedEmailAddress"), readDelegate.User.EmailAddress, "email address")
				testutil.AssertEqual(t, readSuccessResolver.MustGetBool("expectedActive"), readDelegate.Active, "active status")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-READ-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.ReadDelegateRequest {
				return &delegatepb.ReadDelegateRequest{
					Data: &delegatepb.Delegate{
						Id: authorizationUnauthorizedResolver.MustGetString("targetDelegateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "delegate.errors.authorization_failed",
			Assertions: func(t *testing.T, response *delegatepb.ReadDelegateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-READ-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.ReadDelegateRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "delegate.validation.request_required",
			Assertions: func(t *testing.T, response *delegatepb.ReadDelegateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-READ-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.ReadDelegateRequest {
				return &delegatepb.ReadDelegateRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "delegate.validation.data_required",
			Assertions: func(t *testing.T, response *delegatepb.ReadDelegateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-READ-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.ReadDelegateRequest {
				return &delegatepb.ReadDelegateRequest{
					Data: &delegatepb.Delegate{Id: ""},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "delegate.validation.id_required",
			Assertions: func(t *testing.T, response *delegatepb.ReadDelegateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "NotFound",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-READ-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.ReadDelegateRequest {
				return &delegatepb.ReadDelegateRequest{
					Data: &delegatepb.Delegate{
						Id: readNotFoundResolver.MustGetString("nonExistentDelegateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *delegatepb.ReadDelegateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 0, len(response.Data), "response data length for not found")
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
			useCase := createTestReadUseCaseWithAuth(businessType, tc.UseAuth)

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

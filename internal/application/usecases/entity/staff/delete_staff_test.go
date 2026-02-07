//go:build mock_db && mock_auth

// Package staff provides table-driven tests for the staff delete use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, not found errors, validation errors, and nil request handling.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestDeleteStaffUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-ENTITY-STAFF-DELETE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-ENTITY-STAFF-DELETE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-ENTITY-STAFF-DELETE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-ENTITY-STAFF-DELETE-NOT-FOUND-v1.0: NotFound
//   - ESPYNA-TEST-ENTITY-STAFF-DELETE-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-ENTITY-STAFF-DELETE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-ENTITY-STAFF-DELETE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-ENTITY-STAFF-DELETE-INTEGRATION-v1.0: IntegrationTest
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/staff.json
//   - Mock data: packages/copya/data/{businessType}/staff.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/staff.json
package staff

import (
	"context"
	"fmt"
	"strings"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"
	staffpb "leapfor.xyz/esqyma/golang/v1/domain/entity/staff"
)

// Type alias for delete staff test cases
type DeleteStaffTestCase = testutil.GenericTestCase[*staffpb.DeleteStaffRequest, *staffpb.DeleteStaffResponse]

func createTestDeleteStaffUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *DeleteStaffUseCase {
	repositories := DeleteStaffRepositories{
		Staff: entity.NewMockStaffRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := DeleteStaffServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewDeleteStaffUseCase(repositories, services)
}

func TestDeleteStaffUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	deleteSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "staff", "DeleteStaff_Success")
	testutil.AssertTestCaseLoad(t, err, "DeleteStaff_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "staff", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	notFoundResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "staff", "NotFound_DeleteStaff")
	testutil.AssertTestCaseLoad(t, err, "NotFound_DeleteStaff")

	validationErrorEmptyIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "staff", "ValidationError_EmptyId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyId")

	testCases := []DeleteStaffTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-DELETE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.DeleteStaffRequest {
				return &staffpb.DeleteStaffRequest{
					Data: &staffpb.Staff{
						Id: deleteSuccessResolver.MustGetString("targetStaffId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *staffpb.DeleteStaffResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")

				// Verify the staff is actually deleted by attempting to read it
				deleteUseCase := useCase.(*DeleteStaffUseCase)
				standardServices := testutil.CreateStandardServices(false, false)
				readUseCase := NewReadStaffUseCase(
					ReadStaffRepositories{Staff: deleteUseCase.repositories.Staff},
					ReadStaffServices{
						TranslationService: standardServices.TranslationService,
					},
				)
				readReq := &staffpb.ReadStaffRequest{
					Data: &staffpb.Staff{Id: deleteSuccessResolver.MustGetString("targetStaffId")},
				}
				_, readErr := readUseCase.Execute(ctx, readReq)
				testutil.AssertError(t, readErr)
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-DELETE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.DeleteStaffRequest {
				return &staffpb.DeleteStaffRequest{
					Data: &staffpb.Staff{
						Id: deleteSuccessResolver.MustGetString("integrationTestStaffId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *staffpb.DeleteStaffResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-DELETE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.DeleteStaffRequest {
				return &staffpb.DeleteStaffRequest{
					Data: &staffpb.Staff{
						Id: authorizationUnauthorizedResolver.MustGetString("targetStaffId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "staff.errors.authorization_failed",
			Assertions: func(t *testing.T, response *staffpb.DeleteStaffResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NotFound",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-DELETE-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.DeleteStaffRequest {
				return &staffpb.DeleteStaffRequest{
					Data: &staffpb.Staff{
						Id: notFoundResolver.MustGetString("nonExistentStaffId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			Assertions: func(t *testing.T, response *staffpb.DeleteStaffResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
				expectedErrorMessage := notFoundResolver.MustGetString("expectedErrorMessage")
				// Since we already verified err is not nil above, we can safely check its content
				errorContainsExpectedMessage := strings.Contains(err.Error(), expectedErrorMessage)
				testutil.AssertTrue(t, errorContainsExpectedMessage, fmt.Sprintf("error message should contain '%s' but got '%s'", expectedErrorMessage, err.Error()))
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-DELETE-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.DeleteStaffRequest {
				return &staffpb.DeleteStaffRequest{
					Data: &staffpb.Staff{
						Id: validationErrorEmptyIdResolver.MustGetString("emptyId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "staff.validation.id_required",
			Assertions: func(t *testing.T, response *staffpb.DeleteStaffResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-DELETE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.DeleteStaffRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "staff.validation.request_required",
			Assertions: func(t *testing.T, response *staffpb.DeleteStaffResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-DELETE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.DeleteStaffRequest {
				return &staffpb.DeleteStaffRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "staff.validation.request_required",
			Assertions: func(t *testing.T, response *staffpb.DeleteStaffResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
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
			useCase := createTestDeleteStaffUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

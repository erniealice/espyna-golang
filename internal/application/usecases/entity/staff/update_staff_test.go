//go:build mock_db && mock_auth

// Package staff provides table-driven tests for the staff update use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestUpdateStaffUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-ENTITY-STAFF-UPDATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-ENTITY-STAFF-UPDATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-ENTITY-STAFF-UPDATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-ENTITY-STAFF-UPDATE-NOT-FOUND-v1.0: NotFound
//   - ESPYNA-TEST-ENTITY-STAFF-UPDATE-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-ENTITY-STAFF-UPDATE-VALIDATION-EMPTY-EMAIL-v1.0: EmptyEmail
//   - ESPYNA-TEST-ENTITY-STAFF-UPDATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-ENTITY-STAFF-UPDATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-ENTITY-STAFF-UPDATE-VERIFICATION-v1.0: VerifyDateModified
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/staff.json
//   - Mock data: packages/copya/data/{businessType}/staff.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/staff.json
package staff

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	staffpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/staff"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// Type alias for update staff test cases
type UpdateStaffTestCase = testutil.GenericTestCase[*staffpb.UpdateStaffRequest, *staffpb.UpdateStaffResponse]

func createTestUpdateStaffUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *UpdateStaffUseCase {
	repositories := UpdateStaffRepositories{
		Staff: entity.NewMockStaffRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := UpdateStaffServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewUpdateStaffUseCase(repositories, services)
}

func TestUpdateStaffUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	updateSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "staff", "UpdateStaff_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdateStaff_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "staff", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	notFoundResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "staff", "NotFound_UpdateStaff")
	testutil.AssertTestCaseLoad(t, err, "NotFound_UpdateStaff")

	validationErrorEmptyIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "staff", "ValidationError_EmptyId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyId")

	validationErrorEmptyEmailResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "staff", "ValidationError_EmptyEmail")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyEmail")

	testCases := []UpdateStaffTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-UPDATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.UpdateStaffRequest {
				return &staffpb.UpdateStaffRequest{
					Data: &staffpb.Staff{
						Id: updateSuccessResolver.MustGetString("targetStaffId"),
						User: &userpb.User{
							Id:           updateSuccessResolver.MustGetString("targetUserId"),
							FirstName:    updateSuccessResolver.MustGetString("updatedFirstName"),
							LastName:     updateSuccessResolver.MustGetString("updatedLastName"),
							EmailAddress: updateSuccessResolver.MustGetString("updatedEmail"),
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *staffpb.UpdateStaffResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				updatedStaff := response.Data[0]
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("targetStaffId"), updatedStaff.Id, "staff ID")
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("updatedEmail"), updatedStaff.User.EmailAddress, "updated email")
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("updatedFirstName"), updatedStaff.User.FirstName, "updated first name")
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("updatedLastName"), updatedStaff.User.LastName, "updated last name")
				testutil.AssertFieldSet(t, updatedStaff.DateModified, "DateModified")
				testutil.AssertFieldSet(t, updatedStaff.DateModifiedString, "DateModifiedString")

				// Check that DateModified is greater than original timestamp
				if updatedStaff.DateModified != nil {
					originalTime := updateSuccessResolver.MustGetInt64("originalTimestamp")
					testutil.AssertGreaterThan(t, int(*updatedStaff.DateModified), int(originalTime), "DateModified timestamp")
				}
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-UPDATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.UpdateStaffRequest {
				return &staffpb.UpdateStaffRequest{
					Data: &staffpb.Staff{
						Id: updateSuccessResolver.MustGetString("secondTargetStaffId"),
						User: &userpb.User{
							Id:           updateSuccessResolver.MustGetString("secondTargetUserId"),
							FirstName:    updateSuccessResolver.MustGetString("enhancedFirstName"),
							LastName:     updateSuccessResolver.MustGetString("enhancedLastName"),
							EmailAddress: updateSuccessResolver.MustGetString("enhancedEmail"),
						},
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *staffpb.UpdateStaffResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				updatedStaff := response.Data[0]
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("enhancedEmail"), updatedStaff.User.EmailAddress, "enhanced email")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-UPDATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.UpdateStaffRequest {
				return &staffpb.UpdateStaffRequest{
					Data: &staffpb.Staff{
						Id: authorizationUnauthorizedResolver.MustGetString("targetStaffId"),
						User: &userpb.User{
							FirstName:    authorizationUnauthorizedResolver.MustGetString("unauthorizedFirstName"),
							LastName:     authorizationUnauthorizedResolver.MustGetString("unauthorizedLastName"),
							EmailAddress: authorizationUnauthorizedResolver.MustGetString("unauthorizedEmail"),
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "staff.errors.authorization_failed",
			Assertions: func(t *testing.T, response *staffpb.UpdateStaffResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NotFound",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-UPDATE-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.UpdateStaffRequest {
				return &staffpb.UpdateStaffRequest{
					Data: &staffpb.Staff{
						Id: notFoundResolver.MustGetString("nonExistentStaffId"),
						User: &userpb.User{
							FirstName:    notFoundResolver.MustGetString("updateFirstName"),
							LastName:     notFoundResolver.MustGetString("updateLastName"),
							EmailAddress: notFoundResolver.MustGetString("updateEmail"),
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "staff.errors.update_failed",
			Assertions: func(t *testing.T, response *staffpb.UpdateStaffResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-UPDATE-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.UpdateStaffRequest {
				return &staffpb.UpdateStaffRequest{
					Data: &staffpb.Staff{
						Id: validationErrorEmptyIdResolver.MustGetString("emptyId"),
						User: &userpb.User{
							FirstName:    validationErrorEmptyIdResolver.MustGetString("validFirstName"),
							LastName:     validationErrorEmptyIdResolver.MustGetString("validLastName"),
							EmailAddress: validationErrorEmptyIdResolver.MustGetString("validEmail"),
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "staff.validation.id_required",
			Assertions: func(t *testing.T, response *staffpb.UpdateStaffResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "EmptyEmail",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-UPDATE-VALIDATION-EMPTY-EMAIL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.UpdateStaffRequest {
				return &staffpb.UpdateStaffRequest{
					Data: &staffpb.Staff{
						Id: updateSuccessResolver.MustGetString("targetStaffId"),
						User: &userpb.User{
							FirstName:    validationErrorEmptyEmailResolver.MustGetString("validFirstName"),
							LastName:     validationErrorEmptyEmailResolver.MustGetString("validLastName"),
							EmailAddress: validationErrorEmptyEmailResolver.MustGetString("emptyEmail"),
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "staff.validation.email_required",
			Assertions: func(t *testing.T, response *staffpb.UpdateStaffResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty email")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-UPDATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.UpdateStaffRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "staff.validation.request_required",
			Assertions: func(t *testing.T, response *staffpb.UpdateStaffResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-UPDATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.UpdateStaffRequest {
				return &staffpb.UpdateStaffRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "staff.validation.request_required",
			Assertions: func(t *testing.T, response *staffpb.UpdateStaffResponse, err error, useCase interface{}, ctx context.Context) {
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
			useCase := createTestUpdateStaffUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

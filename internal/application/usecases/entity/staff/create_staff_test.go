//go:build mock_db && mock_auth

// Package staff provides table-driven tests for the staff creation use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, data enrichment, and boundary conditions.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestCreateStaffUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-ENTITY-STAFF-CREATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-ENTITY-STAFF-CREATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-ENTITY-STAFF-CREATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-ENTITY-STAFF-CREATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-ENTITY-STAFF-CREATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-ENTITY-STAFF-CREATE-VALIDATION-EMPTY-FIRST-NAME-v1.0: EmptyFirstName
//   - ESPYNA-TEST-ENTITY-STAFF-CREATE-VALIDATION-EMPTY-LAST-NAME-v1.0: EmptyLastName
//   - ESPYNA-TEST-ENTITY-STAFF-CREATE-VALIDATION-EMPTY-EMAIL-v1.0: EmptyEmail
//   - ESPYNA-TEST-ENTITY-STAFF-CREATE-VALIDATION-INVALID-EMAIL-v1.0: InvalidEmail
//   - ESPYNA-TEST-ENTITY-STAFF-CREATE-VALIDATION-NIL-USER-v1.0: NilUser
//   - ESPYNA-TEST-ENTITY-STAFF-CREATE-ENRICHMENT-v1.0: DataEnrichment
//   - ESPYNA-TEST-ENTITY-STAFF-CREATE-VALIDATION-MINIMAL-v1.0: MinimalValidData
//   - ESPYNA-TEST-ENTITY-STAFF-CREATE-VALIDATION-MAXIMAL-v1.0: MaxValidData
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/staff.json
//   - Mock data: packages/copya/data/{businessType}/staff.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/staff.json
package staff

import (
	"context"
	"testing"
	"time"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	staffpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/staff"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// Type alias for create staff test cases
type CreateStaffTestCase = testutil.GenericTestCase[*staffpb.CreateStaffRequest, *staffpb.CreateStaffResponse]

func createTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreateStaffUseCase {
	repositories := CreateStaffRepositories{
		Staff: entity.NewMockStaffRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreateStaffServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreateStaffUseCase(repositories, services)
}

func TestCreateStaffUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "staff", "CreateStaff_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateStaff_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "staff", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	validationErrorEmptyFirstNameResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "staff", "ValidationError_EmptyFirstName")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyFirstName")

	validationErrorEmptyLastNameResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "staff", "ValidationError_EmptyLastName")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyLastName")

	validationErrorEmptyEmailResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "staff", "ValidationError_EmptyEmail")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyEmail")

	validationErrorInvalidEmailResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "staff", "ValidationError_InvalidEmail")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidEmail")

	dataEnrichmentResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "staff", "DataEnrichment_TestStaff")
	testutil.AssertTestCaseLoad(t, err, "DataEnrichment_TestStaff")

	testCases := []CreateStaffTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.CreateStaffRequest {
				return &staffpb.CreateStaffRequest{
					Data: &staffpb.Staff{
						User: &userpb.User{
							FirstName:    createSuccessResolver.MustGetString("newStaffFirstName"),
							LastName:     createSuccessResolver.MustGetString("newStaffLastName"),
							EmailAddress: createSuccessResolver.MustGetString("newStaffEmail"),
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *staffpb.CreateStaffResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdStaff := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newStaffFirstName"), createdStaff.User.FirstName, "staff first name")
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newStaffLastName"), createdStaff.User.LastName, "staff last name")
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newStaffEmail"), createdStaff.User.EmailAddress, "staff email")
				testutil.AssertNonEmptyString(t, createdStaff.Id, "staff ID")
				testutil.AssertNonEmptyString(t, createdStaff.User.Id, "user ID")
				testutil.AssertTrue(t, createdStaff.Active, "staff active status")
				testutil.AssertFieldSet(t, createdStaff.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdStaff.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.CreateStaffRequest {
				return &staffpb.CreateStaffRequest{
					Data: &staffpb.Staff{
						User: &userpb.User{
							FirstName:    createSuccessResolver.MustGetString("validUserFirstName"),
							LastName:     createSuccessResolver.MustGetString("validUserLastName"),
							EmailAddress: createSuccessResolver.MustGetString("validUserEmail"),
						},
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *staffpb.CreateStaffResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdStaff := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("validUserFirstName"), createdStaff.User.FirstName, "staff first name")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-CREATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.CreateStaffRequest {
				return &staffpb.CreateStaffRequest{
					Data: &staffpb.Staff{
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
			Assertions: func(t *testing.T, response *staffpb.CreateStaffResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.CreateStaffRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "staff.validation.request_required",
			Assertions: func(t *testing.T, response *staffpb.CreateStaffResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.CreateStaffRequest {
				return &staffpb.CreateStaffRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "staff.validation.data_required",
			Assertions: func(t *testing.T, response *staffpb.CreateStaffResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyFirstName",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-CREATE-VALIDATION-EMPTY-FIRST-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.CreateStaffRequest {
				return &staffpb.CreateStaffRequest{
					Data: &staffpb.Staff{
						User: &userpb.User{
							FirstName:    validationErrorEmptyFirstNameResolver.MustGetString("emptyFirstName"),
							LastName:     validationErrorEmptyFirstNameResolver.MustGetString("validLastName"),
							EmailAddress: validationErrorEmptyFirstNameResolver.MustGetString("validEmail"),
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "staff.validation.first_name_required",
			Assertions: func(t *testing.T, response *staffpb.CreateStaffResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty first name")
			},
		},
		{
			Name:     "EmptyLastName",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-CREATE-VALIDATION-EMPTY-LAST-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.CreateStaffRequest {
				return &staffpb.CreateStaffRequest{
					Data: &staffpb.Staff{
						User: &userpb.User{
							FirstName:    validationErrorEmptyLastNameResolver.MustGetString("validFirstName"),
							LastName:     validationErrorEmptyLastNameResolver.MustGetString("emptyLastName"),
							EmailAddress: validationErrorEmptyLastNameResolver.MustGetString("validEmail"),
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "staff.validation.last_name_required",
			Assertions: func(t *testing.T, response *staffpb.CreateStaffResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty last name")
			},
		},
		{
			Name:     "EmptyEmail",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-CREATE-VALIDATION-EMPTY-EMAIL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.CreateStaffRequest {
				return &staffpb.CreateStaffRequest{
					Data: &staffpb.Staff{
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
			Assertions: func(t *testing.T, response *staffpb.CreateStaffResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty email")
			},
		},
		{
			Name:     "InvalidEmail",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-CREATE-VALIDATION-INVALID-EMAIL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.CreateStaffRequest {
				return &staffpb.CreateStaffRequest{
					Data: &staffpb.Staff{
						User: &userpb.User{
							FirstName:    validationErrorInvalidEmailResolver.MustGetString("validFirstName"),
							LastName:     validationErrorInvalidEmailResolver.MustGetString("validLastName"),
							EmailAddress: validationErrorInvalidEmailResolver.MustGetString("invalidEmail"),
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "staff.validation.email_invalid",
			Assertions: func(t *testing.T, response *staffpb.CreateStaffResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid email")
			},
		},
		{
			Name:     "NilUser",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-CREATE-VALIDATION-NIL-USER-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.CreateStaffRequest {
				return &staffpb.CreateStaffRequest{
					Data: &staffpb.Staff{
						User: nil,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "staff.validation.user_data_required",
			Assertions: func(t *testing.T, response *staffpb.CreateStaffResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "nil user data")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-CREATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.CreateStaffRequest {
				return &staffpb.CreateStaffRequest{
					Data: &staffpb.Staff{
						User: &userpb.User{
							FirstName:    dataEnrichmentResolver.MustGetString("enrichmentFirstName"),
							LastName:     dataEnrichmentResolver.MustGetString("enrichmentLastName"),
							EmailAddress: dataEnrichmentResolver.MustGetString("enrichmentEmail"),
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *staffpb.CreateStaffResponse, err error, useCase interface{}, ctx context.Context) {
				createdStaff := response.Data[0]
				testutil.AssertNonEmptyString(t, createdStaff.Id, "generated ID")
				testutil.AssertNonEmptyString(t, createdStaff.User.Id, "generated User ID")
				testutil.AssertFieldSet(t, createdStaff.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdStaff.DateCreatedString, "DateCreatedString")
				testutil.AssertTrue(t, createdStaff.Active, "Active")

				// Check timestamp is recent (within 5 seconds)
				if createdStaff.DateCreated != nil {
					now := time.Now().UnixMilli()
					threshold := dataEnrichmentResolver.MustGetInt64("expectedDateCreatedThreshold")
					if *createdStaff.DateCreated < now-threshold || *createdStaff.DateCreated > now+threshold {
						testutil.AssertTimestampInMilliseconds(t, *createdStaff.DateCreated, "DateCreated")
					}
				}
			},
		},
		{
			Name:     "MinimalValidData",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-CREATE-VALIDATION-MINIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.CreateStaffRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "staff", "BoundaryTest_MinimalValid")
				testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MinimalValid")
				return &staffpb.CreateStaffRequest{
					Data: &staffpb.Staff{
						User: &userpb.User{
							FirstName:    resolver.MustGetString("minValidFirstName"),
							LastName:     resolver.MustGetString("minValidLastName"),
							EmailAddress: resolver.MustGetString("minValidEmail"),
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *staffpb.CreateStaffResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdStaff := response.Data[0]
				resolver, _ := copyatestutil.LoadTestCaseFromBusinessType(t, testutil.GetTestBusinessType(), "staff", "BoundaryTest_MinimalValid")
				testutil.AssertStringEqual(t, resolver.MustGetString("minValidFirstName"), createdStaff.User.FirstName, "staff first name")
			},
		},
		{
			Name:     "MaxValidData",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-CREATE-VALIDATION-MAXIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.CreateStaffRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "staff", "BoundaryTest_MaximalValid")
				testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MaximalValid")
				return &staffpb.CreateStaffRequest{
					Data: &staffpb.Staff{
						User: &userpb.User{
							FirstName:    resolver.MustGetString("maxValidFirstName"),
							LastName:     resolver.MustGetString("maxValidLastName"),
							EmailAddress: resolver.MustGetString("maxValidEmail"),
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *staffpb.CreateStaffResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdStaff := response.Data[0]
				resolver, _ := copyatestutil.LoadTestCaseFromBusinessType(t, testutil.GetTestBusinessType(), "staff", "BoundaryTest_MaximalValid")
				testutil.AssertStringEqual(t, resolver.MustGetString("maxValidFirstName"), createdStaff.User.FirstName, "staff first name")
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
			useCase := createTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

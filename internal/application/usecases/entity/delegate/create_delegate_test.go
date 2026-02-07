//go:build mock_db && mock_auth

// Package delegate provides table-driven tests for the delegate creation use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestCreateDelegateUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-ENTITY-DELEGATE-CREATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-ENTITY-DELEGATE-CREATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-ENTITY-DELEGATE-CREATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-ENTITY-DELEGATE-CREATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-ENTITY-DELEGATE-CREATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-ENTITY-DELEGATE-CREATE-VALIDATION-EMPTY-FIRST-NAME-v1.0: EmptyFirstName
//   - ESPYNA-TEST-ENTITY-DELEGATE-CREATE-VALIDATION-EMPTY-LAST-NAME-v1.0: EmptyLastName
//   - ESPYNA-TEST-ENTITY-DELEGATE-CREATE-VALIDATION-EMPTY-EMAIL-v1.0: EmptyEmail
//   - ESPYNA-TEST-ENTITY-DELEGATE-CREATE-VALIDATION-INVALID-EMAIL-v1.0: InvalidEmail
//   - ESPYNA-TEST-ENTITY-DELEGATE-CREATE-VALIDATION-FIRST-NAME-TOO-LONG-v1.0: FirstNameTooLong
//   - ESPYNA-TEST-ENTITY-DELEGATE-CREATE-VALIDATION-LAST-NAME-TOO-LONG-v1.0: LastNameTooLong
//   - ESPYNA-TEST-ENTITY-DELEGATE-CREATE-ENRICHMENT-v1.0: DataEnrichment
//   - ESPYNA-TEST-ENTITY-DELEGATE-CREATE-VALIDATION-MINIMAL-v1.0: MinimalValidData
//   - ESPYNA-TEST-ENTITY-DELEGATE-CREATE-VALIDATION-MAXIMAL-v1.0: MaxValidData
//   - ESPYNA-TEST-ENTITY-DELEGATE-CREATE-TRANSACTION-FAILURE-v1.0: TransactionFailure
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
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockDb "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"
	delegatepb "leapfor.xyz/esqyma/golang/v1/domain/entity/delegate"
	userpb "leapfor.xyz/esqyma/golang/v1/domain/entity/user"
)

// Type alias for create delegate test cases
type CreateDelegateTestCase = testutil.GenericTestCase[*delegatepb.CreateDelegateRequest, *delegatepb.CreateDelegateResponse]

func createTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreateDelegateUseCase {
	mockDelegateRepo := entity.NewMockDelegateRepository(businessType)

	repositories := CreateDelegateRepositories{
		Delegate: mockDelegateRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreateDelegateServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreateDelegateUseCase(repositories, services)
}

func TestCreateDelegateUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "delegate", "CreateDelegate_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateDelegate_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "delegate", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	validationErrorEmptyFirstNameResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "delegate", "ValidationError_EmptyFirstName")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyFirstName")

	validationErrorEmptyLastNameResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "delegate", "ValidationError_EmptyLastName")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyLastName")

	validationErrorEmptyEmailResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "delegate", "ValidationError_EmptyEmail")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyEmail")

	validationErrorInvalidEmailResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "delegate", "ValidationError_InvalidEmail")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidEmail")

	testCases := []CreateDelegateTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.CreateDelegateRequest {
				return &delegatepb.CreateDelegateRequest{
					Data: &delegatepb.Delegate{
						User: &userpb.User{
							FirstName:    createSuccessResolver.MustGetString("newDelegateFirstName"),
							LastName:     createSuccessResolver.MustGetString("newDelegateLastName"),
							EmailAddress: createSuccessResolver.MustGetString("newDelegateEmailAddress"),
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *delegatepb.CreateDelegateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdDelegate := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newDelegateFirstName"), createdDelegate.User.FirstName, "delegate first name")
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newDelegateLastName"), createdDelegate.User.LastName, "delegate last name")
				testutil.AssertNonEmptyString(t, createdDelegate.Id, "delegate ID")
				testutil.AssertTrue(t, createdDelegate.Active, "delegate active status")
				testutil.AssertFieldSet(t, createdDelegate.DateCreated, "DateCreated")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.CreateDelegateRequest {
				return &delegatepb.CreateDelegateRequest{
					Data: &delegatepb.Delegate{
						User: &userpb.User{
							FirstName:    createSuccessResolver.MustGetString("newDelegateFirstName"),
							LastName:     createSuccessResolver.MustGetString("newDelegateLastName"),
							EmailAddress: createSuccessResolver.MustGetString("newDelegateEmailAddress"),
						},
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *delegatepb.CreateDelegateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdDelegate := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newDelegateFirstName"), createdDelegate.User.FirstName, "delegate first name")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-CREATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.CreateDelegateRequest {
				return &delegatepb.CreateDelegateRequest{
					Data: &delegatepb.Delegate{
						User: &userpb.User{
							FirstName:    authorizationUnauthorizedResolver.MustGetString("unauthorizedDelegateFirstName"),
							LastName:     authorizationUnauthorizedResolver.MustGetString("unauthorizedDelegateLastName"),
							EmailAddress: authorizationUnauthorizedResolver.MustGetString("unauthorizedDelegateEmailAddress"),
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "delegate.errors.authorization_failed",
			Assertions: func(t *testing.T, response *delegatepb.CreateDelegateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.CreateDelegateRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "delegate.validation.request_required",
			Assertions: func(t *testing.T, response *delegatepb.CreateDelegateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.CreateDelegateRequest {
				return &delegatepb.CreateDelegateRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "delegate.validation.data_required",
			Assertions: func(t *testing.T, response *delegatepb.CreateDelegateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyFirstName",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-CREATE-VALIDATION-EMPTY-FIRST-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.CreateDelegateRequest {
				return &delegatepb.CreateDelegateRequest{
					Data: &delegatepb.Delegate{
						User: &userpb.User{
							FirstName:    validationErrorEmptyFirstNameResolver.MustGetString("emptyFirstName"),
							LastName:     validationErrorEmptyFirstNameResolver.MustGetString("validLastName"),
							EmailAddress: validationErrorEmptyFirstNameResolver.MustGetString("validEmailAddress"),
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "delegate.validation.first_name_required",
			Assertions: func(t *testing.T, response *delegatepb.CreateDelegateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty first name")
			},
		},
		{
			Name:     "EmptyLastName",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-CREATE-VALIDATION-EMPTY-LAST-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.CreateDelegateRequest {
				return &delegatepb.CreateDelegateRequest{
					Data: &delegatepb.Delegate{
						User: &userpb.User{
							FirstName:    validationErrorEmptyLastNameResolver.MustGetString("validFirstName"),
							LastName:     validationErrorEmptyLastNameResolver.MustGetString("emptyLastName"),
							EmailAddress: validationErrorEmptyLastNameResolver.MustGetString("validEmailAddress"),
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "delegate.validation.last_name_required",
			Assertions: func(t *testing.T, response *delegatepb.CreateDelegateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty last name")
			},
		},
		{
			Name:     "EmptyEmail",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-CREATE-VALIDATION-EMPTY-EMAIL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.CreateDelegateRequest {
				return &delegatepb.CreateDelegateRequest{
					Data: &delegatepb.Delegate{
						User: &userpb.User{
							FirstName:    validationErrorEmptyEmailResolver.MustGetString("validFirstName"),
							LastName:     validationErrorEmptyEmailResolver.MustGetString("validLastName"),
							EmailAddress: validationErrorEmptyEmailResolver.MustGetString("emptyEmailAddress"),
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "delegate.validation.email_required",
			Assertions: func(t *testing.T, response *delegatepb.CreateDelegateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty email")
			},
		},
		{
			Name:     "InvalidEmail",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-CREATE-VALIDATION-INVALID-EMAIL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.CreateDelegateRequest {
				return &delegatepb.CreateDelegateRequest{
					Data: &delegatepb.Delegate{
						User: &userpb.User{
							FirstName:    validationErrorInvalidEmailResolver.MustGetString("validFirstName"),
							LastName:     validationErrorInvalidEmailResolver.MustGetString("validLastName"),
							EmailAddress: validationErrorInvalidEmailResolver.MustGetString("invalidEmailAddress"),
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "delegate.validation.email_invalid",
			Assertions: func(t *testing.T, response *delegatepb.CreateDelegateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid email")
			},
		},
		{
			Name:     "FirstNameTooLong",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-CREATE-VALIDATION-FIRST-NAME-TOO-LONG-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.CreateDelegateRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "delegate", "ValidationError_FirstNameTooLong")
				testutil.AssertTestCaseLoad(t, err, "ValidationError_FirstNameTooLong")
				return &delegatepb.CreateDelegateRequest{
					Data: &delegatepb.Delegate{
						User: &userpb.User{
							FirstName:    resolver.MustGetString("tooLongFirstName"),
							LastName:     resolver.MustGetString("validLastName"),
							EmailAddress: resolver.MustGetString("validEmailAddress"),
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "delegate.validation.first_name_too_long",
			Assertions: func(t *testing.T, response *delegatepb.CreateDelegateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "first name too long")
			},
		},
		{
			Name:     "LastNameTooLong",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-CREATE-VALIDATION-LAST-NAME-TOO-LONG-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.CreateDelegateRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "delegate", "ValidationError_LastNameTooLong")
				testutil.AssertTestCaseLoad(t, err, "ValidationError_LastNameTooLong")
				return &delegatepb.CreateDelegateRequest{
					Data: &delegatepb.Delegate{
						User: &userpb.User{
							FirstName:    resolver.MustGetString("validFirstName"),
							LastName:     resolver.MustGetString("tooLongLastName"),
							EmailAddress: resolver.MustGetString("validEmailAddress"),
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "delegate.validation.last_name_too_long",
			Assertions: func(t *testing.T, response *delegatepb.CreateDelegateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "last name too long")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-CREATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.CreateDelegateRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "delegate", "DataEnrichment_TestDelegate")
				testutil.AssertTestCaseLoad(t, err, "DataEnrichment_TestDelegate")
				return &delegatepb.CreateDelegateRequest{
					Data: &delegatepb.Delegate{
						User: &userpb.User{
							FirstName:    resolver.MustGetString("enrichmentTestFirstName"),
							LastName:     resolver.MustGetString("enrichmentTestLastName"),
							EmailAddress: resolver.MustGetString("enrichmentTestEmailAddress"),
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *delegatepb.CreateDelegateResponse, err error, useCase interface{}, ctx context.Context) {
				createdDelegate := response.Data[0]
				testutil.AssertNonEmptyString(t, createdDelegate.Id, "generated ID")
				testutil.AssertNonEmptyString(t, createdDelegate.UserId, "generated UserId")
				testutil.AssertFieldSet(t, createdDelegate.DateCreated, "DateCreated")
				testutil.AssertTrue(t, createdDelegate.Active, "Active")
				testutil.AssertStringEqual(t, createdDelegate.UserId, createdDelegate.User.Id, "UserId equals User.Id")
			},
		},
		{
			Name:     "MinimalValidData",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-CREATE-VALIDATION-MINIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.CreateDelegateRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "delegate", "BoundaryTest_MinimalValid")
				testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MinimalValid")
				return &delegatepb.CreateDelegateRequest{
					Data: &delegatepb.Delegate{
						User: &userpb.User{
							FirstName:    resolver.MustGetString("minValidFirstName"),
							LastName:     resolver.MustGetString("minValidLastName"),
							EmailAddress: resolver.MustGetString("minValidEmailAddress"),
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *delegatepb.CreateDelegateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdDelegate := response.Data[0]
				resolver, _ := copyatestutil.LoadTestCaseFromBusinessType(t, testutil.GetTestBusinessType(), "delegate", "BoundaryTest_MinimalValid")
				testutil.AssertStringEqual(t, resolver.MustGetString("minValidFirstName"), createdDelegate.User.FirstName, "delegate first name")
			},
		},
		{
			Name:     "MaxValidData",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-CREATE-VALIDATION-MAXIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.CreateDelegateRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "delegate", "BoundaryTest_MaximalValid")
				testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MaximalValid")
				return &delegatepb.CreateDelegateRequest{
					Data: &delegatepb.Delegate{
						User: &userpb.User{
							FirstName:    resolver.MustGetString("maxValidFirstName"),
							LastName:     resolver.MustGetString("maxValidLastName"),
							EmailAddress: resolver.MustGetString("maxValidEmailAddress"),
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *delegatepb.CreateDelegateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdDelegate := response.Data[0]
				testutil.AssertEqual(t, 48, len(createdDelegate.User.FirstName), "first name length")
				testutil.AssertEqual(t, 48, len(createdDelegate.User.LastName), "last name length")
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

func TestCreateDelegateUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-DELEGATE-CREATE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockDelegateRepo := entity.NewMockDelegateRepository(businessType)

	repositories := CreateDelegateRepositories{
		Delegate: mockDelegateRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := CreateDelegateServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	useCase := NewCreateDelegateUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "delegate", "CreateDelegate_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateDelegate_Success")

	req := &delegatepb.CreateDelegateRequest{
		Data: &delegatepb.Delegate{
			User: &userpb.User{
				FirstName:    resolver.MustGetString("newDelegateFirstName"),
				LastName:     resolver.MustGetString("newDelegateLastName"),
				EmailAddress: resolver.MustGetString("newDelegateEmailAddress"),
			},
		},
	}

	_, err = useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}

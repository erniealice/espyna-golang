//go:build mock_db && mock_auth

// Package eventclient provides table-driven tests for the event client creation use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, and boundary conditions.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestCreateEventClientUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-EVENT-EVENT-CLIENT-CREATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-EVENT-EVENT-CLIENT-CREATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-EVENT-EVENT-CLIENT-CREATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-EVENT-EVENT-CLIENT-CREATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-EVENT-EVENT-CLIENT-CREATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-EVENT-EVENT-CLIENT-CREATE-VALIDATION-EMPTY-EVENT-ID-v1.0: EmptyEventId
//   - ESPYNA-TEST-EVENT-EVENT-CLIENT-CREATE-VALIDATION-EMPTY-CLIENT-ID-v1.0: EmptyClientId
//   - ESPYNA-TEST-EVENT-EVENT-CLIENT-CREATE-VALIDATION-INVALID-EVENT-ID-v1.0: InvalidEventId
//   - ESPYNA-TEST-EVENT-EVENT-CLIENT-CREATE-VALIDATION-INVALID-CLIENT-ID-v1.0: InvalidClientId
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/event_client.json
//   - Mock data: packages/copya/data/{businessType}/event_client.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/event_client.json
package eventclient

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockEntity "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"
	mockEvent "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/event"
	eventclientpb "leapfor.xyz/esqyma/golang/v1/domain/event/event_client"
)

// Type alias for create event client test cases
type CreateEventClientTestCase = testutil.GenericTestCase[*eventclientpb.CreateEventClientRequest, *eventclientpb.CreateEventClientResponse]

func createTestCreateEventClientUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreateEventClientUseCase {
	mockEventClientRepo := mockEvent.NewEventClientRepository(businessType)
	mockEventRepo := mockEvent.NewMockEventRepository(businessType)
	mockClientRepo := mockEntity.NewMockClientRepository(businessType)

	repositories := CreateEventClientRepositories{
		EventClient: mockEventClientRepo,
		Event:       mockEventRepo,
		Client:      mockClientRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreateEventClientServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreateEventClientUseCase(repositories, services)
}

func TestCreateEventClientUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "event_client", "CreateEventClient_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateEventClient_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "event_client", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	validationErrorEmptyEventIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "event_client", "ValidationError_EmptyEventId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyEventId")

	validationErrorEmptyClientIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "event_client", "ValidationError_EmptyClientId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyClientId")

	testCases := []CreateEventClientTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-CLIENT-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventclientpb.CreateEventClientRequest {
				return &eventclientpb.CreateEventClientRequest{
					Data: &eventclientpb.EventClient{
						EventId:  createSuccessResolver.MustGetString("validEventId"),
						ClientId: createSuccessResolver.MustGetString("validClientId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *eventclientpb.CreateEventClientResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdEventClient := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("validEventId"), createdEventClient.EventId, "event ID")
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("validClientId"), createdEventClient.ClientId, "client ID")
				testutil.AssertNonEmptyString(t, createdEventClient.Id, "event client ID")
				testutil.AssertTrue(t, createdEventClient.Active, "event client active status")
				testutil.AssertFieldSet(t, createdEventClient.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdEventClient.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-CLIENT-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventclientpb.CreateEventClientRequest {
				return &eventclientpb.CreateEventClientRequest{
					Data: &eventclientpb.EventClient{
						EventId:  createSuccessResolver.MustGetString("validEventId"),
						ClientId: createSuccessResolver.MustGetString("validClientId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *eventclientpb.CreateEventClientResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdEventClient := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("validEventId"), createdEventClient.EventId, "event ID")
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("validClientId"), createdEventClient.ClientId, "client ID")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-CLIENT-CREATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventclientpb.CreateEventClientRequest {
				return &eventclientpb.CreateEventClientRequest{
					Data: &eventclientpb.EventClient{
						EventId:  authorizationUnauthorizedResolver.MustGetString("unauthorizedEventId"),
						ClientId: authorizationUnauthorizedResolver.MustGetString("unauthorizedClientId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "event_client.errors.authorization_failed",
			Assertions: func(t *testing.T, response *eventclientpb.CreateEventClientResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-CLIENT-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventclientpb.CreateEventClientRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "event_client.validation.request_required",
			Assertions: func(t *testing.T, response *eventclientpb.CreateEventClientResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-CLIENT-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventclientpb.CreateEventClientRequest {
				return &eventclientpb.CreateEventClientRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "event_client.validation.data_required",
			Assertions: func(t *testing.T, response *eventclientpb.CreateEventClientResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyEventId",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-CLIENT-CREATE-VALIDATION-EMPTY-EVENT-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventclientpb.CreateEventClientRequest {
				return &eventclientpb.CreateEventClientRequest{
					Data: &eventclientpb.EventClient{
						EventId:  validationErrorEmptyEventIdResolver.MustGetString("emptyEventId"),
						ClientId: validationErrorEmptyEventIdResolver.MustGetString("validClientId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "event_client.validation.event_id_required",
			Assertions: func(t *testing.T, response *eventclientpb.CreateEventClientResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty event ID")
			},
		},
		{
			Name:     "EmptyClientId",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-CLIENT-CREATE-VALIDATION-EMPTY-CLIENT-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventclientpb.CreateEventClientRequest {
				return &eventclientpb.CreateEventClientRequest{
					Data: &eventclientpb.EventClient{
						EventId:  validationErrorEmptyClientIdResolver.MustGetString("validEventId"),
						ClientId: validationErrorEmptyClientIdResolver.MustGetString("emptyClientId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "event_client.validation.client_id_required",
			Assertions: func(t *testing.T, response *eventclientpb.CreateEventClientResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty client ID")
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
			useCase := createTestCreateEventClientUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

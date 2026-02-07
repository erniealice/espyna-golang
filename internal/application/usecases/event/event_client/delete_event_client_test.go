//go:build mock_db && mock_auth

// Package eventclient provides table-driven tests for the event client deletion use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, and validation errors.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestDeleteEventClientUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-EVENT-EVENT-CLIENT-DELETE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-EVENT-EVENT-CLIENT-DELETE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-EVENT-EVENT-CLIENT-DELETE-NOT-FOUND-v1.0: NonExistentEventClient
//   - ESPYNA-TEST-EVENT-EVENT-CLIENT-DELETE-VALIDATION-EMPTY-ID-v1.0: EmptyEventClientId
//   - ESPYNA-TEST-EVENT-EVENT-CLIENT-DELETE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-EVENT-EVENT-CLIENT-DELETE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-EVENT-EVENT-CLIENT-DELETE-AUTHORIZATION-DENIED-v1.0: AuthorizationDenied
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

// Type alias for delete event client test cases
type DeleteEventClientTestCase = testutil.GenericTestCase[*eventclientpb.DeleteEventClientRequest, *eventclientpb.DeleteEventClientResponse]

func createTestDeleteEventClientUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *DeleteEventClientUseCase {
	mockEventClientRepo := mockEvent.NewEventClientRepository(businessType)
	mockEventRepo := mockEvent.NewMockEventRepository(businessType)
	mockClientRepo := mockEntity.NewMockClientRepository(businessType)

	repositories := DeleteEventClientRepositories{
		EventClient: mockEventClientRepo,
		Event:       mockEventRepo,
		Client:      mockClientRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := DeleteEventClientServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewDeleteEventClientUseCase(repositories, services)
}

func TestDeleteEventClientUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers
	commonDataResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "event_client", "EventClient_CommonData")
	testutil.AssertTestCaseLoad(t, err, "EventClient_CommonData")

	successResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "event_client", "DeleteEventClient_Success")
	testutil.AssertTestCaseLoad(t, err, "DeleteEventClient_Success")

	emptyIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "event_client", "ValidationError_EmptyId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyId")

	testCases := []DeleteEventClientTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-CLIENT-DELETE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventclientpb.DeleteEventClientRequest {
				return &eventclientpb.DeleteEventClientRequest{
					Data: &eventclientpb.EventClient{Id: successResolver.MustGetString("targetEventClientId")},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *eventclientpb.DeleteEventClientResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "successful deletion")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-CLIENT-DELETE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventclientpb.DeleteEventClientRequest {
				return &eventclientpb.DeleteEventClientRequest{
					Data: &eventclientpb.EventClient{Id: successResolver.MustGetString("targetEventClientId")},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *eventclientpb.DeleteEventClientResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "successful deletion")
			},
		},
		{
			Name:     "NonExistentEventClient",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-CLIENT-DELETE-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventclientpb.DeleteEventClientRequest {
				return &eventclientpb.DeleteEventClientRequest{
					Data: &eventclientpb.EventClient{Id: commonDataResolver.MustGetString("nonExistentId")},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "event_client.errors.not_found",
			ErrorTags:      map[string]any{"eventClientId": commonDataResolver.MustGetString("nonExistentId")},
		},
		{
			Name:     "EmptyEventClientId",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-CLIENT-DELETE-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventclientpb.DeleteEventClientRequest {
				return &eventclientpb.DeleteEventClientRequest{
					Data: &eventclientpb.EventClient{Id: emptyIdResolver.MustGetString("emptyId")},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "event_client.validation.id_required",
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-CLIENT-DELETE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventclientpb.DeleteEventClientRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "event_client.validation.request_required",
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-CLIENT-DELETE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventclientpb.DeleteEventClientRequest {
				return &eventclientpb.DeleteEventClientRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "event_client.validation.data_required",
		},
		{
			Name:     "AuthorizationDenied",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-CLIENT-DELETE-AUTHORIZATION-DENIED-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventclientpb.DeleteEventClientRequest {
				return &eventclientpb.DeleteEventClientRequest{
					Data: &eventclientpb.EventClient{Id: commonDataResolver.MustGetString("primaryEventClientId")},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "event_client.errors.authorization_failed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			testutil.SetTestCode(t, tc.TestCode)
			testutil.LogTestExecution(t, tc.TestCode, tc.Name, tc.ExpectSuccess)

			ctx := testutil.CreateTestContext()
			businessType := testutil.GetTestBusinessType()
			useCase := createTestDeleteEventClientUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

			req := tc.SetupRequest(t, businessType)
			response, err := useCase.Execute(ctx, req)

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

			testutil.LogTestResult(t, tc.TestCode, tc.Name, actualSuccess, err)
		})
	}
}

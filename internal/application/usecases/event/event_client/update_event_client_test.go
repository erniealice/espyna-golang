//go:build mock_db && mock_auth

// Package eventclient provides table-driven tests for the event client update use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, validation errors, and not found cases. Each test case is
// defined in a table with a specific test code, request setup, and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestUpdateEventClientUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-EVENT-EVENT-CLIENT-UPDATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-EVENT-EVENT-CLIENT-UPDATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-EVENT-EVENT-CLIENT-UPDATE-NOT-FOUND-v1.0: NotFound
//   - ESPYNA-TEST-EVENT-EVENT-CLIENT-UPDATE-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-EVENT-EVENT-CLIENT-UPDATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-EVENT-EVENT-CLIENT-UPDATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-EVENT-EVENT-CLIENT-UPDATE-AUTHORIZATION-v1.0: Unauthorized
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/event_client.json
//   - Mock data: packages/copya/data/{businessType}/event_client.json
package eventclient

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	mockEntity "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	mockEvent "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/event"
	eventclientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_client"
)

// Type alias for update event client test cases
type UpdateEventClientTestCase = testutil.GenericTestCase[*eventclientpb.UpdateEventClientRequest, *eventclientpb.UpdateEventClientResponse]

func createTestUpdateEventClientUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *UpdateEventClientUseCase {
	mockEventClientRepo := mockEvent.NewEventClientRepository(businessType)
	mockEventRepo := mockEvent.NewMockEventRepository(businessType)
	mockClientRepo := mockEntity.NewMockClientRepository(businessType)

	repositories := UpdateEventClientRepositories{
		EventClient: mockEventClientRepo,
		Event:       mockEventRepo,
		Client:      mockClientRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := UpdateEventClientServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewUpdateEventClientUseCase(repositories, services)
}

func TestUpdateEventClientUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers
	commonDataResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "event_client", "EventClient_CommonData")
	testutil.AssertTestCaseLoad(t, err, "EventClient_CommonData")

	successResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "event_client", "UpdateEventClient_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdateEventClient_Success")

	testCases := []UpdateEventClientTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-CLIENT-UPDATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventclientpb.UpdateEventClientRequest {
				return &eventclientpb.UpdateEventClientRequest{
					Data: &eventclientpb.EventClient{
						Id:       successResolver.MustGetString("targetEventClientId"),
						EventId:  successResolver.MustGetString("updatedEventId"),
						ClientId: successResolver.MustGetString("updatedClientId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *eventclientpb.UpdateEventClientResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				updatedEventClient := response.Data[0]
				testutil.AssertStringEqual(t, successResolver.MustGetString("updatedEventId"), updatedEventClient.EventId, "event ID")
				testutil.AssertStringEqual(t, successResolver.MustGetString("updatedClientId"), updatedEventClient.ClientId, "client ID")
				testutil.AssertFieldSet(t, updatedEventClient.DateModified, "DateModified")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-CLIENT-UPDATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventclientpb.UpdateEventClientRequest {
				return &eventclientpb.UpdateEventClientRequest{
					Data: &eventclientpb.EventClient{
						Id:       successResolver.MustGetString("targetEventClientId"),
						EventId:  successResolver.MustGetString("updatedEventId"),
						ClientId: successResolver.MustGetString("updatedClientId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *eventclientpb.UpdateEventClientResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				updatedEventClient := response.Data[0]
				testutil.AssertStringEqual(t, successResolver.MustGetString("updatedEventId"), updatedEventClient.EventId, "event ID")
				testutil.AssertStringEqual(t, successResolver.MustGetString("updatedClientId"), updatedEventClient.ClientId, "client ID")
			},
		},
		{
			Name:     "NotFound",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-CLIENT-UPDATE-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventclientpb.UpdateEventClientRequest {
				return &eventclientpb.UpdateEventClientRequest{
					Data: &eventclientpb.EventClient{
						Id:       commonDataResolver.MustGetString("nonExistentId"),
						EventId:  "some-event",
						ClientId: "some-client",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "event_client.errors.not_found",
			ErrorTags:      map[string]any{"eventClientId": commonDataResolver.MustGetString("nonExistentId")},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-CLIENT-UPDATE-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventclientpb.UpdateEventClientRequest {
				return &eventclientpb.UpdateEventClientRequest{Data: &eventclientpb.EventClient{Id: "", EventId: "some-event", ClientId: "some-client"}}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "event_client.validation.id_required",
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-CLIENT-UPDATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventclientpb.UpdateEventClientRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "event_client.validation.request_required",
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-CLIENT-UPDATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventclientpb.UpdateEventClientRequest {
				return &eventclientpb.UpdateEventClientRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "event_client.validation.data_required",
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-CLIENT-UPDATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventclientpb.UpdateEventClientRequest {
				return &eventclientpb.UpdateEventClientRequest{
					Data: &eventclientpb.EventClient{
						Id:       successResolver.MustGetString("targetEventClientId"),
						EventId:  successResolver.MustGetString("updatedEventId"),
						ClientId: successResolver.MustGetString("updatedClientId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "event_client.errors.authorization_failed",
			Assertions: func(t *testing.T, response *eventclientpb.UpdateEventClientResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			testutil.SetTestCode(t, tc.TestCode)
			testutil.LogTestExecution(t, tc.TestCode, tc.Name, tc.ExpectSuccess)

			ctx := testutil.CreateTestContext()
			businessType := testutil.GetTestBusinessType()
			useCase := createTestUpdateEventClientUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

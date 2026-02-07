//go:build mock_db && mock_auth

// Package event provides table-driven tests for the event deletion use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestDeleteEventUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-EVENT-EVENT-DELETE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-EVENT-EVENT-DELETE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-EVENT-EVENT-DELETE-NOT-FOUND-v1.0: NonExistentEvent
//   - ESPYNA-TEST-EVENT-EVENT-DELETE-VALIDATION-EMPTY-ID-v1.0: EmptyEventId
//   - ESPYNA-TEST-EVENT-EVENT-DELETE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-EVENT-EVENT-DELETE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-EVENT-EVENT-DELETE-AUTHORIZATION-DENIED-v1.0: AuthorizationDenied
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/event.json
//   - Mock data: packages/copya/data/{businessType}/event.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/event.json
package event

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockEvent "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/event"
	eventpb "leapfor.xyz/esqyma/golang/v1/domain/event/event"
)

// Type alias for delete event test cases
type DeleteEventTestCase = testutil.GenericTestCase[*eventpb.DeleteEventRequest, *eventpb.DeleteEventResponse]

func createTestDeleteEventUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *DeleteEventUseCase {
	mockEventRepo := mockEvent.NewMockEventRepository(businessType)

	repositories := DeleteEventRepositories{
		Event: mockEventRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := DeleteEventServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewDeleteEventUseCase(repositories, services)
}

func TestDeleteEventUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers
	commonDataResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "event", "Event_CommonData")
	testutil.AssertTestCaseLoad(t, err, "Event_CommonData")

	successResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "event", "DeleteEvent_Success")
	testutil.AssertTestCaseLoad(t, err, "DeleteEvent_Success")

	emptyIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "event", "ValidationError_EmptyId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyId")

	testCases := []DeleteEventTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-DELETE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventpb.DeleteEventRequest {
				return &eventpb.DeleteEventRequest{
					Data: &eventpb.Event{Id: successResolver.MustGetString("targetEventId")},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *eventpb.DeleteEventResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "successful deletion")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-DELETE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventpb.DeleteEventRequest {
				return &eventpb.DeleteEventRequest{
					Data: &eventpb.Event{Id: successResolver.MustGetString("targetEventId")},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *eventpb.DeleteEventResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "successful deletion")
			},
		},
		{
			Name:     "NonExistentEvent",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-DELETE-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventpb.DeleteEventRequest {
				return &eventpb.DeleteEventRequest{
					Data: &eventpb.Event{Id: commonDataResolver.MustGetString("nonExistentId")},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "event.errors.not_found",
			ErrorTags:      map[string]any{"eventId": commonDataResolver.MustGetString("nonExistentId")},
		},
		{
			Name:     "EmptyEventId",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-DELETE-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventpb.DeleteEventRequest {
				return &eventpb.DeleteEventRequest{
					Data: &eventpb.Event{Id: emptyIdResolver.MustGetString("emptyId")},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "event.validation.id_required",
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-DELETE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventpb.DeleteEventRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "event.validation.request_required",
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-DELETE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventpb.DeleteEventRequest {
				return &eventpb.DeleteEventRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "event.validation.data_required",
		},
		{
			Name:     "AuthorizationDenied",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-DELETE-AUTHORIZATION-DENIED-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventpb.DeleteEventRequest {
				return &eventpb.DeleteEventRequest{
					Data: &eventpb.Event{Id: commonDataResolver.MustGetString("primaryEventId")},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "event.errors.authorization_failed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			testutil.SetTestCode(t, tc.TestCode)
			testutil.LogTestExecution(t, tc.TestCode, tc.Name, tc.ExpectSuccess)

			ctx := testutil.CreateTestContext()
			businessType := testutil.GetTestBusinessType()
			useCase := createTestDeleteEventUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

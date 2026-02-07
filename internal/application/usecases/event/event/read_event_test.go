//go:build mock_db && mock_auth

// Package event provides table-driven tests for the event reading use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, not found, and validation errors. Each test case is defined
// in a table with a specific test code, request setup, and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestReadEventUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-EVENT-EVENT-READ-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-EVENT-EVENT-READ-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-EVENT-EVENT-READ-NOT-FOUND-v1.0: NotFound
//   - ESPYNA-TEST-EVENT-EVENT-READ-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-EVENT-EVENT-READ-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-EVENT-EVENT-READ-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-EVENT-EVENT-READ-AUTHORIZATION-v1.0: Unauthorized
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/event.json
//   - Mock data: packages/copya/data/{businessType}/event.json
package event

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockEvent "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/event"
	eventpb "leapfor.xyz/esqyma/golang/v1/domain/event/event"
)

// Type alias for read event test cases
type ReadEventTestCase = testutil.GenericTestCase[*eventpb.ReadEventRequest, *eventpb.ReadEventResponse]

func createTestReadEventUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *ReadEventUseCase {
	mockEventRepo := mockEvent.NewMockEventRepository(businessType)

	repositories := ReadEventRepositories{
		Event: mockEventRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := ReadEventServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewReadEventUseCase(repositories, services)
}

func TestReadEventUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers
	commonDataResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "event", "Event_CommonData")
	testutil.AssertTestCaseLoad(t, err, "Event_CommonData")

	successResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "event", "ReadEvent_Success")
	testutil.AssertTestCaseLoad(t, err, "ReadEvent_Success")

	testCases := []ReadEventTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-READ-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventpb.ReadEventRequest {
				return &eventpb.ReadEventRequest{
					Data: &eventpb.Event{Id: successResolver.MustGetString("targetEventId")},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *eventpb.ReadEventResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				testutil.AssertStringEqual(t, successResolver.MustGetString("targetEventId"), response.Data[0].Id, "event ID")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-READ-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventpb.ReadEventRequest {
				return &eventpb.ReadEventRequest{
					Data: &eventpb.Event{Id: successResolver.MustGetString("targetEventId")},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *eventpb.ReadEventResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				testutil.AssertStringEqual(t, successResolver.MustGetString("targetEventId"), response.Data[0].Id, "event ID")
			},
		},
		{
			Name:     "NotFound",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-READ-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventpb.ReadEventRequest {
				return &eventpb.ReadEventRequest{
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
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-READ-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventpb.ReadEventRequest {
				return &eventpb.ReadEventRequest{Data: &eventpb.Event{Id: ""}}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "event.validation.id_required",
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-READ-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventpb.ReadEventRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "event.validation.request_required",
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-READ-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventpb.ReadEventRequest {
				return &eventpb.ReadEventRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "event.validation.data_required",
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-READ-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventpb.ReadEventRequest {
				return &eventpb.ReadEventRequest{
					Data: &eventpb.Event{Id: successResolver.MustGetString("targetEventId")},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "event.errors.authorization_failed",
			Assertions: func(t *testing.T, response *eventpb.ReadEventResponse, err error, useCase interface{}, ctx context.Context) {
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
			useCase := createTestReadEventUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

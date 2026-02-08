//go:build mock_db && mock_auth

// Package event provides table-driven tests for the event creation use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestCreateEventUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-EVENT-EVENT-CREATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-EVENT-EVENT-CREATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-EVENT-EVENT-CREATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-EVENT-EVENT-CREATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-EVENT-EVENT-CREATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-EVENT-EVENT-CREATE-VALIDATION-EMPTY-NAME-v1.0: EmptyName
//   - ESPYNA-TEST-EVENT-EVENT-CREATE-VALIDATION-NAME-TOO-LONG-v1.0: NameTooLong
//   - ESPYNA-TEST-EVENT-EVENT-CREATE-VALIDATION-INVALID-TIME-v1.0: InvalidTimeRange
//   - ESPYNA-TEST-EVENT-EVENT-CREATE-VALIDATION-PAST-EVENT-v1.0: PastEvent
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/event.json
//   - Mock data: packages/copya/data/{businessType}/event.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/event.json
package event

import (
	"context"
	"testing"
	"time"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	mockEvent "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/event"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
)

// Type alias for create event test cases
type CreateEventTestCase = testutil.GenericTestCase[*eventpb.CreateEventRequest, *eventpb.CreateEventResponse]

func createTestCreateEventUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreateEventUseCase {
	mockEventRepo := mockEvent.NewMockEventRepository(businessType)

	repositories := CreateEventRepositories{
		Event: mockEventRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreateEventServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreateEventUseCase(repositories, services)
}

func TestCreateEventUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers
	successResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "event", "CreateEvent_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateEvent_Success")

	unauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "event", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	emptyNameResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "event", "ValidationError_EmptyName")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyName")

	nameTooLongResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "event", "ValidationError_NameTooLong")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_NameTooLong")

	testCases := []CreateEventTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventpb.CreateEventRequest {
				return &eventpb.CreateEventRequest{
					Data: &eventpb.Event{
						Name:        successResolver.MustGetString("newEventName"),
						Description: &[]string{successResolver.MustGetString("newEventDescription")}[0],
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *eventpb.CreateEventResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdEvent := response.Data[0]
				testutil.AssertStringEqual(t, successResolver.MustGetString("newEventName"), createdEvent.Name, "event name")
				testutil.AssertNonEmptyString(t, createdEvent.Id, "event ID")
				testutil.AssertTrue(t, createdEvent.Active, "event active status")
				testutil.AssertFieldSet(t, createdEvent.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdEvent.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventpb.CreateEventRequest {
				return &eventpb.CreateEventRequest{
					Data: &eventpb.Event{
						Name:        successResolver.MustGetString("newEventName"),
						Description: &[]string{successResolver.MustGetString("newEventDescription")}[0],
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *eventpb.CreateEventResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdEvent := response.Data[0]
				testutil.AssertStringEqual(t, successResolver.MustGetString("newEventName"), createdEvent.Name, "event name")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-CREATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventpb.CreateEventRequest {
				return &eventpb.CreateEventRequest{
					Data: &eventpb.Event{
						Name: unauthorizedResolver.MustGetString("unauthorizedEventName"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "event.errors.authorization_failed",
			Assertions: func(t *testing.T, response *eventpb.CreateEventResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventpb.CreateEventRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "event.validation.request_required",
			Assertions: func(t *testing.T, response *eventpb.CreateEventResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventpb.CreateEventRequest {
				return &eventpb.CreateEventRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "event.validation.data_required",
			Assertions: func(t *testing.T, response *eventpb.CreateEventResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyName",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-CREATE-VALIDATION-EMPTY-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventpb.CreateEventRequest {
				return &eventpb.CreateEventRequest{
					Data: &eventpb.Event{Name: emptyNameResolver.MustGetString("emptyName")},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "event.validation.name_required",
			Assertions: func(t *testing.T, response *eventpb.CreateEventResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty name")
			},
		},
		{
			Name:     "NameTooLong",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-CREATE-VALIDATION-NAME-TOO-LONG-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventpb.CreateEventRequest {
				return &eventpb.CreateEventRequest{
					Data: &eventpb.Event{Name: nameTooLongResolver.MustGetString("longName")},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "event.validation.name_too_long",
			Assertions: func(t *testing.T, response *eventpb.CreateEventResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too long")
			},
		},
		{
			Name:     "InvalidTimeRange",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-CREATE-VALIDATION-INVALID-TIME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventpb.CreateEventRequest {
				return &eventpb.CreateEventRequest{
					Data: &eventpb.Event{
						Name:             "Invalid Time Event",
						StartDateTimeUtc: time.Now().Add(2 * time.Hour).UnixMilli(),
						EndDateTimeUtc:   time.Now().Add(1 * time.Hour).UnixMilli(),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "event.validation.end_time_after_start_time",
			Assertions: func(t *testing.T, response *eventpb.CreateEventResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid time range")
			},
		},
		{
			Name:     "PastEvent",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-CREATE-VALIDATION-PAST-EVENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventpb.CreateEventRequest {
				return &eventpb.CreateEventRequest{
					Data: &eventpb.Event{
						Name:             "Past Event",
						StartDateTimeUtc: time.Now().Add(-1 * time.Hour).UnixMilli(),
						EndDateTimeUtc:   time.Now().Add(1 * time.Hour).UnixMilli(),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "event.validation.start_time_in_future",
			Assertions: func(t *testing.T, response *eventpb.CreateEventResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "past event")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			testutil.SetTestCode(t, tc.TestCode)
			testutil.LogTestExecution(t, tc.TestCode, tc.Name, tc.ExpectSuccess)

			ctx := testutil.CreateTestContext()
			businessType := testutil.GetTestBusinessType()
			useCase := createTestCreateEventUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

			req := tc.SetupRequest(t, businessType)
			response, err := useCase.Execute(ctx, req)

			actualSuccess := err == nil && tc.ExpectSuccess

			if tc.ExpectSuccess {
				testutil.AssertNoError(t, err)
				testutil.AssertNotNil(t, response, "response")
			} else {
				testutil.AssertError(t, err)
				if tc.ExpectedError != "" {
					testutil.AssertTranslatedError(t, err, tc.ExpectedError, useCase.services.TranslationService, ctx)
				}
			}

			if tc.Assertions != nil {
				tc.Assertions(t, response, err, useCase, ctx)
			}

			testutil.LogTestResult(t, tc.TestCode, tc.Name, actualSuccess, err)
		})
	}
}

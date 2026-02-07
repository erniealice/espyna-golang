//go:build mock_db && mock_auth

// Package eventclient provides table-driven tests for the event client listing use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, and nil requests. Each test case is defined in a table with
// a specific test code, request setup, and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestListEventClientsUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-EVENT-EVENT-CLIENT-LIST-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-EVENT-EVENT-CLIENT-LIST-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-EVENT-EVENT-CLIENT-LIST-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-EVENT-EVENT-CLIENT-LIST-AUTHORIZATION-v1.0: Unauthorized
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/event_client.json
//   - Mock data: packages/copya/data/{businessType}/event_client.json
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

// Type alias for list event clients test cases
type ListEventClientsTestCase = testutil.GenericTestCase[*eventclientpb.ListEventClientsRequest, *eventclientpb.ListEventClientsResponse]

func createTestListEventClientsUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *ListEventClientsUseCase {
	mockEventClientRepo := mockEvent.NewEventClientRepository(businessType)
	mockEventRepo := mockEvent.NewMockEventRepository(businessType)
	mockClientRepo := mockEntity.NewMockClientRepository(businessType)

	repositories := ListEventClientsRepositories{
		EventClient: mockEventClientRepo,
		Event:       mockEventRepo,
		Client:      mockClientRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := ListEventClientsServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewListEventClientsUseCase(repositories, services)
}

func TestListEventClientsUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers
	successResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "event_client", "ListEventClients_Success")
	testutil.AssertTestCaseLoad(t, err, "ListEventClients_Success")

	testCases := []ListEventClientsTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-CLIENT-LIST-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventclientpb.ListEventClientsRequest {
				return &eventclientpb.ListEventClientsRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *eventclientpb.ListEventClientsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertGreaterThanOrEqual(t, len(response.Data), int(successResolver.MustGetInt("expectedCount")), "event client count")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-CLIENT-LIST-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventclientpb.ListEventClientsRequest {
				return &eventclientpb.ListEventClientsRequest{}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *eventclientpb.ListEventClientsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertGreaterThanOrEqual(t, len(response.Data), int(successResolver.MustGetInt("expectedCount")), "event client count")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-CLIENT-LIST-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventclientpb.ListEventClientsRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true, // List use cases should handle nil request gracefully
			Assertions: func(t *testing.T, response *eventclientpb.ListEventClientsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success for nil request")
				testutil.AssertGreaterThanOrEqual(t, len(response.Data), int(successResolver.MustGetInt("expectedCount")), "event client count for nil request")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-EVENT-EVENT-CLIENT-LIST-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *eventclientpb.ListEventClientsRequest {
				return &eventclientpb.ListEventClientsRequest{}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "event_client.errors.authorization_failed",
			Assertions: func(t *testing.T, response *eventclientpb.ListEventClientsResponse, err error, useCase interface{}, ctx context.Context) {
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
			useCase := createTestListEventClientsUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

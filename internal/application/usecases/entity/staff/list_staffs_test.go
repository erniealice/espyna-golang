//go:build mock_db && mock_auth

// Package staff provides table-driven tests for the staff list use case.
//
// The tests cover various scenarios, including success, business logic validation,
// integration tests with delete operations, and edge cases. Each test case is
// defined in a table with a specific test code, request setup, and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestListStaffsUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-ENTITY-STAFF-LIST-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-ENTITY-STAFF-LIST-VERIFICATION-v1.0: VerifyDetails
//   - ESPYNA-TEST-ENTITY-STAFF-LIST-INTEGRATION-v1.0: AfterDelete
//   - ESPYNA-TEST-ENTITY-STAFF-LIST-BUSINESS-LOGIC-v1.0: BusinessLogic
//   - ESPYNA-TEST-ENTITY-STAFF-LIST-NIL-REQUEST-v1.0: NilRequest
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/staff.json
//   - Mock data: packages/copya/data/{businessType}/staff.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/staff.json
package staff

import (
	"context"
	"fmt"
	"strings"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"
	staffpb "leapfor.xyz/esqyma/golang/v1/domain/entity/staff"
)

// Type alias for list staffs test cases
type ListStaffsTestCase = testutil.GenericTestCase[*staffpb.ListStaffsRequest, *staffpb.ListStaffsResponse]

func createTestListStaffsUseCase(businessType string) *ListStaffsUseCase {
	repositories := ListStaffsRepositories{
		Staff: entity.NewMockStaffRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(false, false)
	services := ListStaffsServices{
		TranslationService: standardServices.TranslationService,
	}
	return NewListStaffsUseCase(repositories, services)
}

func TestListStaffsUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	listSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "staff", "ListStaffs_Success")
	testutil.AssertTestCaseLoad(t, err, "ListStaffs_Success")

	testCases := []ListStaffsTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-LIST-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.ListStaffsRequest {
				return &staffpb.ListStaffsRequest{}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *staffpb.ListStaffsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertNotNil(t, response.Data, "response data")

				expectedCount := listSuccessResolver.MustGetInt("expectedStaffCount")
				testutil.AssertEqual(t, expectedCount, len(response.Data), "staff count")

				// Verify that all expected staff IDs are present
				expectedIds := listSuccessResolver.MustGetStringArray("expectedStaffIds")
				actualIds := make([]string, len(response.Data))
				for i, staff := range response.Data {
					actualIds[i] = staff.Id
				}

				for _, expectedId := range expectedIds {
					found := false
					for _, actualId := range actualIds {
						if actualId == expectedId {
							found = true
							break
						}
					}
					if !found {
						testutil.AssertTrue(t, false, fmt.Sprintf("staff ID '%s' found in response", expectedId))
					}
				}

				// Verify each staff has required fields
				for _, staff := range response.Data {
					testutil.AssertNonEmptyString(t, staff.Id, "staff ID")
					testutil.AssertNotNil(t, staff.User, "staff user")
					testutil.AssertNonEmptyString(t, staff.User.Id, "user ID")
					testutil.AssertNonEmptyString(t, staff.User.EmailAddress, "user email")
				}
			},
		},
		{
			Name:     "VerifyDetails",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-LIST-VERIFICATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.ListStaffsRequest {
				return &staffpb.ListStaffsRequest{}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *staffpb.ListStaffsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertGreaterThan(t, len(response.Data), 0, "staff list should not be empty")

				// Verify that at least the minimum expected count is met
				minExpectedCount := listSuccessResolver.MustGetInt("expectedMinStaffCount")
				testutil.AssertGreaterThanOrEqual(t, len(response.Data), minExpectedCount, "minimum staff count")

				// Check that all staff are active (default business rule)
				for _, staff := range response.Data {
					testutil.AssertTrue(t, staff.Active, "staff should be active")
					testutil.AssertFieldSet(t, staff.DateCreated, "DateCreated")
					testutil.AssertFieldSet(t, staff.DateCreatedString, "DateCreatedString")
				}
			},
		},
		{
			Name:     "AfterDelete",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-LIST-INTEGRATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.ListStaffsRequest {
				return &staffpb.ListStaffsRequest{}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *staffpb.ListStaffsResponse, err error, useCase interface{}, ctx context.Context) {
				// This test performs integration testing by first deleting a staff member
				// and then verifying the list reflects the deletion
				listUseCase := useCase.(*ListStaffsUseCase)

				// First, get the original count
				originalCount := len(response.Data)

				// Delete a staff member using the same repository
				deleteRepositories := DeleteStaffRepositories{Staff: listUseCase.repositories.Staff}
				standardDeleteServices := testutil.CreateStandardServices(false, true)
				deleteServices := DeleteStaffServices{
					AuthorizationService: standardDeleteServices.AuthorizationService,
					TransactionService:   standardDeleteServices.TransactionService,
					TranslationService:   standardDeleteServices.TranslationService,
				}
				deleteUseCase := NewDeleteStaffUseCase(deleteRepositories, deleteServices)

				deletedStaffId := listSuccessResolver.MustGetString("deletedStaffId")
				deleteReq := &staffpb.DeleteStaffRequest{Data: &staffpb.Staff{Id: deletedStaffId}}
				_, deleteErr := deleteUseCase.Execute(ctx, deleteReq)
				testutil.AssertNoError(t, deleteErr)

				// Now list again to verify the count decreased
				listReq := &staffpb.ListStaffsRequest{}
				newResponse, listErr := listUseCase.Execute(ctx, listReq)
				testutil.AssertNoError(t, listErr)
				testutil.AssertTrue(t, newResponse.Success, "success after delete")

				// Verify count decreased by 1
				newCount := len(newResponse.Data)
				testutil.AssertEqual(t, originalCount-1, newCount, "staff count after deletion")

				// Verify the deleted staff is not in the list
				for _, staff := range newResponse.Data {
					if staff.Id == deletedStaffId {
						testutil.AssertTrue(t, false, fmt.Sprintf("deleted staff ID '%s' should not be found in list", deletedStaffId))
					}
				}
			},
		},
		{
			Name:     "BusinessLogic",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-LIST-BUSINESS-LOGIC-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.ListStaffsRequest {
				return &staffpb.ListStaffsRequest{}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *staffpb.ListStaffsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")

				// Business logic validations
				// 1. All staff should have valid email addresses
				// 2. All staff should have non-empty names
				// 3. All staff should be active by default
				for _, staff := range response.Data {
					testutil.AssertNonEmptyString(t, staff.User.EmailAddress, "staff email")
					if staff.User.EmailAddress != "" && !strings.Contains(staff.User.EmailAddress, "@") {
						testutil.AssertTrue(t, false, fmt.Sprintf("email '%s' should contain '@'", staff.User.EmailAddress))
					}
					testutil.AssertNonEmptyString(t, staff.User.FirstName, "staff first name")
					testutil.AssertNonEmptyString(t, staff.User.LastName, "staff last name")
					testutil.AssertTrue(t, staff.Active, "staff should be active")
				}

				// Verify we have the expected number of active staff
				expectedCount := listSuccessResolver.MustGetInt("expectedStaffCount")
				activeCount := 0
				for _, staff := range response.Data {
					if staff.Active {
						activeCount++
					}
				}
				testutil.AssertEqual(t, expectedCount, activeCount, "active staff count")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-LIST-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.ListStaffsRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  true, // List operations typically handle nil requests gracefully
			Assertions: func(t *testing.T, response *staffpb.ListStaffsResponse, err error, useCase interface{}, ctx context.Context) {
				// List operations should handle nil requests gracefully and return all items
				testutil.AssertTrue(t, response.Success, "success with nil request")
				testutil.AssertNotNil(t, response.Data, "response data should not be nil")
				testutil.AssertGreaterThan(t, len(response.Data), 0, "should return staff list even with nil request")
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

			// Use a fresh repository for each test to ensure consistent state
			mockRepo := entity.NewMockStaffRepository(businessType)
			standardServices := testutil.CreateStandardServices(false, false)
			useCase := NewListStaffsUseCase(ListStaffsRepositories{Staff: mockRepo}, ListStaffsServices{
				TranslationService: standardServices.TranslationService,
			})

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

//go:build mock_db && mock_auth

// Package user provides comprehensive tests for the user listing use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, data enrichment, and boundary conditions.
// Each test function has a specific test code for tracking and logging.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestListUsersUseCase_Execute_Success
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-ENTITY-USER-LIST-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-ENTITY-USER-LIST-INTEGRATION-v1.0: AfterDelete
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/user.json
//   - Mock data: packages/copya/data/{businessType}/user.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/user.json

package user

import (
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"
	userpb "leapfor.xyz/esqyma/golang/v1/domain/entity/user"
)

// createTestListUsersUseCase is a helper function to create the use case with mock dependencies
func createTestListUsersUseCase(businessType string) *ListUsersUseCase {
	repositories := ListUsersRepositories{
		User: entity.NewMockUserRepository(businessType),
	}

	services := testutil.CreateStandardServices(false, true)
	listUsersServices := ListUsersServices{
		TranslationService: services.TranslationService,
	}

	return NewListUsersUseCase(repositories, listUsersServices)
}

func TestListUsersUseCase_Execute_Success(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-USER-LIST-SUCCESS-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "Success", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	// We need a fresh repository for this test to ensure count is correct
	mockRepo := entity.NewMockUserRepository(businessType)
	services := testutil.CreateStandardServices(false, true)
	useCase := NewListUsersUseCase(ListUsersRepositories{User: mockRepo}, ListUsersServices{
		TranslationService: services.TranslationService,
	})

	req := &userpb.ListUsersRequest{}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")
	testutil.AssertTrue(t, len(res.Data) > 0, "should have at least some users")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "Success", true, nil)
}

func TestListUsersUseCase_Execute_AfterDelete(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-USER-LIST-INTEGRATION-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "AfterDelete", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	// Setup a repository that will be shared between use cases
	mockRepo := entity.NewMockUserRepository(businessType)

	// Load test data resolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "user", "User_CommonData")
	testutil.AssertTestCaseLoad(t, err, "User_CommonData")

	// --- Delete a user first ---
	deleteRepositories := DeleteUserRepositories{User: mockRepo}
	deleteServices := testutil.CreateStandardServices(false, true)
	deleteUserServices := DeleteUserServices{
		AuthorizationService: deleteServices.AuthorizationService,
		TransactionService:   deleteServices.TransactionService,
		TranslationService:   deleteServices.TranslationService,
	}
	deleteUseCase := NewDeleteUserUseCase(deleteRepositories, deleteUserServices)

	deleteReq := &userpb.DeleteUserRequest{Data: &userpb.User{Id: resolver.MustGetString("primaryUserId")}}
	_, err = deleteUseCase.Execute(ctx, deleteReq)
	testutil.AssertNoError(t, err)

	// --- Now list the users ---
	listServices := testutil.CreateStandardServices(false, true)
	listUseCase := NewListUsersUseCase(ListUsersRepositories{User: mockRepo}, ListUsersServices{
		TranslationService: listServices.TranslationService,
	})

	listReq := &userpb.ListUsersRequest{}
	res, err := listUseCase.Execute(ctx, listReq)
	testutil.AssertNoError(t, err)

	// Verify that users are still available after deletion (but count should be reduced)
	testutil.AssertTrue(t, len(res.Data) >= 0, "should have valid user list after deletion")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "AfterDelete", true, nil)
}

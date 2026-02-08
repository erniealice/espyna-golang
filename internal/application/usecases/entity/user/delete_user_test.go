//go:build mock_db && mock_auth

// Package user provides comprehensive tests for the user deletion use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestDeleteUserUseCase_Execute_Success
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-ENTITY-USER-DELETE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-ENTITY-USER-DELETE-NOT-FOUND-v1.0: NotFound
//   - ESPYNA-TEST-ENTITY-USER-DELETE-VALIDATION-ERRORS-v1.0: ValidationErrors
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/user.json
//   - Mock data: packages/copya/data/{businessType}/user.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/user.json

package user

import (
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// createTestDeleteUserUseCase is a helper function to create the use case with mock dependencies
func createTestDeleteUserUseCase(businessType string) *DeleteUserUseCase {
	repositories := DeleteUserRepositories{
		User: entity.NewMockUserRepository(businessType),
	}

	services := testutil.CreateStandardServices(false, true)
	deleteUserServices := DeleteUserServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return NewDeleteUserUseCase(repositories, deleteUserServices)
}

func TestDeleteUserUseCase_Execute_Success(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-USER-DELETE-SUCCESS-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "Success", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeleteUserUseCase(businessType)

	// Load test data resolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "user", "User_CommonData")
	testutil.AssertTestCaseLoad(t, err, "User_CommonData")

	// This ID will be "deleted" from the mock repository
	existingID := resolver.MustGetString("secondaryUserId")

	req := &userpb.DeleteUserRequest{
		Data: &userpb.User{Id: existingID},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")
	testutil.AssertTrue(t, res.Success, "success status")

	// Verify the user is actually deleted - use same repository instance
	readServices := testutil.CreateStandardServices(false, true)
	readUseCase := NewReadUserUseCase(
		ReadUserRepositories{User: useCase.repositories.User},
		ReadUserServices{
			TranslationService: readServices.TranslationService,
		},
	)
	readReq := &userpb.ReadUserRequest{Data: &userpb.User{Id: existingID}}
	_, err = readUseCase.Execute(ctx, readReq)
	testutil.AssertError(t, err)

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "Success", true, nil)
}

func TestDeleteUserUseCase_Execute_NotFound(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-USER-DELETE-NOT-FOUND-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "NotFound", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeleteUserUseCase(businessType)

	// Load test data resolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "user", "User_CommonData")
	testutil.AssertTestCaseLoad(t, err, "User_CommonData")

	nonExistentID := resolver.MustGetString("nonExistentId")
	req := &userpb.DeleteUserRequest{
		Data: &userpb.User{Id: nonExistentID},
	}

	_, err = useCase.Execute(ctx, req)
	testutil.AssertError(t, err)

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "NotFound", true, nil)
}

func TestDeleteUserUseCase_Execute_ValidationErrors(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-USER-DELETE-VALIDATION-ERRORS-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "ValidationErrors", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeleteUserUseCase(businessType)

	testCases := []struct {
		name             string
		user             *userpb.User
		expectedErrorKey string
	}{
		{
			name:             "Nil Data",
			user:             nil,
			expectedErrorKey: "user.validation.request_required",
		},
		{
			name:             "Empty ID",
			user:             &userpb.User{Id: ""},
			expectedErrorKey: "user.validation.id_required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &userpb.DeleteUserRequest{Data: tc.user}
			_, err := useCase.Execute(ctx, req)
			testutil.AssertError(t, err)

			testutil.AssertTranslatedError(t, err, tc.expectedErrorKey, useCase.services.TranslationService, ctx)
		})
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "ValidationErrors", true, nil)
}

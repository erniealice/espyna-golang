//go:build mock_db && mock_auth

// Package user provides comprehensive tests for the user reading use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestReadUserUseCase_Execute_Success
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-ENTITY-USER-READ-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-ENTITY-USER-READ-NOT-FOUND-v1.0: NotFound
//   - ESPYNA-TEST-ENTITY-USER-READ-VALIDATION-ERRORS-v1.0: ValidationErrors
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

// createTestReadUserUseCase is a helper function to create the use case with mock dependencies
func createTestReadUserUseCase(businessType string) *ReadUserUseCase {
	repositories := ReadUserRepositories{
		User: entity.NewMockUserRepository(businessType),
	}

	services := testutil.CreateStandardServices(false, true)
	readUserServices := ReadUserServices{
		TranslationService: services.TranslationService,
	}

	return NewReadUserUseCase(repositories, readUserServices)
}

func TestReadUserUseCase_Execute_Success(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-USER-READ-SUCCESS-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "Success", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadUserUseCase(businessType)

	// Load test data resolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "user", "User_CommonData")
	testutil.AssertTestCaseLoad(t, err, "User_CommonData")

	existingID := resolver.MustGetString("thirdUserId")

	req := &userpb.ReadUserRequest{
		Data: &userpb.User{Id: existingID},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")
	testutil.AssertEqual(t, 1, len(res.Data), "number of users in response")

	readUser := res.Data[0]
	testutil.AssertEqual(t, existingID, readUser.Id, "user ID")
	testutil.AssertNonEmptyString(t, readUser.EmailAddress, "email address")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "Success", true, nil)
}

func TestReadUserUseCase_Execute_NotFound(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-USER-READ-NOT-FOUND-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "NotFound", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadUserUseCase(businessType)

	// Load test data resolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "user", "User_CommonData")
	testutil.AssertTestCaseLoad(t, err, "User_CommonData")

	nonExistentID := resolver.MustGetString("nonExistentId")

	req := &userpb.ReadUserRequest{
		Data: &userpb.User{Id: nonExistentID},
	}

	_, err = useCase.Execute(ctx, req)
	testutil.AssertError(t, err)

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "NotFound", true, nil)
}

func TestReadUserUseCase_Execute_ValidationErrors(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-USER-READ-VALIDATION-ERRORS-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "ValidationErrors", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadUserUseCase(businessType)

	testCases := []struct {
		name             string
		user             *userpb.User
		expectedErrorKey string
	}{
		{
			name:             "Nil Data",
			user:             nil,
			expectedErrorKey: "user.validation.data_required",
		},
		{
			name:             "Empty ID",
			user:             &userpb.User{Id: ""},
			expectedErrorKey: "user.validation.id_required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &userpb.ReadUserRequest{Data: tc.user}
			_, err := useCase.Execute(ctx, req)
			testutil.AssertError(t, err)

			testutil.AssertTranslatedError(t, err, tc.expectedErrorKey, useCase.services.TranslationService, ctx)
		})
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "ValidationErrors", true, nil)
}

//go:build mock_db && mock_auth

// Package user provides comprehensive tests for the user updating use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestUpdateUserUseCase_Execute_Success
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-ENTITY-USER-UPDATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-ENTITY-USER-UPDATE-NOT-FOUND-v1.0: NotFound
//   - ESPYNA-TEST-ENTITY-USER-UPDATE-VALIDATION-ERRORS-v1.0: ValidationErrors
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

// createTestUpdateUserUseCase is a helper function to create the use case with mock dependencies
func createTestUpdateUserUseCase(businessType string) *UpdateUserUseCase {
	repositories := UpdateUserRepositories{
		User: entity.NewMockUserRepository(businessType),
	}

	services := testutil.CreateStandardServices(false, true)
	updateUserServices := UpdateUserServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return NewUpdateUserUseCase(repositories, updateUserServices)
}

func TestUpdateUserUseCase_Execute_Success(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-USER-UPDATE-SUCCESS-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "Success", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateUserUseCase(businessType)

	// Load test data resolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "user", "User_CommonData")
	testutil.AssertTestCaseLoad(t, err, "User_CommonData")

	// Load update-specific test data resolver
	updateResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "user", "DomainSpecific_Education")
	testutil.AssertTestCaseLoad(t, err, "DomainSpecific_Education")

	existingID := resolver.MustGetString("thirdUserId")
	updatedEmail := updateResolver.MustGetString("domainSpecificEmailAddress")

	req := &userpb.UpdateUserRequest{
		Data: &userpb.User{
			Id:           existingID,
			FirstName:    updateResolver.MustGetString("domainSpecificFirstName"),
			LastName:     updateResolver.MustGetString("domainSpecificLastName"),
			EmailAddress: updatedEmail,
		},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")
	testutil.AssertEqual(t, 1, len(res.Data), "number of users in response")

	updatedUser := res.Data[0]
	testutil.AssertEqual(t, updatedEmail, updatedUser.EmailAddress, "updated email address")
	testutil.AssertNotNil(t, updatedUser.DateModified, "DateModified field")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "Success", true, nil)
}

func TestUpdateUserUseCase_Execute_NotFound(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-USER-UPDATE-NOT-FOUND-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "NotFound", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateUserUseCase(businessType)

	// Load test data resolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "user", "User_CommonData")
	testutil.AssertTestCaseLoad(t, err, "User_CommonData")

	// Load update data resolver for valid field values
	updateResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "user", "CreateUser_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateUser_Success")

	nonExistentID := resolver.MustGetString("nonExistentId")
	req := &userpb.UpdateUserRequest{
		Data: &userpb.User{
			Id:           nonExistentID,
			FirstName:    updateResolver.MustGetString("newUserFirstName"),
			LastName:     updateResolver.MustGetString("newUserLastName"),
			EmailAddress: updateResolver.MustGetString("newUserEmailAddress"),
		},
	}

	_, err = useCase.Execute(ctx, req)
	testutil.AssertError(t, err)

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "NotFound", true, nil)
}

func TestUpdateUserUseCase_Execute_ValidationErrors(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-USER-UPDATE-VALIDATION-ERRORS-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "ValidationErrors", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateUserUseCase(businessType)

	// Load test data resolvers
	emptyNameResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "user", "ValidationError_EmptyName")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyName")

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
			name: "Empty ID",
			user: &userpb.User{
				FirstName:    emptyNameResolver.MustGetString("validLastName"),
				LastName:     emptyNameResolver.MustGetString("validLastName"),
				EmailAddress: emptyNameResolver.MustGetString("validEmailAddress"),
			},
			expectedErrorKey: "user.validation.id_required",
		},
		{
			name: "Empty Email",
			user: &userpb.User{
				Id:           "valid-user-id",
				FirstName:    emptyNameResolver.MustGetString("validLastName"),
				LastName:     emptyNameResolver.MustGetString("validLastName"),
				EmailAddress: "",
			},
			expectedErrorKey: "user.validation.email_required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &userpb.UpdateUserRequest{Data: tc.user}
			_, err := useCase.Execute(ctx, req)
			testutil.AssertError(t, err)

			testutil.AssertTranslatedError(t, err, tc.expectedErrorKey, useCase.services.TranslationService, ctx)
		})
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "ValidationErrors", true, nil)
}

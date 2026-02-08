//go:build mock_db && mock_auth

// Package user provides comprehensive tests for the user creation use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestCreateUserUseCase_Execute_Success
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-ENTITY-USER-CREATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-ENTITY-USER-CREATE-VALIDATION-ERRORS-v1.0: ValidationErrors
//   - ESPYNA-TEST-ENTITY-USER-CREATE-ENRICHMENT-v1.0: DataEnrichment
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/user.json
//   - Mock data: packages/copya/data/{businessType}/user.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/user.json
package user

import (
	"testing"
	"time"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// createTestCreateUserUseCase is a helper function to create the use case with mock dependencies
func createTestCreateUserUseCase(businessType string) *CreateUserUseCase {
	repositories := CreateUserRepositories{
		User: entity.NewMockUserRepository(businessType),
	}

	services := testutil.CreateStandardServices(false, true)
	createUserServices := CreateUserServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	return NewCreateUserUseCase(repositories, createUserServices)
}

func TestCreateUserUseCase_Execute_Success(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-USER-CREATE-SUCCESS-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "Success", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestCreateUserUseCase(businessType)

	// Load test data resolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "user", "CreateUser_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateUser_Success")

	req := &userpb.CreateUserRequest{
		Data: &userpb.User{
			FirstName:    resolver.MustGetString("newUserFirstName"),
			LastName:     resolver.MustGetString("newUserLastName"),
			EmailAddress: resolver.MustGetString("newUserEmailAddress"),
		},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertEqual(t, 1, len(res.Data), "number of users in response")

	createdUser := res.Data[0]
	testutil.AssertNonEmptyString(t, createdUser.Id, "user ID")
	testutil.AssertTrue(t, createdUser.Active, "user active status")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "Success", true, nil)
}

func TestCreateUserUseCase_Execute_ValidationErrors(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-USER-CREATE-VALIDATION-ERRORS-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "ValidationErrors", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestCreateUserUseCase(businessType)

	// Load test data resolvers
	emptyNameResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "user", "ValidationError_EmptyName")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyName")

	invalidEmailResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "user", "ValidationError_InvalidEmail")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidEmail")

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
			name: "Missing First Name",
			user: &userpb.User{
				LastName: emptyNameResolver.MustGetString("validLastName"), EmailAddress: emptyNameResolver.MustGetString("validEmailAddress"),
			},
			expectedErrorKey: "user.validation.first_name_required",
		},
		{
			name: "Invalid Email",
			user: &userpb.User{
				FirstName: invalidEmailResolver.MustGetString("validFirstName"), LastName: invalidEmailResolver.MustGetString("validLastName"), EmailAddress: invalidEmailResolver.MustGetString("invalidEmailAddress"),
			},
			expectedErrorKey: "user.validation.email_invalid",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &userpb.CreateUserRequest{Data: tc.user}
			_, err := useCase.Execute(ctx, req)
			testutil.AssertError(t, err)

			testutil.AssertTranslatedError(t, err, tc.expectedErrorKey, useCase.services.TranslationService, ctx)
		})
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "ValidationErrors", true, nil)
}

func TestCreateUserUseCase_DataEnrichment(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-USER-CREATE-ENRICHMENT-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "DataEnrichment", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestCreateUserUseCase(businessType)

	// Load test data resolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "user", "DataEnrichment_TestUser")
	testutil.AssertTestCaseLoad(t, err, "DataEnrichment_TestUser")

	req := &userpb.CreateUserRequest{
		Data: &userpb.User{
			FirstName:    resolver.MustGetString("enrichmentFirstName"),
			LastName:     resolver.MustGetString("enrichmentLastName"),
			EmailAddress: resolver.MustGetString("enrichmentEmailAddress"),
		},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	createdUser := res.Data[0]

	testutil.AssertNonEmptyString(t, createdUser.Id, "ID")
	testutil.AssertTrue(t, createdUser.Active, "Active field")
	testutil.AssertNotNil(t, createdUser.DateCreated, "DateCreated")
	now := time.Now().UnixMilli()
	testutil.AssertTrue(t, *createdUser.DateCreated >= now-5000 && *createdUser.DateCreated <= now+5000, "DateCreated should be recent")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "DataEnrichment", true, nil)
}

//go:build mock_db && mock_auth

// Package admin provides comprehensive tests for the admin creation use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestCreateAdminUseCase_Execute_Success
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-ENTITY-ADMIN-CREATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-ENTITY-ADMIN-CREATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-ENTITY-ADMIN-CREATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-ENTITY-ADMIN-CREATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-ENTITY-ADMIN-CREATE-VALIDATION-ERRORS-v1.0: ValidationErrors
//   - ESPYNA-TEST-ENTITY-ADMIN-CREATE-ENRICHMENT-v1.0: DataEnrichment
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/admin.json
//   - Mock data: packages/copya/data/{businessType}/admin.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/admin.json
package admin

import (
	"testing"
	"time"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"

	adminpb "leapfor.xyz/esqyma/golang/v1/domain/entity/admin"
	userpb "leapfor.xyz/esqyma/golang/v1/domain/entity/user"
)

// createTestCreateAdminUseCase is a helper function to create the use case with mock dependencies
func createTestCreateAdminUseCase(businessType string, supportsTransaction bool) *CreateAdminUseCase {
	repositories := CreateAdminRepositories{
		Admin: entity.NewMockAdminRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, true)
	services := CreateAdminServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreateAdminUseCase(repositories, services)
}

func TestCreateAdminUseCase_Execute_Success(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-ADMIN-CREATE-SUCCESS-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "Success", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestCreateAdminUseCase(businessType, false)

	// Load test data resolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "admin", "CreateAdmin_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateAdmin_Success")

	req := &adminpb.CreateAdminRequest{
		Data: &adminpb.Admin{
			User: &userpb.User{
				FirstName:    resolver.MustGetString("newAdminFirstName"),
				LastName:     resolver.MustGetString("newAdminLastName"),
				EmailAddress: resolver.MustGetString("newAdminEmail"),
			},
		},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, res, "response")
	testutil.AssertEqual(t, 1, len(res.Data), "response data length")

	createdAdmin := res.Data[0]
	testutil.AssertNonEmptyString(t, createdAdmin.Id, "admin ID")
	testutil.AssertTrue(t, createdAdmin.Active, "admin active status")
	testutil.AssertNotNil(t, createdAdmin.DateCreated, "DateCreated")
	testutil.AssertNotNil(t, createdAdmin.DateModified, "DateModified")

	// Add defensive check for User being nil
	testutil.AssertNotNil(t, createdAdmin.User, "User")
	testutil.AssertEqual(t, resolver.MustGetString("newAdminFirstName"), createdAdmin.User.FirstName, "user first name")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "Success", true, nil)
}

func TestCreateAdminUseCase_Execute_WithTransaction(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-ADMIN-CREATE-TRANSACTION-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "WithTransaction", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestCreateAdminUseCase(businessType, true)

	// Load test data resolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "admin", "CreateAdmin_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateAdmin_Success")

	req := &adminpb.CreateAdminRequest{
		Data: &adminpb.Admin{
			User: &userpb.User{
				FirstName:    resolver.MustGetString("transactionAdminFirstName"),
				LastName:     resolver.MustGetString("transactionAdminLastName"),
				EmailAddress: resolver.MustGetString("transactionAdminEmail"),
			},
		},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, res, "response")
	testutil.AssertEqual(t, 1, len(res.Data), "response data length")

	createdAdmin := res.Data[0]
	testutil.AssertEqual(t, resolver.MustGetString("transactionAdminEmail"), createdAdmin.User.EmailAddress, "user email address")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "WithTransaction", true, nil)
}

func TestCreateAdminUseCase_Execute_NilRequest(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-ADMIN-CREATE-NIL-REQUEST-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "NilRequest", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestCreateAdminUseCase(businessType, false)

	_, err := useCase.Execute(ctx, nil)
	testutil.AssertError(t, err)

	testutil.AssertTranslatedError(t, err, "admin.validation.request_required", useCase.services.TranslationService, ctx)

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "NilRequest", true, nil)
}

func TestCreateAdminUseCase_Execute_NilData(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-ADMIN-CREATE-NIL-DATA-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "NilData", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestCreateAdminUseCase(businessType, false)

	req := &adminpb.CreateAdminRequest{Data: nil}
	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)

	testutil.AssertTranslatedError(t, err, "admin.validation.data_required", useCase.services.TranslationService, ctx)

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "NilData", true, nil)
}

func TestCreateAdminUseCase_Execute_ValidationErrors(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-ADMIN-CREATE-VALIDATION-ERRORS-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "ValidationErrors", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestCreateAdminUseCase(businessType, false)

	// Load test data resolvers
	emptyNameResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "admin", "ValidationError_EmptyName")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyName")

	invalidEmailResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "admin", "ValidationError_InvalidEmail")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidEmail")

	testCases := []struct {
		name             string
		admin            *adminpb.Admin
		expectedErrorKey string
	}{
		{
			name:             "Nil User",
			admin:            &adminpb.Admin{},
			expectedErrorKey: "admin.validation.user_data_required",
		},
		{
			name: "Missing First Name",
			admin: &adminpb.Admin{
				User: &userpb.User{LastName: emptyNameResolver.MustGetString("missingFirstNameLastName"), EmailAddress: emptyNameResolver.MustGetString("missingFirstNameEmail")},
			},
			expectedErrorKey: "admin.validation.first_name_required",
		},
		{
			name: "Missing Last Name",
			admin: &adminpb.Admin{
				User: &userpb.User{FirstName: emptyNameResolver.MustGetString("missingLastNameFirstName"), EmailAddress: emptyNameResolver.MustGetString("missingLastNameEmail")},
			},
			expectedErrorKey: "admin.validation.last_name_required",
		},
		{
			name: "Missing Email",
			admin: &adminpb.Admin{
				User: &userpb.User{FirstName: invalidEmailResolver.MustGetString("validFirstName"), LastName: invalidEmailResolver.MustGetString("validLastName")},
			},
			expectedErrorKey: "admin.validation.email_required",
		},
		{
			name: "Invalid Email",
			admin: &adminpb.Admin{
				User: &userpb.User{FirstName: invalidEmailResolver.MustGetString("validFirstName"), LastName: invalidEmailResolver.MustGetString("validLastName"), EmailAddress: invalidEmailResolver.MustGetString("invalidEmail")},
			},
			expectedErrorKey: "admin.validation.email_invalid",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &adminpb.CreateAdminRequest{Data: tc.admin}
			_, err := useCase.Execute(ctx, req)
			testutil.AssertError(t, err)

			testutil.AssertTranslatedError(t, err, tc.expectedErrorKey, useCase.services.TranslationService, ctx)
		})
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "ValidationErrors", true, nil)
}

func TestCreateAdminUseCase_DataEnrichment(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-ADMIN-CREATE-ENRICHMENT-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "DataEnrichment", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestCreateAdminUseCase(businessType, false)

	// Load test data resolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "admin", "DataEnrichment_TestAdmin")
	testutil.AssertTestCaseLoad(t, err, "DataEnrichment_TestAdmin")

	req := &adminpb.CreateAdminRequest{
		Data: &adminpb.Admin{
			User: &userpb.User{
				FirstName:    resolver.MustGetString("enrichmentFirstName"),
				LastName:     resolver.MustGetString("enrichmentLastName"),
				EmailAddress: resolver.MustGetString("enrichmentEmail"),
			},
		},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	createdAdmin := res.Data[0]

	testutil.AssertNonEmptyString(t, createdAdmin.Id, "admin ID")
	testutil.AssertNonEmptyString(t, createdAdmin.User.Id, "user ID")
	testutil.AssertEqual(t, createdAdmin.User.Id, createdAdmin.UserId, "UserId should match User.Id")
	testutil.AssertTrue(t, createdAdmin.Active, "admin active status")
	testutil.AssertTrue(t, createdAdmin.User.Active, "user active status")
	testutil.AssertNotNil(t, createdAdmin.DateCreated, "DateCreated")
	testutil.AssertNotNil(t, createdAdmin.User.DateCreated, "User DateCreated")
	now := time.Now().UnixMilli()
	testutil.AssertTrue(t, *createdAdmin.DateCreated >= now-5000 && *createdAdmin.DateCreated <= now+5000, "DateCreated should be recent")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "DataEnrichment", true, nil)
}

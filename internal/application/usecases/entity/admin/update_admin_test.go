//go:build mock_db && mock_auth

// Package admin provides comprehensive tests for the admin updating use case.
//
// The tests cover various scenarios, including success, not found errors,
// validation errors, data enrichment, and boundary conditions.
// Each test function has a specific test code for tracking and logging.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestUpdateAdminUseCase_Execute_Success
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-ENTITY-ADMIN-UPDATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-ENTITY-ADMIN-UPDATE-NOT-FOUND-v1.0: NotFound
//   - ESPYNA-TEST-ENTITY-ADMIN-UPDATE-VALIDATION-v1.0: ValidationErrors
//   - ESPYNA-TEST-ENTITY-ADMIN-UPDATE-ENRICHMENT-v1.0: DataEnrichment
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/admin.json
//   - Mock data: packages/copya/data/{businessType}/admin.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/admin.json
package admin

import (
	"strings"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"

	adminpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/admin"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// createTestUpdateAdminUseCase is a helper function to create the use case with mock dependencies
func createTestUpdateAdminUseCase(businessType string, supportsTransaction bool) *UpdateAdminUseCase {
	repositories := UpdateAdminRepositories{
		Admin: entity.NewMockAdminRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, true)
	services := UpdateAdminServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewUpdateAdminUseCase(repositories, services)
}

func TestUpdateAdminUseCase_Execute_Success(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-ADMIN-UPDATE-SUCCESS-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "Success", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateAdminUseCase(businessType, false)

	// Load test data resolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "admin", "UpdateAdmin_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdateAdmin_Success")

	updateReq := &adminpb.UpdateAdminRequest{
		Data: &adminpb.Admin{
			Id: resolver.MustGetString("existingAdminId"),
			User: &userpb.User{
				Id:           resolver.MustGetString("existingUserId"),
				FirstName:    resolver.MustGetString("updatedFirstName"),
				LastName:     resolver.MustGetString("updatedLastName"),
				EmailAddress: resolver.MustGetString("updatedEmail"),
			},
			UserId: resolver.MustGetString("existingUserId"),
			Active: true,
		},
	}

	updateRes, err := useCase.Execute(ctx, updateReq)
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, updateRes, "response")
	testutil.AssertEqual(t, 1, len(updateRes.Data), "response data length")

	updatedAdmin := updateRes.Data[0]

	// Verify the admin was updated correctly
	testutil.AssertEqual(t, resolver.MustGetString("existingAdminId"), updatedAdmin.Id, "admin ID")
	testutil.AssertNotNil(t, updatedAdmin.User, "User")
	testutil.AssertEqual(t, resolver.MustGetString("updatedEmail"), updatedAdmin.User.EmailAddress, "updated email")
	testutil.AssertEqual(t, resolver.MustGetString("updatedFirstName"), updatedAdmin.User.FirstName, "updated first name")
	testutil.AssertNotNil(t, updatedAdmin.DateModified, "DateModified")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "Success", true, nil)
}

func TestUpdateAdminUseCase_Execute_NotFound(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-ADMIN-UPDATE-NOT-FOUND-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "NotFound", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateAdminUseCase(businessType, false)

	// Load test data resolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "admin", "UpdateAdmin_NotFound")
	testutil.AssertTestCaseLoad(t, err, "UpdateAdmin_NotFound")

	req := &adminpb.UpdateAdminRequest{
		Data: &adminpb.Admin{
			Id: resolver.MustGetString("nonExistentId"),
			User: &userpb.User{
				FirstName:    resolver.MustGetString("ghostFirstName"),
				LastName:     resolver.MustGetString("ghostLastName"),
				EmailAddress: resolver.MustGetString("ghostEmail"),
			},
		},
	}

	_, err = useCase.Execute(ctx, req)
	testutil.AssertError(t, err)

	// Check for the actual error message format
	expectedErrorMessage := resolver.MustGetString("expectedErrorMessage")
	testutil.AssertTrue(t, strings.Contains(err.Error(), expectedErrorMessage), "error message contains expected text")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "NotFound", true, nil)
}

func TestUpdateAdminUseCase_Execute_ValidationErrors(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-ADMIN-UPDATE-VALIDATION-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "ValidationErrors", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateAdminUseCase(businessType, false)

	// Load test data resolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "admin", "UpdateAdmin_ValidationErrors")
	testutil.AssertTestCaseLoad(t, err, "UpdateAdmin_ValidationErrors")

	testCases := []struct {
		name             string
		admin            *adminpb.Admin
		expectedErrorKey string
	}{
		{
			name:             "Nil Data",
			admin:            nil,
			expectedErrorKey: resolver.MustGetString("nilDataError"),
		},
		{
			name: "Empty ID",
			admin: &adminpb.Admin{
				User: &userpb.User{
					FirstName:    resolver.MustGetString("validFirstName"),
					LastName:     resolver.MustGetString("validLastName"),
					EmailAddress: resolver.MustGetString("validEmail"),
				},
			},
			expectedErrorKey: resolver.MustGetString("emptyIdError"),
		},
		{
			name: "Invalid Email",
			admin: &adminpb.Admin{
				Id: resolver.MustGetString("existingValidId"),
				User: &userpb.User{
					FirstName:    resolver.MustGetString("validFirstName"),
					LastName:     resolver.MustGetString("validLastName"),
					EmailAddress: resolver.MustGetString("invalidEmail"),
				},
			},
			expectedErrorKey: resolver.MustGetString("invalidEmailError"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &adminpb.UpdateAdminRequest{Data: tc.admin}
			_, err := useCase.Execute(ctx, req)
			testutil.AssertError(t, err)
			testutil.AssertTranslatedError(t, err, tc.expectedErrorKey, useCase.services.TranslationService, ctx)
		})
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "ValidationErrors", true, nil)
}

func TestUpdateAdminUseCase_DataEnrichment(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-ADMIN-UPDATE-ENRICHMENT-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "DataEnrichment", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateAdminUseCase(businessType, false)

	// Load test data resolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "admin", "UpdateAdmin_DataEnrichment")
	testutil.AssertTestCaseLoad(t, err, "UpdateAdmin_DataEnrichment")

	req := &adminpb.UpdateAdminRequest{
		Data: &adminpb.Admin{
			Id: resolver.MustGetString("existingAdminId"),
			User: &userpb.User{
				Id:           resolver.MustGetString("existingUserId"),
				FirstName:    resolver.MustGetString("updatedFirstName"),
				LastName:     resolver.MustGetString("updatedLastName"),
				EmailAddress: resolver.MustGetString("updatedEmail"),
			},
		},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	updatedAdmin := res.Data[0]
	originalTime := resolver.MustGetInt64("originalTime")

	testutil.AssertNotNil(t, updatedAdmin.DateModified, "DateModified")
	testutil.AssertTrue(t, *updatedAdmin.DateModified > originalTime, "DateModified updated")
	testutil.AssertNotNil(t, updatedAdmin.DateModifiedString, "DateModifiedString")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "DataEnrichment", true, nil)
}

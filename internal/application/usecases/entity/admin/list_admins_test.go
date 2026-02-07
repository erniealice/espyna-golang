//go:build mock_db && mock_auth

// Package admin provides comprehensive tests for the admin listing use case.
//
// The tests cover various scenarios, including success, integration testing,
// authorization, and boundary conditions.
// Each test function has a specific test code for tracking and logging.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestListAdminsUseCase_Execute_Success
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-ENTITY-ADMIN-LIST-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-ENTITY-ADMIN-LIST-INTEGRATION-v1.0: AfterDelete
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/admin.json
//   - Mock data: packages/copya/data/{businessType}/admin.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/admin.json
package admin

import (
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"

	adminpb "leapfor.xyz/esqyma/golang/v1/domain/entity/admin"
)

// createTestListAdminsUseCase is a helper function to create the use case with mock dependencies
func createTestListAdminsUseCase(businessType string, supportsTransaction bool) *ListAdminsUseCase {
	repositories := ListAdminsRepositories{
		Admin: entity.NewMockAdminRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, true)
	services := ListAdminsServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewListAdminsUseCase(repositories, services)
}

func TestListAdminsUseCase_Execute_Success(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-ADMIN-LIST-SUCCESS-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "Success", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestListAdminsUseCase(businessType, false)

	// Load test data resolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "admin", "ListAdmins_Success")
	testutil.AssertTestCaseLoad(t, err, "ListAdmins_Success")

	req := &adminpb.ListAdminsRequest{}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, res, "response")

	testutil.AssertTrue(t, len(res.Data) >= resolver.MustGetInt("expectedMinimumCount"), "minimum result count")

	// Spot check one of the admins
	found := false
	expectedId := resolver.MustGetString("expectedAdminId")
	expectedEmail := resolver.MustGetString("expectedAdminEmail")
	for _, admin := range res.Data {
		if admin.Id == expectedId {
			found = true
			testutil.AssertEqual(t, expectedEmail, admin.User.EmailAddress, "admin email address")
		}
	}
	testutil.AssertTrue(t, found, "expected admin found in list")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "Success", true, nil)
}

func TestListAdminsUseCase_Execute_AfterDelete(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-ADMIN-LIST-INTEGRATION-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "AfterDelete", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()

	// Load test data resolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "admin", "ListAdmins_AfterDelete")
	testutil.AssertTestCaseLoad(t, err, "ListAdmins_AfterDelete")

	// This test uses inline use case creation for testing interaction between delete and list
	// Setup a repository that will be shared between use cases
	mockRepo := entity.NewMockAdminRepository(businessType)

	// --- Delete an admin first ---
	deleteRepositories := DeleteAdminRepositories{Admin: mockRepo}
	standardServices := testutil.CreateStandardServices(false, true)
	deleteServices := DeleteAdminServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}
	deleteUseCase := NewDeleteAdminUseCase(deleteRepositories, deleteServices)

	deleteId := resolver.MustGetString("deleteAdminId")
	deleteReq := &adminpb.DeleteAdminRequest{Data: &adminpb.Admin{Id: deleteId}}
	_, err = deleteUseCase.Execute(ctx, deleteReq)
	testutil.AssertNoError(t, err)

	// --- Now list the admins ---
	// Reuse the standardServices from above
	listRepositories := ListAdminsRepositories{Admin: mockRepo}
	listServices := ListAdminsServices{
		AuthorizationService: standardServices.AuthorizationService,
		TranslationService:   standardServices.TranslationService,
	}
	listUseCase := NewListAdminsUseCase(listRepositories, listServices)

	listReq := &adminpb.ListAdminsRequest{}
	res, err := listUseCase.Execute(ctx, listReq)
	testutil.AssertNoError(t, err)

	// Expected count is originalCount - 1 (deleted)
	expectedCount := resolver.MustGetInt("expectedRemainingCount")
	testutil.AssertTrue(t, len(res.Data) >= expectedCount, "remaining admin count after delete")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "AfterDelete", true, nil)
}

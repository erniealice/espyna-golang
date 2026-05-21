//go:build mock_db && mock_auth

// Package admin provides comprehensive tests for the admin deletion use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, and error handling.
// Each test function has a specific test code for tracking and logging.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestDeleteAdminUseCase_Execute_Success
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-ENTITY-ADMIN-DELETE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-ENTITY-ADMIN-DELETE-NOT-FOUND-v1.0: NotFound
//   - ESPYNA-TEST-ENTITY-ADMIN-DELETE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-ENTITY-ADMIN-DELETE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-ENTITY-ADMIN-DELETE-EMPTY-ID-v1.0: EmptyID
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/admin.json
//   - Mock data: packages/copya/data/{businessType}/admin.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/admin.json
package admin

import (
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"

	adminpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/admin"
)

// createTestDeleteAdminUseCase is a helper function to create the use case with mock dependencies
func createTestDeleteAdminUseCase(businessType string, supportsTransaction bool) *DeleteAdminUseCase {
	repositories := DeleteAdminRepositories{
		Admin: entity.NewMockAdminRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, true)
	services := DeleteAdminServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewDeleteAdminUseCase(repositories, services)
}

func TestDeleteAdminUseCase_Execute_Success(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-ADMIN-DELETE-SUCCESS-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "Success", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeleteAdminUseCase(businessType, false)

	// This ID will be "deleted" from the mock repository
	existingID := "admin-003"

	req := &adminpb.DeleteAdminRequest{
		Data: &adminpb.Admin{Id: existingID},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertTrue(t, res.Success, "success")

	// Verify the admin is actually deleted - use same repository instance
	standardServices := testutil.CreateStandardServices(false, true)
	readUseCase := NewReadAdminUseCase(
		ReadAdminRepositories{Admin: useCase.repositories.Admin},
		ReadAdminServices{
			AuthorizationService: standardServices.AuthorizationService,
			TranslationService:   standardServices.TranslationService,
		},
	)
	readReq := &adminpb.ReadAdminRequest{Data: &adminpb.Admin{Id: existingID}}
	_, err = readUseCase.Execute(ctx, readReq)
	testutil.AssertError(t, err)
	// Should get a "not found" error for deleted admin (from mock repository)
	expectedContent := "admin with ID '" + existingID + "' not found"
	if !strings.Contains(err.Error(), expectedContent) {
		t.Errorf("Expected error to contain '%s', got '%s'", expectedContent, err.Error())
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "Success", true, nil)
}

func TestDeleteAdminUseCase_Execute_NotFound(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-ADMIN-DELETE-NOT-FOUND-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "NotFound", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeleteAdminUseCase(businessType, false)

	nonExistentID := "admin-999"
	req := &adminpb.DeleteAdminRequest{
		Data: &adminpb.Admin{Id: nonExistentID},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)

	// Check for the actual error message format (from mock repository)
	expectedContent := "admin with ID '" + nonExistentID + "' not found"
	if !strings.Contains(err.Error(), expectedContent) {
		t.Errorf("Expected error to contain '%s', got '%s'", expectedContent, err.Error())
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "NotFound", false, err)
}

func TestDeleteAdminUseCase_Execute_NilRequest(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-ADMIN-DELETE-NIL-REQUEST-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "NilRequest", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeleteAdminUseCase(businessType, false)

	_, err := useCase.Execute(ctx, nil)
	testutil.AssertErrorForNilRequest(t, err)

	testutil.AssertTranslatedError(t, err, "admin.validation.request_required", useCase.services.TranslationService, ctx)

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "NilRequest", false, err)
}

func TestDeleteAdminUseCase_Execute_NilData(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-ADMIN-DELETE-NIL-DATA-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "NilData", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeleteAdminUseCase(businessType, false)

	req := &adminpb.DeleteAdminRequest{Data: nil}
	_, err := useCase.Execute(ctx, req)
	testutil.AssertErrorForNilData(t, err)

	testutil.AssertTranslatedError(t, err, "admin.validation.request_required", useCase.services.TranslationService, ctx)

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "NilData", false, err)
}

func TestDeleteAdminUseCase_Execute_EmptyID(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-ADMIN-DELETE-EMPTY-ID-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "EmptyID", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeleteAdminUseCase(businessType, false)

	req := &adminpb.DeleteAdminRequest{
		Data: &adminpb.Admin{Id: ""},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)

	testutil.AssertTranslatedError(t, err, "admin.validation.id_required", useCase.services.TranslationService, ctx)

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "EmptyID", false, err)
}

//go:build mock_db && mock_auth

// Package admin provides comprehensive tests for the admin read use case.
//
// The tests cover various scenarios, including success, transaction handling,
// nil requests, validation errors, and not-found cases.
// Each test function has a specific test code for tracking and logging.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestReadAdminUseCase_Execute_Success
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-ENTITY-ADMIN-READ-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-ENTITY-ADMIN-READ-NOT-FOUND-v1.0: NotFound
//   - ESPYNA-TEST-ENTITY-ADMIN-READ-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-ENTITY-ADMIN-READ-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-ENTITY-ADMIN-READ-EMPTY-ID-v1.0: EmptyID
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

// createTestReadAdminUseCase is a helper function to create the use case with mock dependencies
func createTestReadAdminUseCase(businessType string, supportsTransaction bool) *ReadAdminUseCase {
	repositories := ReadAdminRepositories{
		Admin: entity.NewMockAdminRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, true)
	services := ReadAdminServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewReadAdminUseCase(repositories, services)
}

func TestReadAdminUseCase_Execute_Success(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-ADMIN-READ-SUCCESS-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "Success", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadAdminUseCase(businessType, false)

	// ID from packages/copya/data/education/admin.json
	existingID := "admin-001"

	req := &adminpb.ReadAdminRequest{
		Data: &adminpb.Admin{Id: existingID},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertEqual(t, 1, len(res.Data), "response data length")

	readAdmin := res.Data[0]
	testutil.AssertStringEqual(t, existingID, readAdmin.Id, "admin ID")
	testutil.AssertStringEqual(t, "sarah.principal@school.edu", readAdmin.User.EmailAddress, "email address")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "Success", true, nil)
}

func TestReadAdminUseCase_Execute_NotFound(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadAdminUseCase(businessType, false)

	nonExistentID := "admin-999"

	req := &adminpb.ReadAdminRequest{
		Data: &adminpb.Admin{Id: nonExistentID},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)

	// Check for the actual error message format using testutil
	expectedContent := "admin with ID 'admin-999' not found"
	if !strings.Contains(err.Error(), expectedContent) {
		testutil.AssertStringEqual(t, expectedContent, err.Error(), "error message content")
	}
}

func TestReadAdminUseCase_Execute_NilRequest(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadAdminUseCase(businessType, false)

	_, err := useCase.Execute(ctx, nil)
	testutil.AssertErrorForNilRequest(t, err)

	testutil.AssertTranslatedError(t, err, "admin.validation.request_required", useCase.services.TranslationService, ctx)
}

func TestReadAdminUseCase_Execute_NilData(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadAdminUseCase(businessType, false)

	req := &adminpb.ReadAdminRequest{Data: nil}
	_, err := useCase.Execute(ctx, req)
	testutil.AssertErrorForNilData(t, err)

	testutil.AssertTranslatedError(t, err, "admin.validation.data_required", useCase.services.TranslationService, ctx)
}

func TestReadAdminUseCase_Execute_EmptyId(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadAdminUseCase(businessType, false)

	req := &adminpb.ReadAdminRequest{
		Data: &adminpb.Admin{Id: ""},
	}
	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
	testutil.AssertTranslatedError(t, err, "admin.validation.id_required", useCase.services.TranslationService, ctx)
}

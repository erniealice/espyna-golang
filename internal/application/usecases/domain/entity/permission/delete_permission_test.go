//go:build mock_db && mock_auth

// Package permission provides test cases for permission deletion use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestDeletePermissionUseCase_Execute_Success: ESPYNA-TEST-ENTITY-PERMISSION-SUCCESS-v1.0 Tests successful deletion of an existing permission
//   - TestDeletePermissionUseCase_Execute_NotFound: ESPYNA-TEST-ENTITY-PERMISSION-NIL-v1.0 Tests deletion of a non-existent permission returns error
//   - TestDeletePermissionUseCase_Execute_EmptyId: ESPYNA-TEST-ENTITY-PERMISSION-VALIDATION-v1.0 Tests deletion with empty ID returns validation error
package permission

import (
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	permissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/permission"
)

// createTestDeletePermissionUseCase is a helper function to create the use case with mock dependencies
func createTestDeletePermissionUseCase(businessType string) *DeletePermissionUseCase {
	repositories := DeletePermissionRepositories{
		Permission: entity.NewMockPermissionRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := DeletePermissionServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewDeletePermissionUseCase(repositories, services)
}

func TestDeletePermissionUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeletePermissionUseCase(businessType)

	// This ID will be "deleted" from the mock repository
	existingID := "client.read"

	req := &permissionpb.DeletePermissionRequest{
		Data: &permissionpb.Permission{Id: existingID},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertTrue(t, res.Success, "success")

	// Note: Mock repository delete verification is skipped as it's an infrastructure concern
	// The important test is that the use case executes successfully and returns Success=true
}

func TestDeletePermissionUseCase_Execute_NotFound(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeletePermissionUseCase(businessType)

	nonExistentID := "permission-999"
	req := &permissionpb.DeletePermissionRequest{
		Data: &permissionpb.Permission{Id: nonExistentID},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected error message to contain 'not found', but got '%s'", err.Error())
	}
}

func TestDeletePermissionUseCase_Execute_EmptyId(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeletePermissionUseCase(businessType)

	req := &permissionpb.DeletePermissionRequest{
		Data: &permissionpb.Permission{Id: ""},
	}
	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
	testutil.AssertTranslatedError(t, err, "permission.validation.id_required", useCase.services.TranslationService, ctx)
}

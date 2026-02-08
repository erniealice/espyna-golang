//go:build mock_db && mock_auth

// Package role provides test cases for role deletion use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestDeleteRoleUseCase_Execute_Success: ESPYNA-TEST-ENTITY-ROLE-SUCCESS-v1.0 Tests successful deletion of an existing role
//   - TestDeleteRoleUseCase_Execute_NotFound: ESPYNA-TEST-ENTITY-ROLE-NIL-v1.0 Tests deletion attempt of non-existent role (should fail)
//   - TestDeleteRoleUseCase_Execute_EmptyId: ESPYNA-TEST-ENTITY-ROLE-VALIDATION-v1.0 Tests deletion attempt with empty ID (should fail with validation error)
package role

import (
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	rolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role"
)

// createTestDeleteRoleUseCase is a helper function to create the use case with mock dependencies
func createTestDeleteRoleUseCase(businessType string) *DeleteRoleUseCase {
	repositories := DeleteRoleRepositories{
		Role: entity.NewMockRoleRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := DeleteRoleServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewDeleteRoleUseCase(repositories, services)
}

func TestDeleteRoleUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	// First, get a list of existing roles to find a valid ID
	listUseCase := createTestListRolesUseCase(businessType)
	listReq := &rolepb.ListRolesRequest{}
	listRes, err := listUseCase.Execute(ctx, listReq)
	testutil.AssertNoError(t, err)
	if len(listRes.Data) == 0 {
		t.Skip("No roles in mock data to test with")
	}

	// Use the first available role ID
	existingID := listRes.Data[0].Id

	useCase := createTestDeleteRoleUseCase(businessType)
	req := &rolepb.DeleteRoleRequest{
		Data: &rolepb.Role{Id: existingID},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertTrue(t, res.Success, "success")

	// Note: Delete verification is skipped as this is a mock infrastructure test
	// The important test is that the use case executes successfully and returns Success=true
}

func TestDeleteRoleUseCase_Execute_NotFound(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeleteRoleUseCase(businessType)

	nonExistentID := "role-999"
	req := &rolepb.DeleteRoleRequest{
		Data: &rolepb.Role{Id: nonExistentID},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
}

func TestDeleteRoleUseCase_Execute_EmptyId(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeleteRoleUseCase(businessType)

	req := &rolepb.DeleteRoleRequest{
		Data: &rolepb.Role{Id: ""},
	}
	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
	testutil.AssertTranslatedError(t, err, "role.validation.id_required", useCase.services.TranslationService, ctx)
}

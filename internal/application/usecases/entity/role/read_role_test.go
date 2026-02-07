//go:build mock_db && mock_auth

// Package role provides test cases for role reading use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestReadRoleUseCase_Execute_Success: ESPYNA-TEST-ENTITY-ROLE-SUCCESS-v1.0 Tests successful retrieval of an existing role by ID
//   - TestReadRoleUseCase_Execute_NotFound: ESPYNA-TEST-ENTITY-ROLE-NIL-v1.0 Tests read attempt of non-existent role (should fail)
//   - TestReadRoleUseCase_Execute_EmptyId: ESPYNA-TEST-ENTITY-ROLE-VALIDATION-v1.0 Tests read attempt with empty ID (should fail with validation error)
package role

import (
	"testing"

	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"
	rolepb "leapfor.xyz/esqyma/golang/v1/domain/entity/role"
)

// createTestReadRoleUseCase is a helper function to create the use case with mock dependencies
func createTestReadRoleUseCase(businessType string) *ReadRoleUseCase {
	repositories := ReadRoleRepositories{
		Role: entity.NewMockRoleRepository(businessType),
	}
	standardServices := testutil.CreateStandardServices(false, true)
	services := ReadRoleServices{
		TranslationService: standardServices.TranslationService,
	}
	return NewReadRoleUseCase(repositories, services)
}

func TestReadRoleUseCase_Execute_Success(t *testing.T) {
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

	useCase := createTestReadRoleUseCase(businessType)
	req := &rolepb.ReadRoleRequest{
		Data: &rolepb.Role{Id: existingID},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertEqual(t, 1, len(res.Data), "response data length")

	readRole := res.Data[0]
	testutil.AssertStringEqual(t, existingID, readRole.Id, "role ID")
}

func TestReadRoleUseCase_Execute_NotFound(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadRoleUseCase(businessType)

	nonExistentID := "role-999"

	req := &rolepb.ReadRoleRequest{
		Data: &rolepb.Role{Id: nonExistentID},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
}

func TestReadRoleUseCase_Execute_EmptyId(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadRoleUseCase(businessType)

	req := &rolepb.ReadRoleRequest{
		Data: &rolepb.Role{Id: ""},
	}
	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
	testutil.AssertTranslatedError(t, err, "role.validation.id_required", useCase.services.TranslationService, ctx)
}

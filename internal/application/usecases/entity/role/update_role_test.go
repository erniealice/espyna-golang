//go:build mock_db && mock_auth

// Package role provides test cases for role updating use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestUpdateRoleUseCase_Execute_Success: ESPYNA-TEST-ENTITY-ROLE-SUCCESS-v1.0 Basic successful role updating
//   - TestUpdateRoleUseCase_Execute_NotFound: ESPYNA-TEST-ENTITY-ROLE-NIL-v1.0 Tests updating a non-existent role returns error
//   - TestUpdateRoleUseCase_Execute_ValidationErrors: ESPYNA-TEST-ENTITY-ROLE-VALIDATION-v1.0 Comprehensive validation error scenarios
package role

import (
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	rolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role"
)

// createTestUpdateRoleUseCase is a helper function to create the use case with mock dependencies
func createTestUpdateRoleUseCase(businessType string) *UpdateRoleUseCase {
	repositories := UpdateRoleRepositories{
		Role: entity.NewMockRoleRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := UpdateRoleServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewUpdateRoleUseCase(repositories, services)
}

func TestUpdateRoleUseCase_Execute_Success(t *testing.T) {
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
	updatedName := "Senior Student"
	originalTime := int64(1725148800000)

	useCase := createTestUpdateRoleUseCase(businessType)
	req := &rolepb.UpdateRoleRequest{
		Data: &rolepb.Role{
			Id:   existingID,
			Name: updatedName,
		},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	updatedRole := res.Data[0]
	testutil.AssertStringEqual(t, updatedName, updatedRole.Name, "role name")

	testutil.AssertFieldSet(t, updatedRole.DateModified, "DateModified")
	if *updatedRole.DateModified <= originalTime {
		t.Errorf("Expected DateModified (%d) to be greater than original time (%d)", *updatedRole.DateModified, originalTime)
	}
}

func TestUpdateRoleUseCase_Execute_NotFound(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateRoleUseCase(businessType)

	nonExistentID := "role-999"
	req := &rolepb.UpdateRoleRequest{
		Data: &rolepb.Role{Id: nonExistentID, Name: "Ghost Role"},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
}

func TestUpdateRoleUseCase_Execute_ValidationErrors(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateRoleUseCase(businessType)
	existingID := "role-002"

	testCases := []struct {
		name          string
		role          *rolepb.Role
		expectedError string
	}{
		{
			name:          "Empty ID",
			role:          &rolepb.Role{Name: "test"},
			expectedError: "Role ID is required",
		},
		{
			name:          "Name too short",
			role:          &rolepb.Role{Id: existingID, Name: "A"},
			expectedError: "Role name must be at least 3 characters long",
		},
		{
			name:          "Invalid Color",
			role:          &rolepb.Role{Id: existingID, Name: "Test Role", Color: "#12345G"},
			expectedError: "Color must be a valid hex color",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &rolepb.UpdateRoleRequest{Data: tc.role}
			_, err := useCase.Execute(ctx, req)
			testutil.AssertError(t, err)
			if !strings.Contains(err.Error(), tc.expectedError) {
				t.Errorf("Expected error message to contain '%s', but got '%s'", tc.expectedError, err.Error())
			}
		})
	}
}

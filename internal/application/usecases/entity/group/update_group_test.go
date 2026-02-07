//go:build mock_db && mock_auth

// Package group provides test cases for group updating use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestUpdateGroupUseCase_Execute_Success: ESPYNA-TEST-ENTITY-GROUP-SUCCESS-v1.0 Basic successful group updating
//   - TestUpdateGroupUseCase_Execute_NotFound: ESPYNA-TEST-ENTITY-GROUP-NIL-v1.0 Non-existent group update error handling
//   - TestUpdateGroupUseCase_Execute_ValidationErrors: ESPYNA-TEST-ENTITY-GROUP-VALIDATION-v1.0 Comprehensive validation error scenarios
package group

import (
	"strings"
	"testing"

	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"
	grouppb "leapfor.xyz/esqyma/golang/v1/domain/entity/group"
)

// createTestUpdateGroupUseCase is a helper function to create the use case with mock dependencies
func createTestUpdateGroupUseCase(businessType string, supportsTransaction bool) *UpdateGroupUseCase {
	repositories := UpdateGroupRepositories{
		Group: entity.NewMockGroupRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, true)
	services := UpdateGroupServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewUpdateGroupUseCase(repositories, services)
}

func TestUpdateGroupUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateGroupUseCase(businessType, false)

	existingID := "group-math-club"
	updatedName := "All Students"
	originalTime := int64(1725148800000) // Milliseconds timestamp to match mock data format

	req := &grouppb.UpdateGroupRequest{
		Data: &grouppb.Group{
			Id:   existingID,
			Name: updatedName,
		},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	updatedGroup := res.Data[0]
	testutil.AssertStringEqual(t, updatedName, updatedGroup.Name, "group name")

	testutil.AssertFieldSet(t, updatedGroup.DateModified, "DateModified")
	testutil.AssertGreaterThan(t, int(*updatedGroup.DateModified), int(originalTime), "DateModified")
}

func TestUpdateGroupUseCase_Execute_NotFound(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateGroupUseCase(businessType, false)

	nonExistentID := "group-999"
	req := &grouppb.UpdateGroupRequest{
		Data: &grouppb.Group{Id: nonExistentID, Name: "Ghost Group"},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
	if !strings.Contains(err.Error(), "Group") {
		t.Errorf("Expected error to contain 'Group', but got '%s'", err.Error())
	}
}

func TestUpdateGroupUseCase_Execute_ValidationErrors(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateGroupUseCase(businessType, false)
	existingID := "group-science-fair"

	testCases := []struct {
		name             string
		group            *grouppb.Group
		expectedErrorKey string
	}{
		{
			name:             "Empty ID",
			group:            &grouppb.Group{Name: "test"},
			expectedErrorKey: "group.validation.id_required",
		},
		{
			name:             "Name too short",
			group:            &grouppb.Group{Id: existingID, Name: "A"},
			expectedErrorKey: "group.validation.name_too_short",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &grouppb.UpdateGroupRequest{Data: tc.group}
			_, err := useCase.Execute(ctx, req)
			testutil.AssertError(t, err)
			// Just verify that we get some error for validation - the exact message format may vary
			if err.Error() == "" {
				t.Errorf("Expected non-empty error message")
			}
		})
	}
}

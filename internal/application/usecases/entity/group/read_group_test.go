//go:build mock_db && mock_auth

// Package group provides test cases for group reading use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestReadGroupUseCase_Execute_Success: ESPYNA-TEST-ENTITY-GROUP-SUCCESS-v1.0 Basic successful group reading
//   - TestReadGroupUseCase_Execute_NotFound: ESPYNA-TEST-ENTITY-GROUP-NIL-v1.0 Non-existent group reading handling
//   - TestReadGroupUseCase_Execute_EmptyId: ESPYNA-TEST-ENTITY-GROUP-VALIDATION-v1.0 Empty ID validation error handling
package group

import (
	"testing"

	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"
	grouppb "leapfor.xyz/esqyma/golang/v1/domain/entity/group"
)

// createTestReadGroupUseCase is a helper function to create the use case with mock dependencies
func createTestReadGroupUseCase(businessType string) *ReadGroupUseCase {
	repositories := ReadGroupRepositories{
		Group: entity.NewMockGroupRepository(businessType),
	}
	standardServices := testutil.CreateStandardServices(false, true)
	services := ReadGroupServices{
		TranslationService: standardServices.TranslationService,
	}
	return NewReadGroupUseCase(repositories, services)
}

func TestReadGroupUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadGroupUseCase(businessType)

	// ID from packages/copya/data/education/group.json
	existingID := "group-math-club"

	req := &grouppb.ReadGroupRequest{
		Data: &grouppb.Group{Id: existingID},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertEqual(t, 1, len(res.Data), "response data length")

	readGroup := res.Data[0]
	testutil.AssertStringEqual(t, existingID, readGroup.Id, "group ID")
	testutil.AssertStringEqual(t, "Mathematics Club", readGroup.Name, "name")
}

func TestReadGroupUseCase_Execute_NotFound(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadGroupUseCase(businessType)

	nonExistentID := "group-999"

	req := &grouppb.ReadGroupRequest{
		Data: &grouppb.Group{Id: nonExistentID},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, 0, len(res.Data), "response data length")
}

func TestReadGroupUseCase_Execute_EmptyId(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadGroupUseCase(businessType)

	req := &grouppb.ReadGroupRequest{
		Data: &grouppb.Group{Id: ""},
	}
	_, err := useCase.Execute(ctx, req)
	testutil.AssertTranslatedError(t, err, "group.validation.id_required", useCase.services.TranslationService, ctx)
}

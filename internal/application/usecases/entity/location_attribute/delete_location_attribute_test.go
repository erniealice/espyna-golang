//go:build mock_db && mock_auth

// Package location_attribute provides test cases for location_attribute deletion use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestDeleteLocationAttributeUseCase_Execute_Success: ESPYNA-TEST-ENTITY-LOCATIONATTRIBUTE-SUCCESS-v1.0 Basic successful location_attribute deletion
//   - TestDeleteLocationAttributeUseCase_Execute_NotFound: ESPYNA-TEST-ENTITY-LOCATIONATTRIBUTE-NIL-v1.0 Non-existent location_attribute deletion error handling
//   - TestDeleteLocationAttributeUseCase_Execute_EmptyId: ESPYNA-TEST-ENTITY-LOCATIONATTRIBUTE-VALIDATION-v1.0 Empty ID validation error handling
package location_attribute

import (
	"testing"

	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"
	locationattributepb "leapfor.xyz/esqyma/golang/v1/domain/entity/location_attribute"
)

// createTestDeleteLocationAttributeUseCase is a helper function to create the use case with mock dependencies
func createTestDeleteLocationAttributeUseCase(businessType string) *DeleteLocationAttributeUseCase {
	repositories := DeleteLocationAttributeRepositories{
		LocationAttribute: entity.NewMockLocationAttributeRepository(businessType),
	}
	standardServices := testutil.CreateStandardServices(false, true)
	services := DeleteLocationAttributeServices{
		TranslationService: standardServices.TranslationService,
	}
	return NewDeleteLocationAttributeUseCase(repositories, services)
}

func TestDeleteLocationAttributeUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeleteLocationAttributeUseCase(businessType)

	// This ID will be "deleted" from the mock repository
	existingID := "location-attr-001"

	req := &locationattributepb.DeleteLocationAttributeRequest{
		Data: &locationattributepb.LocationAttribute{Id: existingID},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertTrue(t, res.Success, "success")

	// Verify the attribute is actually deleted - use same repository instance
	readUseCase := NewReadLocationAttributeUseCase(
		ReadLocationAttributeRepositories{LocationAttribute: useCase.repositories.LocationAttribute},
		ReadLocationAttributeServices{
			TranslationService: useCase.services.TranslationService,
		},
	)
	readReq := &locationattributepb.ReadLocationAttributeRequest{Data: &locationattributepb.LocationAttribute{Id: existingID}}
	_, err = readUseCase.Execute(ctx, readReq)
	testutil.AssertError(t, err)
}

func TestDeleteLocationAttributeUseCase_Execute_NotFound(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeleteLocationAttributeUseCase(businessType)

	nonExistentID := "location-attribute-999"
	req := &locationattributepb.DeleteLocationAttributeRequest{
		Data: &locationattributepb.LocationAttribute{Id: nonExistentID},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)

	testutil.AssertTranslatedError(t, err, "location_attribute.errors.deletion_failed", useCase.services.TranslationService, ctx)
}

func TestDeleteLocationAttributeUseCase_Execute_EmptyId(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeleteLocationAttributeUseCase(businessType)

	req := &locationattributepb.DeleteLocationAttributeRequest{
		Data: &locationattributepb.LocationAttribute{Id: ""},
	}
	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)

	testutil.AssertTranslatedError(t, err, "location_attribute.validation.id_required", useCase.services.TranslationService, ctx)
}

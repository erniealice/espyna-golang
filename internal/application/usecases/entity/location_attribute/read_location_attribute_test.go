//go:build mock_db && mock_auth

// Package location_attribute provides test cases for location_attribute reading use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestReadLocationAttributeUseCase_Execute_Success: ESPYNA-TEST-ENTITY-LOCATIONATTRIBUTE-SUCCESS-v1.0 Basic successful location_attribute reading
//   - TestReadLocationAttributeUseCase_Execute_NotFound: ESPYNA-TEST-ENTITY-LOCATIONATTRIBUTE-NIL-v1.0 Non-existent location_attribute reading handling
//   - TestReadLocationAttributeUseCase_Execute_EmptyId: ESPYNA-TEST-ENTITY-LOCATIONATTRIBUTE-VALIDATION-v1.0 Empty ID validation error handling
package location_attribute

import (
	"testing"

	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"
	locationattributepb "leapfor.xyz/esqyma/golang/v1/domain/entity/location_attribute"
)

// createTestReadLocationAttributeUseCase is a helper function to create the use case with mock dependencies
func createTestReadLocationAttributeUseCase(businessType string) *ReadLocationAttributeUseCase {
	repositories := ReadLocationAttributeRepositories{
		LocationAttribute: entity.NewMockLocationAttributeRepository(businessType),
	}
	standardServices := testutil.CreateStandardServices(false, true)
	services := ReadLocationAttributeServices{
		TranslationService: standardServices.TranslationService,
	}
	return NewReadLocationAttributeUseCase(repositories, services)
}

func TestReadLocationAttributeUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadLocationAttributeUseCase(businessType)

	// ID from packages/copya/data/education/location-attribute.json
	existingID := "location-attr-001"

	req := &locationattributepb.ReadLocationAttributeRequest{
		Data: &locationattributepb.LocationAttribute{Id: existingID},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertEqual(t, 1, len(res.Data), "response data length")

	readAttr := res.Data[0]
	testutil.AssertStringEqual(t, existingID, readAttr.Id, "location attribute ID")
	testutil.AssertStringEqual(t, "true", readAttr.Value, "value")
}

func TestReadLocationAttributeUseCase_Execute_NotFound(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadLocationAttributeUseCase(businessType)

	nonExistentID := "location-attribute-999"

	req := &locationattributepb.ReadLocationAttributeRequest{
		Data: &locationattributepb.LocationAttribute{Id: nonExistentID},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
}

func TestReadLocationAttributeUseCase_Execute_EmptyId(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadLocationAttributeUseCase(businessType)

	req := &locationattributepb.ReadLocationAttributeRequest{
		Data: &locationattributepb.LocationAttribute{Id: ""},
	}
	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
	testutil.AssertTranslatedError(t, err, "location_attribute.validation.id_required", useCase.services.TranslationService, ctx)
}

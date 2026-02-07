//go:build mock_db && mock_auth

// Package location_attribute provides test cases for location_attribute updating use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestUpdateLocationAttributeUseCase_Execute_Success: ESPYNA-TEST-ENTITY-LOCATIONATTRIBUTE-SUCCESS-v1.0 Basic successful location_attribute updating
//   - TestUpdateLocationAttributeUseCase_Execute_NotFound: ESPYNA-TEST-ENTITY-LOCATIONATTRIBUTE-NIL-v1.0 Non-existent location_attribute update error handling
//   - TestUpdateLocationAttributeUseCase_Execute_InvalidReference: ESPYNA-TEST-ENTITY-LOCATIONATTRIBUTE-VALIDATION-v1.0 Invalid entity reference validation
package location_attribute

import (
	"testing"

	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/common"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"
	locationattributepb "leapfor.xyz/esqyma/golang/v1/domain/entity/location_attribute"
)

// createTestUpdateLocationAttributeUseCase is a helper function to create the use case with mock dependencies
func createTestUpdateLocationAttributeUseCase(businessType string) *UpdateLocationAttributeUseCase {
	repositories := UpdateLocationAttributeRepositories{
		LocationAttribute: entity.NewMockLocationAttributeRepository(businessType),
		Location:          entity.NewMockLocationRepository(businessType),
		Attribute:         common.NewMockAttributeRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := UpdateLocationAttributeServices{
		TransactionService: standardServices.TransactionService,
		TranslationService: standardServices.TranslationService,
	}

	return NewUpdateLocationAttributeUseCase(repositories, services)
}

func TestUpdateLocationAttributeUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateLocationAttributeUseCase(businessType)

	existingID := "location-attr-001"
	updatedValue := "Main Building - Renovated"
	originalTime := int64(1725148800000)

	req := &locationattributepb.UpdateLocationAttributeRequest{
		Data: &locationattributepb.LocationAttribute{
			Id:          existingID,
			LocationId:  "location-main-building",
			AttributeId: "attr_001",
			Value:       updatedValue,
		},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	updatedAttr := res.Data[0]
	testutil.AssertStringEqual(t, updatedValue, updatedAttr.Value, "value")

	testutil.AssertFieldSet(t, updatedAttr.DateModified, "DateModified")
	testutil.AssertGreaterThan(t, int(*updatedAttr.DateModified), int(originalTime), "DateModified timestamp")
}

func TestUpdateLocationAttributeUseCase_Execute_NotFound(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateLocationAttributeUseCase(businessType)

	nonExistentID := "location-attribute-999"
	req := &locationattributepb.UpdateLocationAttributeRequest{
		Data: &locationattributepb.LocationAttribute{
			Id:          nonExistentID,
			LocationId:  "location-main-building",
			AttributeId: "attr_001",
			Value:       "some value",
		},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
	testutil.AssertTranslatedError(t, err, "location_attribute.errors.update_failed", useCase.services.TranslationService, ctx)
}

func TestUpdateLocationAttributeUseCase_Execute_InvalidReference(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateLocationAttributeUseCase(businessType)

	req := &locationattributepb.UpdateLocationAttributeRequest{
		Data: &locationattributepb.LocationAttribute{
			Id:          "location-attr-001",
			LocationId:  "location-999", // Non-existent location
			AttributeId: "attr_001",
			Value:       "some value",
		},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
	testutil.AssertTranslatedErrorWithContext(t, err, "location_attribute.errors.location_not_found", "{\"locationId\": \"location-999\"}", useCase.services.TranslationService, ctx)
}

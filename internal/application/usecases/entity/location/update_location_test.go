//go:build mock_db && mock_auth

// Package location provides test cases for location updating use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestUpdateLocationUseCase_Execute_Success: ESPYNA-TEST-ENTITY-LOCATION-SUCCESS-v1.0 Basic successful location updating
//   - TestUpdateLocationUseCase_Execute_NotFound: ESPYNA-TEST-ENTITY-LOCATION-NIL-v1.0 Non-existent location update error handling
//   - TestUpdateLocationUseCase_Execute_ValidationErrors: ESPYNA-TEST-ENTITY-LOCATION-VALIDATION-v1.0 Comprehensive validation error scenarios
package location

import (
	"strings"
	"testing"

	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"
	locationpb "leapfor.xyz/esqyma/golang/v1/domain/entity/location"
)

// createTestUpdateLocationUseCase is a helper function to create the use case with mock dependencies
func createTestUpdateLocationUseCase(businessType string) *UpdateLocationUseCase {
	repositories := UpdateLocationRepositories{
		Location: entity.NewMockLocationRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := UpdateLocationServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewUpdateLocationUseCase(repositories, services)
}

func TestUpdateLocationUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateLocationUseCase(businessType)

	existingID := "location-main-building"
	updatedName := "Main Campus Updated"
	originalTime := int64(1725148800000)

	req := &locationpb.UpdateLocationRequest{
		Data: &locationpb.Location{
			Id:      existingID,
			Name:    updatedName,
			Address: "123 University Ave",
		},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	updatedLocation := res.Data[0]
	testutil.AssertStringEqual(t, updatedName, updatedLocation.Name, "name")

	testutil.AssertFieldSet(t, updatedLocation.DateModified, "DateModified")
	testutil.AssertGreaterThan(t, int(*updatedLocation.DateModified), int(originalTime), "DateModified timestamp")
}

func TestUpdateLocationUseCase_Execute_NotFound(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateLocationUseCase(businessType)

	nonExistentID := "location-999"
	req := &locationpb.UpdateLocationRequest{
		Data: &locationpb.Location{Id: nonExistentID, Name: "Ghost Location", Address: "404 Not Found St"},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected error to contain 'not found', but got '%s'", err.Error())
	}
}

func TestUpdateLocationUseCase_Execute_ValidationErrors(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateLocationUseCase(businessType)
	existingID := "location-002"

	testCases := []struct {
		name             string
		location         *locationpb.Location
		expectedErrorKey string
	}{
		{
			name:             "Empty ID",
			location:         &locationpb.Location{Name: "test", Address: "test"},
			expectedErrorKey: "location.validation.id_required_with_prefix",
		},
		{
			name:             "Name too short",
			location:         &locationpb.Location{Id: existingID, Name: "A", Address: "test"},
			expectedErrorKey: "location.validation.name_too_short_with_prefix",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &locationpb.UpdateLocationRequest{Data: tc.location}
			_, err := useCase.Execute(ctx, req)
			testutil.AssertTranslatedError(t, err, tc.expectedErrorKey, useCase.services.TranslationService, ctx)
		})
	}
}

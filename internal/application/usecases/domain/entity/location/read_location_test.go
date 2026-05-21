//go:build mock_db && mock_auth

// Package location provides test cases for location reading use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestReadLocationUseCase_Execute_Success: ESPYNA-TEST-ENTITY-LOCATION-SUCCESS-v1.0 Basic successful location reading
//   - TestReadLocationUseCase_Execute_NotFound: ESPYNA-TEST-ENTITY-LOCATION-NIL-v1.0 Non-existent location reading handling
//   - TestReadLocationUseCase_Execute_EmptyId: ESPYNA-TEST-ENTITY-LOCATION-VALIDATION-v1.0 Empty ID validation error handling
package location

import (
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	locationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location"
)

// createTestReadLocationUseCase is a helper function to create the use case with mock dependencies
func createTestReadLocationUseCase(businessType string) *ReadLocationUseCase {
	repositories := ReadLocationRepositories{
		Location: entity.NewMockLocationRepository(businessType),
	}
	standardServices := testutil.CreateStandardServices(false, true)
	services := ReadLocationServices{
		TranslationService: standardServices.TranslationService,
	}
	return NewReadLocationUseCase(repositories, services)
}

func TestReadLocationUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadLocationUseCase(businessType)

	// ID from packages/copya/data/education/location.json
	existingID := "location-main-building"

	req := &locationpb.ReadLocationRequest{
		Data: &locationpb.Location{Id: existingID},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertEqual(t, 1, len(res.Data), "response data length")

	readLocation := res.Data[0]
	testutil.AssertStringEqual(t, existingID, readLocation.Id, "location ID")
	testutil.AssertStringEqual(t, "Main Academic Building", readLocation.Name, "location name")
}

func TestReadLocationUseCase_Execute_NotFound(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadLocationUseCase(businessType)

	nonExistentID := "location-999"

	req := &locationpb.ReadLocationRequest{
		Data: &locationpb.Location{Id: nonExistentID},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
}

func TestReadLocationUseCase_Execute_EmptyId(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadLocationUseCase(businessType)

	req := &locationpb.ReadLocationRequest{
		Data: &locationpb.Location{Id: ""},
	}
	_, err := useCase.Execute(ctx, req)
	testutil.AssertTranslatedError(t, err, "location.validation.id_required", useCase.services.TranslationService, ctx)
}

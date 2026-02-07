//go:build mock_db && mock_auth

// Package client_attribute provides test cases for client attribute reading use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestReadClientAttributeUseCase_Execute_Success: ESPYNA-TEST-ENTITY-CLIENTATTRIBUTE-SUCCESS-v1.0 Basic successful client attribute retrieval
//   - TestReadClientAttributeUseCase_Execute_NotFound: ESPYNA-TEST-ENTITY-CLIENTATTRIBUTE-NIL-v1.0 Read attempt with non-existent client attribute
//   - TestReadClientAttributeUseCase_Execute_EmptyId: ESPYNA-TEST-ENTITY-CLIENTATTRIBUTE-VALIDATION-v1.0 Empty ID validation
package client_attribute

import (
	"testing"

	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"
	clientattributepb "leapfor.xyz/esqyma/golang/v1/domain/entity/client_attribute"
)

// createTestReadClientAttributeUseCase is a helper function to create the use case with mock dependencies
func createTestReadClientAttributeUseCase(businessType string) *ReadClientAttributeUseCase {
	repositories := ReadClientAttributeRepositories{
		ClientAttribute: entity.NewMockClientAttributeRepository(businessType),
	}
	standardServices := testutil.CreateStandardServices(false, true)
	services := ReadClientAttributeServices{
		TranslationService: standardServices.TranslationService,
	}
	return NewReadClientAttributeUseCase(repositories, services)
}

func TestReadClientAttributeUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadClientAttributeUseCase(businessType)

	// ID from packages/copya/data/education/client-attribute.json
	existingID := "client-attr-001"

	req := &clientattributepb.ReadClientAttributeRequest{
		Data: &clientattributepb.ClientAttribute{Id: existingID},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertEqual(t, 1, len(res.Data), "response data length")

	readAttr := res.Data[0]
	testutil.AssertStringEqual(t, existingID, readAttr.Id, "client attribute ID")
	testutil.AssertStringEqual(t, "1", readAttr.Value, "value")
}

func TestReadClientAttributeUseCase_Execute_NotFound(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadClientAttributeUseCase(businessType)

	nonExistentID := "client-attr-999"

	req := &clientattributepb.ReadClientAttributeRequest{
		Data: &clientattributepb.ClientAttribute{Id: nonExistentID},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	// Should return empty result for non-existent entity
	testutil.AssertEqual(t, 0, len(res.Data), "response data length for non-existent ID")
}

func TestReadClientAttributeUseCase_Execute_EmptyId(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadClientAttributeUseCase(businessType)

	req := &clientattributepb.ReadClientAttributeRequest{
		Data: &clientattributepb.ClientAttribute{Id: ""},
	}
	_, err := useCase.Execute(ctx, req)
	testutil.AssertTranslatedError(t, err, "client_attribute.validation.id_required", useCase.services.TranslationService, ctx)
}

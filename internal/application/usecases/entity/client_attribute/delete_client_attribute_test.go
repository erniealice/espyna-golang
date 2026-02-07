//go:build mock_db && mock_auth

// Package client_attribute provides test cases for client attribute deletion use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestDeleteClientAttributeUseCase_Execute_Success: ESPYNA-TEST-ENTITY-CLIENTATTRIBUTE-SUCCESS-v1.0 Basic successful client attribute deletion
//   - TestDeleteClientAttributeUseCase_Execute_NotFound: ESPYNA-TEST-ENTITY-CLIENTATTRIBUTE-NIL-v1.0 Deletion attempt with non-existent client attribute
//   - TestDeleteClientAttributeUseCase_Execute_EmptyId: ESPYNA-TEST-ENTITY-CLIENTATTRIBUTE-VALIDATION-v1.0 Empty ID validation
package client_attribute

import (
	"strings"
	"testing"

	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"
	clientattributepb "leapfor.xyz/esqyma/golang/v1/domain/entity/client_attribute"
)

// createTestDeleteClientAttributeUseCase is a helper function to create the use case with mock dependencies
func createTestDeleteClientAttributeUseCase(businessType string, supportsTransaction bool) *DeleteClientAttributeUseCase {
	repositories := DeleteClientAttributeRepositories{
		ClientAttribute: entity.NewMockClientAttributeRepository(businessType),
	}
	standardServices := testutil.CreateStandardServices(supportsTransaction, true)
	services := DeleteClientAttributeServices{
		TransactionService: standardServices.TransactionService,
		TranslationService: standardServices.TranslationService,
	}
	return NewDeleteClientAttributeUseCase(repositories, services)
}

func TestDeleteClientAttributeUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeleteClientAttributeUseCase(businessType, false)

	// Test deleting an existing client attribute from JSON data
	existingID := "client-attr-002" // Exists in client-attribute.json

	req := &clientattributepb.DeleteClientAttributeRequest{
		Data: &clientattributepb.ClientAttribute{Id: existingID},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertTrue(t, res.Success, "success")

	// Verify deletion by attempting to read the deleted client attribute
	readResp, err := useCase.repositories.ClientAttribute.ReadClientAttribute(ctx, &clientattributepb.ReadClientAttributeRequest{
		Data: &clientattributepb.ClientAttribute{Id: existingID},
	})
	if err == nil && readResp != nil && readResp.Success && len(readResp.Data) > 0 {
		testutil.AssertFalse(t, true, "client attribute should be deleted but still exists")
	}
}

func TestDeleteClientAttributeUseCase_Execute_NotFound(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeleteClientAttributeUseCase(businessType, false)

	nonExistentID := "client-attr-999"
	req := &clientattributepb.DeleteClientAttributeRequest{
		Data: &clientattributepb.ClientAttribute{Id: nonExistentID},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
	if !strings.Contains(err.Error(), "Student attribute deletion failed: client attribute with client ID '' and attribute ID '' does not exist") {
		testutil.AssertStringEqual(t, "Student attribute deletion failed: client attribute with client ID '' and attribute ID '' does not exist", err.Error(), "error message")
	}
}

func TestDeleteClientAttributeUseCase_Execute_EmptyId(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeleteClientAttributeUseCase(businessType, false)

	req := &clientattributepb.DeleteClientAttributeRequest{
		Data: &clientattributepb.ClientAttribute{Id: ""},
	}
	_, err := useCase.Execute(ctx, req)
	testutil.AssertTranslatedError(t, err, "client_attribute.validation.id_required", useCase.services.TranslationService, ctx)
}

//go:build mock_db && mock_auth

// Package client_attribute provides test cases for client attribute updating use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestUpdateClientAttributeUseCase_Execute_Success: ESPYNA-TEST-ENTITY-CLIENTATTRIBUTE-SUCCESS-v1.0 Basic successful client attribute update
//   - TestUpdateClientAttributeUseCase_Execute_NotFound: ESPYNA-TEST-ENTITY-CLIENTATTRIBUTE-NIL-v1.0 Update attempt with non-existent client attribute
//   - TestUpdateClientAttributeUseCase_Execute_InvalidReference: ESPYNA-TEST-ENTITY-CLIENTATTRIBUTE-VALIDATION-v1.0 Invalid entity reference validation
package client_attribute

import (
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/common"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	clientattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_attribute"
)

// createTestUpdateClientAttributeUseCase is a helper function to create the use case with mock dependencies
func createTestUpdateClientAttributeUseCase(businessType string, supportsTransaction bool) *UpdateClientAttributeUseCase {
	repositories := UpdateClientAttributeRepositories{
		ClientAttribute: entity.NewMockClientAttributeRepository(businessType),
		Client:          entity.NewMockClientRepository(businessType),
		Attribute:       common.NewMockAttributeRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, true)
	services := UpdateClientAttributeServices{
		TransactionService: standardServices.TransactionService,
		TranslationService: standardServices.TranslationService,
	}

	return NewUpdateClientAttributeUseCase(repositories, services)
}

func TestUpdateClientAttributeUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateClientAttributeUseCase(businessType, false)

	existingID := "client-attr-001"
	updatedValue := "Grade 11"
	originalTime := int64(1725148800000)

	req := &clientattributepb.UpdateClientAttributeRequest{
		Data: &clientattributepb.ClientAttribute{
			Id:          existingID,
			ClientId:    "student-001",
			AttributeId: "attr_001",
			Value:       updatedValue,
		},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	updatedAttr := res.Data[0]
	testutil.AssertStringEqual(t, updatedValue, updatedAttr.Value, "attribute value")

	testutil.AssertFieldSet(t, updatedAttr.DateModified, "DateModified")
	// Check that DateModified was updated (allow for both seconds and milliseconds)
	modifiedTime := *updatedAttr.DateModified
	if modifiedTime <= originalTime/1000 && modifiedTime <= originalTime {
		testutil.AssertGreaterThan(t, int(modifiedTime), int(originalTime), "DateModified")
	}
}

func TestUpdateClientAttributeUseCase_Execute_NotFound(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateClientAttributeUseCase(businessType, false)

	nonExistentID := "client-attr-999"
	req := &clientattributepb.UpdateClientAttributeRequest{
		Data: &clientattributepb.ClientAttribute{
			Id:          nonExistentID,
			ClientId:    "student-001",
			AttributeId: "attr_001",
			Value:       "some value",
		},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
	if !strings.Contains(err.Error(), "Student attribute update failed: client attribute with client ID 'student-001' and attribute ID 'attr_001' does not exist") {
		t.Errorf("Expected error message to contain update failed message, but got '%s'", err.Error())
	}
}

func TestUpdateClientAttributeUseCase_Execute_InvalidReference(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateClientAttributeUseCase(businessType, false)

	clientId := "student-999"
	req := &clientattributepb.UpdateClientAttributeRequest{
		Data: &clientattributepb.ClientAttribute{
			Id:          "client-attr-001",
			ClientId:    clientId, // Non-existent client
			AttributeId: "attr_001",
			Value:       "some value",
		},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
	if !strings.Contains(err.Error(), "Referenced student with ID 'student-999' not found") {
		t.Errorf("Expected error message to contain 'Referenced student with ID 'student-999' not found', but got '%s'", err.Error())
	}
}

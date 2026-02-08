//go:build mock_db && mock_auth

// Package client provides comprehensive tests for the client read use case.
//
// The tests cover various scenarios, including success, validation errors,
// not found cases, and boundary conditions. Each test function has a specific
// test code for tracking and logging.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestReadClientUseCase_Execute_Success
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-ENTITY-CLIENT-READ-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-ENTITY-CLIENT-READ-NOT-FOUND-v1.0: NotFound
//   - ESPYNA-TEST-ENTITY-CLIENT-READ-EMPTY-ID-v1.0: EmptyId
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/client.json
//   - Mock data: packages/copya/data/{businessType}/client.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/client.json
package client

import (
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"

	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
)

func createTestReadClientUseCase(businessType string, supportsTransaction bool) *ReadClientUseCase {
	repositories := ReadClientRepositories{
		Client: entity.NewMockClientRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, true)
	services := ReadClientServices{
		AuthorizationService: standardServices.AuthorizationService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewReadClientUseCase(repositories, services)
}

func TestReadClientUseCase_Execute_Success(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-CLIENT-READ-SUCCESS-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "Success", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadClientUseCase(businessType, false)

	// ID from packages/copya/data/education/client.json
	existingID := "student-001"

	req := &clientpb.ReadClientRequest{
		Data: &clientpb.Client{Id: existingID},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertEqual(t, 1, len(res.Data), "response data length")

	readClient := res.Data[0]
	testutil.AssertStringEqual(t, existingID, readClient.Id, "client ID")
	testutil.AssertStringEqual(t, "john.doe@example.com", readClient.User.EmailAddress, "email address")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "Success", true, nil)
}

func TestReadClientUseCase_Execute_NotFound(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-CLIENT-READ-NOT-FOUND-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "NotFound", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadClientUseCase(businessType, false)

	nonExistentID := "client-999"

	req := &clientpb.ReadClientRequest{
		Data: &clientpb.Client{Id: nonExistentID},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)

	testutil.AssertTranslatedErrorWithContext(t, err, "client.errors.not_found", "{\"clientId\": \""+nonExistentID+"\"}", useCase.services.TranslationService, ctx)

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "NotFound", false, err)
}

func TestReadClientUseCase_Execute_EmptyId(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-CLIENT-READ-EMPTY-ID-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "EmptyId", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadClientUseCase(businessType, false)

	req := &clientpb.ReadClientRequest{
		Data: &clientpb.Client{Id: ""},
	}
	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)

	testutil.AssertTranslatedError(t, err, "client.validation.id_required", useCase.services.TranslationService, ctx)

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "EmptyId", false, err)
}

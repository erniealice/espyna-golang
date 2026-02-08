//go:build mock_db && mock_auth

// Package client provides comprehensive tests for the client updating use case.
//
// The tests cover various scenarios, including success, not found errors, validation errors,
// authorization, and boundary conditions. Each test function has a specific test code
// for tracking and logging.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestUpdateClientUseCase_Execute_Success
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-ENTITY-CLIENT-UPDATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-ENTITY-CLIENT-UPDATE-NOT-FOUND-v1.0: NotFound
//   - ESPYNA-TEST-ENTITY-CLIENT-UPDATE-VALIDATION-v1.0: ValidationErrors
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/client.json
//   - Mock data: packages/copya/data/{businessType}/client.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/client.json
package client

import (
	"strings"
	"testing"
	"time"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"

	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

func createTestUpdateClientUseCase(businessType string, supportsTransaction bool) *UpdateClientUseCase {
	repositories := UpdateClientRepositories{
		Client: entity.NewMockClientRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, true)
	services := UpdateClientServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewUpdateClientUseCase(repositories, services)
}

func TestUpdateClientUseCase_Execute_Success(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-CLIENT-UPDATE-SUCCESS-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "Success", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateClientUseCase(businessType, false)

	// Load test data resolvers
	commonDataResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "client", "Client_CommonData")
	testutil.AssertTestCaseLoad(t, err, "Client_CommonData")

	updateSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "client", "CreateClient_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateClient_Success")

	existingID := commonDataResolver.MustGetString("primaryClientId")
	updatedEmail := updateSuccessResolver.MustGetString("newClientEmailAddress")
	startTime := time.Now().UnixMilli() // Current time before update

	req := &clientpb.UpdateClientRequest{
		Data: &clientpb.Client{
			Id: existingID,
			User: &userpb.User{
				Id:           "user-student-001",
				FirstName:    updateSuccessResolver.MustGetString("newClientFirstName"),
				LastName:     updateSuccessResolver.MustGetString("newClientLastName"),
				EmailAddress: updatedEmail,
			},
		},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	updatedClient := res.Data[0]
	testutil.AssertNotNil(t, updatedClient.User, "updated client user data")
	testutil.AssertEqual(t, updatedEmail, updatedClient.User.EmailAddress, "email should be updated")

	testutil.AssertNotNil(t, updatedClient.DateModified, "DateModified should be set")
	testutil.AssertGreaterThanOrEqual(t, int(*updatedClient.DateModified), int(startTime), "DateModified should be recent")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "Success", true, nil)
}

func TestUpdateClientUseCase_Execute_NotFound(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-CLIENT-UPDATE-NOT-FOUND-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "NotFound", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateClientUseCase(businessType, false)

	// Load test data resolvers
	commonDataResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "client", "Client_CommonData")
	testutil.AssertTestCaseLoad(t, err, "Client_CommonData")

	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "client", "CreateClient_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateClient_Success")

	nonExistentID := commonDataResolver.MustGetString("nonExistentId")
	req := &clientpb.UpdateClientRequest{
		Data: &clientpb.Client{
			Id: nonExistentID,
			User: &userpb.User{
				Id:           "user-999",
				FirstName:    createSuccessResolver.MustGetString("newClientFirstName"),
				LastName:     createSuccessResolver.MustGetString("newClientLastName"),
				EmailAddress: createSuccessResolver.MustGetString("newClientEmailAddress"),
			},
		},
	}

	_, err = useCase.Execute(ctx, req)
	testutil.AssertError(t, err)

	// Check for the actual error message format - using strings.Contains as it was in original
	testutil.AssertTrue(t, strings.Contains(err.Error(), "Student update failed: client with ID '"+nonExistentID+"' not found"), "error should contain expected not found message")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "NotFound", true, nil)
}

func TestUpdateClientUseCase_Execute_ValidationErrors(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-CLIENT-UPDATE-VALIDATION-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "ValidationErrors", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateClientUseCase(businessType, false)

	// Load test data resolvers
	commonDataResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "client", "Client_CommonData")
	testutil.AssertTestCaseLoad(t, err, "Client_CommonData")

	validationEmptyResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "client", "ValidationError_EmptyName")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyName")

	existingID := commonDataResolver.MustGetString("secondaryClientId")

	testCases := []struct {
		name             string
		client           *clientpb.Client
		expectedErrorKey string
	}{
		{
			name: "Empty ID",
			client: &clientpb.Client{
				User: &userpb.User{
					FirstName:    validationEmptyResolver.MustGetString("validLastName"),
					LastName:     validationEmptyResolver.MustGetString("validLastName"),
					EmailAddress: validationEmptyResolver.MustGetString("validEmailAddress"),
				},
			},
			expectedErrorKey: "client.validation.id_required",
		},
		{
			name: "Missing Email",
			client: &clientpb.Client{
				Id: existingID,
				User: &userpb.User{
					FirstName:    validationEmptyResolver.MustGetString("validLastName"),
					LastName:     validationEmptyResolver.MustGetString("validLastName"),
					EmailAddress: "",
				},
			},
			expectedErrorKey: "client.validation.email_required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &clientpb.UpdateClientRequest{Data: tc.client}
			_, err := useCase.Execute(ctx, req)
			testutil.AssertError(t, err)
			testutil.AssertTranslatedError(t, err, tc.expectedErrorKey, useCase.services.TranslationService, ctx)
		})
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "ValidationErrors", true, nil)
}

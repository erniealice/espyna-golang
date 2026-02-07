//go:build mock_db && mock_auth

// Package workspace provides table-driven tests for the workspace creation use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, and boundary conditions.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestCreateWorkspaceUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-ENTITY-WORKSPACE-CREATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-ENTITY-WORKSPACE-CREATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-ENTITY-WORKSPACE-CREATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-ENTITY-WORKSPACE-CREATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-ENTITY-WORKSPACE-CREATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-ENTITY-WORKSPACE-CREATE-VALIDATION-EMPTY-NAME-v1.0: EmptyName
//   - ESPYNA-TEST-ENTITY-WORKSPACE-CREATE-VALIDATION-NAME-TOO-SHORT-v1.0: NameTooShort
//   - ESPYNA-TEST-ENTITY-WORKSPACE-CREATE-ENRICHMENT-v1.0: DataEnrichment
//   - ESPYNA-TEST-ENTITY-WORKSPACE-CREATE-VALIDATION-MINIMAL-v1.0: MinimalValidData
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/workspace.json
//   - Mock data: packages/copya/data/{businessType}/workspace.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/workspace.json

package workspace

import (
	"context"
	"testing"

	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"
	workspacepb "leapfor.xyz/esqyma/golang/v1/domain/entity/workspace"
)

// Type alias for create workspace test cases
type CreateWorkspaceTestCase = testutil.GenericTestCase[*workspacepb.CreateWorkspaceRequest, *workspacepb.CreateWorkspaceResponse]

func createTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreateWorkspaceUseCase {
	repositories := CreateWorkspaceRepositories{
		Workspace: entity.NewMockWorkspaceRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreateWorkspaceServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreateWorkspaceUseCase(repositories, services)
}

func TestCreateWorkspaceUseCase_Execute_TableDriven(t *testing.T) {

	testCases := []CreateWorkspaceTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-ENTITY-WORKSPACE-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workspacepb.CreateWorkspaceRequest {
				return &workspacepb.CreateWorkspaceRequest{
					Data: &workspacepb.Workspace{
						Name: "My Test Workspace",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *workspacepb.CreateWorkspaceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdWorkspace := response.Data[0]
				testutil.AssertStringEqual(t, "My Test Workspace", createdWorkspace.Name, "workspace name")
				testutil.AssertNonEmptyString(t, createdWorkspace.Id, "workspace ID")
				testutil.AssertTrue(t, createdWorkspace.Active, "workspace active status")
				testutil.AssertFieldSet(t, createdWorkspace.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdWorkspace.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-ENTITY-WORKSPACE-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workspacepb.CreateWorkspaceRequest {
				return &workspacepb.CreateWorkspaceRequest{
					Data: &workspacepb.Workspace{
						Name: "Transaction Test Workspace",
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *workspacepb.CreateWorkspaceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdWorkspace := response.Data[0]
				testutil.AssertStringEqual(t, "Transaction Test Workspace", createdWorkspace.Name, "workspace name")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-ENTITY-WORKSPACE-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workspacepb.CreateWorkspaceRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "workspace.validation.request_required",
			Assertions: func(t *testing.T, response *workspacepb.CreateWorkspaceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-ENTITY-WORKSPACE-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workspacepb.CreateWorkspaceRequest {
				return &workspacepb.CreateWorkspaceRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "workspace.validation.data_required",
			Assertions: func(t *testing.T, response *workspacepb.CreateWorkspaceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyName",
			TestCode: "ESPYNA-TEST-ENTITY-WORKSPACE-CREATE-VALIDATION-EMPTY-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workspacepb.CreateWorkspaceRequest {
				return &workspacepb.CreateWorkspaceRequest{
					Data: &workspacepb.Workspace{
						Name: "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "workspace.validation.name_required",
			Assertions: func(t *testing.T, response *workspacepb.CreateWorkspaceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty name")
			},
		},
		{
			Name:     "NameTooShort",
			TestCode: "ESPYNA-TEST-ENTITY-WORKSPACE-CREATE-VALIDATION-NAME-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workspacepb.CreateWorkspaceRequest {
				return &workspacepb.CreateWorkspaceRequest{
					Data: &workspacepb.Workspace{
						Name: "A",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "workspace.validation.name_too_short",
			Assertions: func(t *testing.T, response *workspacepb.CreateWorkspaceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too short")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-ENTITY-WORKSPACE-CREATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workspacepb.CreateWorkspaceRequest {
				return &workspacepb.CreateWorkspaceRequest{
					Data: &workspacepb.Workspace{
						Name: "Data Enrichment Test",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *workspacepb.CreateWorkspaceResponse, err error, useCase interface{}, ctx context.Context) {
				createdWorkspace := response.Data[0]
				testutil.AssertNonEmptyString(t, createdWorkspace.Id, "generated ID")
				testutil.AssertFieldSet(t, createdWorkspace.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdWorkspace.DateCreatedString, "DateCreatedString")
				testutil.AssertTrue(t, createdWorkspace.Active, "Active")
			},
		},
		{
			Name:     "MinimalValidData",
			TestCode: "ESPYNA-TEST-ENTITY-WORKSPACE-CREATE-VALIDATION-MINIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workspacepb.CreateWorkspaceRequest {
				return &workspacepb.CreateWorkspaceRequest{
					Data: &workspacepb.Workspace{
						Name: "AB",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *workspacepb.CreateWorkspaceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdWorkspace := response.Data[0]
				testutil.AssertStringEqual(t, "AB", createdWorkspace.Name, "workspace name")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			// Set test code and log execution start
			testutil.SetTestCode(t, tc.TestCode)
			testutil.LogTestExecution(t, tc.TestCode, tc.Name, tc.ExpectSuccess)

			ctx := testutil.CreateTestContext()
			businessType := testutil.GetTestBusinessType()
			useCase := createTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

			req := tc.SetupRequest(t, businessType)
			response, err := useCase.Execute(ctx, req)

			// Determine actual success/failure
			actualSuccess := err == nil && tc.ExpectSuccess

			if tc.ExpectSuccess {
				testutil.AssertNoError(t, err)
				testutil.AssertNotNil(t, response, "response")
			} else {
				testutil.AssertError(t, err)
				if tc.ExpectedError != "" {
					if tc.ErrorTags != nil {
						testutil.AssertTranslatedErrorWithTags(t, err, tc.ExpectedError, tc.ErrorTags, useCase.services.TranslationService, ctx)
					} else {
						testutil.AssertTranslatedError(t, err, tc.ExpectedError, useCase.services.TranslationService, ctx)
					}
				}
			}

			if tc.Assertions != nil {
				tc.Assertions(t, response, err, useCase, ctx)
			}

			// Log test completion with result
			testutil.LogTestResult(t, tc.TestCode, tc.Name, actualSuccess, err)
		})
	}
}

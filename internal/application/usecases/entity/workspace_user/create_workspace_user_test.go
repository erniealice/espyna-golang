//go:build mock_db && mock_auth

// Package workspace_user provides table-driven tests for the workspace user creation use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestCreateWorkspaceUserUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-ENTITY-WORKSPACE-USER-CREATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-ENTITY-WORKSPACE-USER-CREATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-ENTITY-WORKSPACE-USER-CREATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-ENTITY-WORKSPACE-USER-CREATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-ENTITY-WORKSPACE-USER-CREATE-VALIDATION-EMPTY-WORKSPACE-ID-v1.0: EmptyWorkspaceId
//   - ESPYNA-TEST-ENTITY-WORKSPACE-USER-CREATE-VALIDATION-EMPTY-USER-ID-v1.0: EmptyUserId
//   - ESPYNA-TEST-ENTITY-WORKSPACE-USER-CREATE-VALIDATION-INVALID-WORKSPACE-ID-v1.0: InvalidWorkspaceId
//   - ESPYNA-TEST-ENTITY-WORKSPACE-USER-CREATE-VALIDATION-INVALID-USER-ID-v1.0: InvalidUserId
//   - ESPYNA-TEST-ENTITY-WORKSPACE-USER-CREATE-ENRICHMENT-v1.0: DataEnrichment
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/workspace_user.json
//   - Mock data: packages/copya/data/{businessType}/workspace_user.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/workspace_user.json

package workspace_user

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	mockAuth "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/auth/mock"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	rolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role"
	workspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user"
	workspaceuserrolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user_role"
)

// Type alias for create workspace user test cases
type CreateWorkspaceUserTestCase = testutil.GenericTestCase[*workspaceuserpb.CreateWorkspaceUserRequest, *workspaceuserpb.CreateWorkspaceUserResponse]

func createTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreateWorkspaceUserUseCase {
	repositories := CreateWorkspaceUserRepositories{
		WorkspaceUser: entity.NewMockWorkspaceUserRepository(businessType),
		Workspace:     entity.NewMockWorkspaceRepository(businessType),
		User:          entity.NewMockUserRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreateWorkspaceUserServices{
		AuthorizationService: mockAuth.NewAllowAllAuth().SetUserWorkspaces("test-user", "workspace-elementary", "workspace-middle", "workspace-high"),
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreateWorkspaceUserUseCase(repositories, services)
}

func TestCreateWorkspaceUserUseCase_Execute_TableDriven(t *testing.T) {
	// IDs from packages/copya/data/education/
	existingWorkspaceID := "workspace-elementary"
	existingUserID := "user-student-001"

	testCases := []CreateWorkspaceUserTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-ENTITY-WORKSPACE-USER-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workspaceuserpb.CreateWorkspaceUserRequest {
				return &workspaceuserpb.CreateWorkspaceUserRequest{
					Data: &workspaceuserpb.WorkspaceUser{
						WorkspaceId: existingWorkspaceID,
						UserId:      existingUserID,
						WorkspaceUserRoles: []*workspaceuserrolepb.WorkspaceUserRole{
							{
								Id:     "wsur-001",
								RoleId: "role-student",
								Role:   &rolepb.Role{Id: "role-student", Name: "student"},
								Active: true,
							},
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *workspaceuserpb.CreateWorkspaceUserResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdRel := response.Data[0]
				testutil.AssertStringEqual(t, existingUserID, createdRel.UserId, "user ID")
				testutil.AssertStringEqual(t, existingWorkspaceID, createdRel.WorkspaceId, "workspace ID")
				testutil.AssertNonEmptyString(t, createdRel.Id, "workspace user ID")
				testutil.AssertFieldSet(t, createdRel.DateCreated, "DateCreated")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-ENTITY-WORKSPACE-USER-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workspaceuserpb.CreateWorkspaceUserRequest {
				return &workspaceuserpb.CreateWorkspaceUserRequest{
					Data: &workspaceuserpb.WorkspaceUser{
						WorkspaceId: existingWorkspaceID,
						UserId:      existingUserID,
						WorkspaceUserRoles: []*workspaceuserrolepb.WorkspaceUserRole{
							{
								Id:     "wsur-002",
								RoleId: "role-teacher",
								Role:   &rolepb.Role{Id: "role-teacher", Name: "teacher"},
								Active: true,
							},
						},
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *workspaceuserpb.CreateWorkspaceUserResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdRel := response.Data[0]
				testutil.AssertStringEqual(t, existingUserID, createdRel.UserId, "user ID")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-ENTITY-WORKSPACE-USER-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workspaceuserpb.CreateWorkspaceUserRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "workspace_user.validation.request_required",
			Assertions: func(t *testing.T, response *workspaceuserpb.CreateWorkspaceUserResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-ENTITY-WORKSPACE-USER-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workspaceuserpb.CreateWorkspaceUserRequest {
				return &workspaceuserpb.CreateWorkspaceUserRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "workspace_user.validation.data_required",
			Assertions: func(t *testing.T, response *workspaceuserpb.CreateWorkspaceUserResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyWorkspaceId",
			TestCode: "ESPYNA-TEST-ENTITY-WORKSPACE-USER-CREATE-VALIDATION-EMPTY-WORKSPACE-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workspaceuserpb.CreateWorkspaceUserRequest {
				return &workspaceuserpb.CreateWorkspaceUserRequest{
					Data: &workspaceuserpb.WorkspaceUser{
						WorkspaceId: "",
						UserId:      existingUserID,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "workspace_user.validation.workspace_id_required",
			Assertions: func(t *testing.T, response *workspaceuserpb.CreateWorkspaceUserResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty workspace ID")
			},
		},
		{
			Name:     "EmptyUserId",
			TestCode: "ESPYNA-TEST-ENTITY-WORKSPACE-USER-CREATE-VALIDATION-EMPTY-USER-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workspaceuserpb.CreateWorkspaceUserRequest {
				return &workspaceuserpb.CreateWorkspaceUserRequest{
					Data: &workspaceuserpb.WorkspaceUser{
						WorkspaceId: existingWorkspaceID,
						UserId:      "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "workspace_user.validation.user_id_required",
			Assertions: func(t *testing.T, response *workspaceuserpb.CreateWorkspaceUserResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty user ID")
			},
		},
		{
			Name:     "InvalidWorkspaceId",
			TestCode: "ESPYNA-TEST-ENTITY-WORKSPACE-USER-CREATE-VALIDATION-INVALID-WORKSPACE-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workspaceuserpb.CreateWorkspaceUserRequest {
				return &workspaceuserpb.CreateWorkspaceUserRequest{
					Data: &workspaceuserpb.WorkspaceUser{
						WorkspaceId: "workspace-999",
						UserId:      existingUserID,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "workspace_user.errors.access_denied",
			Assertions: func(t *testing.T, response *workspaceuserpb.CreateWorkspaceUserResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid workspace ID")
			},
		},
		{
			Name:     "InvalidUserId",
			TestCode: "ESPYNA-TEST-ENTITY-WORKSPACE-USER-CREATE-VALIDATION-INVALID-USER-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workspaceuserpb.CreateWorkspaceUserRequest {
				return &workspaceuserpb.CreateWorkspaceUserRequest{
					Data: &workspaceuserpb.WorkspaceUser{
						WorkspaceId: existingWorkspaceID,
						UserId:      "user-999",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "workspace_user.errors.reference_validation_failed",
			Assertions: func(t *testing.T, response *workspaceuserpb.CreateWorkspaceUserResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid user ID")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-ENTITY-WORKSPACE-USER-CREATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workspaceuserpb.CreateWorkspaceUserRequest {
				return &workspaceuserpb.CreateWorkspaceUserRequest{
					Data: &workspaceuserpb.WorkspaceUser{
						WorkspaceId: existingWorkspaceID,
						UserId:      existingUserID,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *workspaceuserpb.CreateWorkspaceUserResponse, err error, useCase interface{}, ctx context.Context) {
				createdRel := response.Data[0]
				testutil.AssertNonEmptyString(t, createdRel.Id, "generated ID")
				testutil.AssertFieldSet(t, createdRel.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdRel.DateCreatedString, "DateCreatedString")
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

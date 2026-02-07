//go:build mock_db && mock_auth

// Package stage provides table-driven tests for the stage creation use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, data enrichment, and foreign key validation.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestCreateStageUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-WORKFLOW-STAGE-CREATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-WORKFLOW-STAGE-CREATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-WORKFLOW-STAGE-CREATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-WORKFLOW-STAGE-CREATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-WORKFLOW-STAGE-CREATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-WORKFLOW-STAGE-CREATE-VALIDATION-EMPTY-NAME-v1.0: EmptyName
//   - ESPYNA-TEST-WORKFLOW-STAGE-CREATE-VALIDATION-EMPTY-WORKFLOW-INSTANCE-ID-v1.0: EmptyWorkflowInstanceId
//   - ESPYNA-TEST-WORKFLOW-STAGE-CREATE-VALIDATION-INVALID-WORKFLOW-INSTANCE-ID-v1.0: InvalidWorkflowInstanceId
//   - ESPYNA-TEST-WORKFLOW-STAGE-CREATE-VALIDATION-NAME-TOO-SHORT-v1.0: NameTooShort
//   - ESPYNA-TEST-WORKFLOW-STAGE-CREATE-VALIDATION-NAME-TOO-LONG-v1.0: NameTooLong
//   - ESPYNA-TEST-WORKFLOW-STAGE-CREATE-VALIDATION-DESCRIPTION-TOO-LONG-v1.0: DescriptionTooLong
//   - ESPYNA-TEST-WORKFLOW-STAGE-CREATE-ENRICHMENT-v1.0: DataEnrichment
//   - ESPYNA-TEST-WORKFLOW-STAGE-CREATE-VALIDATION-MINIMAL-v1.0: MinimalValidData
//   - ESPYNA-TEST-WORKFLOW-STAGE-CREATE-VALIDATION-MAXIMAL-v1.0: MaxValidData
//   - ESPYNA-TEST-WORKFLOW-STAGE-CREATE-TRANSACTION-FAILURE-v1.0: TransactionFailure
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/stage.json
//   - Mock data: packages/copya/data/{businessType}/stage.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/stage.json
package stage

import (
	"context"
	"strings"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockDb "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/workflow"
	stagepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/stage"
)

// Type alias for create stage test cases
type CreateStageTestCase = testutil.GenericTestCase[*stagepb.CreateStageRequest, *stagepb.CreateStageResponse]

func createStageTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreateStageUseCase {
	mockStageRepo := workflow.NewMockStageRepository(businessType)
	mockWorkflowRepo := workflow.NewMockWorkflowRepository(businessType)           // For foreign key validation
	mockStageTemplateRepo := workflow.NewMockStageTemplateRepository(businessType) // For foreign key validation

	repositories := CreateStageRepositories{
		Stage:         mockStageRepo,
		Workflow:      mockWorkflowRepo,
		StageTemplate: mockStageTemplateRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreateStageServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreateStageUseCase(repositories, services)
}

func TestCreateStageUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage", "CreateStage_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateStage_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	validationErrorEmptyNameResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage", "ValidationError_EmptyName")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyName")

	validationErrorEmptyWorkflowInstanceIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage", "ValidationError_EmptyWorkflowInstanceId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyWorkflowInstanceId")

	validationErrorInvalidWorkflowInstanceIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage", "ValidationError_InvalidWorkflowInstanceId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidWorkflowInstanceId")

	testCases := []CreateStageTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.CreateStageRequest {
				return &stagepb.CreateStageRequest{
					Data: &stagepb.Stage{
						Name:            createSuccessResolver.MustGetString("newStageName"),
						Description:     &[]string{createSuccessResolver.MustGetString("newStageDescription")}[0],
						StageTemplateId: createSuccessResolver.MustGetString("validStageTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *stagepb.CreateStageResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdStage := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newStageName"), createdStage.Name, "stage name")
				testutil.AssertNonEmptyString(t, createdStage.Id, "stage ID")
				testutil.AssertTrue(t, createdStage.Active, "stage active status")
				testutil.AssertFieldSet(t, createdStage.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdStage.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.CreateStageRequest {
				return &stagepb.CreateStageRequest{
					Data: &stagepb.Stage{
						Name:            createSuccessResolver.MustGetString("newStageName"),
						Description:     &[]string{createSuccessResolver.MustGetString("newStageDescription")}[0],
						StageTemplateId: createSuccessResolver.MustGetString("validStageTemplateId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *stagepb.CreateStageResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdStage := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newStageName"), createdStage.Name, "stage name")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-CREATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.CreateStageRequest {
				return &stagepb.CreateStageRequest{
					Data: &stagepb.Stage{
						Name:            authorizationUnauthorizedResolver.MustGetString("unauthorizedStageName"),
						Description:     &[]string{authorizationUnauthorizedResolver.MustGetString("unauthorizedStageDescription")}[0],
						StageTemplateId: authorizationUnauthorizedResolver.MustGetString("unauthorizedStageTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "Authorization failed for stages",
			Assertions: func(t *testing.T, response *stagepb.CreateStageResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.CreateStageRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Request is required for stages",
			Assertions: func(t *testing.T, response *stagepb.CreateStageResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.CreateStageRequest {
				return &stagepb.CreateStageRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Stage data is required",
			Assertions: func(t *testing.T, response *stagepb.CreateStageResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyName",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-CREATE-VALIDATION-EMPTY-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.CreateStageRequest {
				return &stagepb.CreateStageRequest{
					Data: &stagepb.Stage{
						Name:            validationErrorEmptyNameResolver.MustGetString("emptyName"),
						StageTemplateId: validationErrorEmptyNameResolver.MustGetString("validStageTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Stage name is required",
			Assertions: func(t *testing.T, response *stagepb.CreateStageResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty name")
			},
		},
		{
			Name:     "EmptyWorkflowInstanceId",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-CREATE-VALIDATION-EMPTY-WORKFLOW-INSTANCE-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.CreateStageRequest {
				return &stagepb.CreateStageRequest{
					Data: &stagepb.Stage{
						Name:               validationErrorEmptyWorkflowInstanceIdResolver.MustGetString("validName"),
						WorkflowInstanceId: validationErrorEmptyWorkflowInstanceIdResolver.MustGetString("emptyWorkflowInstanceId"),
						StageTemplateId:    validationErrorEmptyWorkflowInstanceIdResolver.MustGetString("validStageTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Workflow instance ID is required",
			Assertions: func(t *testing.T, response *stagepb.CreateStageResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty workflow instance ID")
			},
		},
		{
			Name:     "InvalidWorkflowInstanceId",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-CREATE-VALIDATION-INVALID-WORKFLOW-INSTANCE-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.CreateStageRequest {
				return &stagepb.CreateStageRequest{
					Data: &stagepb.Stage{
						Name:               validationErrorInvalidWorkflowInstanceIdResolver.MustGetString("validName"),
						WorkflowInstanceId: validationErrorInvalidWorkflowInstanceIdResolver.MustGetString("invalidWorkflowInstanceId"),
						StageTemplateId:    validationErrorInvalidWorkflowInstanceIdResolver.MustGetString("validStageTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Entity reference validation failed [DEFAULT]: Failed to validate workflow_instance entity reference [DEFAULT]: workflow_instance with ID 'workflow-instance-non-existent-123' not found",
			Assertions: func(t *testing.T, response *stagepb.CreateStageResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid workflow instance ID")
			},
		},
		{
			Name:     "NameTooShort",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-CREATE-VALIDATION-NAME-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.CreateStageRequest {
				return &stagepb.CreateStageRequest{
					Data: &stagepb.Stage{
						Name:            "AB",
						StageTemplateId: createSuccessResolver.MustGetString("validStageTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Stage name must be at least 3 characters long",
			Assertions: func(t *testing.T, response *stagepb.CreateStageResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too short")
			},
		},
		{
			Name:     "NameTooLong",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-CREATE-VALIDATION-NAME-TOO-LONG-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.CreateStageRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage", "ValidationError_NameTooLongGenerated")
				testutil.AssertTestCaseLoad(t, err, "ValidationError_NameTooLongGenerated")
				return &stagepb.CreateStageRequest{
					Data: &stagepb.Stage{
						Name:            resolver.MustGetString("tooLongNameGenerated"),
						StageTemplateId: createSuccessResolver.MustGetString("validStageTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Stage name cannot exceed 100 characters",
			Assertions: func(t *testing.T, response *stagepb.CreateStageResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too long")
			},
		},
		{
			Name:     "DescriptionTooLong",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-CREATE-VALIDATION-DESCRIPTION-TOO-LONG-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.CreateStageRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage", "ValidationError_DescriptionTooLongGenerated")
				testutil.AssertTestCaseLoad(t, err, "ValidationError_DescriptionTooLongGenerated")
				return &stagepb.CreateStageRequest{
					Data: &stagepb.Stage{
						Name:            resolver.MustGetString("validName"),
						Description:     &[]string{resolver.MustGetString("tooLongDescriptionGenerated")}[0],
						StageTemplateId: createSuccessResolver.MustGetString("validStageTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Stage description cannot exceed 1000 characters",
			Assertions: func(t *testing.T, response *stagepb.CreateStageResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "description too long")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-CREATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.CreateStageRequest {
				return &stagepb.CreateStageRequest{
					Data: &stagepb.Stage{
						Name:            "Data Enrichment Test",
						StageTemplateId: createSuccessResolver.MustGetString("validStageTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *stagepb.CreateStageResponse, err error, useCase interface{}, ctx context.Context) {
				createdStage := response.Data[0]
				testutil.AssertNonEmptyString(t, createdStage.Id, "generated ID")
				testutil.AssertFieldSet(t, createdStage.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdStage.DateCreatedString, "DateCreatedString")
				testutil.AssertTrue(t, createdStage.Active, "Active")
			},
		},
		{
			Name:     "MinimalValidData",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-CREATE-VALIDATION-MINIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.CreateStageRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage", "BoundaryTest_MinimalValid")
				testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MinimalValid")
				return &stagepb.CreateStageRequest{
					Data: &stagepb.Stage{
						Name:            resolver.MustGetString("minValidName"),
						StageTemplateId: createSuccessResolver.MustGetString("validStageTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *stagepb.CreateStageResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdStage := response.Data[0]
				resolver, _ := copyatestutil.LoadTestCaseFromBusinessType(t, testutil.GetTestBusinessType(), "stage", "BoundaryTest_MinimalValid")
				testutil.AssertStringEqual(t, resolver.MustGetString("minValidName"), createdStage.Name, "stage name")
			},
		},
		{
			Name:     "MaxValidData",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-CREATE-VALIDATION-MAXIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.CreateStageRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage", "BoundaryTest_MaximalValid")
				testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MaximalValid")
				return &stagepb.CreateStageRequest{
					Data: &stagepb.Stage{
						Name:            resolver.MustGetString("maxValidNameExact100"),
						Description:     &[]string{resolver.MustGetString("maxValidDescriptionExact1000")}[0],
						StageTemplateId: createSuccessResolver.MustGetString("validStageTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *stagepb.CreateStageResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdStage := response.Data[0]
				testutil.AssertEqual(t, 100, len(createdStage.Name), "name length")
				if createdStage.Description != nil {
					testutil.AssertEqual(t, 1000, len(*createdStage.Description), "description length")
				}
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
			useCase := createStageTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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
					if !strings.Contains(err.Error(), tc.ExpectedError) {
						t.Errorf("Expected error containing '%s', got '%s'", tc.ExpectedError, err.Error())
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

func TestCreateStageUseCase_Execute_TransactionFailure(t *testing.T) {
	testName := "TransactionFailure"
	testCode := "ESPYNA-TEST-WORKFLOW-STAGE-CREATE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, testName, false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockStageRepo := workflow.NewMockStageRepository(businessType)
	mockWorkflowRepo := workflow.NewMockWorkflowRepository(businessType)
	mockStageTemplateRepo := workflow.NewMockStageTemplateRepository(businessType)

	repositories := CreateStageRepositories{
		Stage:         mockStageRepo,
		Workflow:      mockWorkflowRepo,
		StageTemplate: mockStageTemplateRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := CreateStageServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	useCase := NewCreateStageUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage", "CreateStage_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateStage_Success")

	req := &stagepb.CreateStageRequest{
		Data: &stagepb.Stage{
			Name:            resolver.MustGetString("newStageName"),
			Description:     &[]string{resolver.MustGetString("newStageDescription")}[0],
			StageTemplateId: resolver.MustGetString("validStageTemplateId"),
		},
	}

	_, err = useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, testName, false, err)
}

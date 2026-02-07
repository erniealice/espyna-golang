//go:build mock_db && mock_auth

// Package activity_template provides table-driven tests for the activity template creation use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, data enrichment, and boundary conditions.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestCreateActivityTemplateUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-CREATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-CREATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-CREATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-CREATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-CREATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-CREATE-VALIDATION-EMPTY-NAME-v1.0: EmptyName
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-CREATE-VALIDATION-EMPTY-STAGE-TEMPLATE-ID-v1.0: EmptyStageTemplateId
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-CREATE-VALIDATION-INVALID-STAGE-TEMPLATE-ID-v1.0: InvalidStageTemplateId
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-CREATE-VALIDATION-NAME-TOO-SHORT-v1.0: NameTooShort
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-CREATE-VALIDATION-NAME-TOO-LONG-v1.0: NameTooLong
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-CREATE-VALIDATION-DESCRIPTION-TOO-LONG-v1.0: DescriptionTooLong
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-CREATE-VALIDATION-INVALID-DURATION-v1.0: InvalidDuration
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-CREATE-VALIDATION-INVALID-ACTION-TYPE-v1.0: InvalidActionType
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-CREATE-ENRICHMENT-v1.0: DataEnrichment
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-CREATE-VALIDATION-MINIMAL-v1.0: MinimalValidData
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-CREATE-VALIDATION-MAXIMAL-v1.0: MaxValidData
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-CREATE-TRANSACTION-FAILURE-v1.0: TransactionFailure
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/activity_template.json
//   - Mock data: packages/copya/data/{businessType}/activity_template.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/activity_template.json
package activity_template

import (
	"context"
	"strings"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockDb "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/workflow"
	activityTemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/activity_template"
)

// Type alias for create activity template test cases
type CreateActivityTemplateTestCase = testutil.GenericTestCase[*activityTemplatepb.CreateActivityTemplateRequest, *activityTemplatepb.CreateActivityTemplateResponse]

func createTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreateActivityTemplateUseCase {
	mockActivityTemplateRepo := workflow.NewMockActivityTemplateRepository(businessType)
	mockStageTemplateRepo := workflow.NewMockStageTemplateRepository(businessType) // For foreign key validation

	repositories := CreateActivityTemplateRepositories{
		ActivityTemplate: mockActivityTemplateRepo,
		StageTemplate:    mockStageTemplateRepo, // Foreign key validation
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreateActivityTemplateServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreateActivityTemplateUseCase(repositories, services)
}

func TestCreateActivityTemplateUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity_template", "CreateActivityTemplate_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateActivityTemplate_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity_template", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	validationErrorEmptyNameResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity_template", "ValidationError_EmptyName")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyName")

	validationErrorEmptyStageTemplateIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity_template", "ValidationError_EmptyStageTemplateId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyStageTemplateId")

	validationErrorInvalidStageTemplateIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity_template", "ValidationError_InvalidStageTemplateId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidStageTemplateId")

	testCases := []CreateActivityTemplateTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.CreateActivityTemplateRequest {
				return &activityTemplatepb.CreateActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Name:                     createSuccessResolver.MustGetString("newActivityTemplateName"),
						Description:              &[]string{createSuccessResolver.MustGetString("newActivityTemplateDescription")}[0],
						StageTemplateId:          createSuccessResolver.MustGetString("validStageTemplateId"),
						EstimatedDurationMinutes: &[]int32{30}[0],
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activityTemplatepb.CreateActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdActivityTemplate := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newActivityTemplateName"), createdActivityTemplate.Name, "activity template name")
				testutil.AssertNonEmptyString(t, createdActivityTemplate.Id, "activity template ID")
				testutil.AssertTrue(t, createdActivityTemplate.Active, "activity template active status")
				testutil.AssertFieldSet(t, createdActivityTemplate.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdActivityTemplate.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.CreateActivityTemplateRequest {
				return &activityTemplatepb.CreateActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Name:                     createSuccessResolver.MustGetString("newActivityTemplateName"),
						Description:              &[]string{createSuccessResolver.MustGetString("newActivityTemplateDescription")}[0],
						StageTemplateId:          createSuccessResolver.MustGetString("validStageTemplateId"),
						EstimatedDurationMinutes: &[]int32{45}[0],
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activityTemplatepb.CreateActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdActivityTemplate := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newActivityTemplateName"), createdActivityTemplate.Name, "activity template name")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-CREATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.CreateActivityTemplateRequest {
				return &activityTemplatepb.CreateActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Name:            authorizationUnauthorizedResolver.MustGetString("unauthorizedActivityTemplateName"),
						Description:     &[]string{authorizationUnauthorizedResolver.MustGetString("unauthorizedActivityTemplateDescription")}[0],
						StageTemplateId: authorizationUnauthorizedResolver.MustGetString("unauthorizedStageTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "Authorization failed for activity templates",
			Assertions: func(t *testing.T, response *activityTemplatepb.CreateActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.CreateActivityTemplateRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Request is required for activity templates",
			Assertions: func(t *testing.T, response *activityTemplatepb.CreateActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.CreateActivityTemplateRequest {
				return &activityTemplatepb.CreateActivityTemplateRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Activity template data is required",
			Assertions: func(t *testing.T, response *activityTemplatepb.CreateActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyName",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-CREATE-VALIDATION-EMPTY-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.CreateActivityTemplateRequest {
				return &activityTemplatepb.CreateActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Name:            validationErrorEmptyNameResolver.MustGetString("emptyName"),
						StageTemplateId: validationErrorEmptyNameResolver.MustGetString("validStageTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Activity template name is required",
			Assertions: func(t *testing.T, response *activityTemplatepb.CreateActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty name")
			},
		},
		{
			Name:     "EmptyStageTemplateId",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-CREATE-VALIDATION-EMPTY-STAGE-TEMPLATE-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.CreateActivityTemplateRequest {
				return &activityTemplatepb.CreateActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Name:            validationErrorEmptyStageTemplateIdResolver.MustGetString("validName"),
						StageTemplateId: validationErrorEmptyStageTemplateIdResolver.MustGetString("emptyStageTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Stage template ID is required",
			Assertions: func(t *testing.T, response *activityTemplatepb.CreateActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty stage template ID")
			},
		},
		{
			Name:     "InvalidStageTemplateId",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-CREATE-VALIDATION-INVALID-STAGE-TEMPLATE-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.CreateActivityTemplateRequest {
				return &activityTemplatepb.CreateActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Name:            validationErrorInvalidStageTemplateIdResolver.MustGetString("validName"),
						StageTemplateId: validationErrorInvalidStageTemplateIdResolver.MustGetString("invalidStageTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Referenced stage template does not exist",
			Assertions: func(t *testing.T, response *activityTemplatepb.CreateActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid stage template ID")
			},
		},
		{
			Name:     "NameTooShort",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-CREATE-VALIDATION-NAME-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.CreateActivityTemplateRequest {
				return &activityTemplatepb.CreateActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Name:            "AB",
						StageTemplateId: createSuccessResolver.MustGetString("validStageTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Activity template name must be at least 2 characters long",
			Assertions: func(t *testing.T, response *activityTemplatepb.CreateActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too short")
			},
		},
		{
			Name:     "NameTooLong",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-CREATE-VALIDATION-NAME-TOO-LONG-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.CreateActivityTemplateRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity_template", "ValidationError_NameTooLongGenerated")
				testutil.AssertTestCaseLoad(t, err, "ValidationError_NameTooLongGenerated")
				return &activityTemplatepb.CreateActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Name:            resolver.MustGetString("tooLongNameGenerated"),
						StageTemplateId: resolver.MustGetString("validStageTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Activity template name cannot exceed 100 characters",
			Assertions: func(t *testing.T, response *activityTemplatepb.CreateActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too long")
			},
		},
		{
			Name:     "DescriptionTooLong",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-CREATE-VALIDATION-DESCRIPTION-TOO-LONG-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.CreateActivityTemplateRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity_template", "ValidationError_DescriptionTooLongGenerated")
				testutil.AssertTestCaseLoad(t, err, "ValidationError_DescriptionTooLongGenerated")
				return &activityTemplatepb.CreateActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Name:            resolver.MustGetString("validName"),
						Description:     &[]string{resolver.MustGetString("tooLongDescriptionGenerated")}[0],
						StageTemplateId: resolver.MustGetString("validStageTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Activity template description cannot exceed 1000 characters",
			Assertions: func(t *testing.T, response *activityTemplatepb.CreateActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "description too long")
			},
		},
		{
			Name:     "InvalidDuration",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-CREATE-VALIDATION-INVALID-DURATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.CreateActivityTemplateRequest {
				return &activityTemplatepb.CreateActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Name:                     "Test Activity Template",
						StageTemplateId:          createSuccessResolver.MustGetString("validStageTemplateId"),
						EstimatedDurationMinutes: &[]int32{-1}[0], // Invalid negative duration
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Estimated minutes cannot be negative",
			Assertions: func(t *testing.T, response *activityTemplatepb.CreateActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid duration")
			},
		},
		{
			Name:     "InvalidActionType",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-CREATE-VALIDATION-INVALID-ACTION-TYPE-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.CreateActivityTemplateRequest {
				return &activityTemplatepb.CreateActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Name:            "Test Activity Template",
						StageTemplateId: createSuccessResolver.MustGetString("validStageTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Action type is required",
			Assertions: func(t *testing.T, response *activityTemplatepb.CreateActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid action type")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-CREATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.CreateActivityTemplateRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity_template", "CreateActivityTemplate_Success")
				testutil.AssertTestCaseLoad(t, err, "CreateActivityTemplate_Success")
				return &activityTemplatepb.CreateActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Name:            "Data Enrichment Test",
						StageTemplateId: resolver.MustGetString("validStageTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activityTemplatepb.CreateActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				createdActivityTemplate := response.Data[0]
				testutil.AssertNonEmptyString(t, createdActivityTemplate.Id, "generated ID")
				testutil.AssertFieldSet(t, createdActivityTemplate.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdActivityTemplate.DateCreatedString, "DateCreatedString")
				testutil.AssertTrue(t, createdActivityTemplate.Active, "Active")
				testutil.AssertNotNil(t, createdActivityTemplate.OrderIndex, "OrderIndex")
				testutil.AssertEqual(t, int32(1), *createdActivityTemplate.OrderIndex, "default order index")
				testutil.AssertNotNil(t, createdActivityTemplate.IsRequired, "IsRequired")
				testutil.AssertTrue(t, *createdActivityTemplate.IsRequired, "default is required")
			},
		},
		{
			Name:     "MinimalValidData",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-CREATE-VALIDATION-MINIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.CreateActivityTemplateRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity_template", "BoundaryTest_MinimalValid")
				testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MinimalValid")
				return &activityTemplatepb.CreateActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Name:            resolver.MustGetString("minValidName"),
						StageTemplateId: resolver.MustGetString("validStageTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activityTemplatepb.CreateActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdActivityTemplate := response.Data[0]
				resolver, _ := copyatestutil.LoadTestCaseFromBusinessType(t, testutil.GetTestBusinessType(), "activity_template", "BoundaryTest_MinimalValid")
				testutil.AssertStringEqual(t, resolver.MustGetString("minValidName"), createdActivityTemplate.Name, "activity template name")
			},
		},
		{
			Name:     "MaxValidData",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-CREATE-VALIDATION-MAXIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.CreateActivityTemplateRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity_template", "BoundaryTest_MaximalValid")
				testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MaximalValid")
				return &activityTemplatepb.CreateActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Name:                     resolver.MustGetString("maxValidNameExact100"),
						Description:              &[]string{resolver.MustGetString("maxValidDescriptionExact1000")}[0],
						StageTemplateId:          resolver.MustGetString("validStageTemplateId"),
						EstimatedDurationMinutes: &[]int32{480}[0], // 8 hours
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activityTemplatepb.CreateActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdActivityTemplate := response.Data[0]
				testutil.AssertEqual(t, 100, len(createdActivityTemplate.Name), "name length")
				if createdActivityTemplate.Description != nil {
					testutil.AssertEqual(t, 1000, len(*createdActivityTemplate.Description), "description length")
				}
				testutil.AssertEqual(t, int32(480), *createdActivityTemplate.EstimatedDurationMinutes, "estimated minutes")
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

func TestCreateActivityTemplateUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-CREATE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockActivityTemplateRepo := workflow.NewMockActivityTemplateRepository(businessType)
	mockStageTemplateRepo := workflow.NewMockStageTemplateRepository(businessType)

	repositories := CreateActivityTemplateRepositories{
		ActivityTemplate: mockActivityTemplateRepo,
		StageTemplate:    mockStageTemplateRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := CreateActivityTemplateServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	useCase := NewCreateActivityTemplateUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity_template", "CreateActivityTemplate_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateActivityTemplate_Success")

	req := &activityTemplatepb.CreateActivityTemplateRequest{
		Data: &activityTemplatepb.ActivityTemplate{
			Name:            resolver.MustGetString("newActivityTemplateName"),
			StageTemplateId: resolver.MustGetString("validStageTemplateId"),
		},
	}

	_, err = useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}

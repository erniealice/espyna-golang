//go:build mock_db && mock_auth

// Package activity_template provides table-driven tests for the activity template deletion use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestDeleteActivityTemplateUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-DELETE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-DELETE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-DELETE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-DELETE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-DELETE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-DELETE-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-DELETE-NOT-FOUND-v1.0: NonExistentId
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-DELETE-VALIDATION-MINIMAL-ID-v1.0: ValidMinimalId
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-DELETE-INTEGRATION-v1.0: RealisticEducationActivityTemplate
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-DELETE-TRANSACTION-FAILURE-v1.0: TransactionFailure
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-DELETE-VALIDATION-INPUT-v1.0: InputValidation
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-DELETE-DOMAIN-SPECIFIC-v1.0: EducationDomainSpecific
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-DELETE-BUSINESS-LOGIC-v1.0: BusinessRuleValidation
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-DELETE-CASCADING-v1.0: CascadingDeletes
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/activity_template.json
//   - Mock data: packages/copya/data/{businessType}/activity_template.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/activity_template.json

package activity_template

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	mockDb "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/workflow"
	activityTemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/activity_template"
)

// Type alias for delete activity template test cases
type DeleteActivityTemplateTestCase = testutil.GenericTestCase[*activityTemplatepb.DeleteActivityTemplateRequest, *activityTemplatepb.DeleteActivityTemplateResponse]

func createTestDeleteActivityTemplateUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *DeleteActivityTemplateUseCase {
	mockActivityTemplateRepo := workflow.NewMockActivityTemplateRepository(businessType)

	repositories := DeleteActivityTemplateRepositories{
		ActivityTemplate: mockActivityTemplateRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := DeleteActivityTemplateServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewDeleteActivityTemplateUseCase(repositories, services)
}

func TestDeleteActivityTemplateUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	activityTemplateCommonDataResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity_template", "ActivityTemplate_CommonData")
	testutil.AssertTestCaseLoad(t, err, "ActivityTemplate_CommonData")

	testCases := []DeleteActivityTemplateTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-DELETE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.DeleteActivityTemplateRequest {
				return &activityTemplatepb.DeleteActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Id: activityTemplateCommonDataResolver.MustGetString("primaryActivityTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activityTemplatepb.DeleteActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-DELETE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.DeleteActivityTemplateRequest {
				return &activityTemplatepb.DeleteActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Id: activityTemplateCommonDataResolver.MustGetString("secondaryActivityTemplateId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activityTemplatepb.DeleteActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-DELETE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.DeleteActivityTemplateRequest {
				return &activityTemplatepb.DeleteActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Id: activityTemplateCommonDataResolver.MustGetString("primaryActivityTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "activity_template.errors.authorization_failed",
			Assertions: func(t *testing.T, response *activityTemplatepb.DeleteActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-DELETE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.DeleteActivityTemplateRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "activity_template.validation.request_required",
			Assertions: func(t *testing.T, response *activityTemplatepb.DeleteActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-DELETE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.DeleteActivityTemplateRequest {
				return &activityTemplatepb.DeleteActivityTemplateRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "activity_template.validation.request_required",
			Assertions: func(t *testing.T, response *activityTemplatepb.DeleteActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-DELETE-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.DeleteActivityTemplateRequest {
				return &activityTemplatepb.DeleteActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Id: "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "activity_template.validation.id_required",
			Assertions: func(t *testing.T, response *activityTemplatepb.DeleteActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "NonExistentId",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-DELETE-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.DeleteActivityTemplateRequest {
				return &activityTemplatepb.DeleteActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Id: activityTemplateCommonDataResolver.MustGetString("nonExistentId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "activity_template.errors.not_found",
			ErrorTags:      map[string]any{"activityTemplateId": activityTemplateCommonDataResolver.MustGetString("nonExistentId")},
			Assertions: func(t *testing.T, response *activityTemplatepb.DeleteActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
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
			useCase := createTestDeleteActivityTemplateUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

func TestDeleteActivityTemplateUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-DELETE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	businessType := testutil.GetTestBusinessType()
	mockRepo := workflow.NewMockActivityTemplateRepository(businessType)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity_template", "ActivityTemplate_CommonData")
	testutil.AssertTestCaseLoad(t, err, "ActivityTemplate_CommonData")

	repositories := DeleteActivityTemplateRepositories{
		ActivityTemplate: mockRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := DeleteActivityTemplateServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewDeleteActivityTemplateUseCase(repositories, services)
	ctx := testutil.CreateTestContext()

	req := &activityTemplatepb.DeleteActivityTemplateRequest{
		Data: &activityTemplatepb.ActivityTemplate{
			Id: resolver.MustGetString("primaryActivityTemplateId"),
		},
	}

	_, err = useCase.Execute(ctx, req)

	testutil.AssertTransactionError(t, err)
	testutil.AssertTranslatedError(t, err, "activity_template.errors.transaction_failed", useCase.services.TranslationService, ctx)

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}

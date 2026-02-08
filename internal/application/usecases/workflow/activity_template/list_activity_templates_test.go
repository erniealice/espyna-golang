//go:build mock_db && mock_auth

// Package activity_template provides table-driven tests for the activity template listing use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestListActivityTemplatesUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-LIST-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-LIST-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-LIST-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-LIST-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-LIST-VALIDATION-EMPTY-v1.0: EmptyRequest
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-LIST-DOMAIN-SPECIFIC-v1.0: EducationDomainSpecific
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-LIST-STRUCTURE-VALIDATION-v1.0: ActivityTemplateStructureValidation
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-LIST-BUSINESS-LOGIC-v1.0: BusinessRulesValidation
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/activity_template.json
//   - Mock data: packages/copya/data/{businessType}/activity_template.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/activity_template.json

package activity_template

import (
	"context"
	"fmt"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/workflow"
	activityTemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/activity_template"
)

// Type alias for list activity templates test cases
type ListActivityTemplatesTestCase = testutil.GenericTestCase[*activityTemplatepb.ListActivityTemplatesRequest, *activityTemplatepb.ListActivityTemplatesResponse]

func createTestListActivityTemplatesUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *ListActivityTemplatesUseCase {
	mockRepo := workflow.NewMockActivityTemplateRepository(businessType)

	repositories := ListActivityTemplatesRepositories{
		ActivityTemplate: mockRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := ListActivityTemplatesServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewListActivityTemplatesUseCase(repositories, services)
}

func TestListActivityTemplatesUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity_template", "CreateActivityTemplate_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateActivityTemplate_Success")

	testCases := []ListActivityTemplatesTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-LIST-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.ListActivityTemplatesRequest {
				return &activityTemplatepb.ListActivityTemplatesRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activityTemplatepb.ListActivityTemplatesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertGreaterThan(t, len(response.Data), 0, "activity templates in response")
				expectedActivityTemplateCount := createSuccessResolver.GetIntWithDefault("expectedActivityTemplateCount", 5)
				testutil.AssertEqual(t, expectedActivityTemplateCount, len(response.Data), "activity template count")

				// Verify activity template structure
				activityTemplateIds := make(map[string]bool)
				for i, activityTemplate := range response.Data {
					testutil.AssertNonEmptyString(t, activityTemplate.Id, fmt.Sprintf("activity template %d ID", i))
					testutil.AssertNonEmptyString(t, activityTemplate.Name, fmt.Sprintf("activity template %d name", i))
					testutil.AssertFieldSet(t, activityTemplate.DateCreated, fmt.Sprintf("activity template %d DateCreated", i))
					activityTemplateIds[activityTemplate.Id] = true
				}

				// Check for expected activity template IDs
				expectedIds := createSuccessResolver.GetStringArrayWithDefault("expectedActivityTemplateIds", []string{})
				for _, expectedId := range expectedIds {
					testutil.AssertTrue(t, activityTemplateIds[expectedId], fmt.Sprintf("expected activity template ID '%s' in response", expectedId))
				}
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-LIST-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.ListActivityTemplatesRequest {
				return &activityTemplatepb.ListActivityTemplatesRequest{}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activityTemplatepb.ListActivityTemplatesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertGreaterThan(t, len(response.Data), 0, "activity templates with transaction")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-LIST-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.ListActivityTemplatesRequest {
				return &activityTemplatepb.ListActivityTemplatesRequest{}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "activity_template.errors.authorization_failed",
			Assertions: func(t *testing.T, response *activityTemplatepb.ListActivityTemplatesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-LIST-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.ListActivityTemplatesRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activityTemplatepb.ListActivityTemplatesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertGreaterThan(t, len(response.Data), 0, "activity templates with nil request")
			},
		},
		{
			Name:     "EmptyRequest",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-LIST-VALIDATION-EMPTY-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.ListActivityTemplatesRequest {
				return &activityTemplatepb.ListActivityTemplatesRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activityTemplatepb.ListActivityTemplatesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				expectedActivityTemplateCount := createSuccessResolver.GetIntWithDefault("expectedActivityTemplateCount", 5)
				testutil.AssertEqual(t, expectedActivityTemplateCount, len(response.Data), "activity template count")
			},
		},
		{
			Name:     "EducationDomainSpecific",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-LIST-DOMAIN-SPECIFIC-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.ListActivityTemplatesRequest {
				return &activityTemplatepb.ListActivityTemplatesRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activityTemplatepb.ListActivityTemplatesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")

				// Verify domain-specific activity templates are returned
				domainActivityTemplateNames := make(map[string]bool)
				for _, activityTemplate := range response.Data {
					domainActivityTemplateNames[activityTemplate.Name] = true
				}

				// Check for expected domain activity template names
				expectedNames := createSuccessResolver.GetStringArrayWithDefault("expectedActivityTemplateNames", []string{})
				for _, expectedName := range expectedNames {
					testutil.AssertTrue(t, domainActivityTemplateNames[expectedName], fmt.Sprintf("expected domain activity template '%s' in response", expectedName))
				}
			},
		},
		{
			Name:     "ActivityTemplateStructureValidation",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-LIST-STRUCTURE-VALIDATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.ListActivityTemplatesRequest {
				return &activityTemplatepb.ListActivityTemplatesRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activityTemplatepb.ListActivityTemplatesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")

				// Validate each activity template has required fields
				expectActive := createSuccessResolver.GetBoolWithDefault("expectActive", true)
				for i, activityTemplate := range response.Data {
					testutil.AssertNonEmptyString(t, activityTemplate.Id, fmt.Sprintf("activity template %d ID", i))
					testutil.AssertNonEmptyString(t, activityTemplate.Name, fmt.Sprintf("activity template %d name", i))
					if expectActive {
						testutil.AssertTrue(t, activityTemplate.Active, fmt.Sprintf("activity template %d active status", i))
					}
					testutil.AssertFieldSet(t, activityTemplate.DateCreated, fmt.Sprintf("activity template %d DateCreated", i))
					testutil.AssertFieldSet(t, activityTemplate.DateCreatedString, fmt.Sprintf("activity template %d DateCreatedString", i))
				}
			},
		},
		{
			Name:     "BusinessRulesValidation",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-LIST-BUSINESS-LOGIC-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.ListActivityTemplatesRequest {
				return &activityTemplatepb.ListActivityTemplatesRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activityTemplatepb.ListActivityTemplatesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")

				// Verify business rules are applied
				expectAllActive := createSuccessResolver.GetBoolWithDefault("expectAllActive", true)
				minNameLength := createSuccessResolver.GetIntWithDefault("minNameLength", 3)
				maxNameLength := createSuccessResolver.GetIntWithDefault("maxNameLength", 100)
				maxDescLength := createSuccessResolver.GetIntWithDefault("maxDescriptionLength", 1000)

				for i, activityTemplate := range response.Data {
					if expectAllActive {
						testutil.AssertTrue(t, activityTemplate.Active, fmt.Sprintf("activity template %d should be active", i))
					}
					testutil.AssertGreaterThan(t, len(activityTemplate.Name), minNameLength-1, fmt.Sprintf("activity template %d name minimum length", i))
					if len(activityTemplate.Name) > maxNameLength {
						testutil.AssertTrue(t, false, fmt.Sprintf("activity template %d name exceeds maximum length of %d characters: %s", i, maxNameLength, activityTemplate.Name))
					}
					if activityTemplate.Description != nil && len(*activityTemplate.Description) > maxDescLength {
						testutil.AssertTrue(t, false, fmt.Sprintf("activity template %d description exceeds maximum length of %d characters", i, maxDescLength))
					}
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
			useCase := createTestListActivityTemplatesUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

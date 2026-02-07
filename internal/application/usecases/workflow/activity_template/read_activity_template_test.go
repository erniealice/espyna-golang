//go:build mock_db && mock_auth

// Package activity_template provides table-driven tests for the activity template reading use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestReadActivityTemplateUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-READ-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-READ-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-READ-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-READ-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-READ-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-READ-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-READ-VALIDATION-ID-TOO-SHORT-v1.0: IdTooShort
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-READ-VALIDATION-ID-TOO-LONG-v1.0: IdTooLong
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-READ-VALIDATION-ID-INVALID-CHARS-v1.0: IdInvalidChars
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-READ-VALIDATION-WHITESPACE-ID-v1.0: WhitespaceId
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-READ-VALIDATION-MINIMAL-ID-v1.0: ValidMinimalId
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-READ-INTEGRATION-v1.0: RealisticEducationActivityTemplate
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-READ-DOMAIN-SPECIFIC-v1.0: EducationDomainSpecific
//
// Test data source: packages/copya/data_test/{businessType}/activity_template.json
// Mock data source: packages/copya/data/{businessType}/activity_template.json
// Translation source: packages/lyngua/translations/{languageCode}/{businessType}/activity_template.json

package activity_template

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/workflow"
	activityTemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/activity_template"
)

// Type alias for read activity template test cases
type ReadActivityTemplateTestCase = testutil.GenericTestCase[*activityTemplatepb.ReadActivityTemplateRequest, *activityTemplatepb.ReadActivityTemplateResponse]

func createTestReadActivityTemplateUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *ReadActivityTemplateUseCase {
	mockRepo := workflow.NewMockActivityTemplateRepository(businessType)

	repositories := ReadActivityTemplateRepositories{
		ActivityTemplate: mockRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := ReadActivityTemplateServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewReadActivityTemplateUseCase(repositories, services)
}

func TestReadActivityTemplateUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity_template", "CreateActivityTemplate_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateActivityTemplate_Success")

	testCases := []ReadActivityTemplateTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-READ-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.ReadActivityTemplateRequest {
				return &activityTemplatepb.ReadActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Id: createSuccessResolver.MustGetString("primaryActivityTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activityTemplatepb.ReadActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				t.Log("Activity template found successfully - mock data is properly configured")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-READ-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.ReadActivityTemplateRequest {
				return &activityTemplatepb.ReadActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Id: createSuccessResolver.MustGetString("secondaryActivityTemplateId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activityTemplatepb.ReadActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				t.Log("Activity template found successfully with transaction")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-READ-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.ReadActivityTemplateRequest {
				return &activityTemplatepb.ReadActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Id: createSuccessResolver.MustGetString("primaryActivityTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "Authorization failed for activity templates",
			Assertions: func(t *testing.T, response *activityTemplatepb.ReadActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-READ-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.ReadActivityTemplateRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Request is required for activity templates",
			Assertions: func(t *testing.T, response *activityTemplatepb.ReadActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-READ-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.ReadActivityTemplateRequest {
				return &activityTemplatepb.ReadActivityTemplateRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Activity template data is required",
			Assertions: func(t *testing.T, response *activityTemplatepb.ReadActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-READ-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.ReadActivityTemplateRequest {
				return &activityTemplatepb.ReadActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Id: "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Activity template ID is required for read operations",
			Assertions: func(t *testing.T, response *activityTemplatepb.ReadActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "IdTooShort",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-READ-VALIDATION-ID-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.ReadActivityTemplateRequest {
				return &activityTemplatepb.ReadActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Id: "ab", // Too short - keeping hardcoded for validation test consistency
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Activity template ID format is invalid",
			Assertions: func(t *testing.T, response *activityTemplatepb.ReadActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "ID too short")
			},
		},
		{
			Name:     "IdTooLong",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-READ-VALIDATION-ID-TOO-LONG-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.ReadActivityTemplateRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity_template", "ValidationError_IdTooLongGenerated")
				testutil.AssertTestCaseLoad(t, err, "ValidationError_IdTooLongGenerated")
				return &activityTemplatepb.ReadActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Id: resolver.MustGetString("tooLongIdGenerated"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Activity template ID format is invalid",
			Assertions: func(t *testing.T, response *activityTemplatepb.ReadActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "ID too long")
			},
		},
		{
			Name:     "IdInvalidChars",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-READ-VALIDATION-ID-INVALID-CHARS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.ReadActivityTemplateRequest {
				return &activityTemplatepb.ReadActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Id: "activity@template#123#invalid", // Invalid characters
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Activity template ID format is invalid",
			Assertions: func(t *testing.T, response *activityTemplatepb.ReadActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid characters")
			},
		},
		{
			Name:     "WhitespaceId",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-READ-VALIDATION-WHITESPACE-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.ReadActivityTemplateRequest {
				return &activityTemplatepb.ReadActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Id: "   ", // Whitespace only
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Activity template ID format is invalid",
			Assertions: func(t *testing.T, response *activityTemplatepb.ReadActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "whitespace only")
			},
		},
		{
			Name:     "ValidMinimalId",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-READ-VALIDATION-MINIMAL-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.ReadActivityTemplateRequest {
				return &activityTemplatepb.ReadActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Id: "abc", // Minimal valid ID (3 characters) - but activity template doesn't exist
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Activity template not found",
			Assertions: func(t *testing.T, response *activityTemplatepb.ReadActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
			},
		},
		{
			Name:     "RealisticActivityTemplate",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-READ-INTEGRATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.ReadActivityTemplateRequest {
				return &activityTemplatepb.ReadActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Id: createSuccessResolver.MustGetString("thirdActivityTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activityTemplatepb.ReadActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				t.Log("Activity template found successfully - realistic activity template test")
				if response != nil && len(response.Data) > 0 {
					activityTemplate := response.Data[0]
					testutil.AssertNonEmptyString(t, activityTemplate.Id, "activity template ID")
					testutil.AssertNonEmptyString(t, activityTemplate.Name, "activity template name")
					testutil.AssertFieldSet(t, activityTemplate.DateCreated, "DateCreated")
					testutil.AssertFieldSet(t, activityTemplate.DateCreatedString, "DateCreatedString")
				}
			},
		},
		{
			Name:     "EducationDomainSpecific",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-READ-DOMAIN-SPECIFIC-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.ReadActivityTemplateRequest {
				return &activityTemplatepb.ReadActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Id: createSuccessResolver.MustGetString("educationActivityTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activityTemplatepb.ReadActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				t.Log("Education domain specific activity template found successfully")
				if response != nil && len(response.Data) > 0 {
					activityTemplate := response.Data[0]
					testutil.AssertStringEqual(t, "Education Activity Template", activityTemplate.Name, "education activity template name")
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
			useCase := createTestReadActivityTemplateUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

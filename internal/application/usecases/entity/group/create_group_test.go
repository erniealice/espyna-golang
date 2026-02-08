//go:build mock_db && mock_auth

// Package group provides test cases for group creation use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestCreateGroupUseCase_Execute_Success: ESPYNA-TEST-ENTITY-GROUP-SUCCESS-v1.0 Basic successful group creation
//   - TestCreateGroupUseCase_Execute_ValidationErrors: ESPYNA-TEST-ENTITY-GROUP-VALIDATION-v1.0 Comprehensive validation error scenarios
//   - TestCreateGroupUseCase_DataEnrichment: ESPYNA-TEST-ENTITY-GROUP-ENRICHMENT-v1.0 Auto-generated fields verification
package group

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	grouppb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/group"
)

// Type alias for create group test cases
type CreateGroupTestCase = testutil.GenericTestCase[*grouppb.CreateGroupRequest, *grouppb.CreateGroupResponse]

func createTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreateGroupUseCase {
	repositories := CreateGroupRepositories{
		Group: entity.NewMockGroupRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreateGroupServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreateGroupUseCase(repositories, services)
}

func TestCreateGroupUseCase_Execute_TableDriven(t *testing.T) {
	testCases := []CreateGroupTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-ENTITY-GROUP-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *grouppb.CreateGroupRequest {
				return &grouppb.CreateGroupRequest{
					Data: &grouppb.Group{
						Name: "Grade 10 Students",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *grouppb.CreateGroupResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdGroup := response.Data[0]
				testutil.AssertStringEqual(t, "Grade 10 Students", createdGroup.Name, "group name")
				testutil.AssertNonEmptyString(t, createdGroup.Id, "group ID")
				testutil.AssertTrue(t, createdGroup.Active, "group active")
				testutil.AssertFieldSet(t, createdGroup.DateCreated, "DateCreated")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-ENTITY-GROUP-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *grouppb.CreateGroupRequest {
				return &grouppb.CreateGroupRequest{
					Data: &grouppb.Group{
						Name: "Transaction Group",
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *grouppb.CreateGroupResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdGroup := response.Data[0]
				testutil.AssertStringEqual(t, "Transaction Group", createdGroup.Name, "group name")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-ENTITY-GROUP-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *grouppb.CreateGroupRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "group.validation.request_required",
			Assertions: func(t *testing.T, response *grouppb.CreateGroupResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-ENTITY-GROUP-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *grouppb.CreateGroupRequest {
				return &grouppb.CreateGroupRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "group.validation.data_required",
			Assertions: func(t *testing.T, response *grouppb.CreateGroupResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyName",
			TestCode: "ESPYNA-TEST-ENTITY-GROUP-CREATE-VALIDATION-EMPTY-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *grouppb.CreateGroupRequest {
				return &grouppb.CreateGroupRequest{
					Data: &grouppb.Group{
						Name: "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "group.validation.name_required",
			Assertions: func(t *testing.T, response *grouppb.CreateGroupResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty name")
			},
		},
		{
			Name:     "NameTooShort",
			TestCode: "ESPYNA-TEST-ENTITY-GROUP-CREATE-VALIDATION-NAME-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *grouppb.CreateGroupRequest {
				return &grouppb.CreateGroupRequest{
					Data: &grouppb.Group{
						Name: "A",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "group.validation.name_too_short",
			Assertions: func(t *testing.T, response *grouppb.CreateGroupResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too short")
			},
		},
		{
			Name:     "ValidMinimumName",
			TestCode: "ESPYNA-TEST-ENTITY-GROUP-CREATE-VALIDATION-MINIMUM-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *grouppb.CreateGroupRequest {
				return &grouppb.CreateGroupRequest{
					Data: &grouppb.Group{
						Name: "AB", // Minimum valid length
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *grouppb.CreateGroupResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdGroup := response.Data[0]
				testutil.AssertStringEqual(t, "AB", createdGroup.Name, "group name")
			},
		},
		{
			Name:     "LongName",
			TestCode: "ESPYNA-TEST-ENTITY-GROUP-CREATE-LONG-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *grouppb.CreateGroupRequest {
				return &grouppb.CreateGroupRequest{
					Data: &grouppb.Group{
						Name: "Grade 12 Advanced Mathematics Students - Section A",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *grouppb.CreateGroupResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdGroup := response.Data[0]
				testutil.AssertStringEqual(t, "Grade 12 Advanced Mathematics Students - Section A", createdGroup.Name, "group name")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-ENTITY-GROUP-CREATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *grouppb.CreateGroupRequest {
				return &grouppb.CreateGroupRequest{
					Data: &grouppb.Group{
						Name: "Enriched Group",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *grouppb.CreateGroupResponse, err error, useCase interface{}, ctx context.Context) {
				createdGroup := response.Data[0]
				testutil.AssertNonEmptyString(t, createdGroup.Id, "generated ID")
				testutil.AssertTrue(t, createdGroup.Active, "Active")
				testutil.AssertFieldSet(t, createdGroup.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdGroup.DateCreatedString, "DateCreatedString")
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

//go:build mock_db && mock_auth

// Package staff provides table-driven tests for the staff read use case.
//
// The tests cover various scenarios, including success, not found errors,
// validation errors, and nil request handling. Each test case is defined
// in a table with a specific test code, request setup, and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestReadStaffUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-ENTITY-STAFF-READ-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-ENTITY-STAFF-READ-NOT-FOUND-v1.0: NotFound
//   - ESPYNA-TEST-ENTITY-STAFF-READ-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-ENTITY-STAFF-READ-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-ENTITY-STAFF-READ-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-ENTITY-STAFF-READ-VERIFICATION-v1.0: VerifyDetails
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/staff.json
//   - Mock data: packages/copya/data/{businessType}/staff.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/staff.json
package staff

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	staffpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/staff"
)

// Type alias for read staff test cases
type ReadStaffTestCase = testutil.GenericTestCase[*staffpb.ReadStaffRequest, *staffpb.ReadStaffResponse]

func createTestReadStaffUseCase(businessType string) *ReadStaffUseCase {
	repositories := ReadStaffRepositories{
		Staff: entity.NewMockStaffRepository(businessType),
	}
	services := testutil.CreateStandardServices(false, false)
	return NewReadStaffUseCase(repositories, ReadStaffServices{
		TranslationService: services.TranslationService,
	})
}

func TestReadStaffUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	readSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "staff", "ReadStaff_Success")
	testutil.AssertTestCaseLoad(t, err, "ReadStaff_Success")

	notFoundResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "staff", "NotFound_ReadStaff")
	testutil.AssertTestCaseLoad(t, err, "NotFound_ReadStaff")

	validationErrorEmptyIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "staff", "ValidationError_EmptyId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyId")

	testCases := []ReadStaffTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-READ-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.ReadStaffRequest {
				return &staffpb.ReadStaffRequest{
					Data: &staffpb.Staff{
						Id: readSuccessResolver.MustGetString("targetStaffId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *staffpb.ReadStaffResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				readStaff := response.Data[0]
				testutil.AssertStringEqual(t, readSuccessResolver.MustGetString("targetStaffId"), readStaff.Id, "staff ID")
				testutil.AssertStringEqual(t, readSuccessResolver.MustGetString("expectedEmail"), readStaff.User.EmailAddress, "staff email")
				testutil.AssertNonEmptyString(t, readStaff.User.Id, "user ID")
				testutil.AssertFieldSet(t, readStaff.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, readStaff.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "VerifyDetails",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-READ-VERIFICATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.ReadStaffRequest {
				return &staffpb.ReadStaffRequest{
					Data: &staffpb.Staff{
						Id: readSuccessResolver.MustGetString("secondTargetStaffId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *staffpb.ReadStaffResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				readStaff := response.Data[0]
				testutil.AssertStringEqual(t, readSuccessResolver.MustGetString("secondTargetStaffId"), readStaff.Id, "staff ID")
				testutil.AssertStringEqual(t, readSuccessResolver.MustGetString("secondExpectedEmail"), readStaff.User.EmailAddress, "staff email")
				testutil.AssertTrue(t, readStaff.Active, "staff active status")
			},
		},
		{
			Name:     "NotFound",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-READ-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.ReadStaffRequest {
				return &staffpb.ReadStaffRequest{
					Data: &staffpb.Staff{
						Id: notFoundResolver.MustGetString("nonExistentStaffId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			Assertions: func(t *testing.T, response *staffpb.ReadStaffResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-READ-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.ReadStaffRequest {
				return &staffpb.ReadStaffRequest{
					Data: &staffpb.Staff{
						Id: validationErrorEmptyIdResolver.MustGetString("emptyId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "staff.validation.id_required",
			Assertions: func(t *testing.T, response *staffpb.ReadStaffResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-READ-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.ReadStaffRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "staff.validation.request_required",
			Assertions: func(t *testing.T, response *staffpb.ReadStaffResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-ENTITY-STAFF-READ-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *staffpb.ReadStaffRequest {
				return &staffpb.ReadStaffRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "staff.validation.request_required",
			Assertions: func(t *testing.T, response *staffpb.ReadStaffResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
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
			useCase := createTestReadStaffUseCase(businessType)

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

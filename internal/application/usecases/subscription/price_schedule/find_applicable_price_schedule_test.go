//go:build mock_db && mock_auth

// Package price_schedule provides table-driven tests for the find applicable price schedule use case.
//
// The tests cover: exact match, open-ended (no date_end), inactive row excluded,
// no match returns found=false with no error, and multiple overlapping matches
// where the latest date_start wins.
//
// Usage:
//   - Run: go test -tags="mock_db,mock_auth" ./internal/application/usecases/subscription/price_schedule/...
//
// Test Codes:
//   - ESPYNA-TEST-SUBSCRIPTION-PRICESCHEDULE-FINDAPPLICABLE-EXACT-MATCH-v1.0
//   - ESPYNA-TEST-SUBSCRIPTION-PRICESCHEDULE-FINDAPPLICABLE-OPEN-ENDED-v1.0
//   - ESPYNA-TEST-SUBSCRIPTION-PRICESCHEDULE-FINDAPPLICABLE-INACTIVE-EXCLUDED-v1.0
//   - ESPYNA-TEST-SUBSCRIPTION-PRICESCHEDULE-FINDAPPLICABLE-NO-MATCH-v1.0
//   - ESPYNA-TEST-SUBSCRIPTION-PRICESCHEDULE-FINDAPPLICABLE-LATEST-DATE-START-WINS-v1.0
//   - ESPYNA-TEST-SUBSCRIPTION-PRICESCHEDULE-FINDAPPLICABLE-NIL-REQUEST-v1.0
//   - ESPYNA-TEST-SUBSCRIPTION-PRICESCHEDULE-FINDAPPLICABLE-MISSING-LOCATION-v1.0
//   - ESPYNA-TEST-SUBSCRIPTION-PRICESCHEDULE-FINDAPPLICABLE-MISSING-DATE-v1.0
package price_schedule

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	mocksubscription "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/subscription"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
)

// FindApplicablePriceScheduleTestCase is a type alias for the generic test case
type FindApplicablePriceScheduleTestCase = testutil.GenericTestCase[*priceschedulepb.FindApplicablePriceScheduleRequest, *priceschedulepb.FindApplicablePriceScheduleResponse]

// buildFindApplicableUseCase constructs the use case with a pre-seeded in-memory repository
func buildFindApplicableUseCase(
	data map[string]*priceschedulepb.PriceSchedule,
	supportsTransaction bool,
	shouldAuthorize bool,
) *FindApplicablePriceScheduleUseCase {
	repo := mocksubscription.NewPriceScheduleRepository(data)

	repos := FindApplicablePriceScheduleRepositories{
		PriceSchedule: repo,
	}

	svc := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := FindApplicablePriceScheduleServices{
		AuthorizationService: svc.AuthorizationService,
		TransactionService:   svc.TransactionService,
		TranslationService:   svc.TranslationService,
	}

	return NewFindApplicablePriceScheduleUseCase(repos, services)
}

// ptr returns a pointer to the given string value — helper for optional proto fields
func ptr(s string) *string { return &s }

func TestFindApplicablePriceScheduleUseCase_Execute(t *testing.T) {
	// Shared fixture data — different test cases pick the rows they need
	activeExact := &priceschedulepb.PriceSchedule{
		Id:         "ps-exact",
		Name:       "Exact Match Schedule",
		Active:     true,
		DateStart:  "2025-01-01",
		DateEnd:    ptr("2025-12-31"),
		LocationId: ptr("loc-1"),
	}
	activeOpenEnded := &priceschedulepb.PriceSchedule{
		Id:         "ps-open",
		Name:       "Open Ended Schedule",
		Active:     true,
		DateStart:  "2024-06-01",
		DateEnd:    nil, // open-ended
		LocationId: ptr("loc-2"),
	}
	inactiveRow := &priceschedulepb.PriceSchedule{
		Id:         "ps-inactive",
		Name:       "Inactive Schedule",
		Active:     false,
		DateStart:  "2025-01-01",
		DateEnd:    ptr("2025-12-31"),
		LocationId: ptr("loc-3"),
	}
	olderRow := &priceschedulepb.PriceSchedule{
		Id:         "ps-older",
		Name:       "Older Schedule",
		Active:     true,
		DateStart:  "2025-01-01",
		DateEnd:    ptr("2025-12-31"),
		LocationId: ptr("loc-4"),
	}
	newerRow := &priceschedulepb.PriceSchedule{
		Id:         "ps-newer",
		Name:       "Newer Schedule",
		Active:     true,
		DateStart:  "2025-06-01",
		DateEnd:    ptr("2025-12-31"),
		LocationId: ptr("loc-4"),
	}

	testCases := []FindApplicablePriceScheduleTestCase{
		{
			Name:     "ExactMatch",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PRICESCHEDULE-FINDAPPLICABLE-EXACT-MATCH-v1.0",
			SetupRequest: func(t *testing.T, _ string) *priceschedulepb.FindApplicablePriceScheduleRequest {
				return &priceschedulepb.FindApplicablePriceScheduleRequest{
					LocationId: "loc-1",
					Date:       "2025-07-15",
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, resp *priceschedulepb.FindApplicablePriceScheduleResponse, err error, _ interface{}, _ context.Context) {
				testutil.AssertNoError(t, err)
				testutil.AssertTrue(t, resp.Found, "found should be true")
				testutil.AssertTrue(t, resp.Success, "success should be true")
				testutil.AssertNotNil(t, resp.PriceSchedule, "price schedule")
				testutil.AssertEqual(t, "ps-exact", resp.PriceSchedule.Id, "price schedule ID")
			},
		},
		{
			Name:     "OpenEnded",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PRICESCHEDULE-FINDAPPLICABLE-OPEN-ENDED-v1.0",
			SetupRequest: func(t *testing.T, _ string) *priceschedulepb.FindApplicablePriceScheduleRequest {
				return &priceschedulepb.FindApplicablePriceScheduleRequest{
					LocationId: "loc-2",
					Date:       "2099-01-01", // far future — open-ended schedule must still match
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, resp *priceschedulepb.FindApplicablePriceScheduleResponse, err error, _ interface{}, _ context.Context) {
				testutil.AssertNoError(t, err)
				testutil.AssertTrue(t, resp.Found, "found should be true for open-ended schedule")
				testutil.AssertNotNil(t, resp.PriceSchedule, "price schedule")
				testutil.AssertEqual(t, "ps-open", resp.PriceSchedule.Id, "price schedule ID")
			},
		},
		{
			Name:     "InactiveRowExcluded",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PRICESCHEDULE-FINDAPPLICABLE-INACTIVE-EXCLUDED-v1.0",
			SetupRequest: func(t *testing.T, _ string) *priceschedulepb.FindApplicablePriceScheduleRequest {
				return &priceschedulepb.FindApplicablePriceScheduleRequest{
					LocationId: "loc-3",
					Date:       "2025-07-15",
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, resp *priceschedulepb.FindApplicablePriceScheduleResponse, err error, _ interface{}, _ context.Context) {
				testutil.AssertNoError(t, err)
				testutil.AssertTrue(t, !resp.Found, "found should be false — inactive rows must be excluded")
				testutil.AssertTrue(t, resp.Success, "success should be true even when not found")
			},
		},
		{
			Name:     "NoMatchReturnsFalseNotError",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PRICESCHEDULE-FINDAPPLICABLE-NO-MATCH-v1.0",
			SetupRequest: func(t *testing.T, _ string) *priceschedulepb.FindApplicablePriceScheduleRequest {
				return &priceschedulepb.FindApplicablePriceScheduleRequest{
					LocationId: "loc-unknown",
					Date:       "2025-07-15",
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, resp *priceschedulepb.FindApplicablePriceScheduleResponse, err error, _ interface{}, _ context.Context) {
				testutil.AssertNoError(t, err)
				testutil.AssertTrue(t, !resp.Found, "found should be false when no price schedule matches")
				testutil.AssertTrue(t, resp.Success, "success must be true — no match is not an error")
			},
		},
		{
			Name:     "MultipleOverlappingMatchesLatestDateStartWins",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PRICESCHEDULE-FINDAPPLICABLE-LATEST-DATE-START-WINS-v1.0",
			SetupRequest: func(t *testing.T, _ string) *priceschedulepb.FindApplicablePriceScheduleRequest {
				return &priceschedulepb.FindApplicablePriceScheduleRequest{
					LocationId: "loc-4",
					Date:       "2025-09-01", // both ps-older and ps-newer cover this date
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, resp *priceschedulepb.FindApplicablePriceScheduleResponse, err error, _ interface{}, _ context.Context) {
				testutil.AssertNoError(t, err)
				testutil.AssertTrue(t, resp.Found, "found should be true")
				testutil.AssertNotNil(t, resp.PriceSchedule, "price schedule")
				testutil.AssertEqual(t, "ps-newer", resp.PriceSchedule.Id, "latest date_start (2025-06-01) wins over older (2025-01-01)")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PRICESCHEDULE-FINDAPPLICABLE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, _ string) *priceschedulepb.FindApplicablePriceScheduleRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			Assertions: func(t *testing.T, resp *priceschedulepb.FindApplicablePriceScheduleResponse, err error, _ interface{}, _ context.Context) {
				testutil.AssertError(t, err)
			},
		},
		{
			Name:     "MissingLocationID",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PRICESCHEDULE-FINDAPPLICABLE-MISSING-LOCATION-v1.0",
			SetupRequest: func(t *testing.T, _ string) *priceschedulepb.FindApplicablePriceScheduleRequest {
				return &priceschedulepb.FindApplicablePriceScheduleRequest{
					LocationId: "",
					Date:       "2025-07-15",
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			Assertions: func(t *testing.T, resp *priceschedulepb.FindApplicablePriceScheduleResponse, err error, _ interface{}, _ context.Context) {
				testutil.AssertError(t, err)
			},
		},
		{
			Name:     "MissingDate",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PRICESCHEDULE-FINDAPPLICABLE-MISSING-DATE-v1.0",
			SetupRequest: func(t *testing.T, _ string) *priceschedulepb.FindApplicablePriceScheduleRequest {
				return &priceschedulepb.FindApplicablePriceScheduleRequest{
					LocationId: "loc-1",
					Date:       "",
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			Assertions: func(t *testing.T, resp *priceschedulepb.FindApplicablePriceScheduleResponse, err error, _ interface{}, _ context.Context) {
				testutil.AssertError(t, err)
			},
		},
	}

	// Seed data shared across all test cases
	seedData := map[string]*priceschedulepb.PriceSchedule{
		activeExact.Id:     activeExact,
		activeOpenEnded.Id: activeOpenEnded,
		inactiveRow.Id:     inactiveRow,
		olderRow.Id:        olderRow,
		newerRow.Id:        newerRow,
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			testutil.SetTestCode(t, tc.TestCode)
			testutil.LogTestExecution(t, tc.TestCode, tc.Name, tc.ExpectSuccess)

			ctx := testutil.CreateTestContext()
			useCase := buildFindApplicableUseCase(seedData, tc.UseTransaction, tc.UseAuth)

			req := tc.SetupRequest(t, testutil.GetTestBusinessType())
			resp, err := useCase.Execute(ctx, req)

			actualSuccess := err == nil && tc.ExpectSuccess

			if tc.ExpectSuccess {
				testutil.AssertNoError(t, err)
				testutil.AssertNotNil(t, resp, "response")
			} else {
				testutil.AssertError(t, err)
			}

			if tc.Assertions != nil {
				tc.Assertions(t, resp, err, useCase, ctx)
			}

			testutil.LogTestResult(t, tc.TestCode, tc.Name, actualSuccess, err)
		})
	}
}

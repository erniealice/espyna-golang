package revenue

// Tests for ListRevenueRunCandidates and GenerateRevenueRun.
//
// TODO: Lift the full v1 test matrix from:
//   docs/plan/20260503-subscription-invoice-run/progress.md § Phase 1 Tests
//
// Pending test coverage:
//   - Long-backlog stress: sub 2022-01-01 monthly AsOfDate 2026-12-31 → 60
//     periods enumerate without OOM within reasonable time
//   - Cross-tenant guard rejects mismatched client_id / workspace_id
//   - Integration: idempotency — same selection twice → second is SKIPPED
//     with error_code=period_already_invoiced
//   - Integration: per-selection short tx rollback on attempt-INSERT failure
//   - Status per D15: errored_count==0 → COMPLETE; >0 → FAILED; skipped only
//     does not flip status to FAILED

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"

	revenuerunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_run"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ---------------------------------------------------------------------------
// enumeratePeriods unit tests
// ---------------------------------------------------------------------------

func TestEnumeratePeriods_EmptyWhenNoStartDate(t *testing.T) {
	windows := enumeratePeriods(
		newTestSub("", ""),
		newTestPlanCycle("month", 1),
		time.Now(),
		time.UTC,
	)
	if len(windows) != 0 {
		t.Errorf("expected empty windows for sub with no start date, got %d", len(windows))
	}
}

func TestEnumeratePeriods_EmptyWhenNoCycle(t *testing.T) {
	windows := enumeratePeriods(
		newTestSub("2026-01-01", ""),
		newTestPlanCycle("", 0), // no cycle
		mustParseDate("2026-03-31"),
		time.UTC,
	)
	if len(windows) != 0 {
		t.Errorf("expected empty windows when plan has no cycle, got %d", len(windows))
	}
}

func TestEnumeratePeriods_OpenEndedMonthly3Periods(t *testing.T) {
	// Sub started 2026-01-01, monthly cycle, AsOfDate 2026-03-31
	// Expected: Jan 1–31, Feb 1–28, Mar 1–31
	windows := enumeratePeriods(
		newTestSub("2026-01-01", ""),
		newTestPlanCycle("month", 1),
		mustParseDate("2026-03-31"),
		time.UTC,
	)
	if len(windows) != 3 {
		t.Fatalf("expected 3 periods, got %d: %+v", len(windows), windows)
	}
	assertPeriod(t, windows[0], "2026-01-01", "2026-01-31")
	assertPeriod(t, windows[1], "2026-02-01", "2026-02-28")
	assertPeriod(t, windows[2], "2026-03-01", "2026-03-31")
}

func TestEnumeratePeriods_AsOfDateBeforeStart(t *testing.T) {
	windows := enumeratePeriods(
		newTestSub("2026-06-01", ""),
		newTestPlanCycle("month", 1),
		mustParseDate("2026-05-15"),
		time.UTC,
	)
	if len(windows) != 0 {
		t.Errorf("expected empty windows when AsOfDate is before sub start, got %d", len(windows))
	}
}

func TestEnumeratePeriods_WeeklyCycle(t *testing.T) {
	// Sub started 2026-01-01, weekly cycle, AsOfDate 2026-01-21
	// Expected 3 weeks: Jan 1–7, Jan 8–14, Jan 15–21
	windows := enumeratePeriods(
		newTestSub("2026-01-01", ""),
		newTestPlanCycle("week", 1),
		mustParseDate("2026-01-21"),
		time.UTC,
	)
	if len(windows) != 3 {
		t.Errorf("expected 3 weekly periods, got %d: %+v", len(windows), windows)
	}
}

func TestEnumeratePeriods_DayCycle(t *testing.T) {
	// Sub started 2026-01-01, 7-day cycle, AsOfDate 2026-01-21 → 3 periods
	windows := enumeratePeriods(
		newTestSub("2026-01-01", ""),
		newTestPlanCycle("day", 7),
		mustParseDate("2026-01-21"),
		time.UTC,
	)
	if len(windows) != 3 {
		t.Errorf("expected 3 daily(7) periods, got %d: %+v", len(windows), windows)
	}
}

func TestEnumeratePeriods_YearlyCycle(t *testing.T) {
	// Sub started 2024-01-01, yearly cycle, AsOfDate 2026-06-15 → 2 full years
	// 2024-01-01..2024-12-31, 2025-01-01..2025-12-31; 2026 partial capped at 2026-06-15
	windows := enumeratePeriods(
		newTestSub("2024-01-01", ""),
		newTestPlanCycle("year", 1),
		mustParseDate("2026-06-15"),
		time.UTC,
	)
	if len(windows) != 3 {
		t.Errorf("expected 3 yearly periods (2 full + 1 partial), got %d: %+v", len(windows), windows)
	}
}

func TestEnumeratePeriods_DeprecatedDurationFallback(t *testing.T) {
	// Plan with no billing_cycle_value/unit but has duration_value/unit
	windows := enumeratePeriods(
		newTestSub("2026-01-01", ""),
		newTestPlanDuration("month", 1),
		mustParseDate("2026-03-31"),
		time.UTC,
	)
	if len(windows) != 3 {
		t.Errorf("expected 3 periods via deprecated duration fallback, got %d", len(windows))
	}
}

func TestEnumeratePeriods_EndDatedSubCapsLastPeriod(t *testing.T) {
	// Sub ended 2026-02-15 mid-cycle; last period should end at Feb 15
	windows := enumeratePeriods(
		newTestSub("2026-01-01", "2026-02-15"),
		newTestPlanCycle("month", 1),
		mustParseDate("2026-03-31"),
		time.UTC,
	)
	if len(windows) != 2 {
		t.Fatalf("expected 2 periods, got %d: %+v", len(windows), windows)
	}
	// Second period capped at 2026-02-15
	if windows[1].End != "2026-02-15" {
		t.Errorf("expected end-dated period to cap at 2026-02-15, got %s", windows[1].End)
	}
}

// Regression: subscription start typed as 2026-01-01 in Asia/Manila is stored
// as 2025-12-31T16:00:00Z. With loc=Manila, the first period must start
// 2026-01-01. With loc=UTC (the silent fallback path), the same instant
// projects to 2025-12-31 — that's the off-by-one bug Surface C/A hit when
// scope.WorkspaceID was not set.
func TestEnumeratePeriods_ManilaTimezoneStart(t *testing.T) {
	manila, err := time.LoadLocation("Asia/Manila")
	if err != nil {
		t.Fatalf("LoadLocation Asia/Manila: %v", err)
	}
	// 2026-01-01 00:00 Manila = 2025-12-31 16:00 UTC
	manilaMidnight := time.Date(2026, 1, 1, 0, 0, 0, 0, manila).UTC()
	sub := &subscriptionpb.Subscription{
		Id:            "sub-manila",
		DateTimeStart: timestamppb.New(manilaMidnight),
	}
	plan := newTestPlanCycle("month", 1)
	asOf := time.Date(2026, 3, 31, 0, 0, 0, 0, manila)

	windowsManila := enumeratePeriods(sub, plan, asOf, manila)
	if len(windowsManila) != 3 {
		t.Fatalf("Manila: expected 3 periods, got %d: %+v", len(windowsManila), windowsManila)
	}
	assertPeriod(t, windowsManila[0], "2026-01-01", "2026-01-31")

	// Demonstrate the bug: same input with loc=UTC enumerates from 2025-12-31.
	windowsUTC := enumeratePeriods(sub, plan, asOf, time.UTC)
	if len(windowsUTC) == 0 || windowsUTC[0].Start != "2025-12-31" {
		t.Errorf("UTC fallback path: expected first period start 2025-12-31 (the bug), got %+v", windowsUTC)
	}
}

// ---------------------------------------------------------------------------
// GenerateRevenueRun stub tests
// ---------------------------------------------------------------------------

func TestGenerateRevenueRun_FilterTokenStubReturnsNotImplemented(t *testing.T) {
	uc := &GenerateRevenueRunUseCase{
		services: GenerateRevenueRunServices{
			Authorizer:  ports.NewNoOpAuthorizer(),
			Translator:  ports.NewNoOpTranslator(),
			IDGenerator: ports.NewNoOpIDGenerator(),
		},
	}
	ws := "ws-1"
	tok := "some-token"
	_, err := uc.Execute(
		context.Background(),
		&revenuerunpb.GenerateRevenueRunRequest{
			Scope:      &revenuerunpb.RevenueRunScope{WorkspaceId: &ws},
			Selections: &revenuerunpb.RevenueRunSelections{FilterToken: &tok},
		},
	)
	if err == nil {
		t.Fatal("expected error for FilterToken, got nil")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("expected 'not implemented' in error, got: %s", err.Error())
	}
}

func TestGenerateRevenueRun_EmptySelectionsReturnsError(t *testing.T) {
	uc := &GenerateRevenueRunUseCase{
		services: GenerateRevenueRunServices{
			Authorizer:  ports.NewNoOpAuthorizer(),
			Translator:  ports.NewNoOpTranslator(),
			IDGenerator: ports.NewNoOpIDGenerator(),
		},
	}
	ws := "ws-1"
	_, err := uc.Execute(
		context.Background(),
		&revenuerunpb.GenerateRevenueRunRequest{
			Scope:      &revenuerunpb.RevenueRunScope{WorkspaceId: &ws},
			Selections: &revenuerunpb.RevenueRunSelections{},
		},
	)
	if err == nil {
		t.Fatal("expected error for empty selections, got nil")
	}
}

// ---------------------------------------------------------------------------
// isIdempotencyConflict unit test
// ---------------------------------------------------------------------------

func TestIsIdempotencyConflict(t *testing.T) {
	cases := []struct {
		msg  string
		want bool
	}{
		{"period_already_invoiced: unique constraint", true},
		{"some other error", false},
	}
	for _, c := range cases {
		err := &runTestError{c.msg}
		got := isIdempotencyConflict(err)
		if got != c.want {
			t.Errorf("isIdempotencyConflict(%q) = %v, want %v", c.msg, got, c.want)
		}
	}
	// nil error is not a conflict
	if isIdempotencyConflict(nil) {
		t.Error("isIdempotencyConflict(nil) should be false")
	}
}

// ---------------------------------------------------------------------------
// buildPeriodMarker round-trip test
// ---------------------------------------------------------------------------

func TestBuildPeriodMarker_RoundTrip(t *testing.T) {
	marker := buildPeriodMarker("2026-01-01", "2026-01-31")
	if marker == "" {
		t.Error("expected non-empty marker")
	}
	// Re-deriving with same inputs must produce identical string (idempotency anchor)
	if buildPeriodMarker("2026-01-01", "2026-01-31") != marker {
		t.Error("buildPeriodMarker is not deterministic")
	}
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

type runTestError struct{ msg string }

func (e *runTestError) Error() string { return e.msg }

// newTestSub builds a minimal Subscription for testing.
// Pass empty strings for start/end to get nil timestamps.
func newTestSub(startDate, endDate string) *subscriptionpb.Subscription {
	sub := &subscriptionpb.Subscription{Id: "sub-test"}
	if startDate != "" {
		t := mustParseDate(startDate)
		ts := timestamppb.New(t)
		sub.DateTimeStart = ts
	}
	if endDate != "" {
		t := mustParseDate(endDate)
		ts := timestamppb.New(t)
		sub.DateTimeEnd = ts
	}
	return sub
}

// newTestPlanCycle builds a minimal PricePlan with billing_cycle_value/unit.
func newTestPlanCycle(unit string, value int) *priceplanpb.PricePlan {
	pp := &priceplanpb.PricePlan{
		Id:          "plan-test",
		BillingKind: priceplanpb.BillingKind_BILLING_KIND_RECURRING,
	}
	if value > 0 && unit != "" {
		v := int32(value)
		pp.BillingCycleValue = &v
		pp.BillingCycleUnit = &unit
	}
	return pp
}

// newTestPlanDuration builds a minimal PricePlan using deprecated duration fields.
func newTestPlanDuration(unit string, value int) *priceplanpb.PricePlan {
	pp := &priceplanpb.PricePlan{
		Id:          "plan-test",
		BillingKind: priceplanpb.BillingKind_BILLING_KIND_RECURRING,
	}
	if value > 0 && unit != "" {
		v := int32(value)
		pp.DurationValue = &v
		pp.DurationUnit = &unit
	}
	return pp
}

func mustParseDate(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic("mustParseDate: " + err.Error())
	}
	return t
}

func assertPeriod(t *testing.T, w periodWindow, wantStart, wantEnd string) {
	t.Helper()
	if w.Start != wantStart {
		t.Errorf("period start: got %s, want %s", w.Start, wantStart)
	}
	if w.End != wantEnd {
		t.Errorf("period end: got %s, want %s", w.End, wantEnd)
	}
}

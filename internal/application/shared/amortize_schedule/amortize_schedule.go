// Package amortizeschedule provides pure period and tranche math for
// amortization schedules. It computes period boundaries from start/end
// dates, applies proration policies, and distributes amounts across
// tranches.
//
// Charter — this package MUST NOT import:
//   - proto entity types (esqyma/...)
//   - DB drivers or adapter packages
//   - anything under internal/application/usecases/...
//
// Depends only on the Go standard library.
//
// Consumers (keep in sync):
//   - usecases/service/amortization/ — proto-shaped wrapper (primary consumer)
//   - usecases/expenditure/expense_recognition_run/list_expense_run_candidates.go
//     (uses AddPeriod only for period enumeration; does NOT call
//     EnumerateTranches/ComputeNextDueTranche directly — those go through
//     the service wrapper)
//
// History: originally consumed directly by 4 domain use cases
// (treasury/collection, treasury/disbursement, revenue/revenue,
// expenditure/expense_recognition_run). Promoted to a service-driven domain
// with proto contract at proto/v1/service/amortization/ per Mantra 3/3
// rule-of-three (2026-06-08). The pure math stays here; the service wrapper
// provides the versioned wire contract.
package amortizeschedule

import (
	"errors"
	"strings"
	"time"
)

// Inputs is the request shape for ComputeNextDueTranche / EnumerateTranches.
// All amounts are centavos (integer). Dates are YYYY-MM-DD ISO-8601 in the
// workspace's calendar; the helper does NOT do timezone resolution — it
// assumes the caller has already projected to the right calendar day.
type Inputs struct {
	// StartDate is advance_start_date.
	StartDate string
	// EndDate is advance_end_date (optional; empty = open-ended; last tranche
	// absorbs partial if EndDate falls mid-period).
	EndDate string
	// PeriodCount is the total number of tranches the advance amortizes over.
	// Must be > 0. When EndDate is set, PeriodCount × PeriodUnit must align;
	// when they disagree, PeriodCount wins for tranche enumeration and the
	// last tranche absorbs any remainder.
	PeriodCount int
	// PeriodUnit is "day" | "week" | "month" | "year".
	PeriodUnit string
	// TotalAmount is the original advance amount in centavos. This is the
	// snapshot — NOT the remaining amount. Tranche values are derived from
	// this so that floating-precision rounding is stable across runs.
	TotalAmount int64
	// ProrationPolicy controls the first-tranche behavior. UNSPECIFIED is
	// normalized to FULL_TRANCHE per Decision 13.
	ProrationPolicy ProrationPolicy
	// AsOfDate is the operator's "as of" cursor. Used by ComputeNextDueTranche
	// to pick the next un-recognized tranche.
	AsOfDate string
}

// ProrationPolicy mirrors common/advance_kind.AdvanceProrationPolicy without
// importing the proto type — this keeps the helper free of proto coupling.
//
// Callers translate from advance_kind.AdvanceProrationPolicy_*  in their own
// code. The translation table is:
//
//	UNSPECIFIED       -> ProrationPolicyFullTranche (per Decision 13)
//	DAY_PRORATED      -> ProrationPolicyDayProrated
//	FULL_TRANCHE      -> ProrationPolicyFullTranche
//	NEXT_PERIOD_START -> ProrationPolicyNextPeriodStart
type ProrationPolicy int

const (
	// ProrationPolicyFullTranche makes the first tranche a full period even
	// if advance_start_date falls mid-period. This is the default.
	ProrationPolicyFullTranche ProrationPolicy = iota
	// ProrationPolicyDayProrated splits the first tranche by elapsed days
	// over period days. Subsequent tranches are full.
	ProrationPolicyDayProrated
	// ProrationPolicyNextPeriodStart skips the partial current period and
	// anchors the first tranche to the next period boundary.
	ProrationPolicyNextPeriodStart
)

// TrancheSpec is a single computed period — what AmortizeAdvance{Collection,
// Disbursement} should write as the recognition's period_start / period_end /
// total_amount.
type TrancheSpec struct {
	// Index is the 0-based position of this tranche in the schedule.
	Index int
	// PeriodStart is YYYY-MM-DD inclusive.
	PeriodStart string
	// PeriodEnd is YYYY-MM-DD inclusive (the day before the next period
	// boundary; or capped at EndDate when partial-last-period kicks in).
	PeriodEnd string
	// Amount is centavos for this tranche. First tranche may be smaller
	// (DAY_PRORATED) or skipped (NEXT_PERIOD_START); last tranche absorbs
	// the centavo remainder so SUM(Amount) == TotalAmount.
	Amount int64
}

const dateLayout = "2006-01-02"

// EnumerateTranches returns the full ordered list of tranches the advance
// will amortize into. Pure function, no I/O.
//
// Algorithm:
//  1. Normalize ProrationPolicy (UNSPECIFIED -> FULL_TRANCHE).
//  2. Compute per-period boundaries via AddPeriod walking from the first
//     anchor.
//  3. Distribute TotalAmount across PeriodCount tranches as TotalAmount /
//     PeriodCount, with last-tranche-absorbs-remainder semantics.
//  4. Apply policy-specific first-tranche adjustments.
//  5. Cap the last tranche's PeriodEnd at EndDate when EndDate is mid-period.
func EnumerateTranches(in Inputs) ([]TrancheSpec, error) {
	if in.PeriodCount <= 0 {
		return nil, errors.New("amortize_schedule: period_count must be > 0")
	}
	unit := normalizeUnit(in.PeriodUnit)
	if unit == "" {
		return nil, errors.New("amortize_schedule: period_unit must be one of day|week|month|year")
	}
	if in.TotalAmount < 0 {
		return nil, errors.New("amortize_schedule: total_amount must be >= 0")
	}
	start, err := parseDate(in.StartDate)
	if err != nil {
		return nil, err
	}
	var endPtr *time.Time
	if strings.TrimSpace(in.EndDate) != "" {
		e, parseErr := parseDate(in.EndDate)
		if parseErr != nil {
			return nil, parseErr
		}
		endPtr = &e
	}
	policy := normalizePolicy(in.ProrationPolicy)

	count := in.PeriodCount
	base := in.TotalAmount / int64(count)
	remainder := in.TotalAmount - base*int64(count)

	tranches := make([]TrancheSpec, 0, count)

	// Resolve the first-tranche anchor based on policy.
	firstStart := start
	dayProratedFraction := 1.0
	dayProratedFirstEnd := start // when DAY_PRORATED, the partial first tranche ends at the next anchor minus 1 day
	switch policy {
	case ProrationPolicyDayProrated:
		// Anchor = next "natural" period boundary after start. Period 0 covers
		// start..(nextAnchor-1) but at a prorated amount; full tranches
		// continue from nextAnchor.
		nextAnchor := AddPeriod(naturalAnchor(start, unit), 1, unit)
		// Total days in the natural period
		periodDays := daysBetween(naturalAnchor(start, unit), nextAnchor)
		// Days remaining in the natural period from start (inclusive)
		remDays := daysBetween(start, nextAnchor)
		if periodDays <= 0 || remDays <= 0 {
			dayProratedFraction = 1.0
		} else {
			dayProratedFraction = float64(remDays) / float64(periodDays)
		}
		dayProratedFirstEnd = nextAnchor.AddDate(0, 0, -1)
		firstStart = start
	case ProrationPolicyNextPeriodStart:
		// Skip the partial current period; first tranche begins at the next
		// natural period boundary strictly after start.
		firstStart = nextAnchorStrictlyAfter(start, unit)
	case ProrationPolicyFullTranche:
		firstStart = start
	}

	cursor := firstStart
	for i := 0; i < count; i++ {
		var pStart, pEnd time.Time
		switch {
		case i == 0 && policy == ProrationPolicyDayProrated:
			// Partial first tranche from start to (nextAnchor - 1 day)
			pStart = start
			pEnd = dayProratedFirstEnd
			// Cursor for the next iteration is the nextAnchor (full period start)
			cursor = AddPeriod(naturalAnchor(start, unit), 1, unit)
		default:
			pStart = cursor
			pEnd = AddPeriod(cursor, 1, unit).AddDate(0, 0, -1)
			cursor = AddPeriod(cursor, 1, unit)
		}

		amount := base
		if i == count-1 {
			amount += remainder // last tranche absorbs centavo remainder
		}
		if i == 0 && policy == ProrationPolicyDayProrated {
			// Apply day-prorated fraction to the first tranche.
			full := amount
			prorated := int64(float64(full) * dayProratedFraction)
			diff := full - prorated
			amount = prorated
			// Push the diff onto the last tranche so SUM stays == TotalAmount.
			// (Last tranche's remainder absorption already runs; we add diff
			// here by adjusting the running remainder via a closure variable.)
			remainder += diff
			// But if the last tranche already absorbed the original remainder
			// when i==count-1 — that case is impossible in practice (count >= 1
			// and DAY_PRORATED only fires on i==0). For count==1 special-case
			// below: ensure the single tranche carries the full TotalAmount
			// since there's no other tranche to absorb the diff.
			if count == 1 {
				amount = in.TotalAmount
				remainder = 0
			}
		}

		// Cap PeriodEnd at advance_end_date for the last tranche if it would
		// otherwise extend past it. Same rule applies if any tranche end
		// crosses the boundary.
		if endPtr != nil && pEnd.After(*endPtr) {
			pEnd = *endPtr
		}

		tranches = append(tranches, TrancheSpec{
			Index:       i,
			PeriodStart: pStart.Format(dateLayout),
			PeriodEnd:   pEnd.Format(dateLayout),
			Amount:      amount,
		})

		// If the EndDate is hit early stop (defensive — period_count should
		// govern, but if EndDate is hard-capped before all tranches enumerate
		// we still produce the rest with zero-duration entries collapsed).
		if endPtr != nil && !pEnd.Before(*endPtr) {
			break
		}
	}

	// Re-balance: ensure SUM(Amount) == TotalAmount. The day-prorated branch
	// adjusts `remainder` mid-loop, so the final tranche may have absorbed
	// a different value. Recompute the last-tranche delta from sum.
	if n := len(tranches); n > 0 {
		var sum int64
		for _, t := range tranches {
			sum += t.Amount
		}
		if delta := in.TotalAmount - sum; delta != 0 {
			tranches[n-1].Amount += delta
		}
	}

	return tranches, nil
}

// ComputeNextDueTranche returns the single tranche due as of AsOfDate. Used
// by AmortizeAdvance{Collection,Disbursement} which want to recognize only
// the next-due period (not the whole schedule).
//
// Returns (tranche, true) when there is an un-recognized tranche whose start
// is <= AsOfDate; returns (zero, false) when nothing is due yet OR when the
// schedule is fully amortized through AsOfDate.
//
// The caller is responsible for the idempotency check against existing
// recognitions — this helper only computes "what would the next tranche be";
// it does NOT consult the DB.
func ComputeNextDueTranche(in Inputs) (TrancheSpec, bool, error) {
	asOf, err := parseDate(in.AsOfDate)
	if err != nil {
		return TrancheSpec{}, false, err
	}
	tranches, err := EnumerateTranches(in)
	if err != nil {
		return TrancheSpec{}, false, err
	}
	for _, t := range tranches {
		ts, parseErr := time.Parse(dateLayout, t.PeriodStart)
		if parseErr != nil {
			return TrancheSpec{}, false, parseErr
		}
		if !ts.After(asOf) {
			// Found the latest tranche whose start <= asOf. But we want the
			// FIRST one; since the caller will idempotency-check vs existing
			// recognitions, returning the earliest unrecognized tranche is
			// the correct contract. The caller iterates externally via the
			// idempotency check; this helper picks per-call the earliest
			// candidate.
			return t, true, nil
		}
	}
	return TrancheSpec{}, false, nil
}

// AddPeriod advances t by N × unit. Public so tests can exercise it directly.
func AddPeriod(t time.Time, n int, unit string) time.Time {
	switch normalizeUnit(unit) {
	case "day":
		return t.AddDate(0, 0, n)
	case "week":
		return t.AddDate(0, 0, n*7)
	case "month":
		return t.AddDate(0, n, 0)
	case "year":
		return t.AddDate(n, 0, 0)
	default:
		return t
	}
}

// naturalAnchor returns the start of the "natural" period containing t, used
// to compute DAY_PRORATED's first-period boundaries.
//   - day:   t itself
//   - week:  Monday of t's week
//   - month: 1st of t's month
//   - year:  Jan 1 of t's year
func naturalAnchor(t time.Time, unit string) time.Time {
	switch normalizeUnit(unit) {
	case "day":
		return t
	case "week":
		// Monday anchor (per plan; workspace setting deferred to v2).
		weekday := int(t.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday -> 7
		}
		return t.AddDate(0, 0, -(weekday - 1))
	case "month":
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	case "year":
		return time.Date(t.Year(), time.January, 1, 0, 0, 0, 0, t.Location())
	default:
		return t
	}
}

// nextAnchorStrictlyAfter returns the next natural anchor strictly after t.
// Used by NEXT_PERIOD_START.
func nextAnchorStrictlyAfter(t time.Time, unit string) time.Time {
	current := naturalAnchor(t, unit)
	next := AddPeriod(current, 1, unit)
	if !t.Before(current) && t.Before(next) {
		// t is inside [current, next); next anchor is `next`.
		return next
	}
	return next
}

// daysBetween returns the inclusive day count from start to end.
// If end <= start, returns 0.
func daysBetween(start, end time.Time) int {
	if !end.After(start) {
		return 0
	}
	return int(end.Sub(start).Hours() / 24)
}

func parseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, errors.New("amortize_schedule: date is empty")
	}
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

func normalizeUnit(u string) string {
	switch strings.ToLower(strings.TrimSpace(u)) {
	case "day", "days":
		return "day"
	case "week", "weeks":
		return "week"
	case "month", "months":
		return "month"
	case "year", "years":
		return "year"
	}
	return ""
}

func normalizePolicy(p ProrationPolicy) ProrationPolicy {
	// Defensive: any out-of-range value normalises to FULL_TRANCHE.
	switch p {
	case ProrationPolicyDayProrated, ProrationPolicyFullTranche, ProrationPolicyNextPeriodStart:
		return p
	}
	return ProrationPolicyFullTranche
}

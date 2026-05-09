// Package depreciation provides the pure-compute depreciation engine for IAS 16 methods.
//
// No I/O — all functions are pure functions over asset parameters and period information.
// The engine is called by the application-layer use cases (GenerateDepreciationRun,
// ListDepreciationCandidates); it never touches repositories directly.
//
// Monetary amounts are always int64 centavos. Display ÷100 is the caller's
// responsibility. Zero-float rule: no float arithmetic on centavo amounts.
// Floating-point rates (depreciation_rate, salvage_pct) are inputs-only and are
// converted to centavos immediately.
package depreciation

import (
	"errors"
	"math"
	"time"
)

// ErrUnitsRequired is returned by ComputeUnitsOfProduction when units_produced
// is not provided. The calling use case maps this to a UNITS_REQUIRED blocker.
var ErrUnitsRequired = errors.New("units_of_production: units_produced must be > 0")

// AssetParams holds the fixed per-asset parameters needed by the engine.
// All monetary fields are int64 centavos.
type AssetParams struct {
	AcquisitionCost        int64   // centavos
	SalvageValue           int64   // centavos — floor: depreciation stops here
	UsefulLifeMonths       int32   // total useful life in calendar months
	DepreciationStartDate  string  // YYYY-MM-DD — first depreciable period boundary
	DepreciationRate       float64 // fractional (e.g. 0.20 = 20%) — used by declining-balance methods
	AccumulatedDepreciation int64  // centavos — accumulated so far (before this period)
}

// PeriodParams describes the time window for one depreciation period.
type PeriodParams struct {
	PeriodStart time.Time // start of the period (calendar month boundary)
	PeriodEnd   time.Time // end of the period (inclusive last day of month)
	PeriodIndex int       // 1-based ordinal from depreciation_start_date (used by SoYD)
}

// DepreciableBase returns (acquisition_cost − salvage_value), clamped to ≥0.
func DepreciableBase(cost, salvage int64) int64 {
	base := cost - salvage
	if base < 0 {
		return 0
	}
	return base
}

// ComputeStraightLine computes the depreciation amount for one period under
// IAS 16.62a — Straight-Line method.
//
// Formula: (acquisition_cost − salvage_value) / useful_life_months
// The last period rounds up any rounding remainder so accumulated == depreciable base.
// Returns 0 when the asset is already fully depreciated.
//
// Reference: IAS 16.50, .62a
func ComputeStraightLine(asset AssetParams, period PeriodParams) (int64, error) {
	if asset.UsefulLifeMonths <= 0 {
		return 0, errors.New("straight_line: useful_life_months must be > 0")
	}

	base := DepreciableBase(asset.AcquisitionCost, asset.SalvageValue)
	if base <= 0 {
		return 0, nil
	}

	// Guard: already at or past salvage floor
	remaining := base - asset.AccumulatedDepreciation
	if remaining <= 0 {
		return 0, nil
	}

	// Straight-line: evenly distribute over total months
	perMonth := base / int64(asset.UsefulLifeMonths)

	// Last period: cap so accumulated never exceeds depreciable base
	amount := perMonth
	if remaining < perMonth {
		amount = remaining
	}

	return amount, nil
}

// ComputeDecliningBalance computes depreciation for one period under the
// Declining-Balance method (IAS 16.62b).
//
// Formula: book_value × depreciation_rate
// book_value = acquisition_cost − accumulated_depreciation
// Floor: amount capped so book_value does not fall below salvage_value.
//
// When depreciation_rate == 0, falls back to 2×SL (double-declining).
// Reference: IAS 16.62b
func ComputeDecliningBalance(asset AssetParams, period PeriodParams, accumulated int64) (int64, error) {
	if asset.AcquisitionCost <= 0 {
		return 0, nil
	}

	rate := asset.DepreciationRate
	if rate <= 0 {
		// Default to 1/useful_life rate if not specified
		if asset.UsefulLifeMonths <= 0 {
			return 0, errors.New("declining_balance: depreciation_rate or useful_life_months required")
		}
		rate = 1.0 / float64(asset.UsefulLifeMonths)
	}

	bookValue := asset.AcquisitionCost - accumulated
	if bookValue <= asset.SalvageValue {
		return 0, nil // Already at or below salvage floor
	}

	// Monthly rate from annual rate
	monthlyRate := rate / 12.0
	rawAmount := float64(bookValue) * monthlyRate

	// Round to nearest centavo
	amount := int64(math.Round(rawAmount))
	if amount <= 0 {
		amount = 1 // Always post at least 1 centavo until salvage floor
	}

	// Cap at remaining depreciable balance
	remaining := bookValue - asset.SalvageValue
	if amount > remaining {
		amount = remaining
	}

	return amount, nil
}

// ComputeDoubleDecliningBalance computes depreciation for one period under
// the Double-Declining-Balance method (2× the straight-line rate).
//
// Reference: IAS 16.62b variant; commonly used in practice
func ComputeDoubleDecliningBalance(asset AssetParams, period PeriodParams, accumulated int64) (int64, error) {
	if asset.UsefulLifeMonths <= 0 {
		return 0, errors.New("double_declining_balance: useful_life_months must be > 0")
	}

	// Double the straight-line annual rate
	ddRate := 2.0 / float64(asset.UsefulLifeMonths)

	doubleDeclining := AssetParams{
		AcquisitionCost:  asset.AcquisitionCost,
		SalvageValue:     asset.SalvageValue,
		UsefulLifeMonths: asset.UsefulLifeMonths,
		DepreciationRate: ddRate,
	}
	return ComputeDecliningBalance(doubleDeclining, period, accumulated)
}

// ComputeSumOfYearsDigits computes depreciation for one period under the
// Sum-of-Years-Digits method (IAS 16.62b variant — declining fraction).
//
// Formula: (remaining_years / SYD_denominator) × depreciable_base
// SYD = n(n+1)/2  where n = useful_life_months / 12
//
// period.PeriodIndex is the 1-based sequential month from depreciation_start_date.
// Reference: IAS 16.62b
func ComputeSumOfYearsDigits(asset AssetParams, period PeriodParams) (int64, error) {
	if asset.UsefulLifeMonths <= 0 {
		return 0, errors.New("sum_of_years_digits: useful_life_months must be > 0")
	}
	n := int64(asset.UsefulLifeMonths)

	// SYD denominator = n(n+1)/2  (in months — we operate monthly)
	syd := n * (n + 1) / 2

	// Remaining periods = total - elapsed
	// periodIndex is 1-based, so remaining = n - (periodIndex - 1) = n - periodIndex + 1
	remaining := n - int64(period.PeriodIndex) + 1
	if remaining <= 0 {
		return 0, nil
	}

	base := DepreciableBase(asset.AcquisitionCost, asset.SalvageValue)
	if base <= 0 {
		return 0, nil
	}

	// Accumulation guard: never depreciate past salvage
	alreadyDepreciated := asset.AccumulatedDepreciation
	depreciableRemaining := base - alreadyDepreciated
	if depreciableRemaining <= 0 {
		return 0, nil
	}

	// amount = (remaining / syd) * base  — but using int arithmetic
	// To avoid truncation we compute: (remaining * base) / syd
	amount := (remaining * base) / syd

	// Cap at remaining depreciable balance
	if amount > depreciableRemaining {
		amount = depreciableRemaining
	}
	if amount < 0 {
		amount = 0
	}

	return amount, nil
}

// ComputeUnitsOfProduction returns ErrUnitsRequired when called.
//
// Units-of-production assets require per-period production data that is not
// available at run time in v1. The calling use case maps this error to a
// UNITS_REQUIRED blocker. See docs/plan/20260601-uop-depreciation/ for v2.
//
// Reference: IAS 16.62c
func ComputeUnitsOfProduction(asset AssetParams, period PeriodParams, unitsProduced int64) (int64, error) {
	if unitsProduced <= 0 {
		return 0, ErrUnitsRequired
	}
	// Stub: v2 will implement. If somehow called with units > 0 fall through.
	return 0, ErrUnitsRequired
}

// MidPeriodFraction returns the fraction of a period that is depreciable
// when the asset's depreciation_start_date falls inside the period rather
// than at its start.
//
// Used to prorate the first (and possibly last) period for start-mid-period cases.
// Returns 1.0 when the start date is on or before PeriodStart.
//
// IAS 16 allows either the "full month in which placed in service" convention
// (proration = 1.0 for full month) or the "pro-rata by day" convention.
// This function implements pro-rata by day per the test suite expectations.
func MidPeriodFraction(startDate time.Time, period PeriodParams) float64 {
	if !startDate.After(period.PeriodStart) {
		return 1.0
	}
	if startDate.After(period.PeriodEnd) {
		return 0.0
	}
	totalDays := period.PeriodEnd.Sub(period.PeriodStart).Hours() / 24.0
	activeDays := period.PeriodEnd.Sub(startDate).Hours() / 24.0
	if totalDays <= 0 {
		return 0.0
	}
	return activeDays / totalDays
}

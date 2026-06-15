package depreciation_run

import (
	"testing"
	"time"
)

// testPeriod builds a PeriodParams for the given month (1-based index from depreciation start).
func testPeriod(year, month int, index int) PeriodParams {
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	// Last day of month
	end := time.Date(year, time.Month(month+1), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
	return PeriodParams{
		PeriodStart: start,
		PeriodEnd:   end,
		PeriodIndex: index,
	}
}

// ---------------------------------------------------------------------------
// StraightLine
// ---------------------------------------------------------------------------

// IAS 16 §62a worked example: acquisition_cost=5_000_000 centavos, salvage=500_000,
// useful_life=60 months → per-period = (5_000_000−500_000)/60 = 75_000 centavos.
func TestComputeStraightLine_Normal(t *testing.T) {
	asset := AssetParams{
		AcquisitionCost:         5_000_000,
		SalvageValue:            500_000,
		UsefulLifeMonths:        60,
		AccumulatedDepreciation: 0,
	}
	period := testPeriod(2025, 1, 1)
	got, err := ComputeStraightLine(asset, period)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	const want int64 = 75_000
	if got != want {
		t.Errorf("ComputeStraightLine = %d, want %d", got, want)
	}
}

func TestComputeStraightLine_SalvageFloor(t *testing.T) {
	// Accumulated is already at depreciable base → amount must be 0
	asset := AssetParams{
		AcquisitionCost:         5_000_000,
		SalvageValue:            500_000,
		UsefulLifeMonths:        60,
		AccumulatedDepreciation: 4_500_000, // fully at salvage
	}
	period := testPeriod(2030, 1, 61)
	got, err := ComputeStraightLine(asset, period)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 0 {
		t.Errorf("expected 0 after salvage floor, got %d", got)
	}
}

func TestComputeStraightLine_LastPeriodCap(t *testing.T) {
	// Almost fully depreciated — only 10_000 remaining; period amount would be 75_000
	asset := AssetParams{
		AcquisitionCost:         5_000_000,
		SalvageValue:            500_000,
		UsefulLifeMonths:        60,
		AccumulatedDepreciation: 4_490_000, // 10_000 remaining to salvage
	}
	period := testPeriod(2030, 1, 60)
	got, err := ComputeStraightLine(asset, period)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	const want int64 = 10_000
	if got != want {
		t.Errorf("last-period cap: got %d, want %d", got, want)
	}
}

func TestComputeStraightLine_ZeroUsefulLife(t *testing.T) {
	asset := AssetParams{
		AcquisitionCost:  5_000_000,
		SalvageValue:     500_000,
		UsefulLifeMonths: 0,
	}
	period := testPeriod(2025, 1, 1)
	_, err := ComputeStraightLine(asset, period)
	if err == nil {
		t.Error("expected error for zero useful life")
	}
}

func TestComputeStraightLine_FullyDepreciated(t *testing.T) {
	// accumulated == depreciable base
	asset := AssetParams{
		AcquisitionCost:         1_000_000,
		SalvageValue:            100_000,
		UsefulLifeMonths:        12,
		AccumulatedDepreciation: 900_000,
	}
	period := testPeriod(2026, 2, 13)
	got, err := ComputeStraightLine(asset, period)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 0 {
		t.Errorf("expected 0 for fully-depreciated asset, got %d", got)
	}
}

// ---------------------------------------------------------------------------
// DecliningBalance
// ---------------------------------------------------------------------------

func TestComputeDecliningBalance_Normal(t *testing.T) {
	// Annual rate 20%, book_value = 5_000_000
	// monthly: 5_000_000 * (0.20/12) = 83_333.33 → 83_333
	asset := AssetParams{
		AcquisitionCost:  5_000_000,
		SalvageValue:     500_000,
		UsefulLifeMonths: 60,
		DepreciationRate: 0.20,
	}
	period := testPeriod(2025, 1, 1)
	got, err := ComputeDecliningBalance(asset, period, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 5_000_000 * (0.20/12) = 83_333.33 → rounded to 83_333
	const want int64 = 83_333
	if got != want {
		t.Errorf("ComputeDecliningBalance = %d, want %d", got, want)
	}
}

func TestComputeDecliningBalance_SalvageFloor(t *testing.T) {
	asset := AssetParams{
		AcquisitionCost:  5_000_000,
		SalvageValue:     500_000,
		UsefulLifeMonths: 60,
		DepreciationRate: 0.20,
	}
	// accumulated brings book_value to exactly salvage
	period := testPeriod(2030, 1, 60)
	got, err := ComputeDecliningBalance(asset, period, 4_500_000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 0 {
		t.Errorf("expected 0 at salvage floor, got %d", got)
	}
}

func TestComputeDecliningBalance_PartialPeriod(t *testing.T) {
	// Book value just above salvage — should cap
	asset := AssetParams{
		AcquisitionCost:  1_000_000,
		SalvageValue:     100_000,
		DepreciationRate: 0.40,
		UsefulLifeMonths: 24,
	}
	// accumulated = 890_000 → book_value = 110_000 → remaining above salvage = 10_000
	got, err := ComputeDecliningBalance(asset, testPeriod(2025, 1, 1), 890_000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// would normally be 110_000 * (0.40/12) ≈ 3_667, but remaining = 10_000
	// so result should be the smaller of 3_667 and 10_000 → 3_667
	const want int64 = 3_667
	if got != want {
		t.Errorf("ComputeDecliningBalance partial: got %d, want %d", got, want)
	}
}

// ---------------------------------------------------------------------------
// DoubleDecliningBalance
// ---------------------------------------------------------------------------

func TestComputeDoubleDecliningBalance_Normal(t *testing.T) {
	// DDB rate = 2 / 60 = 0.0333… annual → monthly = 0.0333/12 ≈ 0.00278
	// monthly: 5_000_000 * (2/(60*12)) = 5_000_000 * 0.002778 = 13_889
	asset := AssetParams{
		AcquisitionCost:  5_000_000,
		SalvageValue:     500_000,
		UsefulLifeMonths: 60,
	}
	got, err := ComputeDoubleDecliningBalance(asset, testPeriod(2025, 1, 1), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 5_000_000 * (2/60) / 12 = 13_888.88… → 13_889
	const want int64 = 13_889
	if got != want {
		t.Errorf("ComputeDoubleDecliningBalance = %d, want %d", got, want)
	}
}

// ---------------------------------------------------------------------------
// SumOfYearsDigits
// ---------------------------------------------------------------------------

// IAS 16 §62b (SoYD) example:
// cost=1_200_000, salvage=200_000, useful_life=5 years (60 months), SYD=60*61/2=1830
// Month 1: remaining=60, amount=(60/1830)*1_000_000
// Integer division: (60 * 1_000_000) / 1830 = 32_786 (floor)
func TestComputeSumOfYearsDigits_Period1(t *testing.T) {
	asset := AssetParams{
		AcquisitionCost:         1_200_000,
		SalvageValue:            200_000,
		UsefulLifeMonths:        60,
		AccumulatedDepreciation: 0,
	}
	period := PeriodParams{PeriodIndex: 1}
	got, err := ComputeSumOfYearsDigits(asset, period)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// (60 * 1_000_000) / 1830 = 32_786 (integer division — floor)
	const want int64 = 32_786
	if got != want {
		t.Errorf("SoYD period 1: got %d, want %d", got, want)
	}
}

func TestComputeSumOfYearsDigits_LastPeriod(t *testing.T) {
	// Last month: remaining = 1
	asset := AssetParams{
		AcquisitionCost:         1_200_000,
		SalvageValue:            200_000,
		UsefulLifeMonths:        60,
		AccumulatedDepreciation: 999_453, // some accumulated
	}
	period := PeriodParams{PeriodIndex: 60}
	got, err := ComputeSumOfYearsDigits(asset, period)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// remaining months = 1; amount = (1 * 1_000_000) / 1830 = 546
	// but cap at depreciableRemaining = 1_000_000 - 999_453 = 547
	const want int64 = 546
	if got != want {
		t.Errorf("SoYD last period: got %d, want %d", got, want)
	}
}

func TestComputeSumOfYearsDigits_SalvageFloor(t *testing.T) {
	asset := AssetParams{
		AcquisitionCost:         1_200_000,
		SalvageValue:            200_000,
		UsefulLifeMonths:        60,
		AccumulatedDepreciation: 1_000_000, // fully at salvage
	}
	period := PeriodParams{PeriodIndex: 61}
	got, err := ComputeSumOfYearsDigits(asset, period)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 0 {
		t.Errorf("SoYD salvage floor: expected 0, got %d", got)
	}
}

// ---------------------------------------------------------------------------
// UnitsOfProduction
// ---------------------------------------------------------------------------

func TestComputeUnitsOfProduction_ReturnsError(t *testing.T) {
	asset := AssetParams{
		AcquisitionCost:  5_000_000,
		SalvageValue:     500_000,
		UsefulLifeMonths: 60,
	}
	period := testPeriod(2025, 1, 1)

	_, err := ComputeUnitsOfProduction(asset, period, 0)
	if err == nil {
		t.Error("expected ErrUnitsRequired, got nil")
	}
	if err != ErrUnitsRequired {
		t.Errorf("expected ErrUnitsRequired sentinel, got: %v", err)
	}
}

func TestComputeUnitsOfProduction_PositiveUnitsStillBlocked(t *testing.T) {
	asset := AssetParams{
		AcquisitionCost:  5_000_000,
		SalvageValue:     500_000,
		UsefulLifeMonths: 60,
	}
	period := testPeriod(2025, 1, 1)
	// Even with units > 0, v1 always returns ErrUnitsRequired
	_, err := ComputeUnitsOfProduction(asset, period, 100)
	if err != ErrUnitsRequired {
		t.Errorf("expected ErrUnitsRequired even with units>0, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// MidPeriodFraction
// ---------------------------------------------------------------------------

func TestMidPeriodFraction_StartOnBoundary(t *testing.T) {
	period := testPeriod(2025, 1, 1)
	frac := MidPeriodFraction(period.PeriodStart, period)
	if frac != 1.0 {
		t.Errorf("start on boundary: expected 1.0, got %f", frac)
	}
}

func TestMidPeriodFraction_StartMidMonth(t *testing.T) {
	period := testPeriod(2025, 1, 1)                          // Jan 1 to Jan 31
	startDate := time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC) // mid-Jan
	frac := MidPeriodFraction(startDate, period)
	// 16 active days out of 30 total (Jan 16 to Jan 31 = 16 days remaining; total=30)
	if frac <= 0.0 || frac >= 1.0 {
		t.Errorf("mid-period fraction should be between 0 and 1, got %f", frac)
	}
}

func TestMidPeriodFraction_StartAfterPeriodEnd(t *testing.T) {
	period := testPeriod(2025, 1, 1)
	startDate := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	frac := MidPeriodFraction(startDate, period)
	if frac != 0.0 {
		t.Errorf("start after period end: expected 0.0, got %f", frac)
	}
}

// ---------------------------------------------------------------------------
// DepreciableBase edge cases
// ---------------------------------------------------------------------------

func TestDepreciableBase_NegativeClampedToZero(t *testing.T) {
	// salvage > cost — should clamp to 0, not negative
	got := DepreciableBase(100_000, 200_000)
	if got != 0 {
		t.Errorf("expected 0 when salvage > cost, got %d", got)
	}
}

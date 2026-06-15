package ph

import (
	"context"
	"testing"
	"time"

	suppliercontractpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract"
	suppliercontractlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_line"
	paycyclepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/pay_cycle"
	ratebandpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/rate_band"
	ratetablepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/rate_table"

	"github.com/erniealice/espyna-golang/internal/application/usecases/service/payroll/payrollcore"
)

// ---------------------------------------------------------------------
// In-memory rate-table fixtures
//
// These are TEST-ONLY. The production calculator MUST go through
// PayslipContext.RateResolver — there are no hardcoded rate values in
// calculator.go.
//
// Sources:
//   - SSS: official 2025 SSS contribution table (effective Jan 2025,
//     current for 2026 per codex-research §2). 117 rows; we seed the
//     ones the tests touch (₱5k, ₱8k, ₱25k, ₱35k cap).
//   - BIR: RR 11-2018 Annex E semi-monthly column (codex-research §3).
// ---------------------------------------------------------------------

const (
	sssTableID = "rt_sss_2025_test"
	birTableID = "rt_bir_2023_semi_monthly_test"
)

// upper returns *int64 helper for RateBand UpperBoundCentavos.
func upper(v int64) *int64 { return &v }

// sssBands returns a small subset of SSS employee-share bands covering
// the test scenarios. Centavos throughout.
func sssBands() []*ratebandpb.RateBand {
	type row struct {
		lower int64
		upper *int64
		ee    int64 // employee total in centavos
	}
	rows := []row{
		// Below 5,250 → MSC 5,000 → EE 250.00
		{0, upper(5_249_99), 250_00},
		// 7,750.00–8,249.99 → MSC 8,000 → EE 400.00
		{7_750_00, upper(8_249_99), 400_00},
		// 24,750.00–25,249.99 → MSC 25,000 → EE 1,250.00
		{24_750_00, upper(25_249_99), 1_250_00},
		// 34,750 and over → MSC 35,000 → EE 1,750.00
		{34_750_00, nil, 1_750_00},
	}
	out := make([]*ratebandpb.RateBand, 0, len(rows))
	for i, r := range rows {
		out = append(out, &ratebandpb.RateBand{
			Id:                  "rb_sss_" + strconv2(i),
			RateTableId:         sssTableID,
			LowerBoundCentavos:  r.lower,
			UpperBoundCentavos:  r.upper,
			RateType:            "fixed",
			FixedAmountCentavos: r.ee,
			Ordinal:             int32(i + 1),
		})
	}
	return out
}

// birSemiMonthlyBands returns the full BIR semi-monthly bracket table
// (RR 11-2018 Annex E) in centavos. This is shared across tests.
func birSemiMonthlyBands() []*ratebandpb.RateBand {
	type row struct {
		lower       int64
		upper       *int64
		fixedCent   int64 // additive constant (centavos)
		basisPoints int32 // 0 means no marginal rate (bracket 1)
	}
	rows := []row{
		// Bracket 1: ≤10,417 → 0
		{0, upper(10_417_00), 0, 0},
		// Bracket 2: 10,417–16,666 → 0 + 15% over 10,417
		{10_417_00, upper(16_666_00), 0, 1500},
		// Bracket 3: 16,667–33,332 → 937.50 + 20% over 16,667
		{16_667_00, upper(33_332_00), 937_50, 2000},
		// Bracket 4: 33,333–83,332 → 4,270.70 + 25% over 33,333
		{33_333_00, upper(83_332_00), 4_270_70, 2500},
		// Bracket 5: 83,333–333,332 → 16,770.70 + 30% over 83,333
		{83_333_00, upper(333_332_00), 16_770_70, 3000},
		// Bracket 6: ≥333,333 → 91,770.70 + 35% over 333,333
		{333_333_00, nil, 91_770_70, 3500},
	}
	out := make([]*ratebandpb.RateBand, 0, len(rows))
	for i, r := range rows {
		rt := "fixed"
		if r.basisPoints > 0 {
			rt = "percentage_of_excess"
		}
		out = append(out, &ratebandpb.RateBand{
			Id:                  "rb_bir_" + strconv2(i),
			RateTableId:         birTableID,
			LowerBoundCentavos:  r.lower,
			UpperBoundCentavos:  r.upper,
			RateType:            rt,
			RateBasisPoints:     r.basisPoints,
			FixedAmountCentavos: r.fixedCent,
			Ordinal:             int32(i + 1),
		})
	}
	return out
}

// strconv2 is a tiny non-importing int-to-string for fixture IDs to
// avoid pulling in strconv just for a test helper.
func strconv2(i int) string {
	if i == 0 {
		return "0"
	}
	const digits = "0123456789"
	var buf [12]byte
	n := len(buf)
	negative := i < 0
	if negative {
		i = -i
	}
	for i > 0 {
		n--
		buf[n] = digits[i%10]
		i /= 10
	}
	s := string(buf[n:])
	if negative {
		return "-" + s
	}
	return s
}

func makeRateResolver() func(ctx context.Context, kind, region string, asOf time.Time) (*ratetablepb.RateTable, []*ratebandpb.RateBand, error) {
	sssTable := &ratetablepb.RateTable{
		Id:               sssTableID,
		ComplianceRegion: "PH",
		Kind:             payrollcore.RateKindSSSEmployeeShare,
		EffectiveFrom:    "2025-01-01",
		VersionLabel:     "2025-01",
		Status:           ratetablepb.RateTableStatus_RATE_TABLE_STATUS_ACTIVE,
	}
	birTable := &ratetablepb.RateTable{
		Id:               birTableID,
		ComplianceRegion: "PH",
		Kind:             payrollcore.RateKindBIRWithholdingSemiMonthly,
		EffectiveFrom:    "2023-01-01",
		VersionLabel:     "2023-01",
		Status:           ratetablepb.RateTableStatus_RATE_TABLE_STATUS_ACTIVE,
	}
	sss := sssBands()
	bir := birSemiMonthlyBands()
	return func(ctx context.Context, kind, region string, asOf time.Time) (*ratetablepb.RateTable, []*ratebandpb.RateBand, error) {
		if region != "PH" {
			t := &ratetablepb.RateTable{}
			return t, nil, nil
		}
		switch kind {
		case payrollcore.RateKindSSSEmployeeShare:
			return sssTable, sss, nil
		case payrollcore.RateKindBIRWithholdingSemiMonthly:
			return birTable, bir, nil
		}
		return &ratetablepb.RateTable{}, nil, nil
	}
}

// makeContext builds a PayslipContext with one BASIC_SALARY line.
func makeContext(monthlyBasicCentavos int64, halfIndex string) *payrollcore.PayslipContext {
	basicKind := suppliercontractlinepb.SupplierContractLineKind_SUPPLIER_CONTRACT_LINE_KIND_BASIC_SALARY
	contract := &suppliercontractpb.SupplierContract{
		Id:          "sc_test_employment",
		WorkspaceId: "ws_test",
	}
	lines := []*suppliercontractlinepb.SupplierContractLine{
		{
			Id:                 "scl_basic",
			SupplierContractId: contract.GetId(),
			Description:        "Monthly Basic",
			Quantity:           1,
			UnitPrice:          monthlyBasicCentavos,
			Kind:               &basicKind,
		},
	}
	return &payrollcore.PayslipContext{
		PayCycle: &paycyclepb.PayCycle{
			Id:          "pc_test",
			WorkspaceId: "ws_test",
			CutoffStart: "2026-04-01",
			CutoffEnd:   "2026-04-15",
			PayDate:     "2026-04-15",
			HalfIndex:   halfIndex,
		},
		EmployeeID:         "sup_emp_test",
		EmploymentContract: contract,
		ContractLines:      lines,
		PayFrequency:       payrollcore.PayFrequencySemiMonthly,
		HalfIndex:          halfIndex,
		RateResolver:       makeRateResolver(),
	}
}

// findLine returns the first line whose LineKind matches; nil if absent.
func findLine(lines []payrollcore.LineResolution, kind string) *payrollcore.LineResolution {
	for i := range lines {
		if lines[i].LineKind == kind {
			return &lines[i]
		}
	}
	return nil
}

// findStatutoryByDescription locates the deduction_statutory line whose
// description starts with prefix (e.g., "SSS", "PhilHealth", "Pag-IBIG").
func findStatutoryByDescription(lines []payrollcore.LineResolution, prefix string) *payrollcore.LineResolution {
	for i := range lines {
		if lines[i].LineKind != payrollcore.LineKindDeductionStatutory {
			continue
		}
		if len(lines[i].Description) < len(prefix) {
			continue
		}
		if lines[i].Description[:len(prefix)] == prefix {
			return &lines[i]
		}
	}
	return nil
}

// ---------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------

// TestCalculate_25k_FirstHalf — monthly_basic = 25,000 PHP, semi-monthly
// first half. Expected (centavos):
//
//   - SSS         = 1,250.00 (MSC 25,000 row)
//   - PhilHealth  =   625.00 (5% × 25,000 / 2)
//   - Pag-IBIG    =   200.00 (2% × cap 10,000)
//   - BIR         =   > 0 (just over the 10,417 threshold)
func TestCalculate_25k_FirstHalf(t *testing.T) {
	c := NewCalculator()
	ctx := makeContext(25_000_00, payrollcore.HalfIndexFirst)
	lines, err := c.Calculate(context.Background(), ctx)
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}

	basic := findLine(lines, payrollcore.LineKindEarningBasic)
	if basic == nil {
		t.Fatalf("no earning_basic line emitted")
	}
	if basic.Amount != 12_500_00 {
		t.Errorf("basic.Amount = %d (want 1,250,000 centavos)", basic.Amount)
	}

	sss := findStatutoryByDescription(lines, "SSS")
	if sss == nil {
		t.Fatalf("no SSS line emitted")
	}
	if sss.Amount != 1_250_00 {
		t.Errorf("SSS.Amount = %d (want 125,000 centavos)", sss.Amount)
	}
	if sss.RateTableID != sssTableID {
		t.Errorf("SSS.RateTableID = %q (want %q)", sss.RateTableID, sssTableID)
	}
	if sss.AppliedBasis != 25_000_00 {
		t.Errorf("SSS.AppliedBasis = %d (want 2,500,000)", sss.AppliedBasis)
	}

	ph := findStatutoryByDescription(lines, "PhilHealth")
	if ph == nil {
		t.Fatalf("no PhilHealth line emitted")
	}
	if ph.Amount != 625_00 {
		t.Errorf("PhilHealth.Amount = %d (want 62,500 centavos)", ph.Amount)
	}

	pg := findStatutoryByDescription(lines, "Pag-IBIG")
	if pg == nil {
		t.Fatalf("no Pag-IBIG line emitted")
	}
	if pg.Amount != 200_00 {
		t.Errorf("Pag-IBIG.Amount = %d (want 20,000 centavos)", pg.Amount)
	}

	tax := findLine(lines, payrollcore.LineKindDeductionTax)
	if tax == nil {
		t.Fatalf("no BIR tax line emitted (expected > 0)")
	}
	if tax.Amount <= 0 {
		t.Errorf("BIR.Amount = %d (want > 0)", tax.Amount)
	}
	// Sanity: half_basic 12,500 - statutory 2,075 = 10,425 → bracket 2:
	// (10,425 - 10,417) × 15% = 8 × 0.15 = 1.20 PHP = 120 centavos.
	if tax.Amount != 120 {
		t.Errorf("BIR.Amount = %d (want 120 centavos based on RR 11-2018 bracket-2 math)", tax.Amount)
	}
}

// TestCalculate_8k_BelowTaxThreshold — monthly_basic = 8,000 PHP. Below
// the BIR ₱10,417 threshold even before statutories, so expected tax = 0
// (no tax line emitted). Statutory deductions still apply.
func TestCalculate_8k_BelowTaxThreshold(t *testing.T) {
	c := NewCalculator()
	ctx := makeContext(8_000_00, payrollcore.HalfIndexFirst)
	lines, err := c.Calculate(context.Background(), ctx)
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}

	basic := findLine(lines, payrollcore.LineKindEarningBasic)
	if basic == nil || basic.Amount != 4_000_00 {
		t.Errorf("basic.Amount = %v (want 400,000)", basic)
	}

	sss := findStatutoryByDescription(lines, "SSS")
	if sss == nil || sss.Amount != 400_00 {
		t.Errorf("SSS.Amount = %v (want 40,000 — MSC 8,000 row EE 400.00)", sss)
	}

	// PhilHealth: 8,000 floored to 10,000 → 10,000 × 2.5% = 250.
	ph := findStatutoryByDescription(lines, "PhilHealth")
	if ph == nil || ph.Amount != 250_00 {
		t.Errorf("PhilHealth.Amount = %v (want 25,000 — floor applied)", ph)
	}

	// Pag-IBIG: monthly 8,000 (>1,500) → 2% × min(8k,10k) = 160.
	pg := findStatutoryByDescription(lines, "Pag-IBIG")
	if pg == nil || pg.Amount != 160_00 {
		t.Errorf("Pag-IBIG.Amount = %v (want 16,000)", pg)
	}

	// Tax line should be absent — taxable < 10,417.
	if tax := findLine(lines, payrollcore.LineKindDeductionTax); tax != nil {
		t.Errorf("unexpected BIR tax line: %+v (want absent because taxable < 10,417)", tax)
	}
}

// TestCalculate_50k_HigherBrackets — monthly_basic = 50,000 PHP.
// Exercises:
//   - SSS top bracket (MSC 35,000 cap → EE 1,750.00)
//   - PhilHealth above floor (5% × 50,000 / 2 = 1,250)
//   - Pag-IBIG ceiling (2% × cap 10,000 = 200)
//   - BIR bracket 3 (16,667–33,332): 937.50 + 20% × (taxable − 16,667)
func TestCalculate_50k_HigherBrackets(t *testing.T) {
	c := NewCalculator()
	ctx := makeContext(50_000_00, payrollcore.HalfIndexFirst)
	lines, err := c.Calculate(context.Background(), ctx)
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}

	sss := findStatutoryByDescription(lines, "SSS")
	if sss == nil || sss.Amount != 1_750_00 {
		t.Errorf("SSS.Amount = %v (want 175,000 — MSC 35,000 cap row)", sss)
	}

	ph := findStatutoryByDescription(lines, "PhilHealth")
	if ph == nil || ph.Amount != 1_250_00 {
		t.Errorf("PhilHealth.Amount = %v (want 125,000)", ph)
	}

	pg := findStatutoryByDescription(lines, "Pag-IBIG")
	if pg == nil || pg.Amount != 200_00 {
		t.Errorf("Pag-IBIG.Amount = %v (want 20,000)", pg)
	}

	// Tax math:
	//   half_basic        = 25,000.00
	//   statutory total   = 1,750 + 1,250 + 200 = 3,200.00
	//   taxable           = 25,000 - 3,200 = 21,800.00
	//   bracket 3         = 937.50 + 20% × (21,800 - 16,667)
	//                     = 937.50 + 0.20 × 5,133
	//                     = 937.50 + 1,026.60
	//                     = 1,964.10 → 196,410 centavos
	tax := findLine(lines, payrollcore.LineKindDeductionTax)
	if tax == nil {
		t.Fatalf("no BIR tax line emitted")
	}
	want := int64(1_964_10)
	if tax.Amount != want {
		t.Errorf("BIR.Amount = %d (want %d centavos = ₱1,964.10)", tax.Amount, want)
	}
	if tax.RateTableID != birTableID {
		t.Errorf("BIR.RateTableID = %q (want %q)", tax.RateTableID, birTableID)
	}
}

// TestCalculate_SecondHalf_NoStatutory — semi-monthly second half
// collects no statutory deductions in the MVP "first-half" strategy.
func TestCalculate_SecondHalf_NoStatutory(t *testing.T) {
	c := NewCalculator()
	ctx := makeContext(25_000_00, payrollcore.HalfIndexSecond)
	lines, err := c.Calculate(context.Background(), ctx)
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}

	for _, ln := range lines {
		if ln.LineKind == payrollcore.LineKindDeductionStatutory {
			t.Errorf("unexpected statutory line on second half: %+v", ln)
		}
	}

	// Basic earning still emitted at half-cycle amount.
	basic := findLine(lines, payrollcore.LineKindEarningBasic)
	if basic == nil || basic.Amount != 12_500_00 {
		t.Errorf("basic.Amount = %v (want 1,250,000)", basic)
	}
}

// TestCalculate_RegistryContract — sanity check that ComplianceRegion
// and Version are pinned correctly.
func TestCalculate_RegistryContract(t *testing.T) {
	c := NewCalculator()
	if got := c.ComplianceRegion(); got != "PH" {
		t.Errorf("ComplianceRegion = %q (want PH)", got)
	}
	if got := c.Version(); got != "PH-2026.04" {
		t.Errorf("Version = %q (want PH-2026.04)", got)
	}
}

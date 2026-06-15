// Package ph implements the PH payroll calculator. ComplianceRegion is
// "PH"; Version is "PH-2026.04". All rate values come from RateResolver
// (RateTable + RateBand). No PH-specific monetary constants live in
// this file beyond the BIR semi-monthly threshold table key strings.
//
// Calculation flow (semi-monthly MVP):
//
//  1. monthly_basic = sum(ContractLines where kind=BASIC_SALARY,
//     using line.UnitPrice).
//  2. Earnings:
//     - earning_basic = half-cycle basic (monthly_basic/2 for first or
//     second half; full monthly_basic if HalfIndexFull).
//     - earning_allowance for each ALLOWANCE_TAXABLE /
//     ALLOWANCE_DE_MINIMIS line, prorated by half.
//  3. Statutory deductions (applied on the first half by default;
//     "full" cycle applies them once):
//     - SSS employee share: lookup fixed_amount by MSC bracket on
//     monthly_basic.
//     - PhilHealth: 5% of clamped(monthly_basic, [10k, 100k]) split
//     50/50; employee share = base × 2.5%.
//     - Pag-IBIG: fund_salary = min(monthly_basic, 10k); employee 1%
//     if monthly_basic ≤ 1,500 else 2% of fund_salary.
//  4. Withholding tax (BIR semi-monthly bracket):
//     taxable = half_basic + taxable_allowances − statutory_this_cycle
//     tax = bracket.fixed + (taxable - bracket.lower) ×
//     bracket.rate_basis_points / 10000.
//  5. Loan amortizations: emit deduction_loan per LOAN_AMORTIZATION
//     line at line.UnitPrice.
//
// Pin rate_table_id and applied_basis on every rate-driven line for
// audit. Net pay derivation lives at the orchestration layer.
package ph

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	suppliercontractlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_line"
	ratebandpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/rate_band"
	ratetablepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/rate_table"

	"github.com/erniealice/espyna-golang/internal/application/usecases/service/payroll/payrollcore"
)

const (
	complianceRegion = "PH"
	version          = "PH-2026.04"

	// PhilHealth floor / ceiling in centavos (PhilHealth Advisory
	// 2025-0002; ₱10,000 floor, ₱100,000 ceiling). These are FRAMING
	// constants, not rate values — they bound the salary base used
	// against the rate-table-driven 5% / 2.5% percentage. The actual
	// 5% / 2.5% / 50-50 split should ultimately also live in a rate
	// table; for MVP semi-monthly we hardcode the well-known
	// percentage split because RateBand cannot express "X% of clamped
	// salary" without a formula evaluator. Tracked as a follow-up.
	philHealthFloorCentavos       int64 = 10_000_00
	philHealthCeilingCentavos     int64 = 100_000_00
	philHealthEmployeeBasisPoints int64 = 250 // 2.5%

	// Pag-IBIG cap in centavos (Pag-IBIG Circular 460; ₱10,000 max
	// fund salary). Same MVP caveat as PhilHealth above.
	pagIBIGCapCentavos             int64 = 10_000_00
	pagIBIGLowSalaryThresholdCent  int64 = 1_500_00
	pagIBIGLowEmployeeBasisPoints  int64 = 100 // 1%
	pagIBIGHighEmployeeBasisPoints int64 = 200 // 2%
)

// Calculator implements payrollcore.PayrollCalculator for the
// Philippines.
type Calculator struct{}

// NewCalculator returns a PH calculator. Stateless; safe to share.
func NewCalculator() *Calculator { return &Calculator{} }

// ComplianceRegion returns "PH".
func (c *Calculator) ComplianceRegion() string { return complianceRegion }

// Version returns the compliance version pin "PH-2026.04".
func (c *Calculator) Version() string { return version }

// Calculate computes earnings, statutory deductions, withholding tax,
// and loan deductions for one employee × one cycle.
func (c *Calculator) Calculate(ctx context.Context, p *payrollcore.PayslipContext) ([]payrollcore.LineResolution, error) {
	if p == nil {
		return nil, fmt.Errorf("ph: nil PayslipContext")
	}
	if p.PayCycle == nil {
		return nil, fmt.Errorf("ph: PayslipContext.PayCycle is nil")
	}
	if p.RateResolver == nil {
		return nil, fmt.Errorf("ph: PayslipContext.RateResolver is nil")
	}

	asOf, err := parsePayDate(p.PayCycle.GetPayDate())
	if err != nil {
		return nil, fmt.Errorf("ph: parse pay_date: %w", err)
	}

	// 1) Aggregate basic salary + allowances + loans from contract lines.
	monthlyBasic := sumLinesOfKind(p.ContractLines, suppliercontractlinepb.SupplierContractLineKind_SUPPLIER_CONTRACT_LINE_KIND_BASIC_SALARY)
	taxableAllowances := allowancesByKind(p.ContractLines,
		suppliercontractlinepb.SupplierContractLineKind_SUPPLIER_CONTRACT_LINE_KIND_ALLOWANCE_TAXABLE)
	deMinimisAllowances := allowancesByKind(p.ContractLines,
		suppliercontractlinepb.SupplierContractLineKind_SUPPLIER_CONTRACT_LINE_KIND_ALLOWANCE_DE_MINIMIS)
	loanLines := linesOfKind(p.ContractLines,
		suppliercontractlinepb.SupplierContractLineKind_SUPPLIER_CONTRACT_LINE_KIND_LOAN_AMORTIZATION)

	// Half-cycle proration. Semi-monthly first/second halves split
	// the month in two; "full" applies the whole month at once.
	prorationFactor := halfFactor(p.PayFrequency, p.HalfIndex)
	halfBasic := int64(float64(monthlyBasic) * prorationFactor)

	results := make([]payrollcore.LineResolution, 0, 8)

	// 2) Earnings — basic.
	results = append(results, payrollcore.LineResolution{
		Description:     "Basic Salary",
		LineKind:        payrollcore.LineKindEarningBasic,
		Amount:          halfBasic,
		Quantity:        1,
		UnitPrice:       halfBasic,
		AppliedBasis:    monthlyBasic,
		ProrationFactor: prorationFactor,
		CalcMetadata:    mustJSON(map[string]any{"monthly_basic": monthlyBasic, "half_basic": halfBasic}),
	})

	// 2) Earnings — allowances (taxable + de-minimis treated alike at
	//    line emission; tax classification is the calculator's job
	//    via taxable_allowance accumulator below).
	taxableAllowanceTotal := emitAllowanceLines(&results, taxableAllowances, prorationFactor, true)
	emitAllowanceLines(&results, deMinimisAllowances, prorationFactor, false)

	// 3) Statutory deductions. Apply only on first / full half; second
	//    half collects nothing for MVP. The orchestration layer can
	//    override via a future "split" strategy.
	statutoryAppliedThisCycle := int64(0)
	if shouldApplyStatutory(p.HalfIndex) {
		sssLine, err := computeSSS(ctx, p, monthlyBasic, asOf)
		if err != nil {
			return nil, err
		}
		results = append(results, sssLine)
		statutoryAppliedThisCycle += sssLine.Amount

		phLine := computePhilHealth(monthlyBasic)
		results = append(results, phLine)
		statutoryAppliedThisCycle += phLine.Amount

		pgLine := computePagIBIG(monthlyBasic)
		results = append(results, pgLine)
		statutoryAppliedThisCycle += pgLine.Amount
	}

	// 4) Withholding tax — semi-monthly bracket.
	if p.PayFrequency == payrollcore.PayFrequencySemiMonthly || p.PayFrequency == payrollcore.PayFrequencyMonthly {
		taxableComp := halfBasic + taxableAllowanceTotal - statutoryAppliedThisCycle
		if taxableComp < 0 {
			taxableComp = 0
		}
		taxLine, err := computeBIRWithholding(ctx, p, taxableComp, asOf)
		if err != nil {
			return nil, err
		}
		if taxLine.Amount > 0 {
			results = append(results, taxLine)
		}
	}

	// 5) Loans — one deduction_loan per LOAN_AMORTIZATION line.
	for _, ln := range loanLines {
		results = append(results, payrollcore.LineResolution{
			Description:     descOrDefault(ln, "Loan Amortization"),
			LineKind:        payrollcore.LineKindDeductionLoan,
			Amount:          ln.GetUnitPrice(),
			Quantity:        1,
			UnitPrice:       ln.GetUnitPrice(),
			AppliedBasis:    monthlyBasic,
			ProrationFactor: 1.0,
			CalcMetadata:    mustJSON(map[string]any{"contract_line_id": ln.GetId()}),
		})
	}

	return results, nil
}

// computeSSS resolves the SSS_EMPLOYEE_SHARE rate table, finds the band
// whose range contains monthly_basic, and emits a fixed-amount
// deduction.
func computeSSS(ctx context.Context, p *payrollcore.PayslipContext, monthlyBasic int64, asOf time.Time) (payrollcore.LineResolution, error) {
	table, bands, err := p.RateResolver(ctx, payrollcore.RateKindSSSEmployeeShare, complianceRegion, asOf)
	if err != nil {
		return payrollcore.LineResolution{}, fmt.Errorf("ph: SSS rate lookup: %w", err)
	}
	band := findBand(bands, monthlyBasic)
	if band == nil {
		return payrollcore.LineResolution{}, fmt.Errorf("ph: SSS no band matches monthly_basic=%d", monthlyBasic)
	}
	if band.GetRateType() != "fixed" {
		return payrollcore.LineResolution{}, fmt.Errorf("ph: SSS band rate_type=%q (want fixed)", band.GetRateType())
	}
	amount := band.GetFixedAmountCentavos()
	meta := map[string]any{
		"msc_centavos":        band.GetUpperBoundCentavos(),
		"ee_share_centavos":   amount,
		"band_lower_centavos": band.GetLowerBoundCentavos(),
		"band_upper_centavos": band.GetUpperBoundCentavos(),
		"rate_table_id":       table.GetId(),
		"rate_table_version":  table.GetVersionLabel(),
	}
	return payrollcore.LineResolution{
		Description:     "SSS Employee Contribution",
		LineKind:        payrollcore.LineKindDeductionStatutory,
		Amount:          amount,
		Quantity:        1,
		UnitPrice:       amount,
		RateTableID:     table.GetId(),
		AppliedBasis:    monthlyBasic,
		ProrationFactor: 1.0,
		CalcMetadata:    mustJSON(meta),
	}, nil
}

// computePhilHealth applies the PhilHealth Advisory 2025-0002 formula:
// 5% of clamped(monthly_basic, [10k, 100k]) split 50/50.
func computePhilHealth(monthlyBasic int64) payrollcore.LineResolution {
	base := monthlyBasic
	if base < philHealthFloorCentavos {
		base = philHealthFloorCentavos
	}
	if base > philHealthCeilingCentavos {
		base = philHealthCeilingCentavos
	}
	// Employee share = base × 2.5% (basis_points 250 / 10000).
	amount := base * philHealthEmployeeBasisPoints / 10000
	meta := map[string]any{
		"premium_base_centavos":   base,
		"employee_basis_points":   philHealthEmployeeBasisPoints,
		"employee_share_centavos": amount,
		"floor_centavos":          philHealthFloorCentavos,
		"ceiling_centavos":        philHealthCeilingCentavos,
	}
	return payrollcore.LineResolution{
		Description:     "PhilHealth Employee Contribution",
		LineKind:        payrollcore.LineKindDeductionStatutory,
		Amount:          amount,
		Quantity:        1,
		UnitPrice:       amount,
		AppliedBasis:    monthlyBasic,
		ProrationFactor: 1.0,
		CalcMetadata:    mustJSON(meta),
	}
}

// computePagIBIG applies Pag-IBIG Circular 460:
//   - fund_salary = min(monthly_basic, 10k)
//   - employee = 1% of fund_salary if monthly_basic ≤ 1,500 else 2%
func computePagIBIG(monthlyBasic int64) payrollcore.LineResolution {
	fundSalary := monthlyBasic
	if fundSalary > pagIBIGCapCentavos {
		fundSalary = pagIBIGCapCentavos
	}
	bp := pagIBIGHighEmployeeBasisPoints
	if monthlyBasic <= pagIBIGLowSalaryThresholdCent {
		bp = pagIBIGLowEmployeeBasisPoints
	}
	amount := fundSalary * bp / 10000
	meta := map[string]any{
		"fund_salary_centavos":    fundSalary,
		"employee_basis_points":   bp,
		"employee_share_centavos": amount,
		"cap_centavos":            pagIBIGCapCentavos,
	}
	return payrollcore.LineResolution{
		Description:     "Pag-IBIG Employee Contribution",
		LineKind:        payrollcore.LineKindDeductionStatutory,
		Amount:          amount,
		Quantity:        1,
		UnitPrice:       amount,
		AppliedBasis:    monthlyBasic,
		ProrationFactor: 1.0,
		CalcMetadata:    mustJSON(meta),
	}
}

// computeBIRWithholding applies the BIR semi-monthly bracket
// (RR 11-2018 Annex E). Bracket math: tax = fixed + (taxable - lower)
// × rate_basis_points / 10000.
func computeBIRWithholding(ctx context.Context, p *payrollcore.PayslipContext, taxableComp int64, asOf time.Time) (payrollcore.LineResolution, error) {
	if taxableComp <= 0 {
		return payrollcore.LineResolution{}, nil
	}
	table, bands, err := p.RateResolver(ctx, payrollcore.RateKindBIRWithholdingSemiMonthly, complianceRegion, asOf)
	if err != nil {
		return payrollcore.LineResolution{}, fmt.Errorf("ph: BIR rate lookup: %w", err)
	}
	band := findBand(bands, taxableComp)
	if band == nil {
		return payrollcore.LineResolution{}, fmt.Errorf("ph: BIR no band matches taxable=%d", taxableComp)
	}
	switch band.GetRateType() {
	case "fixed":
		// Bracket 1 — zero or fixed-only.
		amount := band.GetFixedAmountCentavos()
		return payrollcore.LineResolution{
			Description:     "BIR Withholding Tax",
			LineKind:        payrollcore.LineKindDeductionTax,
			Amount:          amount,
			Quantity:        1,
			UnitPrice:       amount,
			RateTableID:     table.GetId(),
			AppliedBasis:    taxableComp,
			ProrationFactor: 1.0,
			CalcMetadata: mustJSON(map[string]any{
				"taxable_compensation_centavos": taxableComp,
				"bracket_lower_centavos":        band.GetLowerBoundCentavos(),
				"bracket_upper_centavos":        band.GetUpperBoundCentavos(),
				"rate_table_id":                 table.GetId(),
			}),
		}, nil
	case "percentage_of_excess":
		excess := taxableComp - band.GetLowerBoundCentavos()
		if excess < 0 {
			excess = 0
		}
		amount := band.GetFixedAmountCentavos() + (excess * int64(band.GetRateBasisPoints()) / 10000)
		return payrollcore.LineResolution{
			Description:     "BIR Withholding Tax",
			LineKind:        payrollcore.LineKindDeductionTax,
			Amount:          amount,
			Quantity:        1,
			UnitPrice:       amount,
			RateTableID:     table.GetId(),
			AppliedBasis:    taxableComp,
			ProrationFactor: 1.0,
			CalcMetadata: mustJSON(map[string]any{
				"taxable_compensation_centavos": taxableComp,
				"bracket_lower_centavos":        band.GetLowerBoundCentavos(),
				"bracket_upper_centavos":        band.GetUpperBoundCentavos(),
				"bracket_fixed_centavos":        band.GetFixedAmountCentavos(),
				"bracket_basis_points":          band.GetRateBasisPoints(),
				"excess_centavos":               excess,
				"rate_table_id":                 table.GetId(),
			}),
		}, nil
	default:
		return payrollcore.LineResolution{}, fmt.Errorf("ph: BIR unsupported rate_type=%q", band.GetRateType())
	}
}

// emitAllowanceLines appends one earning_allowance line per
// SupplierContractLine. Returns the prorated taxable total (only when
// taxable=true; de-minimis lines return 0 to the tax base).
func emitAllowanceLines(out *[]payrollcore.LineResolution, lines []*suppliercontractlinepb.SupplierContractLine, factor float64, taxable bool) int64 {
	taxableTotal := int64(0)
	for _, ln := range lines {
		monthly := ln.GetUnitPrice()
		amount := int64(float64(monthly) * factor)
		if taxable {
			taxableTotal += amount
		}
		*out = append(*out, payrollcore.LineResolution{
			Description:     descOrDefault(ln, "Allowance"),
			LineKind:        payrollcore.LineKindEarningAllowance,
			Amount:          amount,
			Quantity:        1,
			UnitPrice:       amount,
			AppliedBasis:    monthly,
			ProrationFactor: factor,
			CalcMetadata: mustJSON(map[string]any{
				"contract_line_id": ln.GetId(),
				"monthly_centavos": monthly,
				"taxable":          taxable,
			}),
		})
	}
	return taxableTotal
}

// findBand returns the first band whose [lower, upper] range contains
// value (upper is inclusive; if upper is unset the band is uncapped
// and matches anything ≥ lower).
func findBand(bands []*ratebandpb.RateBand, value int64) *ratebandpb.RateBand {
	for _, b := range bands {
		if value < b.GetLowerBoundCentavos() {
			continue
		}
		if b.UpperBoundCentavos == nil {
			return b
		}
		if value <= *b.UpperBoundCentavos {
			return b
		}
	}
	return nil
}

// halfFactor returns the proration factor applied to monthly amounts
// for the given (frequency, half_index) tuple. Semi-monthly first /
// second = 0.5; full = 1.0.
func halfFactor(frequency, half string) float64 {
	switch frequency {
	case payrollcore.PayFrequencySemiMonthly:
		switch half {
		case payrollcore.HalfIndexFirst, payrollcore.HalfIndexSecond:
			return 0.5
		default:
			return 1.0
		}
	case payrollcore.PayFrequencyMonthly:
		return 1.0
	default:
		// Weekly / biweekly aren't part of MVP; treat as full for now.
		return 1.0
	}
}

// shouldApplyStatutory says whether the current cycle collects the
// full-month statutory deductions. MVP rule: collect on first or full;
// second half collects nothing.
func shouldApplyStatutory(half string) bool {
	switch half {
	case payrollcore.HalfIndexFirst, payrollcore.HalfIndexFull, "":
		return true
	default:
		return false
	}
}

// sumLinesOfKind sums UnitPrice of every contract line whose
// SupplierContractLineKind matches.
func sumLinesOfKind(lines []*suppliercontractlinepb.SupplierContractLine, kind suppliercontractlinepb.SupplierContractLineKind) int64 {
	var total int64
	for _, ln := range lines {
		if ln.GetKind() == kind {
			total += ln.GetUnitPrice()
		}
	}
	return total
}

// linesOfKind filters lines by SupplierContractLineKind.
func linesOfKind(lines []*suppliercontractlinepb.SupplierContractLine, kind suppliercontractlinepb.SupplierContractLineKind) []*suppliercontractlinepb.SupplierContractLine {
	out := make([]*suppliercontractlinepb.SupplierContractLine, 0, len(lines))
	for _, ln := range lines {
		if ln.GetKind() == kind {
			out = append(out, ln)
		}
	}
	return out
}

// allowancesByKind is a thin alias of linesOfKind kept for readability
// at call sites.
func allowancesByKind(lines []*suppliercontractlinepb.SupplierContractLine, kind suppliercontractlinepb.SupplierContractLineKind) []*suppliercontractlinepb.SupplierContractLine {
	return linesOfKind(lines, kind)
}

// descOrDefault returns line.Description, falling back to def.
func descOrDefault(ln *suppliercontractlinepb.SupplierContractLine, def string) string {
	if d := ln.GetDescription(); d != "" {
		return d
	}
	return def
}

// parsePayDate accepts ISO 8601 YYYY-MM-DD; returns UTC midnight.
func parsePayDate(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("empty pay_date")
	}
	return time.Parse("2006-01-02", s)
}

// mustJSON marshals to JSON string; falls back to a minimal placeholder
// on encode failure (keys here are always JSON-safe so this should be
// unreachable, but we don't want a calculator panic on metadata).
func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return `{"error":"json_encode_failed"}`
	}
	return string(b)
}

// Compile-time assertion: *Calculator satisfies the interface.
var _ payrollcore.PayrollCalculator = (*Calculator)(nil)

// Sentinel — _ uses ratetablepb to keep the import even if all
// references are indirect. Some toolchains otherwise warn on the
// import; this is a structural import we need for the interface
// signature in payrollcore.
var _ = (*ratetablepb.RateTable)(nil)

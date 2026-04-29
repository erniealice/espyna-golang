// Package payrollcore holds the PayrollCalculator interface and the
// shared PayslipContext / LineResolution types.
//
// Why a sub-package: the parent `payroll` package is the registry that
// imports the per-jurisdiction implementations (ph, us, eu). Those
// implementations also need access to the shared types. Putting types
// here breaks the import cycle:
//
//	payroll  →  payroll/ph  →  payroll/payrollcore
//	payroll  →  payroll/payrollcore
//
// The parent `payroll` package re-exports the names via type aliases so
// callers can keep writing payroll.PayrollCalculator etc.
package payrollcore

import (
	"context"
	"time"

	suppliercontractpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract"
	suppliercontractlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_line"
	paycyclepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/pay_cycle"
	ratebandpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/rate_band"
	ratetablepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/rate_table"
)

// Line-kind controlled vocabulary. Calculators emit these strings on
// LineResolution.LineKind; the orchestration layer maps them to
// ExpenditureLineItem treatment + sign at materialize time.
const (
	LineKindEarningBasic       = "earning_basic"
	LineKindEarningAllowance   = "earning_allowance"
	LineKindDeductionStatutory = "deduction_statutory"
	LineKindDeductionTax       = "deduction_tax"
	LineKindDeductionLoan      = "deduction_loan"
	LineKindEmployerCost       = "employer_cost"
)

// Rate-table kind controlled vocabulary. Matches RateTable.kind seed
// values used by RateResolver lookups.
const (
	RateKindSSSEmployeeShare          = "SSS_EMPLOYEE_SHARE"
	RateKindSSSEmployerShare          = "SSS_EMPLOYER_SHARE"
	RateKindPhilHealthEmployeeShare   = "PHILHEALTH_EMPLOYEE_SHARE"
	RateKindPagIBIGEmployeeShare      = "PAGIBIG_EMPLOYEE_SHARE"
	RateKindBIRWithholdingSemiMonthly = "BIR_WITHHOLDING_SEMI_MONTHLY"
)

// Pay-frequency controlled vocabulary.
const (
	PayFrequencyWeekly      = "weekly"
	PayFrequencyBiweekly    = "biweekly"
	PayFrequencySemiMonthly = "semi_monthly"
	PayFrequencyMonthly     = "monthly"
)

// Half-index controlled vocabulary. Mirrors PayCycle.half_index.
const (
	HalfIndexFirst  = "first"
	HalfIndexSecond = "second"
	HalfIndexFull   = "full"
)

// LineResolution is one calculator output line. Becomes an
// ExpenditureLineItem at materialize time.
//
// Sign convention: amounts are unsigned (positive); sign relative to
// net pay is implied by LineKind. earning_* contributes to gross;
// deduction_* reduces net; employer_cost is informational.
type LineResolution struct {
	// Description is human-readable line text (translated at view layer).
	Description string

	// LineKind is one of the LineKind* constants.
	LineKind string

	// Amount is the line value in centavos (always non-negative).
	Amount int64

	// Quantity / UnitPrice carry through to ExpenditureLineItem.
	Quantity  float64
	UnitPrice int64

	// RateTableID pins the exact RateTable used for audit. Empty for
	// lines that don't depend on a rate lookup (e.g., basic salary).
	RateTableID string

	// AppliedBasis is the salary basis the calc applied to (e.g.,
	// monthly_basic for SSS, taxable compensation for BIR). Pinned for
	// reproducibility.
	AppliedBasis int64

	// ProrationFactor is the 0..1 fraction of a full period applied.
	ProrationFactor float64

	// CalcMetadata is JSON-encoded jurisdiction-specific data.
	CalcMetadata string
}

// PayslipContext is the full input needed to compute one employee's
// payslip for one cycle.
type PayslipContext struct {
	PayCycle           *paycyclepb.PayCycle
	EmployeeID         string
	EmploymentContract *suppliercontractpb.SupplierContract
	ContractLines      []*suppliercontractlinepb.SupplierContractLine

	// PayFrequency is one of the PayFrequency* constants.
	PayFrequency string

	// HalfIndex is one of the HalfIndex* constants.
	HalfIndex string

	// RateResolver returns the active RateTable + bands for (kind,
	// region) on the as-of date. The single dependency boundary that
	// lets us swap real DB lookup for in-memory test fixtures.
	RateResolver func(ctx context.Context, kind, region string, asOf time.Time) (*ratetablepb.RateTable, []*ratebandpb.RateBand, error)
}

// PayrollCalculator is the per-jurisdiction calculator interface. One
// implementation per compliance_region.
type PayrollCalculator interface {
	// ComplianceRegion returns the canonical region code.
	ComplianceRegion() string

	// Version returns the calculator's compliance version string,
	// pinned to LineResolutions for audit.
	Version() string

	// Calculate produces all LineResolutions for one employee × one
	// cycle, in calculation order.
	Calculate(ctx context.Context, p *PayslipContext) ([]LineResolution, error)
}

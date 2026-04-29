// Package payroll selects the per-jurisdiction PayrollCalculator and
// re-exports the shared types from payroll/payrollcore.
//
// The re-export is structural: the canonical definitions live in
// payrollcore (to avoid an import cycle between this package and the
// per-jurisdiction sub-packages). External callers should keep using
// payroll.PayrollCalculator etc.; the alias keeps that surface intact.
package payroll

import (
	"github.com/erniealice/espyna-golang/internal/application/services/payroll/payrollcore"
)

// Type aliases — canonical definitions in payrollcore.
type (
	PayrollCalculator = payrollcore.PayrollCalculator
	PayslipContext    = payrollcore.PayslipContext
	LineResolution    = payrollcore.LineResolution
)

// Re-exported line-kind vocabulary.
const (
	LineKindEarningBasic       = payrollcore.LineKindEarningBasic
	LineKindEarningAllowance   = payrollcore.LineKindEarningAllowance
	LineKindDeductionStatutory = payrollcore.LineKindDeductionStatutory
	LineKindDeductionTax       = payrollcore.LineKindDeductionTax
	LineKindDeductionLoan      = payrollcore.LineKindDeductionLoan
	LineKindEmployerCost       = payrollcore.LineKindEmployerCost
)

// Re-exported rate-table kind vocabulary.
const (
	RateKindSSSEmployeeShare          = payrollcore.RateKindSSSEmployeeShare
	RateKindSSSEmployerShare          = payrollcore.RateKindSSSEmployerShare
	RateKindPhilHealthEmployeeShare   = payrollcore.RateKindPhilHealthEmployeeShare
	RateKindPagIBIGEmployeeShare      = payrollcore.RateKindPagIBIGEmployeeShare
	RateKindBIRWithholdingSemiMonthly = payrollcore.RateKindBIRWithholdingSemiMonthly
)

// Re-exported pay-frequency vocabulary.
const (
	PayFrequencyWeekly      = payrollcore.PayFrequencyWeekly
	PayFrequencyBiweekly    = payrollcore.PayFrequencyBiweekly
	PayFrequencySemiMonthly = payrollcore.PayFrequencySemiMonthly
	PayFrequencyMonthly     = payrollcore.PayFrequencyMonthly
)

// Re-exported half-index vocabulary.
const (
	HalfIndexFirst  = payrollcore.HalfIndexFirst
	HalfIndexSecond = payrollcore.HalfIndexSecond
	HalfIndexFull   = payrollcore.HalfIndexFull
)

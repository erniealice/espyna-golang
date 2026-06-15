// Package us provides a placeholder PayrollCalculator for US states.
// Calculate panics with a NotImplemented message; the type satisfies
// the PayrollCalculator interface so the registry compiles.
package us

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/usecases/service/payroll/payrollcore"
)

// Calculator is the US stub. Region is the full code, e.g. "US-CA".
type Calculator struct {
	region string
}

// NewCalculator returns the US stub bound to a specific state region.
func NewCalculator(region string) *Calculator {
	return &Calculator{region: region}
}

// ComplianceRegion returns the full US-* region code.
func (c *Calculator) ComplianceRegion() string {
	return c.region
}

// Version returns the stub marker. US calculator is not implemented.
func (c *Calculator) Version() string {
	return "US-STUB"
}

// Calculate panics — US payroll is not implemented for the MVP.
func (c *Calculator) Calculate(ctx context.Context, p *payrollcore.PayslipContext) ([]payrollcore.LineResolution, error) {
	panic(fmt.Errorf("not implemented for compliance_region=%s", c.region))
}

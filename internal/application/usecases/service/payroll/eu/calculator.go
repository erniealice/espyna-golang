// Package eu provides a placeholder PayrollCalculator for EU countries.
// Calculate panics with a NotImplemented message; the type satisfies
// the PayrollCalculator interface so the registry compiles.
package eu

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/usecases/service/payroll/payrollcore"
)

// Calculator is the EU stub. Region is the full code, e.g. "EU-DE".
type Calculator struct {
	region string
}

// NewCalculator returns the EU stub bound to a specific country region.
func NewCalculator(region string) *Calculator {
	return &Calculator{region: region}
}

// ComplianceRegion returns the full EU-* region code.
func (c *Calculator) ComplianceRegion() string {
	return c.region
}

// Version returns the stub marker. EU calculator is not implemented.
func (c *Calculator) Version() string {
	return "EU-STUB"
}

// Calculate panics — EU payroll is not implemented for the MVP.
func (c *Calculator) Calculate(ctx context.Context, p *payrollcore.PayslipContext) ([]payrollcore.LineResolution, error) {
	panic(fmt.Errorf("not implemented for compliance_region=%s", c.region))
}

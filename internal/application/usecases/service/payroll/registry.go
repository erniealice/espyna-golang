package payroll

import (
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/usecases/service/payroll/eu"
	"github.com/erniealice/espyna-golang/internal/application/usecases/service/payroll/ph"
	"github.com/erniealice/espyna-golang/internal/application/usecases/service/payroll/us"
)

// Get returns the PayrollCalculator implementation for a compliance
// region. Selection rules:
//
//   - "PH" → real PH calculator
//   - "US-*" (any state) → US stub (panics on Calculate)
//   - "EU-*" (any country) → EU stub (panics on Calculate)
//   - anything else → panic
//
// Why panic on unknown region: workspace.compliance_region must be
// validated at workspace-creation time. Hitting this default branch
// means upstream validation is broken; silent fallback would hide it.
func Get(complianceRegion string) PayrollCalculator {
	switch {
	case complianceRegion == "PH":
		return ph.NewCalculator()
	case strings.HasPrefix(complianceRegion, "US-"):
		return us.NewCalculator(complianceRegion)
	case strings.HasPrefix(complianceRegion, "EU-"):
		return eu.NewCalculator(complianceRegion)
	default:
		panic(fmt.Errorf("no calculator for compliance_region=%q", complianceRegion))
	}
}

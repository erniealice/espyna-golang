package domain

import (
	"fmt"

	treasuryuc "github.com/erniealice/espyna-golang/internal/application/usecases/treasury"
	"github.com/erniealice/espyna-golang/internal/composition/contracts"
)

// ConfigureTreasuryDomain configures routes for the Treasury domain.
// Currently returns an empty route configuration since all legacy payment entities
// (Payment, PaymentAttribute, PaymentMethod, PaymentProfile) have been removed.
// Their functionality is superseded by Collection (money IN) and Disbursement (money OUT).
func ConfigureTreasuryDomain(treasuryUseCases *treasuryuc.TreasuryUseCases) contracts.DomainRouteConfiguration {
	if treasuryUseCases == nil {
		fmt.Printf("Treasury use cases is NIL\n")
		return contracts.DomainRouteConfiguration{
			Domain:  "treasury",
			Prefix:  "/treasury",
			Enabled: false,
			Routes:  []contracts.RouteConfiguration{},
		}
	}

	// No routes currently — legacy payment entities have been removed.
	// Collection and Disbursement domains handle active payment processing.
	return contracts.DomainRouteConfiguration{
		Domain:  "treasury",
		Prefix:  "/treasury",
		Enabled: false,
		Routes:  []contracts.RouteConfiguration{},
	}
}

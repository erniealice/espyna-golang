package domain

import (
	"fmt"

	paymentuc "github.com/erniealice/espyna-golang/internal/application/usecases/payment"
	"github.com/erniealice/espyna-golang/internal/composition/contracts"
)

// ConfigurePaymentDomain configures routes for the Payment domain.
// Currently returns an empty route configuration since all legacy payment entities
// (Payment, PaymentAttribute, PaymentMethod, PaymentProfile) have been removed.
// Their functionality is superseded by Collection (money IN) and Disbursement (money OUT).
func ConfigurePaymentDomain(paymentUseCases *paymentuc.PaymentUseCases) contracts.DomainRouteConfiguration {
	if paymentUseCases == nil {
		fmt.Printf("Payment use cases is NIL\n")
		return contracts.DomainRouteConfiguration{
			Domain:  "payment",
			Prefix:  "/payment",
			Enabled: false,
			Routes:  []contracts.RouteConfiguration{},
		}
	}

	// No routes currently — legacy payment entities have been removed.
	// Collection and Disbursement domains handle active payment processing.
	return contracts.DomainRouteConfiguration{
		Domain:  "payment",
		Prefix:  "/payment",
		Enabled: false,
		Routes:  []contracts.RouteConfiguration{},
	}
}

package payment

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
)

// PaymentRepositories contains payment domain repositories.
// The legacy Payment, PaymentAttribute, PaymentMethod, and PaymentProfile entities
// have been removed. Their functionality is superseded by:
//   - Collection (money IN) — revenue settlements
//   - Disbursement (money OUT) — expenditure settlements
//
// See the collection and disbursement domains for active payment processing.
type PaymentRepositories struct {
	// Reserved for future payment domain repositories if needed
}

// PaymentUseCases contains all payment-related use cases.
// Currently empty after removal of redundant Payment/PaymentAttribute/PaymentMethod/PaymentProfile
// entities that were superseded by Collection and Disbursement.
type PaymentUseCases struct {
	// All payment entities have been superseded by collection (money IN) and disbursement (money OUT)
	// See expenditure and revenue domains for the business transaction layer
}

// NewUseCases creates payment use cases.
// Currently returns an empty struct since all legacy payment entities have been removed.
func NewUseCases(
	repos PaymentRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idService ports.IDService,
) *PaymentUseCases {
	return &PaymentUseCases{}
}

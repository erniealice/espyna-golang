package domain

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/revenue"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializeRevenue creates all revenue use cases from provider repositories.
// Tax-compute wiring is done separately via the composition layer after both
// revenue and tax initializers have run (see usecases.go for the wiring step).
func InitializeRevenue(
	repos *domain.RevenueRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idSvc ports.IDService,
) (*revenue.RevenueUseCases, error) {
	return revenue.NewUseCases(
		revenue.RevenueRepositories{
			Revenue:          repos.Revenue,
			RevenueLineItem:  repos.RevenueLineItem,
			RevenueCategory:  repos.RevenueCategory,
			RevenueAttribute: repos.RevenueAttribute,
			DeferredRevenue:  repos.DeferredRevenue,
			RevenueTaxLine:   repos.RevenueTaxLine,
			PaymentTerm:      repos.PaymentTerm,
			RevenueRun:       repos.RevenueRun,
			Subscription:     repos.Subscription,
			PricePlan:        repos.PricePlan,
			ProductPricePlan: repos.ProductPricePlan,
			PriceSchedule:    repos.PriceSchedule,
			Client:           repos.Client,
			Workspace:        repos.Workspace,

			// Milestone-billing branch (Phase C — milestone-billing plan §3).
			BillingEvent:     repos.BillingEvent,
			JobTemplatePhase: repos.JobTemplatePhase,
			Job:              repos.Job,

			// Plan B Phase 5a — advance Collection branch.
			TreasuryCollection: repos.TreasuryCollection,
		},
		authSvc,
		txSvc,
		i18nSvc,
		idSvc,
	), nil
}

package domain

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/procurement"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializeProcurement creates all procurement use cases from provider repositories.
// This is composition logic — it wires infrastructure (providers) to application (use cases).
func InitializeProcurement(
	repos *domain.ProcurementRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idSvc ports.IDService,
) (*procurement.ProcurementUseCases, error) {
	return procurement.NewUseCases(
		procurement.ProcurementRepositories{
			CostSchedule:            repos.CostSchedule,
			SupplierPlan:            repos.SupplierPlan,
			CostPlan:                repos.CostPlan,
			SupplierProductPlan:     repos.SupplierProductPlan,
			SupplierProductCostPlan: repos.SupplierProductCostPlan,
			SupplierSubscription:    repos.SupplierSubscription,
			Workspace:               repos.Workspace,
		},
		authSvc,
		txSvc,
		i18nSvc,
		idSvc,
	), nil
}

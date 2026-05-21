package procurement

import (
	// Procurement use cases
	costPlanUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/procurement/cost_plan"
	costScheduleUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/procurement/cost_schedule"
	supplierPlanUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/procurement/supplier_plan"
	supplierProductCostPlanUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/procurement/supplier_product_cost_plan"
	supplierProductPlanUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/procurement/supplier_product_plan"
	supplierSubscriptionUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/procurement/supplier_subscription"

	// Application ports
	"github.com/erniealice/espyna-golang/internal/application/ports"

	// Protobuf domain services for procurement repositories
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
	costplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_plan"
	costschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_schedule"
	supplierplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_plan"
	supplierproductcostplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_product_cost_plan"
	supplierproductplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_product_plan"
	suppliersubscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_subscription"
)

// ProcurementRepositories contains all procurement domain repositories
type ProcurementRepositories struct {
	CostSchedule            costschedulepb.CostScheduleDomainServiceServer
	SupplierPlan            supplierplanpb.SupplierPlanDomainServiceServer
	CostPlan                costplanpb.CostPlanDomainServiceServer
	SupplierProductPlan     supplierproductplanpb.SupplierProductPlanDomainServiceServer
	SupplierProductCostPlan supplierproductcostplanpb.SupplierProductCostPlanDomainServiceServer
	SupplierSubscription    suppliersubscriptionpb.SupplierSubscriptionDomainServiceServer
	Workspace               workspacepb.WorkspaceDomainServiceServer // Cross-domain: currency hard-block
}

// ProcurementUseCases contains all procurement-related use cases
type ProcurementUseCases struct {
	CostSchedule            *costScheduleUseCases.UseCases
	SupplierPlan            *supplierPlanUseCases.UseCases
	CostPlan                *costPlanUseCases.UseCases
	SupplierProductPlan     *supplierProductPlanUseCases.UseCases
	SupplierProductCostPlan *supplierProductCostPlanUseCases.UseCases
	SupplierSubscription    *supplierSubscriptionUseCases.UseCases
}

// NewUseCases creates all procurement use cases with proper constructor injection
func NewUseCases(
	repos ProcurementRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idSvc ports.IDService,
) *ProcurementUseCases {
	costScheduleUC := costScheduleUseCases.NewUseCases(
		costScheduleUseCases.Repositories{CostSchedule: repos.CostSchedule},
		costScheduleUseCases.Services{AuthorizationService: authSvc, TransactionService: txSvc, TranslationService: i18nSvc, IDService: idSvc},
	)

	supplierPlanUC := supplierPlanUseCases.NewUseCases(
		supplierPlanUseCases.Repositories{SupplierPlan: repos.SupplierPlan},
		supplierPlanUseCases.Services{AuthorizationService: authSvc, TransactionService: txSvc, TranslationService: i18nSvc, IDService: idSvc},
	)

	costPlanUC := costPlanUseCases.NewUseCases(
		costPlanUseCases.Repositories{CostPlan: repos.CostPlan, Workspace: repos.Workspace},
		costPlanUseCases.Services{AuthorizationService: authSvc, TransactionService: txSvc, TranslationService: i18nSvc, IDService: idSvc},
	)

	supplierProductPlanUC := supplierProductPlanUseCases.NewUseCases(
		supplierProductPlanUseCases.Repositories{SupplierProductPlan: repos.SupplierProductPlan},
		supplierProductPlanUseCases.Services{AuthorizationService: authSvc, TransactionService: txSvc, TranslationService: i18nSvc, IDService: idSvc},
	)

	supplierProductCostPlanUC := supplierProductCostPlanUseCases.NewUseCases(
		supplierProductCostPlanUseCases.Repositories{SupplierProductCostPlan: repos.SupplierProductCostPlan},
		supplierProductCostPlanUseCases.Services{AuthorizationService: authSvc, TransactionService: txSvc, TranslationService: i18nSvc, IDService: idSvc},
	)

	supplierSubscriptionUC := supplierSubscriptionUseCases.NewUseCases(
		supplierSubscriptionUseCases.Repositories{SupplierSubscription: repos.SupplierSubscription, CostPlan: repos.CostPlan, Workspace: repos.Workspace},
		supplierSubscriptionUseCases.Services{AuthorizationService: authSvc, TransactionService: txSvc, TranslationService: i18nSvc, IDService: idSvc},
	)

	return &ProcurementUseCases{
		CostSchedule:            costScheduleUC,
		SupplierPlan:            supplierPlanUC,
		CostPlan:                costPlanUC,
		SupplierProductPlan:     supplierProductPlanUC,
		SupplierProductCostPlan: supplierProductCostPlanUC,
		SupplierSubscription:    supplierSubscriptionUC,
	}
}

package supplier_product_cost_plan

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	supplierproductcostplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_product_cost_plan"
)

type Repositories struct {
	SupplierProductCostPlan supplierproductcostplanpb.SupplierProductCostPlanDomainServiceServer
}

type Services struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

type UseCases struct {
	CreateSupplierProductCostPlan          *CreateSupplierProductCostPlanUseCase
	ReadSupplierProductCostPlan            *ReadSupplierProductCostPlanUseCase
	UpdateSupplierProductCostPlan          *UpdateSupplierProductCostPlanUseCase
	DeleteSupplierProductCostPlan          *DeleteSupplierProductCostPlanUseCase
	ListSupplierProductCostPlans           *ListSupplierProductCostPlansUseCase
	GetSupplierProductCostPlanListPageData *GetSupplierProductCostPlanListPageDataUseCase
	GetSupplierProductCostPlanItemPageData *GetSupplierProductCostPlanItemPageDataUseCase
}

func NewUseCases(repos Repositories, svcs Services) *UseCases {
	return &UseCases{
		CreateSupplierProductCostPlan: NewCreateSupplierProductCostPlanUseCase(
			CreateSupplierProductCostPlanRepositories{SupplierProductCostPlan: repos.SupplierProductCostPlan},
			CreateSupplierProductCostPlanServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService, IDService: svcs.IDService},
		),
		ReadSupplierProductCostPlan: NewReadSupplierProductCostPlanUseCase(
			ReadSupplierProductCostPlanRepositories{SupplierProductCostPlan: repos.SupplierProductCostPlan},
			ReadSupplierProductCostPlanServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
		UpdateSupplierProductCostPlan: NewUpdateSupplierProductCostPlanUseCase(
			UpdateSupplierProductCostPlanRepositories{SupplierProductCostPlan: repos.SupplierProductCostPlan},
			UpdateSupplierProductCostPlanServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
		DeleteSupplierProductCostPlan: NewDeleteSupplierProductCostPlanUseCase(
			DeleteSupplierProductCostPlanRepositories{SupplierProductCostPlan: repos.SupplierProductCostPlan},
			DeleteSupplierProductCostPlanServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
		ListSupplierProductCostPlans: NewListSupplierProductCostPlansUseCase(
			ListSupplierProductCostPlansRepositories{SupplierProductCostPlan: repos.SupplierProductCostPlan},
			ListSupplierProductCostPlansServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
		GetSupplierProductCostPlanListPageData: NewGetSupplierProductCostPlanListPageDataUseCase(
			GetSupplierProductCostPlanListPageDataRepositories{SupplierProductCostPlan: repos.SupplierProductCostPlan},
			GetSupplierProductCostPlanListPageDataServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
		GetSupplierProductCostPlanItemPageData: NewGetSupplierProductCostPlanItemPageDataUseCase(
			GetSupplierProductCostPlanItemPageDataRepositories{SupplierProductCostPlan: repos.SupplierProductCostPlan},
			GetSupplierProductCostPlanItemPageDataServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
	}
}

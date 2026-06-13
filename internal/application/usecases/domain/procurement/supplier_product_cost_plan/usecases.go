package supplier_product_cost_plan

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	supplierproductcostplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_product_cost_plan"
)

type Repositories struct {
	SupplierProductCostPlan supplierproductcostplanpb.SupplierProductCostPlanDomainServiceServer
}

type Services struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
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
			CreateSupplierProductCostPlanServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator, IDGenerator: svcs.IDGenerator},
		),
		ReadSupplierProductCostPlan: NewReadSupplierProductCostPlanUseCase(
			ReadSupplierProductCostPlanRepositories{SupplierProductCostPlan: repos.SupplierProductCostPlan},
			ReadSupplierProductCostPlanServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		UpdateSupplierProductCostPlan: NewUpdateSupplierProductCostPlanUseCase(
			UpdateSupplierProductCostPlanRepositories{SupplierProductCostPlan: repos.SupplierProductCostPlan},
			UpdateSupplierProductCostPlanServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		DeleteSupplierProductCostPlan: NewDeleteSupplierProductCostPlanUseCase(
			DeleteSupplierProductCostPlanRepositories{SupplierProductCostPlan: repos.SupplierProductCostPlan},
			DeleteSupplierProductCostPlanServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		ListSupplierProductCostPlans: NewListSupplierProductCostPlansUseCase(
			ListSupplierProductCostPlansRepositories{SupplierProductCostPlan: repos.SupplierProductCostPlan},
			ListSupplierProductCostPlansServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		GetSupplierProductCostPlanListPageData: NewGetSupplierProductCostPlanListPageDataUseCase(
			GetSupplierProductCostPlanListPageDataRepositories{SupplierProductCostPlan: repos.SupplierProductCostPlan},
			GetSupplierProductCostPlanListPageDataServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		GetSupplierProductCostPlanItemPageData: NewGetSupplierProductCostPlanItemPageDataUseCase(
			GetSupplierProductCostPlanItemPageDataRepositories{SupplierProductCostPlan: repos.SupplierProductCostPlan},
			GetSupplierProductCostPlanItemPageDataServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
	}
}

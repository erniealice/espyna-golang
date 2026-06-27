package supplier_product_plan

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	supplierproductplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_product_plan"
)

type Repositories struct {
	SupplierProductPlan supplierproductplanpb.SupplierProductPlanDomainServiceServer
}

type Services struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

type UseCases struct {
	CreateSupplierProductPlan          *CreateSupplierProductPlanUseCase
	ReadSupplierProductPlan            *ReadSupplierProductPlanUseCase
	UpdateSupplierProductPlan          *UpdateSupplierProductPlanUseCase
	DeleteSupplierProductPlan          *DeleteSupplierProductPlanUseCase
	ListSupplierProductPlans           *ListSupplierProductPlansUseCase
	GetSupplierProductPlanListPageData *GetSupplierProductPlanListPageDataUseCase
	GetSupplierProductPlanItemPageData *GetSupplierProductPlanItemPageDataUseCase
	ListBySupplierPlan                 *ListBySupplierPlanUseCase
}

func NewUseCases(repos Repositories, svcs Services) *UseCases {
	return &UseCases{
		CreateSupplierProductPlan: NewCreateSupplierProductPlanUseCase(
			CreateSupplierProductPlanRepositories{SupplierProductPlan: repos.SupplierProductPlan},
			CreateSupplierProductPlanServices{ActionGatekeeper: svcs.ActionGatekeeper, Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator, IDGenerator: svcs.IDGenerator},
		),
		ReadSupplierProductPlan: NewReadSupplierProductPlanUseCase(
			ReadSupplierProductPlanRepositories{SupplierProductPlan: repos.SupplierProductPlan},
			ReadSupplierProductPlanServices{ActionGatekeeper: svcs.ActionGatekeeper, Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		UpdateSupplierProductPlan: NewUpdateSupplierProductPlanUseCase(
			UpdateSupplierProductPlanRepositories{SupplierProductPlan: repos.SupplierProductPlan},
			UpdateSupplierProductPlanServices{ActionGatekeeper: svcs.ActionGatekeeper, Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		DeleteSupplierProductPlan: NewDeleteSupplierProductPlanUseCase(
			DeleteSupplierProductPlanRepositories{SupplierProductPlan: repos.SupplierProductPlan},
			DeleteSupplierProductPlanServices{ActionGatekeeper: svcs.ActionGatekeeper, Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		ListSupplierProductPlans: NewListSupplierProductPlansUseCase(
			ListSupplierProductPlansRepositories{SupplierProductPlan: repos.SupplierProductPlan},
			ListSupplierProductPlansServices{ActionGatekeeper: svcs.ActionGatekeeper, Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		GetSupplierProductPlanListPageData: NewGetSupplierProductPlanListPageDataUseCase(
			GetSupplierProductPlanListPageDataRepositories{SupplierProductPlan: repos.SupplierProductPlan},
			GetSupplierProductPlanListPageDataServices{ActionGatekeeper: svcs.ActionGatekeeper, Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		GetSupplierProductPlanItemPageData: NewGetSupplierProductPlanItemPageDataUseCase(
			GetSupplierProductPlanItemPageDataRepositories{SupplierProductPlan: repos.SupplierProductPlan},
			GetSupplierProductPlanItemPageDataServices{ActionGatekeeper: svcs.ActionGatekeeper, Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		ListBySupplierPlan: NewListBySupplierPlanUseCase(
			ListBySupplierPlanRepositories{SupplierProductPlan: repos.SupplierProductPlan},
			ListBySupplierPlanServices{ActionGatekeeper: svcs.ActionGatekeeper, Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
	}
}

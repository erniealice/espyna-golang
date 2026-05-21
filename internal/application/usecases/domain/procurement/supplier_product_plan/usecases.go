package supplier_product_plan

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	supplierproductplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_product_plan"
)

type Repositories struct {
	SupplierProductPlan supplierproductplanpb.SupplierProductPlanDomainServiceServer
}

type Services struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
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
			CreateSupplierProductPlanServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator, IDGenerator: svcs.IDGenerator},
		),
		ReadSupplierProductPlan: NewReadSupplierProductPlanUseCase(
			ReadSupplierProductPlanRepositories{SupplierProductPlan: repos.SupplierProductPlan},
			ReadSupplierProductPlanServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		UpdateSupplierProductPlan: NewUpdateSupplierProductPlanUseCase(
			UpdateSupplierProductPlanRepositories{SupplierProductPlan: repos.SupplierProductPlan},
			UpdateSupplierProductPlanServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		DeleteSupplierProductPlan: NewDeleteSupplierProductPlanUseCase(
			DeleteSupplierProductPlanRepositories{SupplierProductPlan: repos.SupplierProductPlan},
			DeleteSupplierProductPlanServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		ListSupplierProductPlans: NewListSupplierProductPlansUseCase(
			ListSupplierProductPlansRepositories{SupplierProductPlan: repos.SupplierProductPlan},
			ListSupplierProductPlansServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		GetSupplierProductPlanListPageData: NewGetSupplierProductPlanListPageDataUseCase(
			GetSupplierProductPlanListPageDataRepositories{SupplierProductPlan: repos.SupplierProductPlan},
			GetSupplierProductPlanListPageDataServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		GetSupplierProductPlanItemPageData: NewGetSupplierProductPlanItemPageDataUseCase(
			GetSupplierProductPlanItemPageDataRepositories{SupplierProductPlan: repos.SupplierProductPlan},
			GetSupplierProductPlanItemPageDataServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		ListBySupplierPlan: NewListBySupplierPlanUseCase(
			ListBySupplierPlanRepositories{SupplierProductPlan: repos.SupplierProductPlan},
			ListBySupplierPlanServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
	}
}

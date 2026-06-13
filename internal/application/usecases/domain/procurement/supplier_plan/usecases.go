package supplier_plan

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	supplierplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_plan"
)

type Repositories struct {
	SupplierPlan supplierplanpb.SupplierPlanDomainServiceServer
}

type Services struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

type UseCases struct {
	CreateSupplierPlan          *CreateSupplierPlanUseCase
	ReadSupplierPlan            *ReadSupplierPlanUseCase
	UpdateSupplierPlan          *UpdateSupplierPlanUseCase
	DeleteSupplierPlan          *DeleteSupplierPlanUseCase
	ListSupplierPlans           *ListSupplierPlansUseCase
	GetSupplierPlanListPageData *GetSupplierPlanListPageDataUseCase
	GetSupplierPlanItemPageData *GetSupplierPlanItemPageDataUseCase
	SearchSupplierPlansByName   *SearchSupplierPlansByNameUseCase
}

func NewUseCases(repos Repositories, svcs Services) *UseCases {
	return &UseCases{
		CreateSupplierPlan: NewCreateSupplierPlanUseCase(
			CreateSupplierPlanRepositories{SupplierPlan: repos.SupplierPlan},
			CreateSupplierPlanServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator, IDGenerator: svcs.IDGenerator},
		),
		ReadSupplierPlan: NewReadSupplierPlanUseCase(
			ReadSupplierPlanRepositories{SupplierPlan: repos.SupplierPlan},
			ReadSupplierPlanServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		UpdateSupplierPlan: NewUpdateSupplierPlanUseCase(
			UpdateSupplierPlanRepositories{SupplierPlan: repos.SupplierPlan},
			UpdateSupplierPlanServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		DeleteSupplierPlan: NewDeleteSupplierPlanUseCase(
			DeleteSupplierPlanRepositories{SupplierPlan: repos.SupplierPlan},
			DeleteSupplierPlanServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		ListSupplierPlans: NewListSupplierPlansUseCase(
			ListSupplierPlansRepositories{SupplierPlan: repos.SupplierPlan},
			ListSupplierPlansServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		GetSupplierPlanListPageData: NewGetSupplierPlanListPageDataUseCase(
			GetSupplierPlanListPageDataRepositories{SupplierPlan: repos.SupplierPlan},
			GetSupplierPlanListPageDataServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		GetSupplierPlanItemPageData: NewGetSupplierPlanItemPageDataUseCase(
			GetSupplierPlanItemPageDataRepositories{SupplierPlan: repos.SupplierPlan},
			GetSupplierPlanItemPageDataServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		SearchSupplierPlansByName: NewSearchSupplierPlansByNameUseCase(
			SearchSupplierPlansByNameRepositories{SupplierPlan: repos.SupplierPlan},
			SearchSupplierPlansByNameServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
	}
}

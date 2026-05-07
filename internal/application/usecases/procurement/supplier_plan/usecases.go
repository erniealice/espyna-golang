package supplier_plan

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	supplierplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_plan"
)

type Repositories struct {
	SupplierPlan supplierplanpb.SupplierPlanDomainServiceServer
}

type Services struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
			CreateSupplierPlanServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService, IDService: svcs.IDService},
		),
		ReadSupplierPlan: NewReadSupplierPlanUseCase(
			ReadSupplierPlanRepositories{SupplierPlan: repos.SupplierPlan},
			ReadSupplierPlanServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
		UpdateSupplierPlan: NewUpdateSupplierPlanUseCase(
			UpdateSupplierPlanRepositories{SupplierPlan: repos.SupplierPlan},
			UpdateSupplierPlanServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
		DeleteSupplierPlan: NewDeleteSupplierPlanUseCase(
			DeleteSupplierPlanRepositories{SupplierPlan: repos.SupplierPlan},
			DeleteSupplierPlanServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
		ListSupplierPlans: NewListSupplierPlansUseCase(
			ListSupplierPlansRepositories{SupplierPlan: repos.SupplierPlan},
			ListSupplierPlansServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
		GetSupplierPlanListPageData: NewGetSupplierPlanListPageDataUseCase(
			GetSupplierPlanListPageDataRepositories{SupplierPlan: repos.SupplierPlan},
			GetSupplierPlanListPageDataServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
		GetSupplierPlanItemPageData: NewGetSupplierPlanItemPageDataUseCase(
			GetSupplierPlanItemPageDataRepositories{SupplierPlan: repos.SupplierPlan},
			GetSupplierPlanItemPageDataServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
		SearchSupplierPlansByName: NewSearchSupplierPlansByNameUseCase(
			SearchSupplierPlansByNameRepositories{SupplierPlan: repos.SupplierPlan},
			SearchSupplierPlansByNameServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
	}
}

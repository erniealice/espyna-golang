package supplier_product_plan

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	supplierproductplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_product_plan"
)

type Repositories struct {
	SupplierProductPlan supplierproductplanpb.SupplierProductPlanDomainServiceServer
}

type Services struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
			CreateSupplierProductPlanServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService, IDService: svcs.IDService},
		),
		ReadSupplierProductPlan: NewReadSupplierProductPlanUseCase(
			ReadSupplierProductPlanRepositories{SupplierProductPlan: repos.SupplierProductPlan},
			ReadSupplierProductPlanServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
		UpdateSupplierProductPlan: NewUpdateSupplierProductPlanUseCase(
			UpdateSupplierProductPlanRepositories{SupplierProductPlan: repos.SupplierProductPlan},
			UpdateSupplierProductPlanServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
		DeleteSupplierProductPlan: NewDeleteSupplierProductPlanUseCase(
			DeleteSupplierProductPlanRepositories{SupplierProductPlan: repos.SupplierProductPlan},
			DeleteSupplierProductPlanServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
		ListSupplierProductPlans: NewListSupplierProductPlansUseCase(
			ListSupplierProductPlansRepositories{SupplierProductPlan: repos.SupplierProductPlan},
			ListSupplierProductPlansServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
		GetSupplierProductPlanListPageData: NewGetSupplierProductPlanListPageDataUseCase(
			GetSupplierProductPlanListPageDataRepositories{SupplierProductPlan: repos.SupplierProductPlan},
			GetSupplierProductPlanListPageDataServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
		GetSupplierProductPlanItemPageData: NewGetSupplierProductPlanItemPageDataUseCase(
			GetSupplierProductPlanItemPageDataRepositories{SupplierProductPlan: repos.SupplierProductPlan},
			GetSupplierProductPlanItemPageDataServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
		ListBySupplierPlan: NewListBySupplierPlanUseCase(
			ListBySupplierPlanRepositories{SupplierProductPlan: repos.SupplierProductPlan},
			ListBySupplierPlanServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
	}
}

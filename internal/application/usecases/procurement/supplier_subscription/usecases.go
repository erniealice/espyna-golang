package supplier_subscription

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	costplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_plan"
	suppliersubscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_subscription"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
)

// Repositories groups all repository dependencies for supplier_subscription use cases
type Repositories struct {
	SupplierSubscription suppliersubscriptionpb.SupplierSubscriptionDomainServiceServer
	CostPlan             costplanpb.CostPlanDomainServiceServer  // Cross-domain: currency hard-block on create
	Workspace            workspacepb.WorkspaceDomainServiceServer // Cross-domain: currency hard-block on create
}

// Services groups all business service dependencies
type Services struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all supplier_subscription-related use cases
type UseCases struct {
	CreateSupplierSubscription          *CreateSupplierSubscriptionUseCase
	ReadSupplierSubscription            *ReadSupplierSubscriptionUseCase
	UpdateSupplierSubscription          *UpdateSupplierSubscriptionUseCase
	DeleteSupplierSubscription          *DeleteSupplierSubscriptionUseCase
	ListSupplierSubscriptions           *ListSupplierSubscriptionsUseCase
	GetSupplierSubscriptionListPageData *GetSupplierSubscriptionListPageDataUseCase
	GetSupplierSubscriptionItemPageData *GetSupplierSubscriptionItemPageDataUseCase
	CountActiveBySupplierIds            *CountActiveBySupplierIdsUseCase
	ListSupplierSubscriptionsByCostPlan *ListSupplierSubscriptionsByCostPlanUseCase
}

// NewUseCases creates a new collection of supplier_subscription use cases
func NewUseCases(repos Repositories, svcs Services) *UseCases {
	return &UseCases{
		CreateSupplierSubscription: NewCreateSupplierSubscriptionUseCase(
			CreateSupplierSubscriptionRepositories{SupplierSubscription: repos.SupplierSubscription, CostPlan: repos.CostPlan, Workspace: repos.Workspace},
			CreateSupplierSubscriptionServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService, IDService: svcs.IDService},
		),
		ReadSupplierSubscription: NewReadSupplierSubscriptionUseCase(
			ReadSupplierSubscriptionRepositories{SupplierSubscription: repos.SupplierSubscription},
			ReadSupplierSubscriptionServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
		UpdateSupplierSubscription: NewUpdateSupplierSubscriptionUseCase(
			UpdateSupplierSubscriptionRepositories{SupplierSubscription: repos.SupplierSubscription},
			UpdateSupplierSubscriptionServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
		DeleteSupplierSubscription: NewDeleteSupplierSubscriptionUseCase(
			DeleteSupplierSubscriptionRepositories{SupplierSubscription: repos.SupplierSubscription},
			DeleteSupplierSubscriptionServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
		ListSupplierSubscriptions: NewListSupplierSubscriptionsUseCase(
			ListSupplierSubscriptionsRepositories{SupplierSubscription: repos.SupplierSubscription},
			ListSupplierSubscriptionsServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
		GetSupplierSubscriptionListPageData: NewGetSupplierSubscriptionListPageDataUseCase(
			GetSupplierSubscriptionListPageDataRepositories{SupplierSubscription: repos.SupplierSubscription},
			GetSupplierSubscriptionListPageDataServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
		GetSupplierSubscriptionItemPageData: NewGetSupplierSubscriptionItemPageDataUseCase(
			GetSupplierSubscriptionItemPageDataRepositories{SupplierSubscription: repos.SupplierSubscription},
			GetSupplierSubscriptionItemPageDataServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
		CountActiveBySupplierIds: NewCountActiveBySupplierIdsUseCase(
			CountActiveBySupplierIdsRepositories{SupplierSubscription: repos.SupplierSubscription},
			CountActiveBySupplierIdsServices{AuthorizationService: svcs.AuthorizationService, TranslationService: svcs.TranslationService},
		),
		ListSupplierSubscriptionsByCostPlan: NewListSupplierSubscriptionsByCostPlanUseCase(
			ListSupplierSubscriptionsByCostPlanRepositories{SupplierSubscription: repos.SupplierSubscription},
			ListSupplierSubscriptionsByCostPlanServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
	}
}

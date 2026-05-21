package supplier_subscription

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
	costplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_plan"
	suppliersubscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_subscription"
)

// Repositories groups all repository dependencies for supplier_subscription use cases
type Repositories struct {
	SupplierSubscription suppliersubscriptionpb.SupplierSubscriptionDomainServiceServer
	CostPlan             costplanpb.CostPlanDomainServiceServer   // Cross-domain: currency hard-block on create
	Workspace            workspacepb.WorkspaceDomainServiceServer // Cross-domain: currency hard-block on create
}

// Services groups all business service dependencies
type Services struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
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
			CreateSupplierSubscriptionServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator, IDGenerator: svcs.IDGenerator},
		),
		ReadSupplierSubscription: NewReadSupplierSubscriptionUseCase(
			ReadSupplierSubscriptionRepositories{SupplierSubscription: repos.SupplierSubscription},
			ReadSupplierSubscriptionServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		UpdateSupplierSubscription: NewUpdateSupplierSubscriptionUseCase(
			UpdateSupplierSubscriptionRepositories{SupplierSubscription: repos.SupplierSubscription},
			UpdateSupplierSubscriptionServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		DeleteSupplierSubscription: NewDeleteSupplierSubscriptionUseCase(
			DeleteSupplierSubscriptionRepositories{SupplierSubscription: repos.SupplierSubscription},
			DeleteSupplierSubscriptionServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		ListSupplierSubscriptions: NewListSupplierSubscriptionsUseCase(
			ListSupplierSubscriptionsRepositories{SupplierSubscription: repos.SupplierSubscription},
			ListSupplierSubscriptionsServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		GetSupplierSubscriptionListPageData: NewGetSupplierSubscriptionListPageDataUseCase(
			GetSupplierSubscriptionListPageDataRepositories{SupplierSubscription: repos.SupplierSubscription},
			GetSupplierSubscriptionListPageDataServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		GetSupplierSubscriptionItemPageData: NewGetSupplierSubscriptionItemPageDataUseCase(
			GetSupplierSubscriptionItemPageDataRepositories{SupplierSubscription: repos.SupplierSubscription},
			GetSupplierSubscriptionItemPageDataServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		CountActiveBySupplierIds: NewCountActiveBySupplierIdsUseCase(
			CountActiveBySupplierIdsRepositories{SupplierSubscription: repos.SupplierSubscription},
			CountActiveBySupplierIdsServices{Authorizer: svcs.Authorizer, Translator: svcs.Translator},
		),
		ListSupplierSubscriptionsByCostPlan: NewListSupplierSubscriptionsByCostPlanUseCase(
			ListSupplierSubscriptionsByCostPlanRepositories{SupplierSubscription: repos.SupplierSubscription},
			ListSupplierSubscriptionsByCostPlanServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
	}
}

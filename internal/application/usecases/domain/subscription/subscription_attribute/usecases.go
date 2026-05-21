package subscription_attribute

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
	subscriptionattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_attribute"
)

// UseCases contains all subscription attribute-related use cases
type UseCases struct {
	CreateSubscriptionAttribute          *CreateSubscriptionAttributeUseCase
	CreateSubscriptionAttributesByCode   *CreateSubscriptionAttributesByCodeUseCase
	ReadSubscriptionAttribute            *ReadSubscriptionAttributeUseCase
	UpdateSubscriptionAttribute          *UpdateSubscriptionAttributeUseCase
	DeleteSubscriptionAttribute          *DeleteSubscriptionAttributeUseCase
	ListSubscriptionAttributes           *ListSubscriptionAttributesUseCase
	GetSubscriptionAttributeListPageData *GetSubscriptionAttributeListPageDataUseCase
	GetSubscriptionAttributeItemPageData *GetSubscriptionAttributeItemPageDataUseCase
}

// SubscriptionAttributeRepositories groups all repository dependencies for subscription attribute use cases
type SubscriptionAttributeRepositories struct {
	SubscriptionAttribute subscriptionattributepb.SubscriptionAttributeDomainServiceServer // Primary entity repository
	Subscription          subscriptionpb.SubscriptionDomainServiceServer                   // Entity reference validation
	Attribute             attributepb.AttributeDomainServiceServer                         // Entity reference validation
}

// SubscriptionAttributeServices groups all business service dependencies for subscription attribute use cases
type SubscriptionAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// NewUseCases creates a new collection of subscription attribute use cases
func NewUseCases(
	repositories SubscriptionAttributeRepositories,
	services SubscriptionAttributeServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateSubscriptionAttributeRepositories(repositories)
	createServices := CreateSubscriptionAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadSubscriptionAttributeRepositories{
		SubscriptionAttribute: repositories.SubscriptionAttribute,
	}
	readServices := ReadSubscriptionAttributeServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	updateRepos := UpdateSubscriptionAttributeRepositories{
		SubscriptionAttribute: repositories.SubscriptionAttribute,
		Subscription:          repositories.Subscription,
		Attribute:             repositories.Attribute,
	}
	updateServices := UpdateSubscriptionAttributeServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	deleteRepos := DeleteSubscriptionAttributeRepositories{
		SubscriptionAttribute: repositories.SubscriptionAttribute,
	}
	deleteServices := DeleteSubscriptionAttributeServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	listRepos := ListSubscriptionAttributesRepositories{
		SubscriptionAttribute: repositories.SubscriptionAttribute,
	}
	listServices := ListSubscriptionAttributesServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	getListPageDataRepos := GetSubscriptionAttributeListPageDataRepositories{
		SubscriptionAttribute: repositories.SubscriptionAttribute,
	}
	getListPageDataServices := GetSubscriptionAttributeListPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	getItemPageDataRepos := GetSubscriptionAttributeItemPageDataRepositories{
		SubscriptionAttribute: repositories.SubscriptionAttribute,
	}
	getItemPageDataServices := GetSubscriptionAttributeItemPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	createUseCase := NewCreateSubscriptionAttributeUseCase(createRepos, createServices)

	// Build repos for code-based creation (uses Attribute repo for code-to-ID resolution)
	createByCodeRepos := CreateSubscriptionAttributesByCodeRepositories{
		SubscriptionAttribute: repositories.SubscriptionAttribute,
		Attribute:             repositories.Attribute,
	}

	return &UseCases{
		CreateSubscriptionAttribute:          createUseCase,
		CreateSubscriptionAttributesByCode:   NewCreateSubscriptionAttributesByCodeUseCase(createByCodeRepos, createUseCase),
		ReadSubscriptionAttribute:            NewReadSubscriptionAttributeUseCase(readRepos, readServices),
		UpdateSubscriptionAttribute:          NewUpdateSubscriptionAttributeUseCase(updateRepos, updateServices),
		DeleteSubscriptionAttribute:          NewDeleteSubscriptionAttributeUseCase(deleteRepos, deleteServices),
		ListSubscriptionAttributes:           NewListSubscriptionAttributesUseCase(listRepos, listServices),
		GetSubscriptionAttributeListPageData: NewGetSubscriptionAttributeListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetSubscriptionAttributeItemPageData: NewGetSubscriptionAttributeItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}

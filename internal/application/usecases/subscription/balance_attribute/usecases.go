package balance_attribute

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	balancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance"
	balanceattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance_attribute"
)

// UseCases contains all balance attribute-related use cases
type UseCases struct {
	CreateBalanceAttribute          *CreateBalanceAttributeUseCase
	ReadBalanceAttribute            *ReadBalanceAttributeUseCase
	UpdateBalanceAttribute          *UpdateBalanceAttributeUseCase
	DeleteBalanceAttribute          *DeleteBalanceAttributeUseCase
	ListBalanceAttributes           *ListBalanceAttributesUseCase
	GetBalanceAttributeListPageData *GetBalanceAttributeListPageDataUseCase
	GetBalanceAttributeItemPageData *GetBalanceAttributeItemPageDataUseCase
}

// BalanceAttributeRepositories groups all repository dependencies for balance attribute use cases
type BalanceAttributeRepositories struct {
	BalanceAttribute balanceattributepb.BalanceAttributeDomainServiceServer // Primary entity repository
	Balance          balancepb.BalanceDomainServiceServer                   // Entity reference validation
	Attribute        attributepb.AttributeDomainServiceServer               // Entity reference validation
}

// BalanceAttributeServices groups all business service dependencies for balance attribute use cases
type BalanceAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// NewUseCases creates a new collection of balance attribute use cases
func NewUseCases(
	repositories BalanceAttributeRepositories,
	services BalanceAttributeServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateBalanceAttributeRepositories(repositories)
	createServices := CreateBalanceAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadBalanceAttributeRepositories{
		BalanceAttribute: repositories.BalanceAttribute,
	}
	readServices := ReadBalanceAttributeServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	updateRepos := UpdateBalanceAttributeRepositories{
		BalanceAttribute: repositories.BalanceAttribute,
		Balance:          repositories.Balance,
		Attribute:        repositories.Attribute,
	}
	updateServices := UpdateBalanceAttributeServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	deleteRepos := DeleteBalanceAttributeRepositories{
		BalanceAttribute: repositories.BalanceAttribute,
	}
	deleteServices := DeleteBalanceAttributeServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	listRepos := ListBalanceAttributesRepositories{
		BalanceAttribute: repositories.BalanceAttribute,
	}
	listServices := ListBalanceAttributesServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	getListPageDataRepos := GetBalanceAttributeListPageDataRepositories{
		BalanceAttribute: repositories.BalanceAttribute,
	}
	getListPageDataServices := GetBalanceAttributeListPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	getItemPageDataRepos := GetBalanceAttributeItemPageDataRepositories{
		BalanceAttribute: repositories.BalanceAttribute,
	}
	getItemPageDataServices := GetBalanceAttributeItemPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	return &UseCases{
		CreateBalanceAttribute:          NewCreateBalanceAttributeUseCase(createRepos, createServices),
		ReadBalanceAttribute:            NewReadBalanceAttributeUseCase(readRepos, readServices),
		UpdateBalanceAttribute:          NewUpdateBalanceAttributeUseCase(updateRepos, updateServices),
		DeleteBalanceAttribute:          NewDeleteBalanceAttributeUseCase(deleteRepos, deleteServices),
		ListBalanceAttributes:           NewListBalanceAttributesUseCase(listRepos, listServices),
		GetBalanceAttributeListPageData: NewGetBalanceAttributeListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetBalanceAttributeItemPageData: NewGetBalanceAttributeItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}

package balance_attribute

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
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
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// NewUseCases creates a new collection of balance attribute use cases
func NewUseCases(
	repositories BalanceAttributeRepositories,
	services BalanceAttributeServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateBalanceAttributeRepositories(repositories)
	createServices := CreateBalanceAttributeServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadBalanceAttributeRepositories{
		BalanceAttribute: repositories.BalanceAttribute,
	}
	readServices := ReadBalanceAttributeServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateBalanceAttributeRepositories{
		BalanceAttribute: repositories.BalanceAttribute,
		Balance:          repositories.Balance,
		Attribute:        repositories.Attribute,
	}
	updateServices := UpdateBalanceAttributeServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteBalanceAttributeRepositories{
		BalanceAttribute: repositories.BalanceAttribute,
	}
	deleteServices := DeleteBalanceAttributeServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListBalanceAttributesRepositories{
		BalanceAttribute: repositories.BalanceAttribute,
	}
	listServices := ListBalanceAttributesServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := GetBalanceAttributeListPageDataRepositories{
		BalanceAttribute: repositories.BalanceAttribute,
	}
	getListPageDataServices := GetBalanceAttributeListPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getItemPageDataRepos := GetBalanceAttributeItemPageDataRepositories{
		BalanceAttribute: repositories.BalanceAttribute,
	}
	getItemPageDataServices := GetBalanceAttributeItemPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Transactor: services.Transactor,
		Translator: services.Translator,
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

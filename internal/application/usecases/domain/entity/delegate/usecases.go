package delegate

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	delegatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate"
)

// DelegateRepositories groups all repository dependencies for delegate use cases
type DelegateRepositories struct {
	Delegate delegatepb.DelegateDomainServiceServer // Primary entity repository
}

// DelegateServices groups all business service dependencies for delegate use cases
type DelegateServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases contains all delegate-related use cases
type UseCases struct {
	CreateDelegate          *CreateDelegateUseCase
	ReadDelegate            *ReadDelegateUseCase
	UpdateDelegate          *UpdateDelegateUseCase
	DeleteDelegate          *DeleteDelegateUseCase
	ListDelegates           *ListDelegatesUseCase
	GetDelegateListPageData *GetDelegateListPageDataUseCase
	GetDelegateItemPageData *GetDelegateItemPageDataUseCase
}

// NewUseCases creates a new collection of delegate use cases
func NewUseCases(
	repositories DelegateRepositories,
	services DelegateServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateDelegateRepositories(repositories)
	createServices := CreateDelegateServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadDelegateRepositories(repositories)
	readServices := ReadDelegateServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateDelegateRepositories(repositories)
	updateServices := UpdateDelegateServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteDelegateRepositories(repositories)
	deleteServices := DeleteDelegateServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListDelegatesRepositories(repositories)
	listServices := ListDelegatesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := GetDelegateListPageDataRepositories(repositories)
	getListPageDataServices := GetDelegateListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getItemPageDataRepos := GetDelegateItemPageDataRepositories(repositories)
	getItemPageDataServices := GetDelegateItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateDelegate:          NewCreateDelegateUseCase(createRepos, createServices),
		ReadDelegate:            NewReadDelegateUseCase(readRepos, readServices),
		UpdateDelegate:          NewUpdateDelegateUseCase(updateRepos, updateServices),
		DeleteDelegate:          NewDeleteDelegateUseCase(deleteRepos, deleteServices),
		ListDelegates:           NewListDelegatesUseCase(listRepos, listServices),
		GetDelegateListPageData: NewGetDelegateListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetDelegateItemPageData: NewGetDelegateItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}

// NewUseCasesUngrouped creates a new collection of delegate use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(delegateRepo delegatepb.DelegateDomainServiceServer) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := DelegateRepositories{
		Delegate: delegateRepo,
	}

	services := DelegateServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewUseCases(repositories, services)
}

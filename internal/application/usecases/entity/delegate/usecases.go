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
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadDelegateRepositories(repositories)
	readServices := ReadDelegateServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateDelegateRepositories(repositories)
	updateServices := UpdateDelegateServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteDelegateRepositories(repositories)
	deleteServices := DeleteDelegateServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListDelegatesRepositories(repositories)
	listServices := ListDelegatesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetDelegateListPageDataRepositories(repositories)
	getListPageDataServices := GetDelegateListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getItemPageDataRepos := GetDelegateItemPageDataRepositories(repositories)
	getItemPageDataServices := GetDelegateItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
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
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUseCases(repositories, services)
}

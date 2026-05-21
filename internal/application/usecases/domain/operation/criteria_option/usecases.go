package criteria_option

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/criteria_option"
)

// CriteriaOptionRepositories groups all repository dependencies
type CriteriaOptionRepositories struct {
	CriteriaOption pb.CriteriaOptionDomainServiceServer
}

// CriteriaOptionServices groups all business service dependencies
type CriteriaOptionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all criteria_option-related use cases
type UseCases struct {
	CreateCriteriaOption          *CreateCriteriaOptionUseCase
	ReadCriteriaOption            *ReadCriteriaOptionUseCase
	UpdateCriteriaOption          *UpdateCriteriaOptionUseCase
	DeleteCriteriaOption          *DeleteCriteriaOptionUseCase
	ListCriteriaOptions           *ListCriteriaOptionsUseCase
	GetCriteriaOptionListPageData *GetCriteriaOptionListPageDataUseCase
	GetCriteriaOptionItemPageData *GetCriteriaOptionItemPageDataUseCase
	ListByCriteria                *ListByCriteriaUseCase
}

// NewUseCases creates a new collection of criteria_option use cases
func NewUseCases(
	repositories CriteriaOptionRepositories,
	services CriteriaOptionServices,
) *UseCases {
	createRepos := CreateCriteriaOptionRepositories{
		CriteriaOption: repositories.CriteriaOption,
	}
	createServices := CreateCriteriaOptionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadCriteriaOptionRepositories{
		CriteriaOption: repositories.CriteriaOption,
	}
	readServices := ReadCriteriaOptionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateCriteriaOptionRepositories{
		CriteriaOption: repositories.CriteriaOption,
	}
	updateServices := UpdateCriteriaOptionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteCriteriaOptionRepositories{
		CriteriaOption: repositories.CriteriaOption,
	}
	deleteServices := DeleteCriteriaOptionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListCriteriaOptionsRepositories{
		CriteriaOption: repositories.CriteriaOption,
	}
	listServices := ListCriteriaOptionsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listPageDataRepos := GetCriteriaOptionListPageDataRepositories{
		CriteriaOption: repositories.CriteriaOption,
	}
	listPageDataServices := GetCriteriaOptionListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	itemPageDataRepos := GetCriteriaOptionItemPageDataRepositories{
		CriteriaOption: repositories.CriteriaOption,
	}
	itemPageDataServices := GetCriteriaOptionItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listByCriteriaRepos := ListByCriteriaRepositories{
		CriteriaOption: repositories.CriteriaOption,
	}
	listByCriteriaServices := ListByCriteriaServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateCriteriaOption:          NewCreateCriteriaOptionUseCase(createRepos, createServices),
		ReadCriteriaOption:            NewReadCriteriaOptionUseCase(readRepos, readServices),
		UpdateCriteriaOption:          NewUpdateCriteriaOptionUseCase(updateRepos, updateServices),
		DeleteCriteriaOption:          NewDeleteCriteriaOptionUseCase(deleteRepos, deleteServices),
		ListCriteriaOptions:           NewListCriteriaOptionsUseCase(listRepos, listServices),
		GetCriteriaOptionListPageData: NewGetCriteriaOptionListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetCriteriaOptionItemPageData: NewGetCriteriaOptionItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
		ListByCriteria:                NewListByCriteriaUseCase(listByCriteriaRepos, listByCriteriaServices),
	}
}

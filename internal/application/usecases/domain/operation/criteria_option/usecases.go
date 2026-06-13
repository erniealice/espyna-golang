package criteria_option

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/criteria_option"
)

// CriteriaOptionRepositories groups all repository dependencies
type CriteriaOptionRepositories struct {
	CriteriaOption pb.CriteriaOptionDomainServiceServer
}

// CriteriaOptionServices groups all business service dependencies
type CriteriaOptionServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
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
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadCriteriaOptionRepositories{
		CriteriaOption: repositories.CriteriaOption,
	}
	readServices := ReadCriteriaOptionServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateCriteriaOptionRepositories{
		CriteriaOption: repositories.CriteriaOption,
	}
	updateServices := UpdateCriteriaOptionServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteCriteriaOptionRepositories{
		CriteriaOption: repositories.CriteriaOption,
	}
	deleteServices := DeleteCriteriaOptionServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListCriteriaOptionsRepositories{
		CriteriaOption: repositories.CriteriaOption,
	}
	listServices := ListCriteriaOptionsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listPageDataRepos := GetCriteriaOptionListPageDataRepositories{
		CriteriaOption: repositories.CriteriaOption,
	}
	listPageDataServices := GetCriteriaOptionListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	itemPageDataRepos := GetCriteriaOptionItemPageDataRepositories{
		CriteriaOption: repositories.CriteriaOption,
	}
	itemPageDataServices := GetCriteriaOptionItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listByCriteriaRepos := ListByCriteriaRepositories{
		CriteriaOption: repositories.CriteriaOption,
	}
	listByCriteriaServices := ListByCriteriaServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
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

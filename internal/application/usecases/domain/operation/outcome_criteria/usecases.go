package outcome_criteria

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
)

// OutcomeCriteriaRepositories groups all repository dependencies
type OutcomeCriteriaRepositories struct {
	OutcomeCriteria pb.OutcomeCriteriaDomainServiceServer
}

// OutcomeCriteriaServices groups all business service dependencies
type OutcomeCriteriaServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all outcome_criteria-related use cases
type UseCases struct {
	CreateOutcomeCriteria          *CreateOutcomeCriteriaUseCase
	ReadOutcomeCriteria            *ReadOutcomeCriteriaUseCase
	UpdateOutcomeCriteria          *UpdateOutcomeCriteriaUseCase
	DeleteOutcomeCriteria          *DeleteOutcomeCriteriaUseCase
	ListOutcomeCriteria            *ListOutcomeCriteriaUseCase
	GetOutcomeCriteriaListPageData *GetOutcomeCriteriaListPageDataUseCase
	GetOutcomeCriteriaItemPageData *GetOutcomeCriteriaItemPageDataUseCase
	ListByGroup                    *ListByGroupUseCase
	GetCurrentPublished            *GetCurrentPublishedUseCase
	ListByScope                    *ListByScopeUseCase
}

// NewUseCases creates a new collection of outcome_criteria use cases
func NewUseCases(
	repositories OutcomeCriteriaRepositories,
	services OutcomeCriteriaServices,
) *UseCases {
	createRepos := CreateOutcomeCriteriaRepositories{
		OutcomeCriteria: repositories.OutcomeCriteria,
	}
	createServices := CreateOutcomeCriteriaServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadOutcomeCriteriaRepositories{
		OutcomeCriteria: repositories.OutcomeCriteria,
	}
	readServices := ReadOutcomeCriteriaServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateOutcomeCriteriaRepositories{
		OutcomeCriteria: repositories.OutcomeCriteria,
	}
	updateServices := UpdateOutcomeCriteriaServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteOutcomeCriteriaRepositories{
		OutcomeCriteria: repositories.OutcomeCriteria,
	}
	deleteServices := DeleteOutcomeCriteriaServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListOutcomeCriteriaRepositories{
		OutcomeCriteria: repositories.OutcomeCriteria,
	}
	listServices := ListOutcomeCriteriaServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listPageDataRepos := GetOutcomeCriteriaListPageDataRepositories{
		OutcomeCriteria: repositories.OutcomeCriteria,
	}
	listPageDataServices := GetOutcomeCriteriaListPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	itemPageDataRepos := GetOutcomeCriteriaItemPageDataRepositories{
		OutcomeCriteria: repositories.OutcomeCriteria,
	}
	itemPageDataServices := GetOutcomeCriteriaItemPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listByGroupRepos := ListByGroupRepositories{
		OutcomeCriteria: repositories.OutcomeCriteria,
	}
	listByGroupServices := ListByGroupServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getCurrentPublishedRepos := GetCurrentPublishedRepositories{
		OutcomeCriteria: repositories.OutcomeCriteria,
	}
	getCurrentPublishedServices := GetCurrentPublishedServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listByScopeRepos := ListByScopeRepositories{
		OutcomeCriteria: repositories.OutcomeCriteria,
	}
	listByScopeServices := ListByScopeServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateOutcomeCriteria:          NewCreateOutcomeCriteriaUseCase(createRepos, createServices),
		ReadOutcomeCriteria:            NewReadOutcomeCriteriaUseCase(readRepos, readServices),
		UpdateOutcomeCriteria:          NewUpdateOutcomeCriteriaUseCase(updateRepos, updateServices),
		DeleteOutcomeCriteria:          NewDeleteOutcomeCriteriaUseCase(deleteRepos, deleteServices),
		ListOutcomeCriteria:            NewListOutcomeCriteriaUseCase(listRepos, listServices),
		GetOutcomeCriteriaListPageData: NewGetOutcomeCriteriaListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetOutcomeCriteriaItemPageData: NewGetOutcomeCriteriaItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
		ListByGroup:                    NewListByGroupUseCase(listByGroupRepos, listByGroupServices),
		GetCurrentPublished:            NewGetCurrentPublishedUseCase(getCurrentPublishedRepos, getCurrentPublishedServices),
		ListByScope:                    NewListByScopeUseCase(listByScopeRepos, listByScopeServices),
	}
}

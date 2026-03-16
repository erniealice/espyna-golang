package outcome_criteria

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
)

// OutcomeCriteriaRepositories groups all repository dependencies
type OutcomeCriteriaRepositories struct {
	OutcomeCriteria pb.OutcomeCriteriaDomainServiceServer
}

// OutcomeCriteriaServices groups all business service dependencies
type OutcomeCriteriaServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadOutcomeCriteriaRepositories{
		OutcomeCriteria: repositories.OutcomeCriteria,
	}
	readServices := ReadOutcomeCriteriaServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateOutcomeCriteriaRepositories{
		OutcomeCriteria: repositories.OutcomeCriteria,
	}
	updateServices := UpdateOutcomeCriteriaServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteOutcomeCriteriaRepositories{
		OutcomeCriteria: repositories.OutcomeCriteria,
	}
	deleteServices := DeleteOutcomeCriteriaServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListOutcomeCriteriaRepositories{
		OutcomeCriteria: repositories.OutcomeCriteria,
	}
	listServices := ListOutcomeCriteriaServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listPageDataRepos := GetOutcomeCriteriaListPageDataRepositories{
		OutcomeCriteria: repositories.OutcomeCriteria,
	}
	listPageDataServices := GetOutcomeCriteriaListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	itemPageDataRepos := GetOutcomeCriteriaItemPageDataRepositories{
		OutcomeCriteria: repositories.OutcomeCriteria,
	}
	itemPageDataServices := GetOutcomeCriteriaItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listByGroupRepos := ListByGroupRepositories{
		OutcomeCriteria: repositories.OutcomeCriteria,
	}
	listByGroupServices := ListByGroupServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getCurrentPublishedRepos := GetCurrentPublishedRepositories{
		OutcomeCriteria: repositories.OutcomeCriteria,
	}
	getCurrentPublishedServices := GetCurrentPublishedServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listByScopeRepos := ListByScopeRepositories{
		OutcomeCriteria: repositories.OutcomeCriteria,
	}
	listByScopeServices := ListByScopeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
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

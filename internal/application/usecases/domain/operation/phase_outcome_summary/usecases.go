package phase_outcome_summary

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/phase_outcome_summary"
)

// PhaseOutcomeSummaryRepositories groups all repository dependencies
type PhaseOutcomeSummaryRepositories struct {
	PhaseOutcomeSummary pb.PhaseOutcomeSummaryDomainServiceServer
}

// PhaseOutcomeSummaryServices groups all business service dependencies
type PhaseOutcomeSummaryServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases contains all phase_outcome_summary-related use cases
type UseCases struct {
	CreatePhaseOutcomeSummary          *CreatePhaseOutcomeSummaryUseCase
	ReadPhaseOutcomeSummary            *ReadPhaseOutcomeSummaryUseCase
	UpdatePhaseOutcomeSummary          *UpdatePhaseOutcomeSummaryUseCase
	DeletePhaseOutcomeSummary          *DeletePhaseOutcomeSummaryUseCase
	ListPhaseOutcomeSummaries          *ListPhaseOutcomeSummariesUseCase
	GetPhaseOutcomeSummaryListPageData *GetPhaseOutcomeSummaryListPageDataUseCase
	GetPhaseOutcomeSummaryItemPageData *GetPhaseOutcomeSummaryItemPageDataUseCase
	GetByJobPhase                      *GetByJobPhaseUseCase
	ListByJob                          *ListByJobUseCase
}

// NewUseCases creates a new collection of phase_outcome_summary use cases
func NewUseCases(
	repositories PhaseOutcomeSummaryRepositories,
	services PhaseOutcomeSummaryServices,
) *UseCases {
	createRepos := CreatePhaseOutcomeSummaryRepositories{
		PhaseOutcomeSummary: repositories.PhaseOutcomeSummary,
	}
	createServices := CreatePhaseOutcomeSummaryServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadPhaseOutcomeSummaryRepositories{
		PhaseOutcomeSummary: repositories.PhaseOutcomeSummary,
	}
	readServices := ReadPhaseOutcomeSummaryServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdatePhaseOutcomeSummaryRepositories{
		PhaseOutcomeSummary: repositories.PhaseOutcomeSummary,
	}
	updateServices := UpdatePhaseOutcomeSummaryServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeletePhaseOutcomeSummaryRepositories{
		PhaseOutcomeSummary: repositories.PhaseOutcomeSummary,
	}
	deleteServices := DeletePhaseOutcomeSummaryServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListPhaseOutcomeSummariesRepositories{
		PhaseOutcomeSummary: repositories.PhaseOutcomeSummary,
	}
	listServices := ListPhaseOutcomeSummariesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listPageDataRepos := GetPhaseOutcomeSummaryListPageDataRepositories{
		PhaseOutcomeSummary: repositories.PhaseOutcomeSummary,
	}
	listPageDataServices := GetPhaseOutcomeSummaryListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	itemPageDataRepos := GetPhaseOutcomeSummaryItemPageDataRepositories{
		PhaseOutcomeSummary: repositories.PhaseOutcomeSummary,
	}
	itemPageDataServices := GetPhaseOutcomeSummaryItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getByJobPhaseRepos := GetByJobPhaseRepositories{
		PhaseOutcomeSummary: repositories.PhaseOutcomeSummary,
	}
	getByJobPhaseServices := GetByJobPhaseServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listByJobRepos := ListByJobRepositories{
		PhaseOutcomeSummary: repositories.PhaseOutcomeSummary,
	}
	listByJobServices := ListByJobServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreatePhaseOutcomeSummary:          NewCreatePhaseOutcomeSummaryUseCase(createRepos, createServices),
		ReadPhaseOutcomeSummary:            NewReadPhaseOutcomeSummaryUseCase(readRepos, readServices),
		UpdatePhaseOutcomeSummary:          NewUpdatePhaseOutcomeSummaryUseCase(updateRepos, updateServices),
		DeletePhaseOutcomeSummary:          NewDeletePhaseOutcomeSummaryUseCase(deleteRepos, deleteServices),
		ListPhaseOutcomeSummaries:          NewListPhaseOutcomeSummariesUseCase(listRepos, listServices),
		GetPhaseOutcomeSummaryListPageData: NewGetPhaseOutcomeSummaryListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetPhaseOutcomeSummaryItemPageData: NewGetPhaseOutcomeSummaryItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
		GetByJobPhase:                      NewGetByJobPhaseUseCase(getByJobPhaseRepos, getByJobPhaseServices),
		ListByJob:                          NewListByJobUseCase(listByJobRepos, listByJobServices),
	}
}

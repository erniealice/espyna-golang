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
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadPhaseOutcomeSummaryRepositories{
		PhaseOutcomeSummary: repositories.PhaseOutcomeSummary,
	}
	readServices := ReadPhaseOutcomeSummaryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdatePhaseOutcomeSummaryRepositories{
		PhaseOutcomeSummary: repositories.PhaseOutcomeSummary,
	}
	updateServices := UpdatePhaseOutcomeSummaryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeletePhaseOutcomeSummaryRepositories{
		PhaseOutcomeSummary: repositories.PhaseOutcomeSummary,
	}
	deleteServices := DeletePhaseOutcomeSummaryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListPhaseOutcomeSummariesRepositories{
		PhaseOutcomeSummary: repositories.PhaseOutcomeSummary,
	}
	listServices := ListPhaseOutcomeSummariesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listPageDataRepos := GetPhaseOutcomeSummaryListPageDataRepositories{
		PhaseOutcomeSummary: repositories.PhaseOutcomeSummary,
	}
	listPageDataServices := GetPhaseOutcomeSummaryListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	itemPageDataRepos := GetPhaseOutcomeSummaryItemPageDataRepositories{
		PhaseOutcomeSummary: repositories.PhaseOutcomeSummary,
	}
	itemPageDataServices := GetPhaseOutcomeSummaryItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getByJobPhaseRepos := GetByJobPhaseRepositories{
		PhaseOutcomeSummary: repositories.PhaseOutcomeSummary,
	}
	getByJobPhaseServices := GetByJobPhaseServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listByJobRepos := ListByJobRepositories{
		PhaseOutcomeSummary: repositories.PhaseOutcomeSummary,
	}
	listByJobServices := ListByJobServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
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

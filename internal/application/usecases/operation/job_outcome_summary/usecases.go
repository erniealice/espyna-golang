package job_outcome_summary

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary"
)

// JobOutcomeSummaryRepositories groups all repository dependencies
type JobOutcomeSummaryRepositories struct {
	JobOutcomeSummary pb.JobOutcomeSummaryDomainServiceServer
}

// JobOutcomeSummaryServices groups all business service dependencies
type JobOutcomeSummaryServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all job_outcome_summary-related use cases
type UseCases struct {
	CreateJobOutcomeSummary          *CreateJobOutcomeSummaryUseCase
	ReadJobOutcomeSummary            *ReadJobOutcomeSummaryUseCase
	UpdateJobOutcomeSummary          *UpdateJobOutcomeSummaryUseCase
	DeleteJobOutcomeSummary          *DeleteJobOutcomeSummaryUseCase
	ListJobOutcomeSummaries          *ListJobOutcomeSummariesUseCase
	GetJobOutcomeSummaryListPageData *GetJobOutcomeSummaryListPageDataUseCase
	GetJobOutcomeSummaryItemPageData *GetJobOutcomeSummaryItemPageDataUseCase
	GetByJob                         *GetByJobUseCase
}

// NewUseCases creates a new collection of job_outcome_summary use cases
func NewUseCases(
	repositories JobOutcomeSummaryRepositories,
	services JobOutcomeSummaryServices,
) *UseCases {
	createRepos := CreateJobOutcomeSummaryRepositories{
		JobOutcomeSummary: repositories.JobOutcomeSummary,
	}
	createServices := CreateJobOutcomeSummaryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadJobOutcomeSummaryRepositories{
		JobOutcomeSummary: repositories.JobOutcomeSummary,
	}
	readServices := ReadJobOutcomeSummaryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateJobOutcomeSummaryRepositories{
		JobOutcomeSummary: repositories.JobOutcomeSummary,
	}
	updateServices := UpdateJobOutcomeSummaryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteJobOutcomeSummaryRepositories{
		JobOutcomeSummary: repositories.JobOutcomeSummary,
	}
	deleteServices := DeleteJobOutcomeSummaryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListJobOutcomeSummariesRepositories{
		JobOutcomeSummary: repositories.JobOutcomeSummary,
	}
	listServices := ListJobOutcomeSummariesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listPageDataRepos := GetJobOutcomeSummaryListPageDataRepositories{
		JobOutcomeSummary: repositories.JobOutcomeSummary,
	}
	listPageDataServices := GetJobOutcomeSummaryListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	itemPageDataRepos := GetJobOutcomeSummaryItemPageDataRepositories{
		JobOutcomeSummary: repositories.JobOutcomeSummary,
	}
	itemPageDataServices := GetJobOutcomeSummaryItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getByJobRepos := GetByJobRepositories{
		JobOutcomeSummary: repositories.JobOutcomeSummary,
	}
	getByJobServices := GetByJobServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateJobOutcomeSummary:          NewCreateJobOutcomeSummaryUseCase(createRepos, createServices),
		ReadJobOutcomeSummary:            NewReadJobOutcomeSummaryUseCase(readRepos, readServices),
		UpdateJobOutcomeSummary:          NewUpdateJobOutcomeSummaryUseCase(updateRepos, updateServices),
		DeleteJobOutcomeSummary:          NewDeleteJobOutcomeSummaryUseCase(deleteRepos, deleteServices),
		ListJobOutcomeSummaries:          NewListJobOutcomeSummariesUseCase(listRepos, listServices),
		GetJobOutcomeSummaryListPageData: NewGetJobOutcomeSummaryListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetJobOutcomeSummaryItemPageData: NewGetJobOutcomeSummaryItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
		GetByJob:                         NewGetByJobUseCase(getByJobRepos, getByJobServices),
	}
}

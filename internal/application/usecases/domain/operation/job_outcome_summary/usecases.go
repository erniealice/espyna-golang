package job_outcome_summary

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary"
)

// JobOutcomeSummaryRepositories groups all repository dependencies
type JobOutcomeSummaryRepositories struct {
	JobOutcomeSummary pb.JobOutcomeSummaryDomainServiceServer
}

// JobOutcomeSummaryServices groups all business service dependencies
type JobOutcomeSummaryServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
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
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadJobOutcomeSummaryRepositories{
		JobOutcomeSummary: repositories.JobOutcomeSummary,
	}
	readServices := ReadJobOutcomeSummaryServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateJobOutcomeSummaryRepositories{
		JobOutcomeSummary: repositories.JobOutcomeSummary,
	}
	updateServices := UpdateJobOutcomeSummaryServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteJobOutcomeSummaryRepositories{
		JobOutcomeSummary: repositories.JobOutcomeSummary,
	}
	deleteServices := DeleteJobOutcomeSummaryServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListJobOutcomeSummariesRepositories{
		JobOutcomeSummary: repositories.JobOutcomeSummary,
	}
	listServices := ListJobOutcomeSummariesServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listPageDataRepos := GetJobOutcomeSummaryListPageDataRepositories{
		JobOutcomeSummary: repositories.JobOutcomeSummary,
	}
	listPageDataServices := GetJobOutcomeSummaryListPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	itemPageDataRepos := GetJobOutcomeSummaryItemPageDataRepositories{
		JobOutcomeSummary: repositories.JobOutcomeSummary,
	}
	itemPageDataServices := GetJobOutcomeSummaryItemPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getByJobRepos := GetByJobRepositories{
		JobOutcomeSummary: repositories.JobOutcomeSummary,
	}
	getByJobServices := GetByJobServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
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

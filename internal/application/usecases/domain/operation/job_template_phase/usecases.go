package job_template_phase

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
)

// JobTemplatePhaseRepositories groups all repository dependencies
type JobTemplatePhaseRepositories struct {
	JobTemplatePhase pb.JobTemplatePhaseDomainServiceServer
}

// JobTemplatePhaseServices groups all business service dependencies
type JobTemplatePhaseServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all job_template_phase-related use cases
type UseCases struct {
	CreateJobTemplatePhase          *CreateJobTemplatePhaseUseCase
	ReadJobTemplatePhase            *ReadJobTemplatePhaseUseCase
	UpdateJobTemplatePhase          *UpdateJobTemplatePhaseUseCase
	DeleteJobTemplatePhase          *DeleteJobTemplatePhaseUseCase
	ListJobTemplatePhases           *ListJobTemplatePhasesUseCase
	GetJobTemplatePhaseListPageData *GetJobTemplatePhaseListPageDataUseCase
	GetJobTemplatePhaseItemPageData *GetJobTemplatePhaseItemPageDataUseCase
	ListByJobTemplate               *ListByJobTemplateUseCase
}

// NewUseCases creates a new collection of job_template_phase use cases
func NewUseCases(
	repositories JobTemplatePhaseRepositories,
	services JobTemplatePhaseServices,
) *UseCases {
	createRepos := CreateJobTemplatePhaseRepositories{
		JobTemplatePhase: repositories.JobTemplatePhase,
	}
	createServices := CreateJobTemplatePhaseServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadJobTemplatePhaseRepositories{
		JobTemplatePhase: repositories.JobTemplatePhase,
	}
	readServices := ReadJobTemplatePhaseServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateJobTemplatePhaseRepositories{
		JobTemplatePhase: repositories.JobTemplatePhase,
	}
	updateServices := UpdateJobTemplatePhaseServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteJobTemplatePhaseRepositories{
		JobTemplatePhase: repositories.JobTemplatePhase,
	}
	deleteServices := DeleteJobTemplatePhaseServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListJobTemplatePhasesRepositories{
		JobTemplatePhase: repositories.JobTemplatePhase,
	}
	listServices := ListJobTemplatePhasesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listPageDataRepos := GetJobTemplatePhaseListPageDataRepositories{
		JobTemplatePhase: repositories.JobTemplatePhase,
	}
	listPageDataServices := GetJobTemplatePhaseListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	itemPageDataRepos := GetJobTemplatePhaseItemPageDataRepositories{
		JobTemplatePhase: repositories.JobTemplatePhase,
	}
	itemPageDataServices := GetJobTemplatePhaseItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listByJobTemplateRepos := ListByJobTemplateRepositories{
		JobTemplatePhase: repositories.JobTemplatePhase,
	}
	listByJobTemplateServices := ListByJobTemplateServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateJobTemplatePhase:          NewCreateJobTemplatePhaseUseCase(createRepos, createServices),
		ReadJobTemplatePhase:            NewReadJobTemplatePhaseUseCase(readRepos, readServices),
		UpdateJobTemplatePhase:          NewUpdateJobTemplatePhaseUseCase(updateRepos, updateServices),
		DeleteJobTemplatePhase:          NewDeleteJobTemplatePhaseUseCase(deleteRepos, deleteServices),
		ListJobTemplatePhases:           NewListJobTemplatePhasesUseCase(listRepos, listServices),
		GetJobTemplatePhaseListPageData: NewGetJobTemplatePhaseListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetJobTemplatePhaseItemPageData: NewGetJobTemplatePhaseItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
		ListByJobTemplate:               NewListByJobTemplateUseCase(listByJobTemplateRepos, listByJobTemplateServices),
	}
}

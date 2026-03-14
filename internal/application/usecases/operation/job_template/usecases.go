package job_template

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
)

// JobTemplateRepositories groups all repository dependencies
type JobTemplateRepositories struct {
	JobTemplate pb.JobTemplateDomainServiceServer
}

// JobTemplateServices groups all business service dependencies
type JobTemplateServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all job-template-related use cases
type UseCases struct {
	CreateJobTemplate          *CreateJobTemplateUseCase
	ReadJobTemplate            *ReadJobTemplateUseCase
	UpdateJobTemplate          *UpdateJobTemplateUseCase
	DeleteJobTemplate          *DeleteJobTemplateUseCase
	ListJobTemplates           *ListJobTemplatesUseCase
	GetJobTemplateListPageData *GetJobTemplateListPageDataUseCase
	GetJobTemplateItemPageData *GetJobTemplateItemPageDataUseCase
}

// NewUseCases creates a new collection of job template use cases
func NewUseCases(
	repositories JobTemplateRepositories,
	services JobTemplateServices,
) *UseCases {
	createRepos := CreateJobTemplateRepositories{JobTemplate: repositories.JobTemplate}
	createServices := CreateJobTemplateServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadJobTemplateRepositories{JobTemplate: repositories.JobTemplate}
	readServices := ReadJobTemplateServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateJobTemplateRepositories{JobTemplate: repositories.JobTemplate}
	updateServices := UpdateJobTemplateServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteJobTemplateRepositories{JobTemplate: repositories.JobTemplate}
	deleteServices := DeleteJobTemplateServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListJobTemplatesRepositories{JobTemplate: repositories.JobTemplate}
	listServices := ListJobTemplatesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listPageDataRepos := GetJobTemplateListPageDataRepositories{JobTemplate: repositories.JobTemplate}
	listPageDataServices := GetJobTemplateListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	itemPageDataRepos := GetJobTemplateItemPageDataRepositories{JobTemplate: repositories.JobTemplate}
	itemPageDataServices := GetJobTemplateItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateJobTemplate:          NewCreateJobTemplateUseCase(createRepos, createServices),
		ReadJobTemplate:            NewReadJobTemplateUseCase(readRepos, readServices),
		UpdateJobTemplate:          NewUpdateJobTemplateUseCase(updateRepos, updateServices),
		DeleteJobTemplate:          NewDeleteJobTemplateUseCase(deleteRepos, deleteServices),
		ListJobTemplates:           NewListJobTemplatesUseCase(listRepos, listServices),
		GetJobTemplateListPageData: NewGetJobTemplateListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetJobTemplateItemPageData: NewGetJobTemplateItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
	}
}

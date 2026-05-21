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
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
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
	InstantiateJobsFromPlan    *InstantiateJobsFromPlanUseCase
}

// NewUseCases creates a new collection of job template use cases
func NewUseCases(
	repositories JobTemplateRepositories,
	services JobTemplateServices,
) *UseCases {
	createRepos := CreateJobTemplateRepositories{JobTemplate: repositories.JobTemplate}
	createServices := CreateJobTemplateServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadJobTemplateRepositories{JobTemplate: repositories.JobTemplate}
	readServices := ReadJobTemplateServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateJobTemplateRepositories{JobTemplate: repositories.JobTemplate}
	updateServices := UpdateJobTemplateServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteJobTemplateRepositories{JobTemplate: repositories.JobTemplate}
	deleteServices := DeleteJobTemplateServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListJobTemplatesRepositories{JobTemplate: repositories.JobTemplate}
	listServices := ListJobTemplatesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listPageDataRepos := GetJobTemplateListPageDataRepositories{JobTemplate: repositories.JobTemplate}
	listPageDataServices := GetJobTemplateListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	itemPageDataRepos := GetJobTemplateItemPageDataRepositories{JobTemplate: repositories.JobTemplate}
	itemPageDataServices := GetJobTemplateItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
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

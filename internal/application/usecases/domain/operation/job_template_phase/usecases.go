package job_template_phase

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
)

// JobTemplatePhaseRepositories groups all repository dependencies
type JobTemplatePhaseRepositories struct {
	JobTemplatePhase pb.JobTemplatePhaseDomainServiceServer
}

// JobTemplatePhaseServices groups all business service dependencies
type JobTemplatePhaseServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
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
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadJobTemplatePhaseRepositories{
		JobTemplatePhase: repositories.JobTemplatePhase,
	}
	readServices := ReadJobTemplatePhaseServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateJobTemplatePhaseRepositories{
		JobTemplatePhase: repositories.JobTemplatePhase,
	}
	updateServices := UpdateJobTemplatePhaseServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteJobTemplatePhaseRepositories{
		JobTemplatePhase: repositories.JobTemplatePhase,
	}
	deleteServices := DeleteJobTemplatePhaseServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListJobTemplatePhasesRepositories{
		JobTemplatePhase: repositories.JobTemplatePhase,
	}
	listServices := ListJobTemplatePhasesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listPageDataRepos := GetJobTemplatePhaseListPageDataRepositories{
		JobTemplatePhase: repositories.JobTemplatePhase,
	}
	listPageDataServices := GetJobTemplatePhaseListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	itemPageDataRepos := GetJobTemplatePhaseItemPageDataRepositories{
		JobTemplatePhase: repositories.JobTemplatePhase,
	}
	itemPageDataServices := GetJobTemplatePhaseItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listByJobTemplateRepos := ListByJobTemplateRepositories{
		JobTemplatePhase: repositories.JobTemplatePhase,
	}
	listByJobTemplateServices := ListByJobTemplateServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
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

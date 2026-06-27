package job_template_task

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_task"
)

// JobTemplateTaskRepositories groups all repository dependencies
type JobTemplateTaskRepositories struct {
	JobTemplateTask pb.JobTemplateTaskDomainServiceServer
}

// JobTemplateTaskServices groups all business service dependencies
type JobTemplateTaskServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all job_template_task-related use cases
type UseCases struct {
	CreateJobTemplateTask          *CreateJobTemplateTaskUseCase
	ReadJobTemplateTask            *ReadJobTemplateTaskUseCase
	UpdateJobTemplateTask          *UpdateJobTemplateTaskUseCase
	DeleteJobTemplateTask          *DeleteJobTemplateTaskUseCase
	ListJobTemplateTasks           *ListJobTemplateTasksUseCase
	GetJobTemplateTaskListPageData *GetJobTemplateTaskListPageDataUseCase
	GetJobTemplateTaskItemPageData *GetJobTemplateTaskItemPageDataUseCase
	ListByPhase                    *ListByPhaseUseCase
}

// NewUseCases creates a new collection of job_template_task use cases
func NewUseCases(
	repositories JobTemplateTaskRepositories,
	services JobTemplateTaskServices,
) *UseCases {
	createRepos := CreateJobTemplateTaskRepositories{
		JobTemplateTask: repositories.JobTemplateTask,
	}
	createServices := CreateJobTemplateTaskServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadJobTemplateTaskRepositories{
		JobTemplateTask: repositories.JobTemplateTask,
	}
	readServices := ReadJobTemplateTaskServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateJobTemplateTaskRepositories{
		JobTemplateTask: repositories.JobTemplateTask,
	}
	updateServices := UpdateJobTemplateTaskServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteJobTemplateTaskRepositories{
		JobTemplateTask: repositories.JobTemplateTask,
	}
	deleteServices := DeleteJobTemplateTaskServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListJobTemplateTasksRepositories{
		JobTemplateTask: repositories.JobTemplateTask,
	}
	listServices := ListJobTemplateTasksServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listPageDataRepos := GetJobTemplateTaskListPageDataRepositories{
		JobTemplateTask: repositories.JobTemplateTask,
	}
	listPageDataServices := GetJobTemplateTaskListPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	itemPageDataRepos := GetJobTemplateTaskItemPageDataRepositories{
		JobTemplateTask: repositories.JobTemplateTask,
	}
	itemPageDataServices := GetJobTemplateTaskItemPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listByPhaseRepos := ListByPhaseRepositories{
		JobTemplateTask: repositories.JobTemplateTask,
	}
	listByPhaseServices := ListByPhaseServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateJobTemplateTask:          NewCreateJobTemplateTaskUseCase(createRepos, createServices),
		ReadJobTemplateTask:            NewReadJobTemplateTaskUseCase(readRepos, readServices),
		UpdateJobTemplateTask:          NewUpdateJobTemplateTaskUseCase(updateRepos, updateServices),
		DeleteJobTemplateTask:          NewDeleteJobTemplateTaskUseCase(deleteRepos, deleteServices),
		ListJobTemplateTasks:           NewListJobTemplateTasksUseCase(listRepos, listServices),
		GetJobTemplateTaskListPageData: NewGetJobTemplateTaskListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetJobTemplateTaskItemPageData: NewGetJobTemplateTaskItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
		ListByPhase:                    NewListByPhaseUseCase(listByPhaseRepos, listByPhaseServices),
	}
}

package job_template_task

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_task"
)

// JobTemplateTaskRepositories groups all repository dependencies
type JobTemplateTaskRepositories struct {
	JobTemplateTask pb.JobTemplateTaskDomainServiceServer
}

// JobTemplateTaskServices groups all business service dependencies
type JobTemplateTaskServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadJobTemplateTaskRepositories{
		JobTemplateTask: repositories.JobTemplateTask,
	}
	readServices := ReadJobTemplateTaskServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateJobTemplateTaskRepositories{
		JobTemplateTask: repositories.JobTemplateTask,
	}
	updateServices := UpdateJobTemplateTaskServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteJobTemplateTaskRepositories{
		JobTemplateTask: repositories.JobTemplateTask,
	}
	deleteServices := DeleteJobTemplateTaskServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListJobTemplateTasksRepositories{
		JobTemplateTask: repositories.JobTemplateTask,
	}
	listServices := ListJobTemplateTasksServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listPageDataRepos := GetJobTemplateTaskListPageDataRepositories{
		JobTemplateTask: repositories.JobTemplateTask,
	}
	listPageDataServices := GetJobTemplateTaskListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	itemPageDataRepos := GetJobTemplateTaskItemPageDataRepositories{
		JobTemplateTask: repositories.JobTemplateTask,
	}
	itemPageDataServices := GetJobTemplateTaskItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listByPhaseRepos := ListByPhaseRepositories{
		JobTemplateTask: repositories.JobTemplateTask,
	}
	listByPhaseServices := ListByPhaseServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
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

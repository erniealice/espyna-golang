package task_outcome

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

// TaskOutcomeRepositories groups all repository dependencies
type TaskOutcomeRepositories struct {
	TaskOutcome pb.TaskOutcomeDomainServiceServer
}

// TaskOutcomeServices groups all business service dependencies
type TaskOutcomeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all task_outcome-related use cases
type UseCases struct {
	CreateTaskOutcome          *CreateTaskOutcomeUseCase
	ReadTaskOutcome            *ReadTaskOutcomeUseCase
	UpdateTaskOutcome          *UpdateTaskOutcomeUseCase
	DeleteTaskOutcome          *DeleteTaskOutcomeUseCase
	ListTaskOutcomes           *ListTaskOutcomesUseCase
	GetTaskOutcomeListPageData *GetTaskOutcomeListPageDataUseCase
	GetTaskOutcomeItemPageData *GetTaskOutcomeItemPageDataUseCase
	ListByJobTask              *ListByJobTaskUseCase
	ListByJobPhase             *ListByJobPhaseUseCase
	ListByJob                  *ListByJobUseCase
}

// NewUseCases creates a new collection of task_outcome use cases
func NewUseCases(
	repositories TaskOutcomeRepositories,
	services TaskOutcomeServices,
) *UseCases {
	createRepos := CreateTaskOutcomeRepositories{
		TaskOutcome: repositories.TaskOutcome,
	}
	createServices := CreateTaskOutcomeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadTaskOutcomeRepositories{
		TaskOutcome: repositories.TaskOutcome,
	}
	readServices := ReadTaskOutcomeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateTaskOutcomeRepositories{
		TaskOutcome: repositories.TaskOutcome,
	}
	updateServices := UpdateTaskOutcomeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteTaskOutcomeRepositories{
		TaskOutcome: repositories.TaskOutcome,
	}
	deleteServices := DeleteTaskOutcomeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListTaskOutcomesRepositories{
		TaskOutcome: repositories.TaskOutcome,
	}
	listServices := ListTaskOutcomesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listPageDataRepos := GetTaskOutcomeListPageDataRepositories{
		TaskOutcome: repositories.TaskOutcome,
	}
	listPageDataServices := GetTaskOutcomeListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	itemPageDataRepos := GetTaskOutcomeItemPageDataRepositories{
		TaskOutcome: repositories.TaskOutcome,
	}
	itemPageDataServices := GetTaskOutcomeItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listByJobTaskRepos := ListByJobTaskRepositories{
		TaskOutcome: repositories.TaskOutcome,
	}
	listByJobTaskServices := ListByJobTaskServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listByJobPhaseRepos := ListByJobPhaseRepositories{
		TaskOutcome: repositories.TaskOutcome,
	}
	listByJobPhaseServices := ListByJobPhaseServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listByJobRepos := ListByJobRepositories{
		TaskOutcome: repositories.TaskOutcome,
	}
	listByJobServices := ListByJobServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateTaskOutcome:          NewCreateTaskOutcomeUseCase(createRepos, createServices),
		ReadTaskOutcome:            NewReadTaskOutcomeUseCase(readRepos, readServices),
		UpdateTaskOutcome:          NewUpdateTaskOutcomeUseCase(updateRepos, updateServices),
		DeleteTaskOutcome:          NewDeleteTaskOutcomeUseCase(deleteRepos, deleteServices),
		ListTaskOutcomes:           NewListTaskOutcomesUseCase(listRepos, listServices),
		GetTaskOutcomeListPageData: NewGetTaskOutcomeListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetTaskOutcomeItemPageData: NewGetTaskOutcomeItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
		ListByJobTask:              NewListByJobTaskUseCase(listByJobTaskRepos, listByJobTaskServices),
		ListByJobPhase:             NewListByJobPhaseUseCase(listByJobPhaseRepos, listByJobPhaseServices),
		ListByJob:                  NewListByJobUseCase(listByJobRepos, listByJobServices),
	}
}

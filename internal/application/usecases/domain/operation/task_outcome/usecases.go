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
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
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
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadTaskOutcomeRepositories{
		TaskOutcome: repositories.TaskOutcome,
	}
	readServices := ReadTaskOutcomeServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateTaskOutcomeRepositories{
		TaskOutcome: repositories.TaskOutcome,
	}
	updateServices := UpdateTaskOutcomeServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteTaskOutcomeRepositories{
		TaskOutcome: repositories.TaskOutcome,
	}
	deleteServices := DeleteTaskOutcomeServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListTaskOutcomesRepositories{
		TaskOutcome: repositories.TaskOutcome,
	}
	listServices := ListTaskOutcomesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listPageDataRepos := GetTaskOutcomeListPageDataRepositories{
		TaskOutcome: repositories.TaskOutcome,
	}
	listPageDataServices := GetTaskOutcomeListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	itemPageDataRepos := GetTaskOutcomeItemPageDataRepositories{
		TaskOutcome: repositories.TaskOutcome,
	}
	itemPageDataServices := GetTaskOutcomeItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listByJobTaskRepos := ListByJobTaskRepositories{
		TaskOutcome: repositories.TaskOutcome,
	}
	listByJobTaskServices := ListByJobTaskServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listByJobPhaseRepos := ListByJobPhaseRepositories{
		TaskOutcome: repositories.TaskOutcome,
	}
	listByJobPhaseServices := ListByJobPhaseServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listByJobRepos := ListByJobRepositories{
		TaskOutcome: repositories.TaskOutcome,
	}
	listByJobServices := ListByJobServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
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

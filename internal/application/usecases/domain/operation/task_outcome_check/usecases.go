package task_outcome_check

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome_check"
)

// TaskOutcomeCheckRepositories groups all repository dependencies
type TaskOutcomeCheckRepositories struct {
	TaskOutcomeCheck pb.TaskOutcomeCheckDomainServiceServer
}

// TaskOutcomeCheckServices groups all business service dependencies
type TaskOutcomeCheckServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases contains all task_outcome_check-related use cases
type UseCases struct {
	CreateTaskOutcomeCheck          *CreateTaskOutcomeCheckUseCase
	ReadTaskOutcomeCheck            *ReadTaskOutcomeCheckUseCase
	UpdateTaskOutcomeCheck          *UpdateTaskOutcomeCheckUseCase
	DeleteTaskOutcomeCheck          *DeleteTaskOutcomeCheckUseCase
	ListTaskOutcomeChecks           *ListTaskOutcomeChecksUseCase
	GetTaskOutcomeCheckListPageData *GetTaskOutcomeCheckListPageDataUseCase
	GetTaskOutcomeCheckItemPageData *GetTaskOutcomeCheckItemPageDataUseCase
	ListByTaskOutcome               *ListByTaskOutcomeUseCase
}

// NewUseCases creates a new collection of task_outcome_check use cases
func NewUseCases(
	repositories TaskOutcomeCheckRepositories,
	services TaskOutcomeCheckServices,
) *UseCases {
	createRepos := CreateTaskOutcomeCheckRepositories{
		TaskOutcomeCheck: repositories.TaskOutcomeCheck,
	}
	createServices := CreateTaskOutcomeCheckServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadTaskOutcomeCheckRepositories{
		TaskOutcomeCheck: repositories.TaskOutcomeCheck,
	}
	readServices := ReadTaskOutcomeCheckServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateTaskOutcomeCheckRepositories{
		TaskOutcomeCheck: repositories.TaskOutcomeCheck,
	}
	updateServices := UpdateTaskOutcomeCheckServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteTaskOutcomeCheckRepositories{
		TaskOutcomeCheck: repositories.TaskOutcomeCheck,
	}
	deleteServices := DeleteTaskOutcomeCheckServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListTaskOutcomeChecksRepositories{
		TaskOutcomeCheck: repositories.TaskOutcomeCheck,
	}
	listServices := ListTaskOutcomeChecksServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listPageDataRepos := GetTaskOutcomeCheckListPageDataRepositories{
		TaskOutcomeCheck: repositories.TaskOutcomeCheck,
	}
	listPageDataServices := GetTaskOutcomeCheckListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	itemPageDataRepos := GetTaskOutcomeCheckItemPageDataRepositories{
		TaskOutcomeCheck: repositories.TaskOutcomeCheck,
	}
	itemPageDataServices := GetTaskOutcomeCheckItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listByTaskOutcomeRepos := ListByTaskOutcomeRepositories{
		TaskOutcomeCheck: repositories.TaskOutcomeCheck,
	}
	listByTaskOutcomeServices := ListByTaskOutcomeServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateTaskOutcomeCheck:          NewCreateTaskOutcomeCheckUseCase(createRepos, createServices),
		ReadTaskOutcomeCheck:            NewReadTaskOutcomeCheckUseCase(readRepos, readServices),
		UpdateTaskOutcomeCheck:          NewUpdateTaskOutcomeCheckUseCase(updateRepos, updateServices),
		DeleteTaskOutcomeCheck:          NewDeleteTaskOutcomeCheckUseCase(deleteRepos, deleteServices),
		ListTaskOutcomeChecks:           NewListTaskOutcomeChecksUseCase(listRepos, listServices),
		GetTaskOutcomeCheckListPageData: NewGetTaskOutcomeCheckListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetTaskOutcomeCheckItemPageData: NewGetTaskOutcomeCheckItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
		ListByTaskOutcome:               NewListByTaskOutcomeUseCase(listByTaskOutcomeRepos, listByTaskOutcomeServices),
	}
}

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
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadTaskOutcomeCheckRepositories{
		TaskOutcomeCheck: repositories.TaskOutcomeCheck,
	}
	readServices := ReadTaskOutcomeCheckServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateTaskOutcomeCheckRepositories{
		TaskOutcomeCheck: repositories.TaskOutcomeCheck,
	}
	updateServices := UpdateTaskOutcomeCheckServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteTaskOutcomeCheckRepositories{
		TaskOutcomeCheck: repositories.TaskOutcomeCheck,
	}
	deleteServices := DeleteTaskOutcomeCheckServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListTaskOutcomeChecksRepositories{
		TaskOutcomeCheck: repositories.TaskOutcomeCheck,
	}
	listServices := ListTaskOutcomeChecksServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listPageDataRepos := GetTaskOutcomeCheckListPageDataRepositories{
		TaskOutcomeCheck: repositories.TaskOutcomeCheck,
	}
	listPageDataServices := GetTaskOutcomeCheckListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	itemPageDataRepos := GetTaskOutcomeCheckItemPageDataRepositories{
		TaskOutcomeCheck: repositories.TaskOutcomeCheck,
	}
	itemPageDataServices := GetTaskOutcomeCheckItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listByTaskOutcomeRepos := ListByTaskOutcomeRepositories{
		TaskOutcomeCheck: repositories.TaskOutcomeCheck,
	}
	listByTaskOutcomeServices := ListByTaskOutcomeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
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

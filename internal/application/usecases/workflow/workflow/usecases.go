package workflow

import (
	"leapfor.xyz/espyna/internal/application/ports"
	workflowpb "leapfor.xyz/esqyma/golang/v1/domain/workflow/workflow"
)

// WorkflowRepositories groups all repository dependencies for workflow use cases
type WorkflowRepositories struct {
	Workflow workflowpb.WorkflowDomainServiceServer // Primary entity repository
}

// WorkflowServices groups all business service dependencies for workflow use cases
type WorkflowServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService // Required for Create use case
}

// UseCases contains all workflow-related use cases

type UseCases struct {
	CreateWorkflow *CreateWorkflowUseCase

	ReadWorkflow *ReadWorkflowUseCase

	UpdateWorkflow *UpdateWorkflowUseCase

	DeleteWorkflow *DeleteWorkflowUseCase

	ListWorkflows *ListWorkflowsUseCase

	GetWorkflowListPageData *GetWorkflowListPageDataUseCase

	GetWorkflowItemPageData *GetWorkflowItemPageDataUseCase
}

// NewUseCases creates a new collection of workflow use cases

func NewUseCases(

	repositories WorkflowRepositories,

	services WorkflowServices,

) *UseCases {

	// Build individual grouped parameters for each use case

	createRepos := CreateWorkflowRepositories(repositories)

	createServices := CreateWorkflowServices{

		AuthorizationService: services.AuthorizationService,

		TransactionService: services.TransactionService,

		TranslationService: services.TranslationService,

		IDService: services.IDService,
	}

	readRepos := ReadWorkflowRepositories(repositories)

	readServices := ReadWorkflowServices{

		AuthorizationService: services.AuthorizationService,

		TransactionService: services.TransactionService,

		TranslationService: services.TranslationService,
	}

	updateRepos := UpdateWorkflowRepositories(repositories)

	updateServices := UpdateWorkflowServices{

		AuthorizationService: services.AuthorizationService,

		TransactionService: services.TransactionService,

		TranslationService: services.TranslationService,
	}

	deleteRepos := DeleteWorkflowRepositories(repositories)

	deleteServices := DeleteWorkflowServices{

		AuthorizationService: services.AuthorizationService,

		TransactionService: services.TransactionService,

		TranslationService: services.TranslationService,
	}

	listRepos := ListWorkflowsRepositories(repositories)

	listServices := ListWorkflowsServices{

		AuthorizationService: services.AuthorizationService,

		TransactionService: services.TransactionService,

		TranslationService: services.TranslationService,
	}

	getListPageDataRepos := GetWorkflowListPageDataRepositories{

		Workflow: repositories.Workflow,
	}

	getListPageDataServices := GetWorkflowListPageDataServices{

		TransactionService: services.TransactionService,

		TranslationService: services.TranslationService,
	}

	getItemPageDataRepos := GetWorkflowItemPageDataRepositories{

		Workflow: repositories.Workflow,
	}

	getItemPageDataServices := GetWorkflowItemPageDataServices{

		TransactionService: services.TransactionService,

		TranslationService: services.TranslationService,
	}

	return &UseCases{

		CreateWorkflow: NewCreateWorkflowUseCase(createRepos, createServices),

		ReadWorkflow: NewReadWorkflowUseCase(readRepos, readServices),

		UpdateWorkflow: NewUpdateWorkflowUseCase(updateRepos, updateServices),

		DeleteWorkflow: NewDeleteWorkflowUseCase(deleteRepos, deleteServices),

		ListWorkflows: NewListWorkflowsUseCase(listRepos, listServices),

		GetWorkflowListPageData: NewGetWorkflowListPageDataUseCase(getListPageDataRepos, getListPageDataServices),

		GetWorkflowItemPageData: NewGetWorkflowItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}

}

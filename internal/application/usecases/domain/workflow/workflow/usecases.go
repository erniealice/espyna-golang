package workflow

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	workflowpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/workflow"
)

// WorkflowRepositories groups all repository dependencies for workflow use cases
type WorkflowRepositories struct {
	Workflow workflowpb.WorkflowDomainServiceServer // Primary entity repository
}

// WorkflowServices groups all business service dependencies for workflow use cases
type WorkflowServices struct {
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator // Required for Create use case
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

		Authorizer: services.Authorizer,

		Transactor: services.Transactor,

		Translator: services.Translator,

		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadWorkflowRepositories(repositories)

	readServices := ReadWorkflowServices{

		Authorizer: services.Authorizer,

		Transactor: services.Transactor,

		Translator: services.Translator,
	}

	updateRepos := UpdateWorkflowRepositories(repositories)

	updateServices := UpdateWorkflowServices{

		Authorizer: services.Authorizer,

		Transactor: services.Transactor,

		Translator: services.Translator,
	}

	deleteRepos := DeleteWorkflowRepositories(repositories)

	deleteServices := DeleteWorkflowServices{

		Authorizer: services.Authorizer,

		Transactor: services.Transactor,

		Translator: services.Translator,
	}

	listRepos := ListWorkflowsRepositories(repositories)

	listServices := ListWorkflowsServices{

		Authorizer: services.Authorizer,

		Transactor: services.Transactor,

		Translator: services.Translator,
	}

	getListPageDataRepos := GetWorkflowListPageDataRepositories{

		Workflow: repositories.Workflow,
	}

	getListPageDataServices := GetWorkflowListPageDataServices{

		Transactor: services.Transactor,

		Translator: services.Translator,
	}

	getItemPageDataRepos := GetWorkflowItemPageDataRepositories{

		Workflow: repositories.Workflow,
	}

	getItemPageDataServices := GetWorkflowItemPageDataServices{

		Transactor: services.Transactor,

		Translator: services.Translator,
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

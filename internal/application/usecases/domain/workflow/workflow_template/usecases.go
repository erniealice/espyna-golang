package workflow_template

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
	workflow_templatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/workflow_template"
)

// WorkflowTemplateRepositories groups all repository dependencies for workflow template use cases
type WorkflowTemplateRepositories struct {
	WorkflowTemplate workflow_templatepb.WorkflowTemplateDomainServiceServer // Primary entity repository
	Workspace        workspacepb.WorkspaceDomainServiceServer                // Workspace repository for foreign key validation
}

// WorkflowTemplateServices groups all business service dependencies for workflow template use cases
type WorkflowTemplateServices struct {
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator // Required for Create use case
}

// UseCases contains all workflow template-related use cases
type UseCases struct {
	CreateWorkflowTemplate          *CreateWorkflowTemplateUseCase
	ReadWorkflowTemplate            *ReadWorkflowTemplateUseCase
	UpdateWorkflowTemplate          *UpdateWorkflowTemplateUseCase
	DeleteWorkflowTemplate          *DeleteWorkflowTemplateUseCase
	ListWorkflowTemplates           *ListWorkflowTemplatesUseCase
	GetWorkflowTemplateListPageData *GetWorkflowTemplateListPageDataUseCase
	GetWorkflowTemplateItemPageData *GetWorkflowTemplateItemPageDataUseCase
}

// NewUseCases creates a new collection of workflow template use cases
func NewUseCases(
	repositories WorkflowTemplateRepositories,
	services WorkflowTemplateServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateWorkflowTemplateRepositories(repositories)
	createServices := CreateWorkflowTemplateServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadWorkflowTemplateRepositories(repositories)
	readServices := ReadWorkflowTemplateServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateWorkflowTemplateRepositories(repositories)
	updateServices := UpdateWorkflowTemplateServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteWorkflowTemplateRepositories(repositories)
	deleteServices := DeleteWorkflowTemplateServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListWorkflowTemplatesRepositories(repositories)
	listServices := ListWorkflowTemplatesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := GetWorkflowTemplateListPageDataRepositories{
		WorkflowTemplate: repositories.WorkflowTemplate,
	}
	getListPageDataServices := GetWorkflowTemplateListPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getItemPageDataRepos := GetWorkflowTemplateItemPageDataRepositories{
		WorkflowTemplate: repositories.WorkflowTemplate,
	}
	getItemPageDataServices := GetWorkflowTemplateItemPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateWorkflowTemplate:          NewCreateWorkflowTemplateUseCase(createRepos, createServices),
		ReadWorkflowTemplate:            NewReadWorkflowTemplateUseCase(readRepos, readServices),
		UpdateWorkflowTemplate:          NewUpdateWorkflowTemplateUseCase(updateRepos, updateServices),
		DeleteWorkflowTemplate:          NewDeleteWorkflowTemplateUseCase(deleteRepos, deleteServices),
		ListWorkflowTemplates:           NewListWorkflowTemplatesUseCase(listRepos, listServices),
		GetWorkflowTemplateListPageData: NewGetWorkflowTemplateListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetWorkflowTemplateItemPageData: NewGetWorkflowTemplateItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}

package workflow_template

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
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
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService // Required for Create use case
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadWorkflowTemplateRepositories(repositories)
	readServices := ReadWorkflowTemplateServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateWorkflowTemplateRepositories(repositories)
	updateServices := UpdateWorkflowTemplateServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteWorkflowTemplateRepositories(repositories)
	deleteServices := DeleteWorkflowTemplateServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListWorkflowTemplatesRepositories(repositories)
	listServices := ListWorkflowTemplatesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetWorkflowTemplateListPageDataRepositories{
		WorkflowTemplate: repositories.WorkflowTemplate,
	}
	getListPageDataServices := GetWorkflowTemplateListPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	getItemPageDataRepos := GetWorkflowTemplateItemPageDataRepositories{
		WorkflowTemplate: repositories.WorkflowTemplate,
	}
	getItemPageDataServices := GetWorkflowTemplateItemPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
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

package stage_template

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	stageTemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/stage_template"
	workflowTemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/workflow_template"
)

// StageTemplateRepositories groups all repository dependencies for stage template use cases
type StageTemplateRepositories struct {
	StageTemplate    stageTemplatepb.StageTemplateDomainServiceServer       // Primary entity repository
	WorkflowTemplate workflowTemplatepb.WorkflowTemplateDomainServiceServer // Foreign key reference
}

// StageTemplateServices groups all business service dependencies for stage template use cases
type StageTemplateServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator // Required for Create use case
}

// UseCases contains all stage template-related use cases
type UseCases struct {
	CreateStageTemplate          *CreateStageTemplateUseCase
	ReadStageTemplate            *ReadStageTemplateUseCase
	UpdateStageTemplate          *UpdateStageTemplateUseCase
	DeleteStageTemplate          *DeleteStageTemplateUseCase
	ListStageTemplates           *ListStageTemplatesUseCase
	GetStageTemplateListPageData *GetStageTemplateListPageDataUseCase
	GetStageTemplateItemPageData *GetStageTemplateItemPageDataUseCase
	// GetStageTemplatesByWorkflow  *GetStageTemplatesByWorkflowUseCase // TODO: Implement
}

// NewUseCases creates a new collection of stage template use cases
func NewUseCases(
	repositories StageTemplateRepositories,
	services StageTemplateServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateStageTemplateRepositories(repositories)
	createServices := CreateStageTemplateServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadStageTemplateRepositories{
		StageTemplate: repositories.StageTemplate,
	}
	readServices := ReadStageTemplateServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateStageTemplateRepositories{
		StageTemplate:    repositories.StageTemplate,
		WorkflowTemplate: repositories.WorkflowTemplate,
	}
	updateServices := UpdateStageTemplateServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteStageTemplateRepositories{
		StageTemplate: repositories.StageTemplate,
	}
	deleteServices := DeleteStageTemplateServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListStageTemplatesRepositories{
		StageTemplate: repositories.StageTemplate,
	}
	listServices := ListStageTemplatesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := GetStageTemplateListPageDataRepositories{
		StageTemplate:    repositories.StageTemplate,
		WorkflowTemplate: repositories.WorkflowTemplate,
	}
	getListPageDataServices := GetStageTemplateListPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getItemPageDataRepos := GetStageTemplateItemPageDataRepositories{
		StageTemplate:    repositories.StageTemplate,
		WorkflowTemplate: repositories.WorkflowTemplate,
	}
	getItemPageDataServices := GetStageTemplateItemPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	// TODO: Implement when GetStageTemplatesByWorkflow use case is available
	// getByWorkflowRepos := GetStageTemplatesByWorkflowRepositories(repositories)
	// getByWorkflowServices := GetStageTemplatesByWorkflowServices{
	// 	Authorizer: services.Authorizer,
	// 	Transactor:   services.Transactor,
	// 	Translator:   services.Translator,
	// }

	return &UseCases{
		CreateStageTemplate:          NewCreateStageTemplateUseCase(createRepos, createServices),
		ReadStageTemplate:            NewReadStageTemplateUseCase(readRepos, readServices),
		UpdateStageTemplate:          NewUpdateStageTemplateUseCase(updateRepos, updateServices),
		DeleteStageTemplate:          NewDeleteStageTemplateUseCase(deleteRepos, deleteServices),
		ListStageTemplates:           NewListStageTemplatesUseCase(listRepos, listServices),
		GetStageTemplateListPageData: NewGetStageTemplateListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetStageTemplateItemPageData: NewGetStageTemplateItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
		// GetStageTemplatesByWorkflow:  NewGetStageTemplatesByWorkflowUseCase(getByWorkflowRepos, getByWorkflowServices), // TODO: Implement
	}
}

// NewUseCasesUngrouped creates a new collection of stage template use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(stageTemplateRepo stageTemplatepb.StageTemplateDomainServiceServer, workflowTemplateRepo workflowTemplatepb.WorkflowTemplateDomainServiceServer) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := StageTemplateRepositories{
		StageTemplate:    stageTemplateRepo,
		WorkflowTemplate: workflowTemplateRepo,
	}

	services := StageTemplateServices{
		Authorizer:  nil, // Will be injected later by container
		Transactor:  ports.NewNoOpTransactor(),
		Translator:  ports.NewNoOpTranslator(),
		IDGenerator: ports.NewNoOpIDGenerator(),
	}

	return NewUseCases(repositories, services)
}

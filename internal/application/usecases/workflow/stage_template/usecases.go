package stage_template

import (
	"leapfor.xyz/espyna/internal/application/ports"
	stageTemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/stage_template"
	workflowTemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/workflow_template"
)

// StageTemplateRepositories groups all repository dependencies for stage template use cases
type StageTemplateRepositories struct {
	StageTemplate    stageTemplatepb.StageTemplateDomainServiceServer       // Primary entity repository
	WorkflowTemplate workflowTemplatepb.WorkflowTemplateDomainServiceServer // Foreign key reference
}

// StageTemplateServices groups all business service dependencies for stage template use cases
type StageTemplateServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService // Required for Create use case
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadStageTemplateRepositories{
		StageTemplate: repositories.StageTemplate,
	}
	readServices := ReadStageTemplateServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateStageTemplateRepositories{
		StageTemplate:    repositories.StageTemplate,
		WorkflowTemplate: repositories.WorkflowTemplate,
	}
	updateServices := UpdateStageTemplateServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteStageTemplateRepositories{
		StageTemplate: repositories.StageTemplate,
	}
	deleteServices := DeleteStageTemplateServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListStageTemplatesRepositories{
		StageTemplate: repositories.StageTemplate,
	}
	listServices := ListStageTemplatesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetStageTemplateListPageDataRepositories{
		StageTemplate:    repositories.StageTemplate,
		WorkflowTemplate: repositories.WorkflowTemplate,
	}
	getListPageDataServices := GetStageTemplateListPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	getItemPageDataRepos := GetStageTemplateItemPageDataRepositories{
		StageTemplate:    repositories.StageTemplate,
		WorkflowTemplate: repositories.WorkflowTemplate,
	}
	getItemPageDataServices := GetStageTemplateItemPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	// TODO: Implement when GetStageTemplatesByWorkflow use case is available
	// getByWorkflowRepos := GetStageTemplatesByWorkflowRepositories(repositories)
	// getByWorkflowServices := GetStageTemplatesByWorkflowServices{
	// 	AuthorizationService: services.AuthorizationService,
	// 	TransactionService:   services.TransactionService,
	// 	TranslationService:   services.TranslationService,
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
		AuthorizationService: nil, // Will be injected later by container
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewUseCases(repositories, services)
}

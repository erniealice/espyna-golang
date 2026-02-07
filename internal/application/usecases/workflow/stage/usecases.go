package stage

import (
	"leapfor.xyz/espyna/internal/application/ports"
	stagepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/stage"
	stageTemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/stage_template"
	workflowpb "leapfor.xyz/esqyma/golang/v1/domain/workflow/workflow"
)

// StageRepositories groups all repository dependencies for stage use cases
type StageRepositories struct {
	Stage         stagepb.StageDomainServiceServer                 // Primary entity repository
	Workflow      workflowpb.WorkflowDomainServiceServer           // Foreign key reference
	StageTemplate stageTemplatepb.StageTemplateDomainServiceServer // Foreign key reference
}

// StageServices groups all business service dependencies for stage use cases
type StageServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService // Required for Create use case
}

// UseCases contains all stage-related use cases
type UseCases struct {
	CreateStage *CreateStageUseCase
	ReadStage   *ReadStageUseCase
	UpdateStage *UpdateStageUseCase
	DeleteStage *DeleteStageUseCase
	ListStages  *ListStagesUseCase
}

// NewUseCases creates a new collection of stage use cases
func NewUseCases(
	repositories StageRepositories,
	services StageServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateStageRepositories{
		Stage:         repositories.Stage,
		Workflow:      repositories.Workflow,
		StageTemplate: repositories.StageTemplate,
	}
	createServices := CreateStageServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}
	readRepos := ReadStageRepositories{
		Stage: repositories.Stage,
	}
	readServices := ReadStageServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateStageRepositories{
		Stage: repositories.Stage,
	}
	updateServices := UpdateStageServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteStageRepositories{
		Stage: repositories.Stage,
	}
	deleteServices := DeleteStageServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListStagesRepositories{
		Stage: repositories.Stage,
	}
	listServices := ListStagesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateStage: NewCreateStageUseCase(createRepos, createServices),
		ReadStage:   NewReadStageUseCase(readRepos, readServices),
		UpdateStage: NewUpdateStageUseCase(updateRepos, updateServices),
		DeleteStage: NewDeleteStageUseCase(deleteRepos, deleteServices),
		ListStages:  NewListStagesUseCase(listRepos, listServices),
	}
}

package stage

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	stagepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/stage"
	stageTemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/stage_template"
	workflowpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/workflow"
)

// StageRepositories groups all repository dependencies for stage use cases
type StageRepositories struct {
	Stage         stagepb.StageDomainServiceServer                 // Primary entity repository
	Workflow      workflowpb.WorkflowDomainServiceServer           // Foreign key reference
	StageTemplate stageTemplatepb.StageTemplateDomainServiceServer // Foreign key reference
}

// StageServices groups all business service dependencies for stage use cases
type StageServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator // Required for Create use case
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
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}
	readRepos := ReadStageRepositories{
		Stage: repositories.Stage,
	}
	readServices := ReadStageServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateStageRepositories{
		Stage: repositories.Stage,
	}
	updateServices := UpdateStageServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteStageRepositories{
		Stage: repositories.Stage,
	}
	deleteServices := DeleteStageServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListStagesRepositories{
		Stage: repositories.Stage,
	}
	listServices := ListStagesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateStage: NewCreateStageUseCase(createRepos, createServices),
		ReadStage:   NewReadStageUseCase(readRepos, readServices),
		UpdateStage: NewUpdateStageUseCase(updateRepos, updateServices),
		DeleteStage: NewDeleteStageUseCase(deleteRepos, deleteServices),
		ListStages:  NewListStagesUseCase(listRepos, listServices),
	}
}

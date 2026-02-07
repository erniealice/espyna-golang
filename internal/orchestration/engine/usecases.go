package engine

import (
	"context"
	"leapfor.xyz/espyna/internal/application/ports"

	activitypb "leapfor.xyz/esqyma/golang/v1/domain/workflow/activity"
	activitytemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/activity_template"
	stagepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/stage"
	stagetemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/stage_template"
	workflowpb "leapfor.xyz/esqyma/golang/v1/domain/workflow/workflow"
	workflowtemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/workflow_template"
	enginepb "leapfor.xyz/esqyma/golang/v1/orchestration/engine"
)

// EngineRepositories groups all repository dependencies for engine use cases
type EngineRepositories struct {
	Workflow         workflowpb.WorkflowDomainServiceServer
	WorkflowTemplate workflowtemplatepb.WorkflowTemplateDomainServiceServer
	Stage            stagepb.StageDomainServiceServer
	StageTemplate    stagetemplatepb.StageTemplateDomainServiceServer
	Activity         activitypb.ActivityDomainServiceServer
	ActivityTemplate activitytemplatepb.ActivityTemplateDomainServiceServer
}

// EngineServices groups all business service dependencies for engine use cases
type EngineServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
	ExecutorRegistry     ports.ExecutorRegistry
}

// EngineUseCases contains all workflow engine-related use cases and implements
// the WorkflowEngineService port for the orchestration layer.
type EngineUseCases struct {
	startWorkflowUC    *StartWorkflowFromTemplateUseCase
	executeActivityUC  *ExecuteActivityUseCase
	advanceWorkflowUC  *AdvanceWorkflowUseCase
	getStatusUC        *GetWorkflowStatusUseCase
	continueWorkflowUC *ContinueWorkflowUseCase
	runToCompletionUC  *RunToCompletionUseCase
}

// NewUseCases creates a new collection of engine use cases
func NewUseCases(
	repositories EngineRepositories,
	services EngineServices,
) *EngineUseCases {
	// Create shared template cache for all use cases
	cache := NewTemplateCache(repositories)

	// Create base use cases first
	startUC := NewStartWorkflowFromTemplateUseCase(repositories, services, cache)
	statusUC := NewGetWorkflowStatusUseCase(repositories, services)
	executeUC := NewExecuteActivityUseCase(repositories, services, cache)
	advanceUC := NewAdvanceWorkflowUseCase(repositories, services, cache)

	return &EngineUseCases{
		startWorkflowUC:    startUC,
		executeActivityUC:  executeUC,
		advanceWorkflowUC:  advanceUC,
		getStatusUC:        statusUC,
		continueWorkflowUC: NewContinueWorkflowUseCase(repositories, services, cache),
		runToCompletionUC:  NewRunToCompletionUseCase(repositories, services, cache, startUC, statusUC, executeUC, advanceUC),
	}
}

// Statically check that EngineUseCases implements the WorkflowEngineService interface
var _ ports.WorkflowEngineService = (*EngineUseCases)(nil)

// StartWorkflowFromTemplate implements ports.WorkflowEngineService
func (e *EngineUseCases) StartWorkflowFromTemplate(ctx context.Context, req *enginepb.StartWorkflowRequest) (*enginepb.StartWorkflowResponse, error) {
	return e.startWorkflowUC.Execute(ctx, req)
}

// ExecuteActivity implements ports.WorkflowEngineService
func (e *EngineUseCases) ExecuteActivity(ctx context.Context, req *enginepb.ExecuteActivityRequest) (*enginepb.ExecuteActivityResponse, error) {
	return e.executeActivityUC.Execute(ctx, req)
}

// AdvanceWorkflow implements ports.WorkflowEngineService
func (e *EngineUseCases) AdvanceWorkflow(ctx context.Context, req *enginepb.AdvanceWorkflowRequest) (*enginepb.AdvanceWorkflowResponse, error) {
	return e.advanceWorkflowUC.Execute(ctx, req)
}

// GetWorkflowStatus implements ports.WorkflowEngineService
func (e *EngineUseCases) GetWorkflowStatus(ctx context.Context, req *enginepb.GetWorkflowStatusRequest) (*enginepb.GetWorkflowStatusResponse, error) {
	return e.getStatusUC.Execute(ctx, req)
}

// ContinueWorkflow implements ports.WorkflowEngineService
func (e *EngineUseCases) ContinueWorkflow(ctx context.Context, req *enginepb.ContinueWorkflowRequest) (*enginepb.ContinueWorkflowResponse, error) {
	return e.continueWorkflowUC.Execute(ctx, req)
}

// RunToCompletion implements ports.WorkflowEngineService
func (e *EngineUseCases) RunToCompletion(ctx context.Context, req *enginepb.RunToCompletionRequest) (*enginepb.RunToCompletionResponse, error) {
	return e.runToCompletionUC.Execute(ctx, req)
}

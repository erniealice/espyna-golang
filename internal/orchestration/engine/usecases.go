package engine

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"

	activitypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/activity"
	activitytemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/activity_template"
	stagepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/stage"
	stagetemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/stage_template"
	workflowpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/workflow"
	workflowtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/workflow_template"
	enginepb "github.com/erniealice/esqyma/pkg/schema/v1/orchestration/engine"
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
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ExecutorRegistry ports.ExecutorRegistry
}

// EngineUseCases contains all workflow engine-related use cases and implements
// the WorkflowEngineService port for the orchestration layer.
// It also implements WorkflowAssigneeQueryService (Q-EIB-IFACE) when an
// AssigneeQueryRepository is wired via SetAssigneeQueryRepository.
type EngineUseCases struct {
	startWorkflowUC    *StartWorkflowFromTemplateUseCase
	executeActivityUC  *ExecuteActivityUseCase
	advanceWorkflowUC  *AdvanceWorkflowUseCase
	getStatusUC        *GetWorkflowStatusUseCase
	continueWorkflowUC *ContinueWorkflowUseCase
	runToCompletionUC  *RunToCompletionUseCase

	// Engine identity bridge (Q-EIB-BRIDGE): read-only query for pending
	// activities assigned to a workspace user through the user_id bridge.
	// Set via SetAssigneeQueryRepository after adapter initialization.
	listAssigneeActivitiesUC *ListPendingActivitiesForAssigneeUseCase
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

// Statically check that EngineUseCases implements the WorkflowAssigneeQueryService interface
var _ ports.WorkflowAssigneeQueryService = (*EngineUseCases)(nil)

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

// SetAssigneeQueryRepository wires the identity bridge adapter so that
// EngineUseCases can serve WorkflowAssigneeQueryService. This setter
// pattern allows the adapter to be initialized after the engine use cases
// are created (the postgres adapter needs the DB connection, which is
// available after provider initialization).
//
// The parameter is typed as `any` so the container (which cannot import
// the engine package) can call this method via a structural interface
// assertion. The concrete value must implement AssigneeQueryRepository.
func (e *EngineUseCases) SetAssigneeQueryRepository(repo any) {
	if repo == nil {
		return
	}
	typedRepo, ok := repo.(AssigneeQueryRepository)
	if !ok {
		return
	}
	e.listAssigneeActivitiesUC = NewListPendingActivitiesForAssigneeUseCase(typedRepo)
}

// ListPendingActivitiesForAssignee implements ports.WorkflowAssigneeQueryService.
// Returns pending engine activities assigned to the human behind the given
// workspace_user_id. The bridge resolves through workspace_user.user_id.
//
// If the assignee query repository has not been wired (via
// SetAssigneeQueryRepository), returns an empty result — fail-closed.
func (e *EngineUseCases) ListPendingActivitiesForAssignee(
	ctx context.Context,
	req *ports.ListPendingActivitiesForAssigneeRequest,
) (*ports.ListPendingActivitiesForAssigneeResponse, error) {
	if e.listAssigneeActivitiesUC == nil {
		// Assignee query not wired — fail-closed with empty result.
		return &ports.ListPendingActivitiesForAssigneeResponse{
			Activities: nil,
			Total:      0,
		}, nil
	}
	return e.listAssigneeActivitiesUC.Execute(ctx, req)
}

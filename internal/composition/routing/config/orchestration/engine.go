package orchestration

import (
	"context"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	"leapfor.xyz/espyna/internal/composition/contracts"

	enginepb "leapfor.xyz/esqyma/golang/v1/orchestration/engine"
)

// ============================================================================
// Adapter wrappers to convert interface methods to UseCaseExecutor pattern
// ============================================================================

// startWorkflowAdapter wraps StartWorkflowFromTemplate to implement UseCaseExecutor
type startWorkflowAdapter struct {
	service ports.WorkflowEngineService
}

func (a *startWorkflowAdapter) Execute(ctx context.Context, req *enginepb.StartWorkflowRequest) (*enginepb.StartWorkflowResponse, error) {
	return a.service.StartWorkflowFromTemplate(ctx, req)
}

// getWorkflowStatusAdapter wraps GetWorkflowStatus to implement UseCaseExecutor
type getWorkflowStatusAdapter struct {
	service ports.WorkflowEngineService
}

func (a *getWorkflowStatusAdapter) Execute(ctx context.Context, req *enginepb.GetWorkflowStatusRequest) (*enginepb.GetWorkflowStatusResponse, error) {
	return a.service.GetWorkflowStatus(ctx, req)
}

// continueWorkflowAdapter wraps ContinueWorkflow to implement UseCaseExecutor
type continueWorkflowAdapter struct {
	service ports.WorkflowEngineService
}

func (a *continueWorkflowAdapter) Execute(ctx context.Context, req *enginepb.ContinueWorkflowRequest) (*enginepb.ContinueWorkflowResponse, error) {
	return a.service.ContinueWorkflow(ctx, req)
}

// executeActivityAdapter wraps ExecuteActivity to implement UseCaseExecutor
type executeActivityAdapter struct {
	service ports.WorkflowEngineService
}

func (a *executeActivityAdapter) Execute(ctx context.Context, req *enginepb.ExecuteActivityRequest) (*enginepb.ExecuteActivityResponse, error) {
	return a.service.ExecuteActivity(ctx, req)
}

// advanceWorkflowAdapter wraps AdvanceWorkflow to implement UseCaseExecutor
type advanceWorkflowAdapter struct {
	service ports.WorkflowEngineService
}

func (a *advanceWorkflowAdapter) Execute(ctx context.Context, req *enginepb.AdvanceWorkflowRequest) (*enginepb.AdvanceWorkflowResponse, error) {
	return a.service.AdvanceWorkflow(ctx, req)
}

// runToCompletionAdapter wraps RunToCompletion to implement UseCaseExecutor
type runToCompletionAdapter struct {
	service ports.WorkflowEngineService
}

func (a *runToCompletionAdapter) Execute(ctx context.Context, req *enginepb.RunToCompletionRequest) (*enginepb.RunToCompletionResponse, error) {
	return a.service.RunToCompletion(ctx, req)
}

// ============================================================================
// Route Configuration
// ============================================================================

// ConfigureWorkflowEngine configures routes for the Workflow Engine (Orchestration layer).
// This is separate from domain routing because orchestration coordinates use cases
// rather than exposing domain CRUD operations.
func ConfigureWorkflowEngine(engineService ports.WorkflowEngineService) contracts.DomainRouteConfiguration {
	if engineService == nil {
		fmt.Printf("⚠️  Workflow engine service is NIL\n")
		return contracts.DomainRouteConfiguration{
			Domain:  "orchestration",
			Prefix:  "/orchestration",
			Enabled: false,
			Routes:  []contracts.RouteConfiguration{},
		}
	}

	fmt.Printf("✅ Workflow engine service is properly initialized!\n")

	routes := []contracts.RouteConfiguration{
		{
			Method:  "POST",
			Path:    "/api/workflow/engine/start",
			Handler: contracts.NewGenericHandler(&startWorkflowAdapter{service: engineService}, &enginepb.StartWorkflowRequest{}),
		},
		{
			Method:  "POST",
			Path:    "/api/workflow/engine/status",
			Handler: contracts.NewGenericHandler(&getWorkflowStatusAdapter{service: engineService}, &enginepb.GetWorkflowStatusRequest{}),
		},
		{
			Method:  "POST",
			Path:    "/api/workflow/engine/continue",
			Handler: contracts.NewGenericHandler(&continueWorkflowAdapter{service: engineService}, &enginepb.ContinueWorkflowRequest{}),
		},
		{
			Method:  "POST",
			Path:    "/api/workflow/engine/execute-activity",
			Handler: contracts.NewGenericHandler(&executeActivityAdapter{service: engineService}, &enginepb.ExecuteActivityRequest{}),
		},
		{
			Method:  "POST",
			Path:    "/api/workflow/engine/advance",
			Handler: contracts.NewGenericHandler(&advanceWorkflowAdapter{service: engineService}, &enginepb.AdvanceWorkflowRequest{}),
		},
		{
			Method:  "POST",
			Path:    "/api/workflow/engine/run",
			Handler: contracts.NewGenericHandler(&runToCompletionAdapter{service: engineService}, &enginepb.RunToCompletionRequest{}),
		},
	}

	return contracts.DomainRouteConfiguration{
		Domain:  "orchestration",
		Prefix:  "/orchestration",
		Enabled: true,
		Routes:  routes,
	}
}

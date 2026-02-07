package domain

import (
	"context"

	enginepb "leapfor.xyz/esqyma/golang/v1/orchestration/engine"
)

// WorkflowEngineService defines the port for the orchestration engine.
// The application layer uses this interface to start and manage workflows,
// without knowing the details of the orchestration implementation.
type WorkflowEngineService interface {
	StartWorkflowFromTemplate(ctx context.Context, req *enginepb.StartWorkflowRequest) (*enginepb.StartWorkflowResponse, error)
	ExecuteActivity(ctx context.Context, req *enginepb.ExecuteActivityRequest) (*enginepb.ExecuteActivityResponse, error)
	AdvanceWorkflow(ctx context.Context, req *enginepb.AdvanceWorkflowRequest) (*enginepb.AdvanceWorkflowResponse, error)
	GetWorkflowStatus(ctx context.Context, req *enginepb.GetWorkflowStatusRequest) (*enginepb.GetWorkflowStatusResponse, error)
	ContinueWorkflow(ctx context.Context, req *enginepb.ContinueWorkflowRequest) (*enginepb.ContinueWorkflowResponse, error)
	RunToCompletion(ctx context.Context, req *enginepb.RunToCompletionRequest) (*enginepb.RunToCompletionResponse, error)
}

// ActivityExecutor defines the interface for executing a bound use case dynamically
type ActivityExecutor interface {
	// Execute takes a map (resolved from dynamic context), converts it to the specific Proto request,
	// runs the underlying use case, and returns the result as a map.
	Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)
}

// ExecutorRegistry defines the interface for looking up executors by their use case code
type ExecutorRegistry interface {
	// GetExecutor returns an executor for the given activity code (e.g., "entity.client.create")
	GetExecutor(activityCode string) (ActivityExecutor, error)
}

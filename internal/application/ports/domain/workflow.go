package domain

import (
	"context"

	activitypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/activity"
	enginepb "github.com/erniealice/esqyma/pkg/schema/v1/orchestration/engine"
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

// ListPendingActivitiesForAssigneeRequest carries the identity inputs for the
// engine-identity bridge query. Both fields are sourced from session context
// (workspace_user_id = staff principal_id; workspace_id from workspace path
// middleware). They are NEVER taken from the wire / request params.
type ListPendingActivitiesForAssigneeRequest struct {
	// WorkspaceUserID is the session's principal_id (the active workspace_user hat).
	WorkspaceUserID string
	// WorkspaceID is the active workspace, sourced from context.
	WorkspaceID string
	// Limit caps the number of results. Zero means use the adapter default.
	Limit int
	// Offset for pagination.
	Offset int
}

// ListPendingActivitiesForAssigneeResponse wraps the query results.
type ListPendingActivitiesForAssigneeResponse struct {
	Activities []*activitypb.Activity
	Total      int
}

// WorkflowAssigneeQueryService is a read-only query port that resolves engine
// activities assigned to a given workspace user. It bridges the two identity
// surfaces: record-level workspace_user.id and engine-level Activity.assigned_to
// (a global user.id) by JOINing through workspace_user.user_id.
//
// This is a SEPARATE port from WorkflowEngineService (Q-EIB-IFACE): the engine
// execution port (Start/Advance/Execute/Status/Continue/RunToCompletion) is an
// orthogonal write concern. Keeping them apart caps the compile blast radius —
// existing engine mocks need no stub for this method.
type WorkflowAssigneeQueryService interface {
	// ListPendingActivitiesForAssignee returns pending engine activities
	// assigned to the human behind the given workspace_user_id, scoped to
	// the given workspace_id. The bridge join is:
	//   activity.assigned_to = workspace_user.user_id
	//   WHERE workspace_user.id = req.WorkspaceUserID
	//     AND work_request.workspace_id = req.WorkspaceID
	//
	// Fail-closed: empty WorkspaceUserID or WorkspaceID returns an empty
	// result with no SQL executed.
	ListPendingActivitiesForAssignee(ctx context.Context, req *ListPendingActivitiesForAssigneeRequest) (*ListPendingActivitiesForAssigneeResponse, error)
}

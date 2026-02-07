package engine

import (
	"context"
	"fmt"

	enginepb "leapfor.xyz/esqyma/golang/v1/orchestration/engine"
)

// RunToCompletionUseCase executes a workflow from start to finish in a single call.
// It loops through all automated activities until completion or error.
// This is particularly useful for handlers that need immediate execution results.
type RunToCompletionUseCase struct {
	repositories EngineRepositories
	services     EngineServices
	cache        *TemplateCache
	startUC      *StartWorkflowFromTemplateUseCase
	statusUC     *GetWorkflowStatusUseCase
	executeUC    *ExecuteActivityUseCase
	advanceUC    *AdvanceWorkflowUseCase
}

// NewRunToCompletionUseCase creates a new RunToCompletion use case instance.
func NewRunToCompletionUseCase(
	repositories EngineRepositories,
	services EngineServices,
	cache *TemplateCache,
	startUC *StartWorkflowFromTemplateUseCase,
	statusUC *GetWorkflowStatusUseCase,
	executeUC *ExecuteActivityUseCase,
	advanceUC *AdvanceWorkflowUseCase,
) *RunToCompletionUseCase {
	return &RunToCompletionUseCase{
		repositories: repositories,
		services:     services,
		cache:        cache,
		startUC:      startUC,
		statusUC:     statusUC,
		executeUC:    executeUC,
		advanceUC:    advanceUC,
	}
}

// Execute runs the workflow to completion by iterating through stages and activities.
// The algorithm:
//  1. Start workflow from template
//  2. Loop until complete or max iterations reached:
//     a. Get current status
//     b. Return if workflow is completed
//     c. If no pending activity, try to advance to next stage
//     d. If pending activity is manual (human task), return error
//     e. Execute the pending automated activity
func (uc *RunToCompletionUseCase) Execute(ctx context.Context, req *enginepb.RunToCompletionRequest) (*enginepb.RunToCompletionResponse, error) {
	// Set default max iterations for safety
	maxIterations := int32(100)
	if req.MaxIterations != nil && *req.MaxIterations > 0 {
		maxIterations = *req.MaxIterations
	}

	// 1. Start workflow from template
	startResp, err := uc.startUC.Execute(ctx, &enginepb.StartWorkflowRequest{
		WorkflowTemplateId: req.WorkflowTemplateId,
		InputJson:          req.InputJson,
		WorkspaceId:        req.WorkspaceId,
	})
	if err != nil {
		return nil, fmt.Errorf("start workflow: %w", err)
	}
	if !startResp.Success {
		return &enginepb.RunToCompletionResponse{
			Success: false,
			Error:   startResp.Error,
		}, nil
	}

	workflowId := startResp.Workflow.Id
	var iterations int32 = 0

	// 2. Loop until complete or max iterations reached
	for iterations < maxIterations {
		iterations++

		// Get current status
		statusResp, err := uc.statusUC.Execute(ctx, &enginepb.GetWorkflowStatusRequest{
			WorkflowId: workflowId,
		})
		if err != nil {
			return nil, fmt.Errorf("get status: %w", err)
		}
		if !statusResp.Success {
			return &enginepb.RunToCompletionResponse{
				Success: false,
				Error:   statusResp.Error,
			}, nil
		}
		if statusResp.Workflow == nil {
			return nil, fmt.Errorf("workflow not found in status response")
		}

		// Check if workflow is complete
		if statusResp.Workflow.Status == "completed" {
			outputJSON := ""
			if statusResp.Workflow.ContextJson != nil {
				outputJSON = *statusResp.Workflow.ContextJson
			}
			return &enginepb.RunToCompletionResponse{
				Success:    true,
				Workflow:   statusResp.Workflow,
				OutputJson: outputJSON,
			}, nil
		}

		// Check for pending activity
		pendingActivityID := ""
		if statusResp.PendingActivityId != nil {
			pendingActivityID = *statusResp.PendingActivityId
		}

		if pendingActivityID == "" {
			// No pending activity, try to advance to next stage
			advResp, err := uc.advanceUC.Execute(ctx, &enginepb.AdvanceWorkflowRequest{
				WorkflowId: workflowId,
			})
			if err != nil {
				return nil, fmt.Errorf("advance workflow: %w", err)
			}
			if advResp.WorkflowCompleted {
				// Fetch final workflow state
				finalStatus, _ := uc.statusUC.Execute(ctx, &enginepb.GetWorkflowStatusRequest{
					WorkflowId: workflowId,
				})
				outputJSON := ""
				if finalStatus != nil && finalStatus.Workflow != nil && finalStatus.Workflow.ContextJson != nil {
					outputJSON = *finalStatus.Workflow.ContextJson
				}
				return &enginepb.RunToCompletionResponse{
					Success:    true,
					Workflow:   finalStatus.Workflow,
					OutputJson: outputJSON,
				}, nil
			}
			// Continue to next iteration after advancing
			continue
		}

		// Check if activity is manual (cannot auto-execute)
		for _, activity := range statusResp.Activities {
			if activity.Id == pendingActivityID {
				// Fetch activity template to check type via cache
				template, err := uc.cache.GetActivityTemplate(ctx, activity.ActivityTemplateId)
				if err == nil && template != nil {
					activityType := template.ActivityType
					if activityType == "manual" || activityType == "human_task" || activityType == "approval" {
						return nil, fmt.Errorf("workflow has manual activity %s that cannot be auto-executed", activity.Name)
					}
				}
				break
			}
		}

		// Execute the pending activity
		execResp, err := uc.executeUC.Execute(ctx, &enginepb.ExecuteActivityRequest{
			ActivityId: pendingActivityID,
			WorkflowId: workflowId,
		})
		if err != nil {
			return nil, fmt.Errorf("execute activity: %w", err)
		}
		if !execResp.Success {
			return &enginepb.RunToCompletionResponse{
				Success: false,
				Error:   execResp.Error,
			}, nil
		}
	}

	return nil, fmt.Errorf("max iterations (%d) exceeded", maxIterations)
}

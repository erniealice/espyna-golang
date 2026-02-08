package engine

import (
	"context"
	"errors"
	"fmt"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	activitypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/activity"
	activitytemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/activity_template"
	stagepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/stage"
	workflowpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/workflow"
	enginepb "github.com/erniealice/esqyma/pkg/schema/v1/orchestration/engine"
)

// GetWorkflowStatusUseCase retrieves the current state of a workflow
type GetWorkflowStatusUseCase struct {
	repositories EngineRepositories
	services     EngineServices
}

// NewGetWorkflowStatusUseCase creates a new use case
func NewGetWorkflowStatusUseCase(repos EngineRepositories, svcs EngineServices) *GetWorkflowStatusUseCase {
	return &GetWorkflowStatusUseCase{
		repositories: repos,
		services:     svcs,
	}
}

// Execute retrieves workflow status including current stage and activities
func (uc *GetWorkflowStatusUseCase) Execute(ctx context.Context, req *enginepb.GetWorkflowStatusRequest) (*enginepb.GetWorkflowStatusResponse, error) {
	if req == nil || req.WorkflowId == "" {
		return nil, errors.New("workflow_id is required")
	}

	// 1. Fetch Workflow
	workflowRes, err := uc.repositories.Workflow.ReadWorkflow(ctx, &workflowpb.ReadWorkflowRequest{
		Data: &workflowpb.Workflow{Id: req.WorkflowId},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow: %w", err)
	}
	if workflowRes == nil || !workflowRes.Success || len(workflowRes.Data) == 0 {
		return &enginepb.GetWorkflowStatusResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "WORKFLOW_NOT_FOUND",
				Message: fmt.Sprintf("Workflow not found: %s", req.WorkflowId),
			},
		}, nil
	}
	workflow := workflowRes.Data[0]

	// 2. Find current stage (most recent non-completed stage)
	stagesRes, err := uc.repositories.Stage.ListStages(ctx, &stagepb.ListStagesRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				{
					Field: "workflow_id",
					FilterType: &commonpb.TypedFilter_StringFilter{
						StringFilter: &commonpb.StringFilter{
							Value:    req.WorkflowId,
							Operator: commonpb.StringOperator_STRING_EQUALS,
						},
					},
				},
			},
		},
		Sort: &commonpb.SortRequest{
			Fields: []*commonpb.SortField{
				{
					Field:     "date_created",
					Direction: commonpb.SortDirection_DESC,
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list stages: %w", err)
	}

	var currentStage *stagepb.Stage
	if len(stagesRes.Data) > 0 {
		// Find the most recent non-completed stage, or the latest one
		for _, stage := range stagesRes.Data {
			if stage.Status != "completed" {
				currentStage = stage
				break
			}
		}
		// If all stages are completed, use the most recent one
		if currentStage == nil {
			currentStage = stagesRes.Data[0]
		}
	}

	// 3. Fetch activities for current stage
	var activities []*activitypb.Activity
	var pendingActivityId string

	if currentStage != nil {
		activitiesRes, err := uc.repositories.Activity.ListActivities(ctx, &activitypb.ListActivitiesRequest{
			Filters: &commonpb.FilterRequest{
				Filters: []*commonpb.TypedFilter{
					{
						Field: "stage_id",
						FilterType: &commonpb.TypedFilter_StringFilter{
							StringFilter: &commonpb.StringFilter{
								Value:    currentStage.Id,
								Operator: commonpb.StringOperator_STRING_EQUALS,
							},
						},
					},
				},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list activities: %w", err)
		}

		activities = activitiesRes.Data

		// Find pending human activity
		for _, activity := range activities {
			if activity.Status == "pending" {
				// Check if this is a human task by looking at the template
				templateRes, _ := uc.repositories.ActivityTemplate.ReadActivityTemplate(ctx, &activitytemplatepb.ReadActivityTemplateRequest{
					Data: &activitytemplatepb.ActivityTemplate{Id: activity.ActivityTemplateId},
				})
				if templateRes != nil && len(templateRes.Data) > 0 {
					template := templateRes.Data[0]
					if template.ActivityType == "human_task" || template.ActivityType == "approval" {
						pendingActivityId = activity.Id
						break
					}
				}
				// If no template or not human_task, still mark first pending as potential
				if pendingActivityId == "" {
					pendingActivityId = activity.Id
				}
			}
		}
	}

	response := &enginepb.GetWorkflowStatusResponse{
		Success:      true,
		Workflow:     workflow,
		CurrentStage: currentStage,
		Activities:   activities,
	}

	if pendingActivityId != "" {
		response.PendingActivityId = &pendingActivityId
	}

	return response, nil
}

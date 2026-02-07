package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	activitypb "leapfor.xyz/esqyma/golang/v1/domain/workflow/activity"
	stagepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/stage"
	stagetemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/stage_template"
	workflowpb "leapfor.xyz/esqyma/golang/v1/domain/workflow/workflow"
	enginepb "leapfor.xyz/esqyma/golang/v1/orchestration/engine"
)

// AdvanceWorkflowUseCase handles checking progress and moving to next steps
type AdvanceWorkflowUseCase struct {
	repositories EngineRepositories
	services     EngineServices
	cache        *TemplateCache
}

// NewAdvanceWorkflowUseCase creates a new use case
func NewAdvanceWorkflowUseCase(repos EngineRepositories, svcs EngineServices, cache *TemplateCache) *AdvanceWorkflowUseCase {
	return &AdvanceWorkflowUseCase{
		repositories: repos,
		services:     svcs,
		cache:        cache,
	}
}

// Execute checks current state and advances if possible
func (uc *AdvanceWorkflowUseCase) Execute(ctx context.Context, req *enginepb.AdvanceWorkflowRequest) (*enginepb.AdvanceWorkflowResponse, error) {
	if req == nil || req.WorkflowId == "" {
		return nil, errors.New("workflow_id is required")
	}

	// 1. Fetch Workflow
	workflowRes, err := uc.repositories.Workflow.ReadWorkflow(ctx, &workflowpb.ReadWorkflowRequest{
		Data: &workflowpb.Workflow{Id: req.WorkflowId},
	})
	if err != nil || !workflowRes.Success || len(workflowRes.Data) == 0 {
		return nil, fmt.Errorf("failed to read workflow: %w", err)
	}
	workflow := workflowRes.Data[0]

	// 2. Find Current Stage (latest active/pending stage)
	// Simplified logic: We assume the stage matching current_stage_index is the active one
	// Or we look for the last created stage
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
		currentStage = stagesRes.Data[0]
	}

	if currentStage == nil {
		return nil, fmt.Errorf("no stages found for workflow")
	}

	// 3. Check if current stage is complete
	// Logic: Check if all activities in this stage are completed
	// Fetch activities for this stage
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

	allActivitiesComplete := true
	for _, act := range activitiesRes.Data {
		if act.Status != "completed" && act.Status != "skipped" {
			allActivitiesComplete = false
			break
		}
	}

	// If stage activities are not done, we can't advance stage.
	// But maybe we need to create the activities if they don't exist yet?
	if len(activitiesRes.Data) == 0 {
		// Need to instantiate activities from template
		// This happens when stage is first created
		// StartWorkflow only created the Stage, so we need to create activities here.

		// Fetch activity templates for this stage's template via cache
		activityTemplates, err := uc.cache.GetActivityTemplatesForStage(ctx, currentStage.StageTemplateId)
		if err != nil {
			return nil, fmt.Errorf("failed to list activity templates: %w", err)
		}

		// Get stage template to capture its order_index for hierarchical context storage
		stageTemplate, err := uc.cache.GetStageTemplate(ctx, currentStage.StageTemplateId)
		if err != nil {
			return nil, fmt.Errorf("failed to get stage template: %w", err)
		}

		// Create activity instances from templates
		now := time.Now()
		for _, activityTemplate := range activityTemplates {
			activityID := uc.services.IDService.GenerateID()
			activity := &activitypb.Activity{
				Id:                       activityID,
				StageId:                  currentStage.Id,
				ActivityTemplateId:       activityTemplate.Id,
				Name:                     activityTemplate.Name,
				Description:              activityTemplate.Description,
				Status:                   "pending",
				Priority:                 "medium", // Default priority
				EstimatedDurationMinutes: activityTemplate.EstimatedDurationMinutes,
				InputDataJson:            activityTemplate.InputSchemaJson,
				Active:                   true,
				DateCreated:              &[]int64{now.UnixMilli()}[0],
				DateCreatedString:        &[]string{now.Format(time.RFC3339)}[0],
				OrderIndex:               activityTemplate.OrderIndex, // Copy from activity template
				StageOrderIndex:          stageTemplate.OrderIndex,    // Copy from stage template
			}

			// Set assignee if specified in template
			if activityTemplate.DefaultAssigneeId != nil && *activityTemplate.DefaultAssigneeId != "" {
				activity.AssignedTo = activityTemplate.DefaultAssigneeId
				activity.DateAssigned = &[]int64{now.UnixMilli()}[0]
				activity.DateAssignedString = &[]string{now.Format(time.RFC3339)}[0]
			}

			_, createErr := uc.repositories.Activity.CreateActivity(ctx, &activitypb.CreateActivityRequest{
				Data: activity,
			})
			if createErr != nil {
				// Log error but continue creating other activities
				fmt.Printf("Warning: failed to create activity from template %s: %v\n", activityTemplate.Id, createErr)
			}
		}

		// Return to indicate activities were created, workflow can now proceed
		return &enginepb.AdvanceWorkflowResponse{
			Success:     true,
			NextStageId: currentStage.Id,
		}, nil
	}

	if !allActivitiesComplete {
		return &enginepb.AdvanceWorkflowResponse{
			Success:     true,
			NextStageId: currentStage.Id, // Still on same stage
		}, nil
	}

	// 4. Current Stage is Done. Close it.
	if currentStage.Status != "completed" {
		currentStage.Status = "completed"
		uc.repositories.Stage.UpdateStage(ctx, &stagepb.UpdateStageRequest{Data: currentStage})
	}

	// 5. Find Next Stage Template via cache
	// Get all stage templates for workflow (sorted by order_index)
	// Find current template and determine next one
	stageTemplates, err := uc.cache.GetStageTemplates(ctx, *workflow.WorkflowTemplateId)
	if err != nil {
		return nil, fmt.Errorf("failed to get stage templates: %w", err)
	}

	// Find current template and next template by order_index
	var currentTemplate, nextTemplate *stagetemplatepb.StageTemplate
	for _, tmpl := range stageTemplates {
		if tmpl.Id == currentStage.StageTemplateId {
			currentTemplate = tmpl
		}
	}

	if currentTemplate == nil {
		return nil, fmt.Errorf("current stage template not found")
	}

	nextIndex := int32(1)
	if currentTemplate.OrderIndex != nil {
		nextIndex = *currentTemplate.OrderIndex + 1
	}

	// Find template with next order_index
	for _, tmpl := range stageTemplates {
		if tmpl.OrderIndex != nil && *tmpl.OrderIndex == nextIndex {
			nextTemplate = tmpl
			break
		}
	}

	// If no next stage, workflow is complete
	if nextTemplate == nil {
		workflow.Status = "completed"
		uc.repositories.Workflow.UpdateWorkflow(ctx, &workflowpb.UpdateWorkflowRequest{Data: workflow})
		return &enginepb.AdvanceWorkflowResponse{
			Success:           true,
			WorkflowCompleted: true,
		}, nil
	}

	// 6. Create Next Stage
	newStageId := uc.services.IDService.GenerateID()
	now := time.Now()
	newStage := &stagepb.Stage{
		Id:              newStageId,
		WorkflowId:      workflow.Id,
		StageTemplateId: nextTemplate.Id,
		Status:          "pending",
		Active:          true,
		DateCreated:     &[]int64{now.UnixMilli()}[0],
	}
	uc.repositories.Stage.CreateStage(ctx, &stagepb.CreateStageRequest{Data: newStage})

	return &enginepb.AdvanceWorkflowResponse{
		Success:     true,
		NextStageId: newStageId,
	}, nil
}

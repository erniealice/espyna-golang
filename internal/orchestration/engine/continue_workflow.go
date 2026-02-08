package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	activitypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/activity"
	stagepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/stage"
	stagetemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/stage_template"
	workflowpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/workflow"
	enginepb "github.com/erniealice/esqyma/pkg/schema/v1/orchestration/engine"
)

// ContinueWorkflowUseCase handles submitting human input to continue a paused workflow
type ContinueWorkflowUseCase struct {
	repositories    EngineRepositories
	services        EngineServices
	cache           *TemplateCache
	schemaProcessor *SchemaProcessor
}

// NewContinueWorkflowUseCase creates a new use case
func NewContinueWorkflowUseCase(repos EngineRepositories, svcs EngineServices, cache *TemplateCache) *ContinueWorkflowUseCase {
	return &ContinueWorkflowUseCase{
		repositories:    repos,
		services:        svcs,
		cache:           cache,
		schemaProcessor: NewSchemaProcessor(),
	}
}

// Execute processes human input and continues the workflow
func (uc *ContinueWorkflowUseCase) Execute(ctx context.Context, req *enginepb.ContinueWorkflowRequest) (*enginepb.ContinueWorkflowResponse, error) {
	if req == nil {
		return nil, errors.New("request is required")
	}
	if req.WorkflowId == "" {
		return nil, errors.New("workflow_id is required")
	}
	if req.ActivityId == "" {
		return nil, errors.New("activity_id is required")
	}

	// 1. Fetch Activity
	activityRes, err := uc.repositories.Activity.ReadActivity(ctx, &activitypb.ReadActivityRequest{
		Data: &activitypb.Activity{Id: req.ActivityId},
	})
	if err != nil || !activityRes.Success || len(activityRes.Data) == 0 {
		return uc.errorResponse("ACTIVITY_NOT_FOUND", fmt.Sprintf("Activity not found: %s", req.ActivityId)), nil
	}
	activity := activityRes.Data[0]

	// Validate activity is pending
	if activity.Status != "pending" {
		return uc.errorResponse("INVALID_ACTIVITY_STATE", fmt.Sprintf("Activity is not pending, current status: %s", activity.Status)), nil
	}

	// 2. Fetch Activity Template for schema validation via cache
	template, err := uc.cache.GetActivityTemplate(ctx, activity.ActivityTemplateId)
	if err != nil {
		return uc.errorResponse("TEMPLATE_NOT_FOUND", fmt.Sprintf("Activity template not found: %v", err)), nil
	}

	// 3. Fetch Workflow for context
	workflowRes, err := uc.repositories.Workflow.ReadWorkflow(ctx, &workflowpb.ReadWorkflowRequest{
		Data: &workflowpb.Workflow{Id: req.WorkflowId},
	})
	if err != nil || !workflowRes.Success || len(workflowRes.Data) == 0 {
		return uc.errorResponse("WORKFLOW_NOT_FOUND", "Workflow not found"), nil
	}
	workflow := workflowRes.Data[0]

	// 4. Validate input against template schema
	var validatedInput map[string]any
	if template.InputSchemaJson != nil && *template.InputSchemaJson != "" {
		validatedInput, err = uc.schemaProcessor.ValidateInput(req.InputJson, *template.InputSchemaJson)
		if err != nil {
			return uc.errorResponse("VALIDATION_FAILED", fmt.Sprintf("Input validation failed: %v", err)), nil
		}
	} else {
		// No schema, parse input as-is
		if req.InputJson != "" {
			if err := json.Unmarshal([]byte(req.InputJson), &validatedInput); err != nil {
				return uc.errorResponse("INVALID_INPUT", "Failed to parse input JSON"), nil
			}
		} else {
			validatedInput = make(map[string]any)
		}
	}

	// 5. Merge input into workflow context
	var workflowContext map[string]any
	if workflow.ContextJson != nil && *workflow.ContextJson != "" {
		json.Unmarshal([]byte(*workflow.ContextJson), &workflowContext)
	}
	if workflowContext == nil {
		workflowContext = make(map[string]any)
	}

	// Merge validated input into context
	for k, v := range validatedInput {
		workflowContext[k] = v
	}

	// 6. Execute use case if defined
	var output map[string]any
	useCaseCode := ""
	if template.UseCaseCode != nil {
		useCaseCode = *template.UseCaseCode
	}

	if useCaseCode != "" {
		// Resolve inputs from context using schema
		resolvedInput, err := uc.schemaProcessor.Resolve(workflowContext, safeString(template.InputSchemaJson))
		if err != nil {
			return uc.errorResponse("SCHEMA_RESOLUTION_FAILED", fmt.Sprintf("Failed to resolve input: %v", err)), nil
		}

		executor, err := uc.services.ExecutorRegistry.GetExecutor(useCaseCode)
		if err != nil {
			return uc.errorResponse("EXECUTOR_NOT_FOUND", fmt.Sprintf("Executor not found: %v", err)), nil
		}

		// Wrap input for executor (all use cases expect {data: {...}})
		wrappedInput := wrapInputForExecutor(resolvedInput)

		// Log for debugging
		fmt.Printf("[ContinueWorkflow] Wrapped input for executor: data keys = %v\n", getKeys(wrappedInput["data"].(map[string]interface{})))

		output, err = executor.Execute(ctx, wrappedInput)
		if err != nil {
			return uc.errorResponse("EXECUTION_FAILED", fmt.Sprintf("Execution failed: %v", err)), nil
		}

		// Map outputs back to context
		if template.OutputSchemaJson != nil && *template.OutputSchemaJson != "" {
			outputUpdates, err := uc.schemaProcessor.Resolve(output, *template.OutputSchemaJson)
			if err == nil {
				for k, v := range outputUpdates {
					workflowContext[k] = v
				}
			}
		}
	}

	// 7. Update workflow context
	newContextBytes, _ := json.Marshal(workflowContext)
	newContextStr := string(newContextBytes)
	workflow.ContextJson = &newContextStr
	uc.repositories.Workflow.UpdateWorkflow(ctx, &workflowpb.UpdateWorkflowRequest{Data: workflow})

	// 8. Mark activity as completed
	now := time.Now()
	activity.Status = "completed"
	activity.DateCompleted = &[]int64{now.UnixMilli()}[0]

	inputStr := req.InputJson
	activity.InputDataJson = &inputStr

	if output != nil {
		outputBytes, _ := json.Marshal(output)
		outputStr := string(outputBytes)
		activity.OutputDataJson = &outputStr
	}

	uc.repositories.Activity.UpdateActivity(ctx, &activitypb.UpdateActivityRequest{Data: activity})

	// 9. Check if we should auto-advance the workflow
	workflowAdvanced := false
	var nextPendingActivityId string

	// Check if all activities in current stage are complete
	stageComplete, nextActivity := uc.checkStageCompletion(ctx, activity.StageId)

	if stageComplete {
		// Mark stage as completed and try to advance
		stageRes, _ := uc.repositories.Stage.ReadStage(ctx, &stagepb.ReadStageRequest{
			Data: &stagepb.Stage{Id: activity.StageId},
		})
		if stageRes != nil && len(stageRes.Data) > 0 {
			stage := stageRes.Data[0]
			stage.Status = "completed"
			uc.repositories.Stage.UpdateStage(ctx, &stagepb.UpdateStageRequest{Data: stage})

			// Try to create next stage
			nextStage := uc.createNextStage(ctx, workflow, stage.StageTemplateId)
			if nextStage != nil {
				workflowAdvanced = true
				// Create activities for the new stage and find first pending
				nextPendingActivityId = uc.createStageActivities(ctx, nextStage.Id, workflow.Id)
			} else {
				// No more stages, workflow is complete
				workflow.Status = "completed"
				uc.repositories.Workflow.UpdateWorkflow(ctx, &workflowpb.UpdateWorkflowRequest{Data: workflow})
				workflowAdvanced = true
			}
		}
	} else if nextActivity != "" {
		nextPendingActivityId = nextActivity
	}

	// Build response
	response := &enginepb.ContinueWorkflowResponse{
		Success:          true,
		WorkflowAdvanced: workflowAdvanced,
	}

	if output != nil {
		outputBytes, _ := json.Marshal(output)
		outputStr := string(outputBytes)
		response.OutputJson = &outputStr
	}

	if nextPendingActivityId != "" {
		response.NextPendingActivityId = &nextPendingActivityId
	}

	return response, nil
}

func (uc *ContinueWorkflowUseCase) errorResponse(code, message string) *enginepb.ContinueWorkflowResponse {
	return &enginepb.ContinueWorkflowResponse{
		Success: false,
		Error: &commonpb.Error{
			Code:    code,
			Message: message,
		},
	}
}

func (uc *ContinueWorkflowUseCase) checkStageCompletion(ctx context.Context, stageId string) (bool, string) {
	activitiesRes, _ := uc.repositories.Activity.ListActivities(ctx, &activitypb.ListActivitiesRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				{
					Field: "stage_id",
					FilterType: &commonpb.TypedFilter_StringFilter{
						StringFilter: &commonpb.StringFilter{
							Value:    stageId,
							Operator: commonpb.StringOperator_STRING_EQUALS,
						},
					},
				},
			},
		},
	})

	if activitiesRes == nil {
		return true, ""
	}

	var nextPending string
	allComplete := true
	for _, act := range activitiesRes.Data {
		if act.Status != "completed" && act.Status != "skipped" {
			allComplete = false
			if act.Status == "pending" && nextPending == "" {
				nextPending = act.Id
			}
		}
	}

	return allComplete, nextPending
}

func (uc *ContinueWorkflowUseCase) createNextStage(ctx context.Context, workflow *workflowpb.Workflow, currentTemplateId string) *stagepb.Stage {
	if workflow.WorkflowTemplateId == nil {
		return nil
	}

	// Get all stage templates via cache (sorted by order_index)
	stageTemplates, err := uc.cache.GetStageTemplates(ctx, *workflow.WorkflowTemplateId)
	if err != nil {
		return nil
	}

	// Find current template and determine next one
	var currentTemplate, nextTemplate *stagetemplatepb.StageTemplate
	for _, tmpl := range stageTemplates {
		if tmpl.Id == currentTemplateId {
			currentTemplate = tmpl
		}
	}

	if currentTemplate == nil {
		return nil
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

	if nextTemplate == nil {
		return nil
	}

	// Create new stage
	now := time.Now()
	newStage := &stagepb.Stage{
		Id:              uc.services.IDService.GenerateID(),
		WorkflowId:      workflow.Id,
		StageTemplateId: nextTemplate.Id,
		Status:          "pending",
		Active:          true,
		DateCreated:     &[]int64{now.UnixMilli()}[0],
	}

	uc.repositories.Stage.CreateStage(ctx, &stagepb.CreateStageRequest{Data: newStage})

	return newStage
}

func (uc *ContinueWorkflowUseCase) createStageActivities(ctx context.Context, stageId, workflowId string) string {
	// Get stage to find template
	stageRes, _ := uc.repositories.Stage.ReadStage(ctx, &stagepb.ReadStageRequest{
		Data: &stagepb.Stage{Id: stageId},
	})
	if stageRes == nil || len(stageRes.Data) == 0 {
		return ""
	}
	stage := stageRes.Data[0]

	// Get activity templates for this stage template via cache
	activityTemplates, err := uc.cache.GetActivityTemplatesForStage(ctx, stage.StageTemplateId)
	if err != nil || len(activityTemplates) == 0 {
		return ""
	}

	var firstPendingId string
	now := time.Now()

	for _, template := range activityTemplates {
		activity := &activitypb.Activity{
			Id:                 uc.services.IDService.GenerateID(),
			StageId:            stageId,
			ActivityTemplateId: template.Id,
			Name:               template.Name,
			Status:             "pending",
			Priority:           "normal",
			Active:             true,
			DateCreated:        &[]int64{now.UnixMilli()}[0],
		}

		uc.repositories.Activity.CreateActivity(ctx, &activitypb.CreateActivityRequest{Data: activity})

		if firstPendingId == "" {
			firstPendingId = activity.Id
		}
	}

	return firstPendingId
}

func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

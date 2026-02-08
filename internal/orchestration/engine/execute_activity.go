package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	activitypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/activity"
	workflowpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/workflow"
	enginepb "github.com/erniealice/esqyma/pkg/schema/v1/orchestration/engine"
)

// ExecuteActivityUseCase handles the execution of a single workflow activity
type ExecuteActivityUseCase struct {
	repositories    EngineRepositories
	services        EngineServices
	cache           *TemplateCache
	schemaProcessor *SchemaProcessor
	celEvaluator    *CELEvaluator
}

// NewExecuteActivityUseCase creates a new use case
func NewExecuteActivityUseCase(repos EngineRepositories, svcs EngineServices, cache *TemplateCache) *ExecuteActivityUseCase {
	celEvaluator, err := NewCELEvaluator()
	if err != nil {
		log.Printf("[WARN] Failed to create CEL evaluator: %v (conditions will be skipped)", err)
	}
	return &ExecuteActivityUseCase{
		repositories:    repos,
		services:        svcs,
		cache:           cache,
		schemaProcessor: NewSchemaProcessor(),
		celEvaluator:    celEvaluator,
	}
}

// Execute performs the activity execution logic
func (uc *ExecuteActivityUseCase) Execute(ctx context.Context, req *enginepb.ExecuteActivityRequest) (*enginepb.ExecuteActivityResponse, error) {
	totalStart := time.Now()
	if req == nil {
		return nil, errors.New("request is required")
	}

	// 1. Fetch Activity
	t1 := time.Now()
	activityRes, err := uc.repositories.Activity.ReadActivity(ctx, &activitypb.ReadActivityRequest{
		Data: &activitypb.Activity{Id: req.ActivityId},
	})
	if err != nil || !activityRes.Success || len(activityRes.Data) == 0 {
		return nil, fmt.Errorf("failed to read activity: %w", err)
	}
	activity := activityRes.Data[0]
	log.Printf("[⏱️ Activity] ReadActivity: %v", time.Since(t1))

	// 2. Fetch Activity Template (for schema and use case code) via cache
	t2 := time.Now()
	template, err := uc.cache.GetActivityTemplate(ctx, activity.ActivityTemplateId)
	if err != nil {
		return nil, fmt.Errorf("failed to read activity template: %w", err)
	}
	log.Printf("[⏱️ Activity] GetActivityTemplate (cached): %v", time.Since(t2))

	// 3. Fetch Workflow (for Context)
	t3 := time.Now()
	workflowRes, err := uc.repositories.Workflow.ReadWorkflow(ctx, &workflowpb.ReadWorkflowRequest{
		Data: &workflowpb.Workflow{Id: req.WorkflowId},
	})
	if err != nil || !workflowRes.Success || len(workflowRes.Data) == 0 {
		return nil, fmt.Errorf("failed to read workflow: %w", err)
	}
	workflow := workflowRes.Data[0]
	log.Printf("[⏱️ Activity] ReadWorkflow: %v", time.Since(t3))

	// 4. Update Status to In Progress (SKIP - reduces Firestore writes by 50%)
	// Note: Status update is handled at completion; in_progress tracking can be done via workflow status
	activity.Status = "in_progress"
	// Commented out to improve performance - unnecessary intermediate state
	// uc.repositories.Activity.UpdateActivity(ctx, &activitypb.UpdateActivityRequest{Data: activity})

	// 5. Resolve Inputs
	var workflowContext map[string]any
	if workflow.ContextJson != nil && *workflow.ContextJson != "" {
		json.Unmarshal([]byte(*workflow.ContextJson), &workflowContext)
	}
	if workflowContext == nil {
		workflowContext = make(map[string]any)
	}

	// 5.5. Evaluate Condition Expression (if present)
	// This allows activities to be conditionally skipped based on workflow state
	if template.ConditionExpression != nil && *template.ConditionExpression != "" {
		if uc.celEvaluator != nil {
			shouldExecute, err := uc.celEvaluator.EvaluateCondition(*template.ConditionExpression, workflowContext)
			if err != nil {
				// Fail-open: log warning and proceed with execution
				log.Printf("[WARN] Condition evaluation failed for activity %s: %v (proceeding with execution)", activity.Id, err)
			} else if !shouldExecute {
				log.Printf("[INFO] Condition not met for activity %s: %s (skipping)", activity.Id, *template.ConditionExpression)
				return uc.skipActivity(ctx, activity, fmt.Sprintf("Condition not met: %s", *template.ConditionExpression))
			}
		} else {
			log.Printf("[WARN] CEL evaluator not available, skipping condition check for activity %s", activity.Id)
		}
	}

	// Get input_mapping from ConfigurationJson (YAML workflow approach)
	// Format: {"target_field": "$.input.source_field"} (JSONPath-style)
	inputMappingJson := ""
	if template.ConfigurationJson != nil && *template.ConfigurationJson != "" {
		var config map[string]interface{}
		if err := json.Unmarshal([]byte(*template.ConfigurationJson), &config); err == nil {
			if inputMapping, ok := config["input_mapping"]; ok {
				if mappingBytes, err := json.Marshal(inputMapping); err == nil {
					inputMappingJson = string(mappingBytes)
				}
			}
		}
	}

	resolvedInput, err := uc.schemaProcessor.Resolve(workflowContext, inputMappingJson)
	if err != nil {
		return uc.failActivity(ctx, activity, fmt.Sprintf("Schema resolution failed: %v", err))
	}

	// 6. Execute via Registry
	useCaseCode := ""
	if template.UseCaseCode != nil {
		useCaseCode = *template.UseCaseCode
	}

	if useCaseCode == "" {
		return uc.failActivity(ctx, activity, "No use case code defined in template")
	}

	executor, err := uc.services.ExecutorRegistry.GetExecutor(useCaseCode)
	if err != nil {
		return uc.failActivity(ctx, activity, fmt.Sprintf("Executor not found: %v", err))
	}

	// Wrap input for executor (all use cases expect {data: {...}})
	wrappedInput := wrapInputForExecutor(resolvedInput)

	// Log for debugging
	fmt.Printf("[ExecuteActivity] Wrapped input for executor: data keys = %v\n", getKeys(wrappedInput["data"].(map[string]interface{})))

	t4 := time.Now()
	output, err := executor.Execute(ctx, wrappedInput)
	if err != nil {
		return uc.failActivity(ctx, activity, fmt.Sprintf("Execution failed: %v", err))
	}
	log.Printf("[⏱️ Activity] UseCase %s: %v", useCaseCode, time.Since(t4))

	// 6.5. Unwrap output for schema mapping
	// Most use cases return {success: true, data: [Entity]}, so we extract
	// the first item from data array for output schema mapping
	outputContext := output
	if dataField, ok := output["data"]; ok {
		if dataArr, ok := dataField.([]interface{}); ok && len(dataArr) > 0 {
			if firstItem, ok := dataArr[0].(map[string]interface{}); ok {
				outputContext = firstItem
				fmt.Printf("[ExecuteActivity] Unwrapped data[0] for output mapping: keys = %v\n", getKeys(firstItem))
			}
		}
	}

	// 7. Map Outputs back to Context
	outputSchemaJson := ""
	if template.OutputSchemaJson != nil {
		outputSchemaJson = *template.OutputSchemaJson
	}

	// We treat the output of the use case as the "context" for resolving the output schema
	// This maps UseCaseResponse -> WorkflowContextKey
	resolvedOutputUpdates, err := uc.schemaProcessor.Resolve(outputContext, outputSchemaJson)
	if err != nil {
		return uc.failActivity(ctx, activity, fmt.Sprintf("Output resolution failed: %v", err))
	}

	// Store output in hierarchical structure: stage[x].activity[y].output
	// This enables JSONPath-style lookups like $.stage[0].activity[1].output.client_id
	stageIndex := int32(0)
	if activity.StageOrderIndex != nil {
		stageIndex = *activity.StageOrderIndex
	}
	activityIndex := int32(0)
	if activity.OrderIndex != nil {
		activityIndex = *activity.OrderIndex
	}

	// Ensure stage structure exists
	if workflowContext["stage"] == nil {
		workflowContext["stage"] = make(map[string]any)
	}
	stages, ok := workflowContext["stage"].(map[string]any)
	if !ok {
		stages = make(map[string]any)
		workflowContext["stage"] = stages
	}

	stageKey := fmt.Sprintf("%d", stageIndex)
	if stages[stageKey] == nil {
		stages[stageKey] = map[string]any{
			"activity": make(map[string]any),
		}
	}
	stage, ok := stages[stageKey].(map[string]any)
	if !ok {
		stage = map[string]any{"activity": make(map[string]any)}
		stages[stageKey] = stage
	}

	if stage["activity"] == nil {
		stage["activity"] = make(map[string]any)
	}
	activities, ok := stage["activity"].(map[string]any)
	if !ok {
		activities = make(map[string]any)
		stage["activity"] = activities
	}

	activityKey := fmt.Sprintf("%d", activityIndex)
	activities[activityKey] = map[string]any{
		"name":   activity.Name,
		"output": resolvedOutputUpdates,
	}

	// 8. Save Workflow Context
	t5 := time.Now()
	newContextBytes, _ := json.Marshal(workflowContext)
	newContextStr := string(newContextBytes)
	workflow.ContextJson = &newContextStr
	uc.repositories.Workflow.UpdateWorkflow(ctx, &workflowpb.UpdateWorkflowRequest{Data: workflow})
	log.Printf("[⏱️ Activity] UpdateWorkflowContext: %v", time.Since(t5))

	// 9. Complete Activity
	t6 := time.Now()
	activity.Status = "completed"
	// Save inputs/outputs for audit (simplified, ideally strictly limited size)
	inputBytes, _ := json.Marshal(resolvedInput)
	inputStr := string(inputBytes)
	activity.InputDataJson = &inputStr

	outputBytes, _ := json.Marshal(output)
	outputStr := string(outputBytes)
	activity.OutputDataJson = &outputStr

	uc.repositories.Activity.UpdateActivity(ctx, &activitypb.UpdateActivityRequest{Data: activity})
	log.Printf("[⏱️ Activity] UpdateActivityComplete: %v", time.Since(t6))
	log.Printf("[⏱️ Activity] TOTAL for %s: %v", useCaseCode, time.Since(totalStart))

	return &enginepb.ExecuteActivityResponse{
		Success:    true,
		OutputJson: outputStr,
	}, nil
}

func (uc *ExecuteActivityUseCase) failActivity(ctx context.Context, activity *activitypb.Activity, msg string) (*enginepb.ExecuteActivityResponse, error) {
	activity.Status = "failed"
	// We could also log error message to activity if field exists
	uc.repositories.Activity.UpdateActivity(ctx, &activitypb.UpdateActivityRequest{Data: activity})

	return &enginepb.ExecuteActivityResponse{
		Success: false,
		Error: &commonpb.Error{
			Code:    "ACTIVITY_EXECUTION_FAILED",
			Message: msg,
		},
	}, nil
}

// skipActivity marks an activity as skipped (condition not met) and returns a success response
func (uc *ExecuteActivityUseCase) skipActivity(ctx context.Context, activity *activitypb.Activity, reason string) (*enginepb.ExecuteActivityResponse, error) {
	activity.Status = "skipped"
	reasonStr := reason
	activity.OutputDataJson = &reasonStr
	uc.repositories.Activity.UpdateActivity(ctx, &activitypb.UpdateActivityRequest{Data: activity})

	return &enginepb.ExecuteActivityResponse{
		Success:    true, // Skipped is still a successful completion (not a failure)
		OutputJson: fmt.Sprintf(`{"skipped": true, "reason": "%s"}`, reason),
	}, nil
}

// wrapInputForExecutor wraps resolved input under "data" key for protobuf compatibility
// All use case requests follow the pattern: message XxxRequest { MessageType data = 1; }
func wrapInputForExecutor(resolvedInput map[string]interface{}) map[string]interface{} {
	if resolvedInput == nil {
		resolvedInput = make(map[string]interface{})
	}
	return map[string]interface{}{
		"data": resolvedInput,
	}
}

// getKeys extracts keys from a map for logging purposes
func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

package engine

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	stagepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/stage"
	stagetemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/stage_template"
	workflowpb "leapfor.xyz/esqyma/golang/v1/domain/workflow/workflow"
	enginepb "leapfor.xyz/esqyma/golang/v1/orchestration/engine"
)

// StartWorkflowFromTemplateUseCase handles the creation of a workflow instance from a template
type StartWorkflowFromTemplateUseCase struct {
	repositories    EngineRepositories
	services        EngineServices
	cache           *TemplateCache
	schemaProcessor *SchemaProcessor
}

// NewStartWorkflowFromTemplateUseCase creates a new use case
func NewStartWorkflowFromTemplateUseCase(repos EngineRepositories, svcs EngineServices, cache *TemplateCache) *StartWorkflowFromTemplateUseCase {
	return &StartWorkflowFromTemplateUseCase{
		repositories:    repos,
		services:        svcs,
		cache:           cache,
		schemaProcessor: NewSchemaProcessor(),
	}
}

// Execute creates a workflow instance from a template and starts the first stage
func (uc *StartWorkflowFromTemplateUseCase) Execute(ctx context.Context, req *enginepb.StartWorkflowRequest) (*enginepb.StartWorkflowResponse, error) {
	totalStart := time.Now()
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow_engine.validation.request_required", "Request is required [DEFAULT]"))
	}

	if req.WorkflowTemplateId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow_engine.validation.template_id_required", "Workflow template ID is required [DEFAULT]"))
	}

	// 1. Fetch WorkflowTemplate (via cache)
	t1 := time.Now()
	template, err := uc.cache.GetWorkflowTemplate(ctx, req.WorkflowTemplateId)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow template: %w", err)
	}
	log.Printf("[⏱️ StartWorkflow] GetWorkflowTemplate (cached): %v", time.Since(t1))

	// 2. Validate and enrich input against template schema
	var validatedInputJson string
	if template.InputSchemaJson != nil && *template.InputSchemaJson != "" {
		enrichedJson, err := uc.schemaProcessor.ValidateInputToJson(req.InputJson, *template.InputSchemaJson)
		if err != nil {
			return nil, fmt.Errorf("input validation failed: %w", err)
		}
		validatedInputJson = enrichedJson
	} else {
		// No schema defined, use input as-is
		validatedInputJson = req.InputJson
	}

	// 3. Create Workflow instance
	t2 := time.Now()
	workflowID := uc.services.IDService.GenerateID()
	now := time.Now()

	// Wrap input under "input" key for YAML workflow JSONPath access ($.input.field)
	wrappedContextJson := fmt.Sprintf(`{"input":%s}`, validatedInputJson)

	workflow := &workflowpb.Workflow{
		Id:                 workflowID,
		Name:               fmt.Sprintf("%s - %s", template.Name, now.Format("2006-01-02 15:04:05")),
		WorkflowTemplateId: &req.WorkflowTemplateId,
		ContextJson:        &wrappedContextJson,
		CurrentStageIndex:  &[]int32{0}[0],
		Status:             "in_progress",
		Active:             true,
		WorkspaceId:        template.WorkspaceId,
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
	}

	createWorkflowRes, err := uc.repositories.Workflow.CreateWorkflow(ctx, &workflowpb.CreateWorkflowRequest{
		Data: workflow,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create workflow instance: %w", err)
	}
	if createWorkflowRes == nil || len(createWorkflowRes.Data) == 0 {
		return nil, fmt.Errorf("failed to create workflow instance: no data returned")
	}
	log.Printf("[⏱️ StartWorkflow] CreateWorkflow: %v", time.Since(t2))

	// 4. Fetch StageTemplates to find the first one (via cache)
	// Note: Cache handles filtering and sorting by order_index
	t3 := time.Now()
	stageTemplates, err := uc.cache.GetStageTemplates(ctx, req.WorkflowTemplateId)
	if err != nil {
		return nil, fmt.Errorf("failed to list stage templates: %w", err)
	}
	log.Printf("[⏱️ StartWorkflow] GetStageTemplates (cached): %v", time.Since(t3))
	// No stage templates is allowed - workflow can start without predefined stages
	// Cache returns templates already sorted by order_index, so first one is the lowest
	var firstStageTemplate *stagetemplatepb.StageTemplate
	if len(stageTemplates) > 0 {
		firstStageTemplate = stageTemplates[0]
	}

	// 5. Create first Stage instance if we have a stage template (Lazy instantiation)
	if firstStageTemplate != nil {
		t4 := time.Now()
		stageID := uc.services.IDService.GenerateID()
		stage := &stagepb.Stage{
			Id:              stageID,
			WorkflowId:      workflowID,
			StageTemplateId: firstStageTemplate.Id,
			Status:          "pending", // Should probably be in_progress once activities start
			Active:          true,
			DateCreated:     &[]int64{now.UnixMilli()}[0],
		}

		_, err := uc.repositories.Stage.CreateStage(ctx, &stagepb.CreateStageRequest{
			Data: stage,
		})
		if err != nil {
			// Log error but we already created the workflow
			fmt.Printf("Warning: failed to create initial stage: %v\n", err)
		}
		log.Printf("[⏱️ StartWorkflow] CreateStage: %v", time.Since(t4))
	}

	log.Printf("[⏱️ StartWorkflow] TOTAL for %s: %v", req.WorkflowTemplateId, time.Since(totalStart))
	return &enginepb.StartWorkflowResponse{
			Workflow: workflow,
			Success:  true,
		},
		nil
}

package workflow

import (
	// Workflow domain use cases
	activityUseCases "leapfor.xyz/espyna/internal/application/usecases/workflow/activity"
	activityTemplateUseCases "leapfor.xyz/espyna/internal/application/usecases/workflow/activity_template"
	stageUseCases "leapfor.xyz/espyna/internal/application/usecases/workflow/stage"
	stageTemplateUseCases "leapfor.xyz/espyna/internal/application/usecases/workflow/stage_template"
	workflowUseCases "leapfor.xyz/espyna/internal/application/usecases/workflow/workflow"
	workflowTemplateUseCases "leapfor.xyz/espyna/internal/application/usecases/workflow/workflow_template"
)

// WorkflowUseCases contains all workflow domain use cases.
// Note: The workflow engine is NOT part of this aggregate - it's an orchestration
// concern managed by the Container as a first-class service (Container.services.WorkflowEngine).
type WorkflowUseCases struct {
	Workflow         *workflowUseCases.UseCases
	WorkflowTemplate *workflowTemplateUseCases.UseCases
	Stage            *stageUseCases.UseCases
	StageTemplate    *stageTemplateUseCases.UseCases
	Activity         *activityUseCases.UseCases
	ActivityTemplate *activityTemplateUseCases.UseCases
}

// NewUseCases creates a new collection of workflow domain use cases.
// This is a simple aggregation constructor - no dependency wiring logic here.
func NewUseCases(
	workflowUseCases *workflowUseCases.UseCases,
	workflowTemplateUseCases *workflowTemplateUseCases.UseCases,
	stageUseCases *stageUseCases.UseCases,
	stageTemplateUseCases *stageTemplateUseCases.UseCases,
	activityUseCases *activityUseCases.UseCases,
	activityTemplateUseCases *activityTemplateUseCases.UseCases,
) *WorkflowUseCases {
	return &WorkflowUseCases{
		Workflow:         workflowUseCases,
		WorkflowTemplate: workflowTemplateUseCases,
		Stage:            stageUseCases,
		StageTemplate:    stageTemplateUseCases,
		Activity:         activityUseCases,
		ActivityTemplate: activityTemplateUseCases,
	}
}

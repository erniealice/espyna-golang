package domain

import (
	"leapfor.xyz/espyna/internal/application/ports"
	"leapfor.xyz/espyna/internal/application/usecases"
	"leapfor.xyz/espyna/internal/orchestration/workflow/executor"
)

// RegisterWorkflowUseCases registers all workflow domain use cases with the registry.
func RegisterWorkflowUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Workflow == nil {
		return
	}

	registerWorkflowCoreUseCases(useCases, register)
	registerWorkflowTemplateUseCases(useCases, register)
	registerStageUseCases(useCases, register)
	registerStageTemplateUseCases(useCases, register)
	registerActivityUseCases(useCases, register)
	registerActivityTemplateUseCases(useCases, register)
}

func registerWorkflowCoreUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Workflow.Workflow == nil {
		return
	}
	if useCases.Workflow.Workflow.CreateWorkflow != nil {
		register("workflow.workflow.create", executor.New(useCases.Workflow.Workflow.CreateWorkflow.Execute))
	}
	if useCases.Workflow.Workflow.ReadWorkflow != nil {
		register("workflow.workflow.read", executor.New(useCases.Workflow.Workflow.ReadWorkflow.Execute))
	}
	if useCases.Workflow.Workflow.UpdateWorkflow != nil {
		register("workflow.workflow.update", executor.New(useCases.Workflow.Workflow.UpdateWorkflow.Execute))
	}
	if useCases.Workflow.Workflow.DeleteWorkflow != nil {
		register("workflow.workflow.delete", executor.New(useCases.Workflow.Workflow.DeleteWorkflow.Execute))
	}
	if useCases.Workflow.Workflow.ListWorkflows != nil {
		register("workflow.workflow.list", executor.New(useCases.Workflow.Workflow.ListWorkflows.Execute))
	}
}

func registerWorkflowTemplateUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Workflow.WorkflowTemplate == nil {
		return
	}
	if useCases.Workflow.WorkflowTemplate.CreateWorkflowTemplate != nil {
		register("workflow.workflow_template.create", executor.New(useCases.Workflow.WorkflowTemplate.CreateWorkflowTemplate.Execute))
	}
	if useCases.Workflow.WorkflowTemplate.ReadWorkflowTemplate != nil {
		register("workflow.workflow_template.read", executor.New(useCases.Workflow.WorkflowTemplate.ReadWorkflowTemplate.Execute))
	}
	if useCases.Workflow.WorkflowTemplate.UpdateWorkflowTemplate != nil {
		register("workflow.workflow_template.update", executor.New(useCases.Workflow.WorkflowTemplate.UpdateWorkflowTemplate.Execute))
	}
	if useCases.Workflow.WorkflowTemplate.DeleteWorkflowTemplate != nil {
		register("workflow.workflow_template.delete", executor.New(useCases.Workflow.WorkflowTemplate.DeleteWorkflowTemplate.Execute))
	}
	if useCases.Workflow.WorkflowTemplate.ListWorkflowTemplates != nil {
		register("workflow.workflow_template.list", executor.New(useCases.Workflow.WorkflowTemplate.ListWorkflowTemplates.Execute))
	}
}

func registerStageUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Workflow.Stage == nil {
		return
	}
	if useCases.Workflow.Stage.CreateStage != nil {
		register("workflow.stage.create", executor.New(useCases.Workflow.Stage.CreateStage.Execute))
	}
	if useCases.Workflow.Stage.ReadStage != nil {
		register("workflow.stage.read", executor.New(useCases.Workflow.Stage.ReadStage.Execute))
	}
	if useCases.Workflow.Stage.UpdateStage != nil {
		register("workflow.stage.update", executor.New(useCases.Workflow.Stage.UpdateStage.Execute))
	}
	if useCases.Workflow.Stage.DeleteStage != nil {
		register("workflow.stage.delete", executor.New(useCases.Workflow.Stage.DeleteStage.Execute))
	}
	if useCases.Workflow.Stage.ListStages != nil {
		register("workflow.stage.list", executor.New(useCases.Workflow.Stage.ListStages.Execute))
	}
}

func registerStageTemplateUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Workflow.StageTemplate == nil {
		return
	}
	if useCases.Workflow.StageTemplate.CreateStageTemplate != nil {
		register("workflow.stage_template.create", executor.New(useCases.Workflow.StageTemplate.CreateStageTemplate.Execute))
	}
	if useCases.Workflow.StageTemplate.ReadStageTemplate != nil {
		register("workflow.stage_template.read", executor.New(useCases.Workflow.StageTemplate.ReadStageTemplate.Execute))
	}
	if useCases.Workflow.StageTemplate.UpdateStageTemplate != nil {
		register("workflow.stage_template.update", executor.New(useCases.Workflow.StageTemplate.UpdateStageTemplate.Execute))
	}
	if useCases.Workflow.StageTemplate.DeleteStageTemplate != nil {
		register("workflow.stage_template.delete", executor.New(useCases.Workflow.StageTemplate.DeleteStageTemplate.Execute))
	}
	if useCases.Workflow.StageTemplate.ListStageTemplates != nil {
		register("workflow.stage_template.list", executor.New(useCases.Workflow.StageTemplate.ListStageTemplates.Execute))
	}
}

func registerActivityUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Workflow.Activity == nil {
		return
	}
	if useCases.Workflow.Activity.CreateActivity != nil {
		register("workflow.activity.create", executor.New(useCases.Workflow.Activity.CreateActivity.Execute))
	}
	if useCases.Workflow.Activity.ReadActivity != nil {
		register("workflow.activity.read", executor.New(useCases.Workflow.Activity.ReadActivity.Execute))
	}
	if useCases.Workflow.Activity.UpdateActivity != nil {
		register("workflow.activity.update", executor.New(useCases.Workflow.Activity.UpdateActivity.Execute))
	}
	if useCases.Workflow.Activity.DeleteActivity != nil {
		register("workflow.activity.delete", executor.New(useCases.Workflow.Activity.DeleteActivity.Execute))
	}
	if useCases.Workflow.Activity.ListActivities != nil {
		register("workflow.activity.list", executor.New(useCases.Workflow.Activity.ListActivities.Execute))
	}
}

func registerActivityTemplateUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Workflow.ActivityTemplate == nil {
		return
	}
	if useCases.Workflow.ActivityTemplate.CreateActivityTemplate != nil {
		register("workflow.activity_template.create", executor.New(useCases.Workflow.ActivityTemplate.CreateActivityTemplate.Execute))
	}
	if useCases.Workflow.ActivityTemplate.ReadActivityTemplate != nil {
		register("workflow.activity_template.read", executor.New(useCases.Workflow.ActivityTemplate.ReadActivityTemplate.Execute))
	}
	if useCases.Workflow.ActivityTemplate.UpdateActivityTemplate != nil {
		register("workflow.activity_template.update", executor.New(useCases.Workflow.ActivityTemplate.UpdateActivityTemplate.Execute))
	}
	if useCases.Workflow.ActivityTemplate.DeleteActivityTemplate != nil {
		register("workflow.activity_template.delete", executor.New(useCases.Workflow.ActivityTemplate.DeleteActivityTemplate.Execute))
	}
	if useCases.Workflow.ActivityTemplate.ListActivityTemplates != nil {
		register("workflow.activity_template.list", executor.New(useCases.Workflow.ActivityTemplate.ListActivityTemplates.Execute))
	}
}

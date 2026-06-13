package domain

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/workflow"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"

	// Workflow domain use cases
	activityUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/workflow/activity"
	activityTemplateUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/workflow/activity_template"
	stageUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/workflow/stage"
	stageTemplateUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/workflow/stage_template"
	workflowUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/workflow/workflow"
	workflowTemplateUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/workflow/workflow_template"

	// Orchestration layer - workflow engine
	engineUseCases "github.com/erniealice/espyna-golang/internal/orchestration/engine"
)

// InitializeWorkflow creates all workflow domain use cases from provider repositories.
// Note: The WorkflowEngineService is NOT passed here - it's an orchestration concern.
// If specific use cases need the engine (e.g., CreateWorkflowUseCase), it should be
// injected later via a setter after the engine is initialized by the Container.
func InitializeWorkflow(
	repos *domain.WorkflowRepositories,
	authSvc ports.Authorizer,
	txSvc ports.Transactor,
	i18nSvc ports.Translator,
	idSvc ports.IDGenerator,
	actionGate *actiongate.ActionGatekeeper,
) (*workflow.WorkflowUseCases, error) {
	// Create individual domain use cases with proper dependency injection
	workflowUC := workflowUseCases.NewUseCases(
		workflowUseCases.WorkflowRepositories{
			Workflow: repos.Workflow,
		},
		workflowUseCases.WorkflowServices{
			Authorizer:       authSvc,
			Transactor:       txSvc,
			Translator:       i18nSvc,
			IDGenerator:      idSvc,
			ActionGatekeeper: actionGate,
		},
	)

	stageTemplateUC := stageTemplateUseCases.NewUseCases(
		stageTemplateUseCases.StageTemplateRepositories{
			StageTemplate:    repos.StageTemplate,
			WorkflowTemplate: repos.WorkflowTemplate,
		},
		stageTemplateUseCases.StageTemplateServices{
			Authorizer:       authSvc,
			Transactor:       txSvc,
			Translator:       i18nSvc,
			IDGenerator:      idSvc,
			ActionGatekeeper: actionGate,
		},
	)

	workflowTemplateUC := workflowTemplateUseCases.NewUseCases(
		workflowTemplateUseCases.WorkflowTemplateRepositories{
			WorkflowTemplate: repos.WorkflowTemplate,
		},
		workflowTemplateUseCases.WorkflowTemplateServices{
			Authorizer:       authSvc,
			Transactor:       txSvc,
			Translator:       i18nSvc,
			IDGenerator:      idSvc,
			ActionGatekeeper: actionGate,
		},
	)

	activityTemplateUC := activityTemplateUseCases.NewUseCases(
		activityTemplateUseCases.ActivityTemplateRepositories{
			ActivityTemplate: repos.ActivityTemplate,
			StageTemplate:    repos.StageTemplate,
		},
		activityTemplateUseCases.ActivityTemplateServices{
			Authorizer:       authSvc,
			Transactor:       txSvc,
			Translator:       i18nSvc,
			IDGenerator:      idSvc,
			ActionGatekeeper: actionGate,
		},
	)

	stageUC := stageUseCases.NewUseCases(
		stageUseCases.StageRepositories{
			Stage:         repos.Stage,
			Workflow:      repos.Workflow,
			StageTemplate: repos.StageTemplate,
		},
		stageUseCases.StageServices{
			Authorizer:       authSvc,
			Transactor:       txSvc,
			Translator:       i18nSvc,
			IDGenerator:      idSvc,
			ActionGatekeeper: actionGate,
		},
	)

	activityUC := activityUseCases.NewUseCases(
		activityUseCases.ActivityRepositories{
			Activity: repos.Activity,
			Stage:    repos.Stage,
		},
		activityUseCases.ActivityServices{
			Authorizer:       authSvc,
			Transactor:       txSvc,
			Translator:       i18nSvc,
			IDGenerator:      idSvc,
			ActionGatekeeper: actionGate,
		},
	)

	// Aggregate all workflow use cases using the domain's constructor
	return workflow.NewUseCases(
		workflowUC,
		workflowTemplateUC,
		stageUC,
		stageTemplateUC,
		activityUC,
		activityTemplateUC,
	), nil
}

// InitializeWorkflowEngine creates the workflow engine use cases
// This must be called AFTER all domains are initialized because the engine needs
// access to the UseCaseRegistry which maps use_case_codes to actual domain use cases.
// The engine enables dynamic workflow execution by binding activities to use cases.
func InitializeWorkflowEngine(
	repos *domain.WorkflowRepositories,
	authSvc ports.Authorizer,
	txSvc ports.Transactor,
	i18nSvc ports.Translator,
	idSvc ports.IDGenerator,
	executorRegistry ports.ExecutorRegistry,
) (ports.WorkflowEngineService, error) {
	engineUC := engineUseCases.NewUseCases(
		engineUseCases.EngineRepositories{
			Workflow:         repos.Workflow,
			WorkflowTemplate: repos.WorkflowTemplate,
			Stage:            repos.Stage,
			StageTemplate:    repos.StageTemplate,
			Activity:         repos.Activity,
			ActivityTemplate: repos.ActivityTemplate,
		},
		engineUseCases.EngineServices{
			Authorizer:       authSvc,
			Transactor:       txSvc,
			Translator:       i18nSvc,
			IDGenerator:      idSvc,
			ExecutorRegistry: executorRegistry,
		},
	)

	return engineUC, nil
}

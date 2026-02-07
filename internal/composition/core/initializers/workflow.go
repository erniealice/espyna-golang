package initializers

import (
	"leapfor.xyz/espyna/internal/application/ports"
	"leapfor.xyz/espyna/internal/application/usecases/workflow"
	"leapfor.xyz/espyna/internal/composition/providers/domain"

	// Workflow domain use cases
	activityUseCases "leapfor.xyz/espyna/internal/application/usecases/workflow/activity"
	activityTemplateUseCases "leapfor.xyz/espyna/internal/application/usecases/workflow/activity_template"
	stageUseCases "leapfor.xyz/espyna/internal/application/usecases/workflow/stage"
	stageTemplateUseCases "leapfor.xyz/espyna/internal/application/usecases/workflow/stage_template"
	workflowUseCases "leapfor.xyz/espyna/internal/application/usecases/workflow/workflow"
	workflowTemplateUseCases "leapfor.xyz/espyna/internal/application/usecases/workflow/workflow_template"

	// Orchestration layer - workflow engine
	engineUseCases "leapfor.xyz/espyna/internal/orchestration/engine"
)

// InitializeWorkflow creates all workflow domain use cases from provider repositories.
// Note: The WorkflowEngineService is NOT passed here - it's an orchestration concern.
// If specific use cases need the engine (e.g., CreateWorkflowUseCase), it should be
// injected later via a setter after the engine is initialized by the Container.
func InitializeWorkflow(
	repos *domain.WorkflowRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idSvc ports.IDService,
) (*workflow.WorkflowUseCases, error) {
	// Create individual domain use cases with proper dependency injection
	workflowUC := workflowUseCases.NewUseCases(
		workflowUseCases.WorkflowRepositories{
			Workflow: repos.Workflow,
		},
		workflowUseCases.WorkflowServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idSvc,
		},
	)

	stageTemplateUC := stageTemplateUseCases.NewUseCases(
		stageTemplateUseCases.StageTemplateRepositories{
			StageTemplate:    repos.StageTemplate,
			WorkflowTemplate: repos.WorkflowTemplate,
		},
		stageTemplateUseCases.StageTemplateServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idSvc,
		},
	)

	workflowTemplateUC := workflowTemplateUseCases.NewUseCases(
		workflowTemplateUseCases.WorkflowTemplateRepositories{
			WorkflowTemplate: repos.WorkflowTemplate,
		},
		workflowTemplateUseCases.WorkflowTemplateServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idSvc,
		},
	)

	activityTemplateUC := activityTemplateUseCases.NewUseCases(
		activityTemplateUseCases.ActivityTemplateRepositories{
			ActivityTemplate: repos.ActivityTemplate,
			StageTemplate:    repos.StageTemplate,
		},
		activityTemplateUseCases.ActivityTemplateServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idSvc,
		},
	)

	stageUC := stageUseCases.NewUseCases(
		stageUseCases.StageRepositories{
			Stage:         repos.Stage,
			Workflow:      repos.Workflow,
			StageTemplate: repos.StageTemplate,
		},
		stageUseCases.StageServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idSvc,
		},
	)

	activityUC := activityUseCases.NewUseCases(
		activityUseCases.ActivityRepositories{
			Activity: repos.Activity,
			Stage:    repos.Stage,
		},
		activityUseCases.ActivityServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idSvc,
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
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idSvc ports.IDService,
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
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idSvc,
			ExecutorRegistry:     executorRegistry,
		},
	)

	return engineUC, nil
}

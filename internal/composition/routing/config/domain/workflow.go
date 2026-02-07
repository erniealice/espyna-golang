package domain

import (
	"fmt"

	workflowuc "leapfor.xyz/espyna/internal/application/usecases/workflow"
	"leapfor.xyz/espyna/internal/composition/contracts"

	activitypb "leapfor.xyz/esqyma/golang/v1/domain/workflow/activity"
	activitytemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/activity_template"
	stagepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/stage"
	stagetemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/stage_template"
	workflowpb "leapfor.xyz/esqyma/golang/v1/domain/workflow/workflow"
	workflowtemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/workflow_template"
)

// ConfigureWorkflowDomain configures routes for the Workflow domain use cases.
// Note: Engine routes are NOT configured here - orchestration is a separate concern
// and should be configured via a dedicated orchestration routing configuration if needed.
func ConfigureWorkflowDomain(workflowUseCases *workflowuc.WorkflowUseCases) contracts.DomainRouteConfiguration {
	if workflowUseCases == nil {
		fmt.Printf("⚠️  Workflow use cases is NIL\n")
		return contracts.DomainRouteConfiguration{
			Domain:  "workflow",
			Prefix:  "/workflow",
			Enabled: false,
			Routes:  []contracts.RouteConfiguration{},
		}
	}

	fmt.Printf("✅ Workflow use cases are properly initialized!\n")

	routes := []contracts.RouteConfiguration{}

	// Workflow routes
	if workflowUseCases != nil && workflowUseCases.Workflow != nil {
		routes = append(routes,
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/workflow/workflow/create",
				Handler: contracts.NewGenericHandler(workflowUseCases.Workflow.CreateWorkflow, &workflowpb.CreateWorkflowRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/workflow/workflow/read",
				Handler: contracts.NewGenericHandler(workflowUseCases.Workflow.ReadWorkflow, &workflowpb.ReadWorkflowRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/workflow/workflow/update",
				Handler: contracts.NewGenericHandler(workflowUseCases.Workflow.UpdateWorkflow, &workflowpb.UpdateWorkflowRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/workflow/workflow/delete",
				Handler: contracts.NewGenericHandler(workflowUseCases.Workflow.DeleteWorkflow, &workflowpb.DeleteWorkflowRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/workflow/workflow/list",
				Handler: contracts.NewGenericHandler(workflowUseCases.Workflow.ListWorkflows, &workflowpb.ListWorkflowsRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/workflow/workflow/get-list-page-data",
				Handler: contracts.NewGenericHandler(workflowUseCases.Workflow.GetWorkflowListPageData, &workflowpb.GetWorkflowListPageDataRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/workflow/workflow/get-item-page-data",
				Handler: contracts.NewGenericHandler(workflowUseCases.Workflow.GetWorkflowItemPageData, &workflowpb.GetWorkflowItemPageDataRequest{}),
			},
		)
	}

	// WorkflowTemplate routes
	if workflowUseCases != nil && workflowUseCases.WorkflowTemplate != nil {
		routes = append(routes,
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/workflow/workflow-template/create",
				Handler: contracts.NewGenericHandler(workflowUseCases.WorkflowTemplate.CreateWorkflowTemplate, &workflowtemplatepb.CreateWorkflowTemplateRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/workflow/workflow-template/read",
				Handler: contracts.NewGenericHandler(workflowUseCases.WorkflowTemplate.ReadWorkflowTemplate, &workflowtemplatepb.ReadWorkflowTemplateRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/workflow/workflow-template/update",
				Handler: contracts.NewGenericHandler(workflowUseCases.WorkflowTemplate.UpdateWorkflowTemplate, &workflowtemplatepb.UpdateWorkflowTemplateRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/workflow/workflow-template/delete",
				Handler: contracts.NewGenericHandler(workflowUseCases.WorkflowTemplate.DeleteWorkflowTemplate, &workflowtemplatepb.DeleteWorkflowTemplateRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/workflow/workflow-template/list",
				Handler: contracts.NewGenericHandler(workflowUseCases.WorkflowTemplate.ListWorkflowTemplates, &workflowtemplatepb.ListWorkflowTemplatesRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/workflow/workflow-template/get-list-page-data",
				Handler: contracts.NewGenericHandler(workflowUseCases.WorkflowTemplate.GetWorkflowTemplateListPageData, &workflowtemplatepb.GetWorkflowTemplateListPageDataRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/workflow/workflow-template/get-item-page-data",
				Handler: contracts.NewGenericHandler(workflowUseCases.WorkflowTemplate.GetWorkflowTemplateItemPageData, &workflowtemplatepb.GetWorkflowTemplateItemPageDataRequest{}),
			},
		)
	}

	// StageTemplate routes
	if workflowUseCases != nil && workflowUseCases.StageTemplate != nil {
		routes = append(routes,
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/workflow/stage-template/create",
				Handler: contracts.NewGenericHandler(workflowUseCases.StageTemplate.CreateStageTemplate, &stagetemplatepb.CreateStageTemplateRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/workflow/stage-template/read",
				Handler: contracts.NewGenericHandler(workflowUseCases.StageTemplate.ReadStageTemplate, &stagetemplatepb.ReadStageTemplateRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/workflow/stage-template/update",
				Handler: contracts.NewGenericHandler(workflowUseCases.StageTemplate.UpdateStageTemplate, &stagetemplatepb.UpdateStageTemplateRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/workflow/stage-template/delete",
				Handler: contracts.NewGenericHandler(workflowUseCases.StageTemplate.DeleteStageTemplate, &stagetemplatepb.DeleteStageTemplateRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/workflow/stage-template/list",
				Handler: contracts.NewGenericHandler(workflowUseCases.StageTemplate.ListStageTemplates, &stagetemplatepb.ListStageTemplatesRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/workflow/stage-template/get-list-page-data",
				Handler: contracts.NewGenericHandler(workflowUseCases.StageTemplate.GetStageTemplateListPageData, &stagetemplatepb.GetStageTemplateListPageDataRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/workflow/stage-template/get-item-page-data",
				Handler: contracts.NewGenericHandler(workflowUseCases.StageTemplate.GetStageTemplateItemPageData, &stagetemplatepb.GetStageTemplateItemPageDataRequest{}),
			},
		)
	}

	// ActivityTemplate routes
	if workflowUseCases != nil && workflowUseCases.ActivityTemplate != nil {
		routes = append(routes,
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/workflow/activity-template/create",
				Handler: contracts.NewGenericHandler(workflowUseCases.ActivityTemplate.CreateActivityTemplate, &activitytemplatepb.CreateActivityTemplateRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/workflow/activity-template/read",
				Handler: contracts.NewGenericHandler(workflowUseCases.ActivityTemplate.ReadActivityTemplate, &activitytemplatepb.ReadActivityTemplateRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/workflow/activity-template/update",
				Handler: contracts.NewGenericHandler(workflowUseCases.ActivityTemplate.UpdateActivityTemplate, &activitytemplatepb.UpdateActivityTemplateRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/workflow/activity-template/delete",
				Handler: contracts.NewGenericHandler(workflowUseCases.ActivityTemplate.DeleteActivityTemplate, &activitytemplatepb.DeleteActivityTemplateRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/workflow/activity-template/list",
				Handler: contracts.NewGenericHandler(workflowUseCases.ActivityTemplate.ListActivityTemplates, &activitytemplatepb.ListActivityTemplatesRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/workflow/activity-template/get-list-page-data",
				Handler: contracts.NewGenericHandler(workflowUseCases.ActivityTemplate.GetActivityTemplateListPageData, &activitytemplatepb.GetActivityTemplateListPageDataRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/workflow/activity-template/get-item-page-data",
				Handler: contracts.NewGenericHandler(workflowUseCases.ActivityTemplate.GetActivityTemplateItemPageData, &activitytemplatepb.GetActivityTemplateItemPageDataRequest{}),
			},
		)
	}

	// Activity module routes
	if workflowUseCases.Activity != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/workflow/activity/create",
			Handler: contracts.NewGenericHandler(workflowUseCases.Activity.CreateActivity, &activitypb.CreateActivityRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/workflow/activity/read",
			Handler: contracts.NewGenericHandler(workflowUseCases.Activity.ReadActivity, &activitypb.ReadActivityRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/workflow/activity/update",
			Handler: contracts.NewGenericHandler(workflowUseCases.Activity.UpdateActivity, &activitypb.UpdateActivityRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/workflow/activity/delete",
			Handler: contracts.NewGenericHandler(workflowUseCases.Activity.DeleteActivity, &activitypb.DeleteActivityRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/workflow/activity/list",
			Handler: contracts.NewGenericHandler(workflowUseCases.Activity.ListActivities, &activitypb.ListActivitiesRequest{}),
		})

		// routes = append(routes, contracts.RouteConfiguration{
		// 	Method:  "POST",
		// 	Path:    "/api/workflow/activity/get-list-page-data",
		// 	Handler: contracts.NewGenericHandler(workflowUseCases.Activity.GetActivityListPageData, &activitypb.GetActivityListPageDataRequest{}),
		// })

		// routes = append(routes, contracts.RouteConfiguration{
		// 	Method:  "POST",
		// 	Path:    "/api/workflow/activity/get-item-page-data",
		// 	Handler: contracts.NewGenericHandler(workflowUseCases.Activity.GetActivityItemPageData, &activitypb.GetActivityItemPageDataRequest{}),
		// })
	}

	// Stage module routes
	if workflowUseCases.Stage != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/workflow/stage/create",
			Handler: contracts.NewGenericHandler(workflowUseCases.Stage.CreateStage, &stagepb.CreateStageRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/workflow/stage/read",
			Handler: contracts.NewGenericHandler(workflowUseCases.Stage.ReadStage, &stagepb.ReadStageRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/workflow/stage/update",
			Handler: contracts.NewGenericHandler(workflowUseCases.Stage.UpdateStage, &stagepb.UpdateStageRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/workflow/stage/delete",
			Handler: contracts.NewGenericHandler(workflowUseCases.Stage.DeleteStage, &stagepb.DeleteStageRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/workflow/stage/list",
			Handler: contracts.NewGenericHandler(workflowUseCases.Stage.ListStages, &stagepb.ListStagesRequest{}),
		})

		// routes = append(routes, contracts.RouteConfiguration{
		// 	Method:  "POST",
		// 	Path:    "/api/workflow/stage/get-list-page-data",
		// 	Handler: contracts.NewGenericHandler(workflowUseCases.Stage.GetStageListPageData, &stagepb.GetStageListPageDataRequest{}),
		// })

		// routes = append(routes, contracts.RouteConfiguration{
		// 	Method:  "POST",
		// 	Path:    "/api/workflow/stage/get-item-page-data",
		// 	Handler: contracts.NewGenericHandler(workflowUseCases.Stage.GetStageItemPageData, &stagepb.GetStageItemPageDataRequest{}),
		// })
	}

	return contracts.DomainRouteConfiguration{
		Domain:  "workflow",
		Prefix:  "/workflow",
		Enabled: true,
		Routes:  routes,
	}
}

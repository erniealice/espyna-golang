package domain

import (
	"fmt"

	operationuc "github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation"
	"github.com/erniealice/espyna-golang/internal/composition/contracts"

	evaluationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation"
	evaluationtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation_template"
	jobpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
	jobtaskpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_task"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
)

// ConfigureOperationDomain configures routes for the Operation domain.
func ConfigureOperationDomain(operationUseCases *operationuc.OperationUseCases) contracts.DomainRouteConfiguration {
	if operationUseCases == nil {
		fmt.Printf("Operation use cases is NIL\n")
		return contracts.DomainRouteConfiguration{
			Domain:  "operation",
			Prefix:  "/operation",
			Enabled: false,
			Routes:  []contracts.RouteConfiguration{},
		}
	}

	routes := []contracts.RouteConfiguration{}

	// Job routes
	if operationUseCases.Job != nil {
		if operationUseCases.Job.CreateJob != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/job/create",
				Handler: contracts.NewGenericHandler(operationUseCases.Job.CreateJob, &jobpb.CreateJobRequest{}),
			})
		}
		if operationUseCases.Job.ReadJob != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/job/read",
				Handler: contracts.NewGenericHandler(operationUseCases.Job.ReadJob, &jobpb.ReadJobRequest{}),
			})
		}
		if operationUseCases.Job.UpdateJob != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/job/update",
				Handler: contracts.NewGenericHandler(operationUseCases.Job.UpdateJob, &jobpb.UpdateJobRequest{}),
			})
		}
		if operationUseCases.Job.DeleteJob != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/job/delete",
				Handler: contracts.NewGenericHandler(operationUseCases.Job.DeleteJob, &jobpb.DeleteJobRequest{}),
			})
		}
		if operationUseCases.Job.ListJobs != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/job/list",
				Handler: contracts.NewGenericHandler(operationUseCases.Job.ListJobs, &jobpb.ListJobsRequest{}),
			})
		}
		if operationUseCases.Job.GetJobListPageData != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/job/get-list-page-data",
				Handler: contracts.NewGenericHandler(operationUseCases.Job.GetJobListPageData, &jobpb.GetJobListPageDataRequest{}),
			})
		}
		if operationUseCases.Job.GetJobItemPageData != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/job/get-item-page-data",
				Handler: contracts.NewGenericHandler(operationUseCases.Job.GetJobItemPageData, &jobpb.GetJobItemPageDataRequest{}),
			})
		}
		if operationUseCases.Job.UpdateJobStatus != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/job/update-status",
				Handler: contracts.NewGenericHandler(operationUseCases.Job.UpdateJobStatus, &jobpb.UpdateJobStatusRequest{}),
			})
		}
	}

	// JobTemplate routes
	if operationUseCases.JobTemplate != nil {
		if operationUseCases.JobTemplate.CreateJobTemplate != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/job-template/create",
				Handler: contracts.NewGenericHandler(operationUseCases.JobTemplate.CreateJobTemplate, &jobtemplatepb.CreateJobTemplateRequest{}),
			})
		}
		if operationUseCases.JobTemplate.ReadJobTemplate != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/job-template/read",
				Handler: contracts.NewGenericHandler(operationUseCases.JobTemplate.ReadJobTemplate, &jobtemplatepb.ReadJobTemplateRequest{}),
			})
		}
		if operationUseCases.JobTemplate.UpdateJobTemplate != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/job-template/update",
				Handler: contracts.NewGenericHandler(operationUseCases.JobTemplate.UpdateJobTemplate, &jobtemplatepb.UpdateJobTemplateRequest{}),
			})
		}
		if operationUseCases.JobTemplate.DeleteJobTemplate != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/job-template/delete",
				Handler: contracts.NewGenericHandler(operationUseCases.JobTemplate.DeleteJobTemplate, &jobtemplatepb.DeleteJobTemplateRequest{}),
			})
		}
		if operationUseCases.JobTemplate.ListJobTemplates != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/job-template/list",
				Handler: contracts.NewGenericHandler(operationUseCases.JobTemplate.ListJobTemplates, &jobtemplatepb.ListJobTemplatesRequest{}),
			})
		}
		if operationUseCases.JobTemplate.GetJobTemplateListPageData != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/job-template/get-list-page-data",
				Handler: contracts.NewGenericHandler(operationUseCases.JobTemplate.GetJobTemplateListPageData, &jobtemplatepb.GetJobTemplateListPageDataRequest{}),
			})
		}
		if operationUseCases.JobTemplate.GetJobTemplateItemPageData != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/job-template/get-item-page-data",
				Handler: contracts.NewGenericHandler(operationUseCases.JobTemplate.GetJobTemplateItemPageData, &jobtemplatepb.GetJobTemplateItemPageDataRequest{}),
			})
		}
	}

	// JobTask routes
	if operationUseCases.JobTask != nil {
		if operationUseCases.JobTask.CreateJobTask != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/job-task/create",
				Handler: contracts.NewGenericHandler(operationUseCases.JobTask.CreateJobTask, &jobtaskpb.CreateJobTaskRequest{}),
			})
		}
		if operationUseCases.JobTask.ReadJobTask != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/job-task/read",
				Handler: contracts.NewGenericHandler(operationUseCases.JobTask.ReadJobTask, &jobtaskpb.ReadJobTaskRequest{}),
			})
		}
		if operationUseCases.JobTask.UpdateJobTask != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/job-task/update",
				Handler: contracts.NewGenericHandler(operationUseCases.JobTask.UpdateJobTask, &jobtaskpb.UpdateJobTaskRequest{}),
			})
		}
		if operationUseCases.JobTask.DeleteJobTask != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/job-task/delete",
				Handler: contracts.NewGenericHandler(operationUseCases.JobTask.DeleteJobTask, &jobtaskpb.DeleteJobTaskRequest{}),
			})
		}
		if operationUseCases.JobTask.ListJobTasks != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/job-task/list",
				Handler: contracts.NewGenericHandler(operationUseCases.JobTask.ListJobTasks, &jobtaskpb.ListJobTasksRequest{}),
			})
		}
		if operationUseCases.JobTask.GetJobTaskListPageData != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/job-task/get-list-page-data",
				Handler: contracts.NewGenericHandler(operationUseCases.JobTask.GetJobTaskListPageData, &jobtaskpb.GetJobTaskListPageDataRequest{}),
			})
		}
	}

	// Evaluation routes (Performance Evaluation 20260604 v1).
	if operationUseCases.Evaluation != nil {
		if operationUseCases.Evaluation.CreateEvaluation != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/evaluation/create",
				Handler: contracts.NewGenericHandler(operationUseCases.Evaluation.CreateEvaluation, &evaluationpb.CreateEvaluationRequest{}),
			})
		}
		if operationUseCases.Evaluation.ReadEvaluation != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/evaluation/read",
				Handler: contracts.NewGenericHandler(operationUseCases.Evaluation.ReadEvaluation, &evaluationpb.ReadEvaluationRequest{}),
			})
		}
		if operationUseCases.Evaluation.UpdateEvaluation != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/evaluation/update",
				Handler: contracts.NewGenericHandler(operationUseCases.Evaluation.UpdateEvaluation, &evaluationpb.UpdateEvaluationRequest{}),
			})
		}
		if operationUseCases.Evaluation.DeleteEvaluation != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/evaluation/delete",
				Handler: contracts.NewGenericHandler(operationUseCases.Evaluation.DeleteEvaluation, &evaluationpb.DeleteEvaluationRequest{}),
			})
		}
		if operationUseCases.Evaluation.ListEvaluations != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/evaluation/list",
				Handler: contracts.NewGenericHandler(operationUseCases.Evaluation.ListEvaluations, &evaluationpb.ListEvaluationsRequest{}),
			})
		}
		if operationUseCases.Evaluation.GetEvaluationListPageData != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/evaluation/get-list-page-data",
				Handler: contracts.NewGenericHandler(operationUseCases.Evaluation.GetEvaluationListPageData, &evaluationpb.GetEvaluationListPageDataRequest{}),
			})
		}
		if operationUseCases.Evaluation.GetEvaluationItemPageData != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/evaluation/get-item-page-data",
				Handler: contracts.NewGenericHandler(operationUseCases.Evaluation.GetEvaluationItemPageData, &evaluationpb.GetEvaluationItemPageDataRequest{}),
			})
		}
	}

	// EvaluationTemplate routes.
	if operationUseCases.EvaluationTemplate != nil {
		if operationUseCases.EvaluationTemplate.Create != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/evaluation-template/create",
				Handler: contracts.NewGenericHandler(operationUseCases.EvaluationTemplate.Create, &evaluationtemplatepb.CreateEvaluationTemplateRequest{}),
			})
		}
		if operationUseCases.EvaluationTemplate.List != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/evaluation-template/list",
				Handler: contracts.NewGenericHandler(operationUseCases.EvaluationTemplate.List, &evaluationtemplatepb.ListEvaluationTemplatesRequest{}),
			})
		}
		if operationUseCases.EvaluationTemplate.GetListPage != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/evaluation-template/get-list-page-data",
				Handler: contracts.NewGenericHandler(operationUseCases.EvaluationTemplate.GetListPage, &evaluationtemplatepb.GetEvaluationTemplateListPageDataRequest{}),
			})
		}
	}

	// WorkRequest routes (20260604-requests-workflow v1) — wired when use cases land.
	// WorkRequestType routes (20260604-requests-workflow v1) — wired when use cases land.

	return contracts.DomainRouteConfiguration{
		Domain:  "operation",
		Prefix:  "/operation",
		Enabled: len(routes) > 0,
		Routes:  routes,
	}
}

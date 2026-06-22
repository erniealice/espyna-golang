package domain

import (
	"fmt"

	operationuc "github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation"
	"github.com/erniealice/espyna-golang/internal/composition/contracts"

	evaluationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation"
	evaluationtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation_template"
	jobpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
	joboutcomelinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_line"
	jobtaskpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_task"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	reportingcheckpointpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/reporting_checkpoint"
	scorescalepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale"
	scorescalebandpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale_band"
	scoringcomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/scoring_component"
	scoringcomponentcriteriapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/scoring_component_criteria"
	scoringschemepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/scoring_scheme"
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

	// Education grading (20260616 v1) — single-repo CRUD entities.

	// ScoringScheme routes.
	if operationUseCases.ScoringScheme != nil {
		if operationUseCases.ScoringScheme.CreateScoringScheme != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/scoring-scheme/create",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoringScheme.CreateScoringScheme, &scoringschemepb.CreateScoringSchemeRequest{}),
			})
		}
		if operationUseCases.ScoringScheme.ReadScoringScheme != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/scoring-scheme/read",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoringScheme.ReadScoringScheme, &scoringschemepb.ReadScoringSchemeRequest{}),
			})
		}
		if operationUseCases.ScoringScheme.UpdateScoringScheme != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/scoring-scheme/update",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoringScheme.UpdateScoringScheme, &scoringschemepb.UpdateScoringSchemeRequest{}),
			})
		}
		if operationUseCases.ScoringScheme.DeleteScoringScheme != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/scoring-scheme/delete",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoringScheme.DeleteScoringScheme, &scoringschemepb.DeleteScoringSchemeRequest{}),
			})
		}
		if operationUseCases.ScoringScheme.ListScoringSchemes != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/scoring-scheme/list",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoringScheme.ListScoringSchemes, &scoringschemepb.ListScoringSchemesRequest{}),
			})
		}
		if operationUseCases.ScoringScheme.GetScoringSchemeListPageData != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/scoring-scheme/get-list-page-data",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoringScheme.GetScoringSchemeListPageData, &scoringschemepb.GetScoringSchemeListPageDataRequest{}),
			})
		}
		if operationUseCases.ScoringScheme.GetScoringSchemeItemPageData != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/scoring-scheme/get-item-page-data",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoringScheme.GetScoringSchemeItemPageData, &scoringschemepb.GetScoringSchemeItemPageDataRequest{}),
			})
		}
	}

	// ScoringComponent routes.
	if operationUseCases.ScoringComponent != nil {
		if operationUseCases.ScoringComponent.CreateScoringComponent != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/scoring-component/create",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoringComponent.CreateScoringComponent, &scoringcomponentpb.CreateScoringComponentRequest{}),
			})
		}
		if operationUseCases.ScoringComponent.ReadScoringComponent != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/scoring-component/read",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoringComponent.ReadScoringComponent, &scoringcomponentpb.ReadScoringComponentRequest{}),
			})
		}
		if operationUseCases.ScoringComponent.UpdateScoringComponent != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/scoring-component/update",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoringComponent.UpdateScoringComponent, &scoringcomponentpb.UpdateScoringComponentRequest{}),
			})
		}
		if operationUseCases.ScoringComponent.DeleteScoringComponent != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/scoring-component/delete",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoringComponent.DeleteScoringComponent, &scoringcomponentpb.DeleteScoringComponentRequest{}),
			})
		}
		if operationUseCases.ScoringComponent.ListScoringComponents != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/scoring-component/list",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoringComponent.ListScoringComponents, &scoringcomponentpb.ListScoringComponentsRequest{}),
			})
		}
		if operationUseCases.ScoringComponent.GetScoringComponentListPageData != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/scoring-component/get-list-page-data",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoringComponent.GetScoringComponentListPageData, &scoringcomponentpb.GetScoringComponentListPageDataRequest{}),
			})
		}
		if operationUseCases.ScoringComponent.GetScoringComponentItemPageData != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/scoring-component/get-item-page-data",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoringComponent.GetScoringComponentItemPageData, &scoringcomponentpb.GetScoringComponentItemPageDataRequest{}),
			})
		}
	}

	// ScoringComponentCriteria routes.
	if operationUseCases.ScoringComponentCriteria != nil {
		if operationUseCases.ScoringComponentCriteria.CreateScoringComponentCriteria != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/scoring-component-criteria/create",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoringComponentCriteria.CreateScoringComponentCriteria, &scoringcomponentcriteriapb.CreateScoringComponentCriteriaRequest{}),
			})
		}
		if operationUseCases.ScoringComponentCriteria.ReadScoringComponentCriteria != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/scoring-component-criteria/read",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoringComponentCriteria.ReadScoringComponentCriteria, &scoringcomponentcriteriapb.ReadScoringComponentCriteriaRequest{}),
			})
		}
		if operationUseCases.ScoringComponentCriteria.UpdateScoringComponentCriteria != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/scoring-component-criteria/update",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoringComponentCriteria.UpdateScoringComponentCriteria, &scoringcomponentcriteriapb.UpdateScoringComponentCriteriaRequest{}),
			})
		}
		if operationUseCases.ScoringComponentCriteria.DeleteScoringComponentCriteria != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/scoring-component-criteria/delete",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoringComponentCriteria.DeleteScoringComponentCriteria, &scoringcomponentcriteriapb.DeleteScoringComponentCriteriaRequest{}),
			})
		}
		if operationUseCases.ScoringComponentCriteria.ListScoringComponentCriterias != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/scoring-component-criteria/list",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoringComponentCriteria.ListScoringComponentCriterias, &scoringcomponentcriteriapb.ListScoringComponentCriteriasRequest{}),
			})
		}
		if operationUseCases.ScoringComponentCriteria.GetScoringComponentCriteriaListPageData != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/scoring-component-criteria/get-list-page-data",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoringComponentCriteria.GetScoringComponentCriteriaListPageData, &scoringcomponentcriteriapb.GetScoringComponentCriteriaListPageDataRequest{}),
			})
		}
		if operationUseCases.ScoringComponentCriteria.GetScoringComponentCriteriaItemPageData != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/scoring-component-criteria/get-item-page-data",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoringComponentCriteria.GetScoringComponentCriteriaItemPageData, &scoringcomponentcriteriapb.GetScoringComponentCriteriaItemPageDataRequest{}),
			})
		}
	}

	// ScoreScale routes.
	if operationUseCases.ScoreScale != nil {
		if operationUseCases.ScoreScale.CreateScoreScale != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/score-scale/create",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoreScale.CreateScoreScale, &scorescalepb.CreateScoreScaleRequest{}),
			})
		}
		if operationUseCases.ScoreScale.ReadScoreScale != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/score-scale/read",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoreScale.ReadScoreScale, &scorescalepb.ReadScoreScaleRequest{}),
			})
		}
		if operationUseCases.ScoreScale.UpdateScoreScale != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/score-scale/update",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoreScale.UpdateScoreScale, &scorescalepb.UpdateScoreScaleRequest{}),
			})
		}
		if operationUseCases.ScoreScale.DeleteScoreScale != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/score-scale/delete",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoreScale.DeleteScoreScale, &scorescalepb.DeleteScoreScaleRequest{}),
			})
		}
		if operationUseCases.ScoreScale.ListScoreScales != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/score-scale/list",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoreScale.ListScoreScales, &scorescalepb.ListScoreScalesRequest{}),
			})
		}
		if operationUseCases.ScoreScale.GetScoreScaleListPageData != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/score-scale/get-list-page-data",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoreScale.GetScoreScaleListPageData, &scorescalepb.GetScoreScaleListPageDataRequest{}),
			})
		}
		if operationUseCases.ScoreScale.GetScoreScaleItemPageData != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/score-scale/get-item-page-data",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoreScale.GetScoreScaleItemPageData, &scorescalepb.GetScoreScaleItemPageDataRequest{}),
			})
		}
	}

	// ScoreScaleBand routes.
	if operationUseCases.ScoreScaleBand != nil {
		if operationUseCases.ScoreScaleBand.CreateScoreScaleBand != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/score-scale-band/create",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoreScaleBand.CreateScoreScaleBand, &scorescalebandpb.CreateScoreScaleBandRequest{}),
			})
		}
		if operationUseCases.ScoreScaleBand.ReadScoreScaleBand != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/score-scale-band/read",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoreScaleBand.ReadScoreScaleBand, &scorescalebandpb.ReadScoreScaleBandRequest{}),
			})
		}
		if operationUseCases.ScoreScaleBand.UpdateScoreScaleBand != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/score-scale-band/update",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoreScaleBand.UpdateScoreScaleBand, &scorescalebandpb.UpdateScoreScaleBandRequest{}),
			})
		}
		if operationUseCases.ScoreScaleBand.DeleteScoreScaleBand != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/score-scale-band/delete",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoreScaleBand.DeleteScoreScaleBand, &scorescalebandpb.DeleteScoreScaleBandRequest{}),
			})
		}
		if operationUseCases.ScoreScaleBand.ListScoreScaleBands != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/score-scale-band/list",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoreScaleBand.ListScoreScaleBands, &scorescalebandpb.ListScoreScaleBandsRequest{}),
			})
		}
		if operationUseCases.ScoreScaleBand.GetScoreScaleBandListPageData != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/score-scale-band/get-list-page-data",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoreScaleBand.GetScoreScaleBandListPageData, &scorescalebandpb.GetScoreScaleBandListPageDataRequest{}),
			})
		}
		if operationUseCases.ScoreScaleBand.GetScoreScaleBandItemPageData != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/score-scale-band/get-item-page-data",
				Handler: contracts.NewGenericHandler(operationUseCases.ScoreScaleBand.GetScoreScaleBandItemPageData, &scorescalebandpb.GetScoreScaleBandItemPageDataRequest{}),
			})
		}
	}

	// JobOutcomeLine routes.
	if operationUseCases.JobOutcomeLine != nil {
		if operationUseCases.JobOutcomeLine.CreateJobOutcomeLine != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/job-outcome-line/create",
				Handler: contracts.NewGenericHandler(operationUseCases.JobOutcomeLine.CreateJobOutcomeLine, &joboutcomelinepb.CreateJobOutcomeLineRequest{}),
			})
		}
		if operationUseCases.JobOutcomeLine.ReadJobOutcomeLine != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/job-outcome-line/read",
				Handler: contracts.NewGenericHandler(operationUseCases.JobOutcomeLine.ReadJobOutcomeLine, &joboutcomelinepb.ReadJobOutcomeLineRequest{}),
			})
		}
		if operationUseCases.JobOutcomeLine.UpdateJobOutcomeLine != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/job-outcome-line/update",
				Handler: contracts.NewGenericHandler(operationUseCases.JobOutcomeLine.UpdateJobOutcomeLine, &joboutcomelinepb.UpdateJobOutcomeLineRequest{}),
			})
		}
		if operationUseCases.JobOutcomeLine.DeleteJobOutcomeLine != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/job-outcome-line/delete",
				Handler: contracts.NewGenericHandler(operationUseCases.JobOutcomeLine.DeleteJobOutcomeLine, &joboutcomelinepb.DeleteJobOutcomeLineRequest{}),
			})
		}
		if operationUseCases.JobOutcomeLine.ListJobOutcomeLines != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/job-outcome-line/list",
				Handler: contracts.NewGenericHandler(operationUseCases.JobOutcomeLine.ListJobOutcomeLines, &joboutcomelinepb.ListJobOutcomeLinesRequest{}),
			})
		}
		if operationUseCases.JobOutcomeLine.GetJobOutcomeLineListPageData != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/job-outcome-line/get-list-page-data",
				Handler: contracts.NewGenericHandler(operationUseCases.JobOutcomeLine.GetJobOutcomeLineListPageData, &joboutcomelinepb.GetJobOutcomeLineListPageDataRequest{}),
			})
		}
		if operationUseCases.JobOutcomeLine.GetJobOutcomeLineItemPageData != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/job-outcome-line/get-item-page-data",
				Handler: contracts.NewGenericHandler(operationUseCases.JobOutcomeLine.GetJobOutcomeLineItemPageData, &joboutcomelinepb.GetJobOutcomeLineItemPageDataRequest{}),
			})
		}
	}

	// ReportingCheckpoint routes.
	if operationUseCases.ReportingCheckpoint != nil {
		if operationUseCases.ReportingCheckpoint.CreateReportingCheckpoint != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/reporting-checkpoint/create",
				Handler: contracts.NewGenericHandler(operationUseCases.ReportingCheckpoint.CreateReportingCheckpoint, &reportingcheckpointpb.CreateReportingCheckpointRequest{}),
			})
		}
		if operationUseCases.ReportingCheckpoint.ReadReportingCheckpoint != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/reporting-checkpoint/read",
				Handler: contracts.NewGenericHandler(operationUseCases.ReportingCheckpoint.ReadReportingCheckpoint, &reportingcheckpointpb.ReadReportingCheckpointRequest{}),
			})
		}
		if operationUseCases.ReportingCheckpoint.UpdateReportingCheckpoint != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/reporting-checkpoint/update",
				Handler: contracts.NewGenericHandler(operationUseCases.ReportingCheckpoint.UpdateReportingCheckpoint, &reportingcheckpointpb.UpdateReportingCheckpointRequest{}),
			})
		}
		if operationUseCases.ReportingCheckpoint.DeleteReportingCheckpoint != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/reporting-checkpoint/delete",
				Handler: contracts.NewGenericHandler(operationUseCases.ReportingCheckpoint.DeleteReportingCheckpoint, &reportingcheckpointpb.DeleteReportingCheckpointRequest{}),
			})
		}
		if operationUseCases.ReportingCheckpoint.ListReportingCheckpoints != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/reporting-checkpoint/list",
				Handler: contracts.NewGenericHandler(operationUseCases.ReportingCheckpoint.ListReportingCheckpoints, &reportingcheckpointpb.ListReportingCheckpointsRequest{}),
			})
		}
		if operationUseCases.ReportingCheckpoint.GetReportingCheckpointListPageData != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/reporting-checkpoint/get-list-page-data",
				Handler: contracts.NewGenericHandler(operationUseCases.ReportingCheckpoint.GetReportingCheckpointListPageData, &reportingcheckpointpb.GetReportingCheckpointListPageDataRequest{}),
			})
		}
		if operationUseCases.ReportingCheckpoint.GetReportingCheckpointItemPageData != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/operation/reporting-checkpoint/get-item-page-data",
				Handler: contracts.NewGenericHandler(operationUseCases.ReportingCheckpoint.GetReportingCheckpointItemPageData, &reportingcheckpointpb.GetReportingCheckpointItemPageDataRequest{}),
			})
		}
	}

	return contracts.DomainRouteConfiguration{
		Domain:  "operation",
		Prefix:  "/operation",
		Enabled: len(routes) > 0,
		Routes:  routes,
	}
}

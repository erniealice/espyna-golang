package job_template

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	jobpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
	jobphasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
	jobtaskpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_task"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	jtphasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
	jttaskpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_task"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
)

// InstantiateJobsFromPlanRepositories groups all repository dependencies for this use case.
type InstantiateJobsFromPlanRepositories struct {
	ProductPlan      productplanpb.ProductPlanDomainServiceServer
	JobTemplate      jobtemplatepb.JobTemplateDomainServiceServer
	JobTemplatePhase jtphasepb.JobTemplatePhaseDomainServiceServer
	JobTemplateTask  jttaskpb.JobTemplateTaskDomainServiceServer
	Job              jobpb.JobDomainServiceServer
	JobPhase         jobphasepb.JobPhaseDomainServiceServer
	JobTask          jobtaskpb.JobTaskDomainServiceServer
}

// InstantiateJobsFromPlanServices groups all business service dependencies.
type InstantiateJobsFromPlanServices struct {
	TransactionService ports.TransactionService
	IDService          ports.IDService
}

// InstantiateJobsFromPlanUseCase creates a Job hierarchy from JobTemplates
// linked to ProductPlans for the given plan.
type InstantiateJobsFromPlanUseCase struct {
	repositories InstantiateJobsFromPlanRepositories
	services     InstantiateJobsFromPlanServices
}

// NewInstantiateJobsFromPlanUseCase creates a new InstantiateJobsFromPlanUseCase.
func NewInstantiateJobsFromPlanUseCase(
	repos InstantiateJobsFromPlanRepositories,
	services InstantiateJobsFromPlanServices,
) *InstantiateJobsFromPlanUseCase {
	return &InstantiateJobsFromPlanUseCase{repositories: repos, services: services}
}

// templateHierarchy holds a resolved job template with its phases and tasks.
type templateHierarchy struct {
	template *jobtemplatepb.JobTemplate
	phases   []*phaseWithTasks
}

type phaseWithTasks struct {
	phase *jtphasepb.JobTemplatePhase
	tasks []*jttaskpb.JobTemplateTask
}

// InstantiateJobsFromPlan reads ProductPlans for planID, resolves linked
// JobTemplate hierarchies, and creates Jobs + JobPhases + JobTasks in a
// single transaction.
func (uc *InstantiateJobsFromPlanUseCase) InstantiateJobsFromPlan(
	ctx context.Context, planID, clientID, subscriptionID, workspaceID string,
) error {
	// 1. List ProductPlans for the given plan.
	listResp, err := uc.repositories.ProductPlan.ListByPlan(ctx, &productplanpb.ListProductPlansByPlanRequest{PlanId: planID})
	if err != nil {
		return fmt.Errorf("InstantiateJobsFromPlan: list product plans: %w", err)
	}

	// 2. Filter to those with a linked job template.
	var qualifying []*productplanpb.ProductPlan
	for _, pp := range listResp.GetProductPlans() {
		if pp.GetJobTemplateId() != "" {
			qualifying = append(qualifying, pp)
		}
	}
	if len(qualifying) == 0 {
		return nil
	}

	// 3. Read template hierarchies OUTSIDE the transaction (reads use raw DB).
	hierarchies := make([]*templateHierarchy, 0, len(qualifying))
	for _, pp := range qualifying {
		templateID := pp.GetJobTemplateId()

		tmplResp, err := uc.repositories.JobTemplate.ReadJobTemplate(ctx, &jobtemplatepb.ReadJobTemplateRequest{
			Data: &jobtemplatepb.JobTemplate{Id: templateID},
		})
		if err != nil {
			return fmt.Errorf("InstantiateJobsFromPlan: read job template %s: %w", templateID, err)
		}
		if len(tmplResp.GetData()) == 0 {
			return fmt.Errorf("InstantiateJobsFromPlan: job template %s not found", templateID)
		}
		tmpl := tmplResp.GetData()[0]

		phasesResp, err := uc.repositories.JobTemplatePhase.ListByJobTemplate(ctx, &jtphasepb.ListByJobTemplateRequest{
			JobTemplateId: templateID,
		})
		if err != nil {
			return fmt.Errorf("InstantiateJobsFromPlan: list phases for template %s: %w", templateID, err)
		}

		pwt := make([]*phaseWithTasks, 0, len(phasesResp.GetJobTemplatePhases()))
		for _, phase := range phasesResp.GetJobTemplatePhases() {
			tasksResp, err := uc.repositories.JobTemplateTask.ListByPhase(ctx, &jttaskpb.ListJobTemplateTasksByPhaseRequest{
				JobTemplatePhaseId: phase.Id,
			})
			if err != nil {
				return fmt.Errorf("InstantiateJobsFromPlan: list tasks for phase %s: %w", phase.Id, err)
			}
			pwt = append(pwt, &phaseWithTasks{
				phase: phase,
				tasks: tasksResp.GetJobTemplateTasks(),
			})
		}

		hierarchies = append(hierarchies, &templateHierarchy{
			template: tmpl,
			phases:   pwt,
		})
	}

	// 4. Write all Jobs + phases + tasks in a single transaction.
	count := 0
	writeFunc := func(txCtx context.Context) error {
		now := time.Now()
		dc := now.UnixMilli()
		dcs := now.Format(time.RFC3339)

		for _, h := range hierarchies {
			// Generate job ID.
			var jobID string
			if uc.services.IDService != nil {
				jobID = uc.services.IDService.GenerateID()
			} else {
				jobID = fmt.Sprintf("job-%d", now.UnixNano())
			}

			templateID := h.template.Id
			jobData := &jobpb.Job{
				Id:                 jobID,
				Name:               h.template.Name,
				JobTemplateId:      &templateID,
				OriginType:         enumspb.OriginType_ORIGIN_TYPE_SUBSCRIPTION,
				OriginId:           &subscriptionID,
				ClientId:           &clientID,
				Status:             enumspb.JobStatus_JOB_STATUS_DRAFT,
				WorkspaceId:        &workspaceID,
				Active:             true,
				DateCreated:        &dc,
				DateCreatedString:  &dcs,
				DateModified:       &dc,
				DateModifiedString: &dcs,
			}

			// Copy default type fields from template if set.
			if h.template.DefaultFulfillmentType != nil {
				ft := *h.template.DefaultFulfillmentType
				jobData.FulfillmentType = ft
			}
			if h.template.DefaultCostFlowType != nil {
				ct := *h.template.DefaultCostFlowType
				jobData.CostFlowType = ct
			}
			if h.template.DefaultBillingRuleType != nil {
				bt := *h.template.DefaultBillingRuleType
				jobData.BillingRuleType = bt
			}

			if _, err := uc.repositories.Job.CreateJob(txCtx, &jobpb.CreateJobRequest{Data: jobData}); err != nil {
				return fmt.Errorf("InstantiateJobsFromPlan: create job for template %s: %w", templateID, err)
			}

			// Create phases and tasks.
			for _, pwt := range h.phases {
				var phaseID string
				if uc.services.IDService != nil {
					phaseID = uc.services.IDService.GenerateID()
				} else {
					phaseID = fmt.Sprintf("phase-%d", now.UnixNano())
				}

				phaseData := &jobphasepb.JobPhase{
					Id:                 phaseID,
					JobId:              jobID,
					Name:               pwt.phase.Name,
					PhaseOrder:         pwt.phase.PhaseOrder,
					Status:             jobphasepb.PhaseStatus_PHASE_STATUS_PENDING,
					Active:             true,
					DateCreated:        &dc,
					DateCreatedString:  &dcs,
					DateModified:       &dc,
					DateModifiedString: &dcs,
				}
				if _, err := uc.repositories.JobPhase.CreateJobPhase(txCtx, &jobphasepb.CreateJobPhaseRequest{Data: phaseData}); err != nil {
					return fmt.Errorf("InstantiateJobsFromPlan: create job phase for template phase %s: %w", pwt.phase.Id, err)
				}

				for _, task := range pwt.tasks {
					var taskID string
					if uc.services.IDService != nil {
						taskID = uc.services.IDService.GenerateID()
					} else {
						taskID = fmt.Sprintf("task-%d", now.UnixNano())
					}

					taskData := &jobtaskpb.JobTask{
						Id:                 taskID,
						JobPhaseId:         phaseID,
						Name:               task.Name,
						StepOrder:          task.StepOrder,
						Status:             jobtaskpb.TaskStatus_TASK_STATUS_PENDING,
						IsAdHoc:            false,
						Active:             true,
						DateCreated:        &dc,
						DateCreatedString:  &dcs,
						DateModified:       &dc,
						DateModifiedString: &dcs,
					}
					if _, err := uc.repositories.JobTask.CreateJobTask(txCtx, &jobtaskpb.CreateJobTaskRequest{Data: taskData}); err != nil {
						return fmt.Errorf("InstantiateJobsFromPlan: create job task for template task %s: %w", task.Id, err)
					}
				}
			}

			count++
		}
		return nil
	}

	if uc.services.TransactionService != nil {
		if err := uc.services.TransactionService.ExecuteInTransaction(ctx, writeFunc); err != nil {
			return err
		}
	} else {
		if err := writeFunc(ctx); err != nil {
			return err
		}
	}

	log.Printf("InstantiateJobsFromPlan: created %d jobs for plan %s (subscription %s)", count, planID, subscriptionID)
	return nil
}

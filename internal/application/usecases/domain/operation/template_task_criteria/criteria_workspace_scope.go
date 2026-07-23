package template_task_criteria

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	jobtemplatephasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
	jobtemplatetaskpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_task"
	outcomecriteriapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
)

// template_task_criteria carries NO workspace_id column; its tenant scope is
// inherited from the pinned job_template_task via the
// task -> job_template_phase -> job_template chain. These helpers resolve that
// chain so every use case can fail-closed against cross-workspace rubric
// pinning (red-team HIGH #4 — cross-workspace IDOR).
//
// The pinned rubric criterion (outcome_criteria_id) DOES carry its own
// workspace_id, and must ALSO be proven in-workspace so a crafted POST cannot
// pin a foreign workspace's criterion onto a task the caller legitimately owns.

// scopeRepos bundles the repos required to prove both ends of the pin are
// in-workspace: the task chain (task -> phase -> template) and the criterion.
type scopeRepos struct {
	JobTemplateTask  jobtemplatetaskpb.JobTemplateTaskDomainServiceServer
	JobTemplatePhase jobtemplatephasepb.JobTemplatePhaseDomainServiceServer
	JobTemplate      jobtemplatepb.JobTemplateDomainServiceServer
	OutcomeCriteria  outcomecriteriapb.OutcomeCriteriaDomainServiceServer
}

// taskChainWorkspace resolves the owning job_template's workspace_id for a
// given job_template_task_id by walking task -> phase -> template. Any missing
// link (nil repo, empty id, read error, empty result) returns found=false so
// callers fail-closed.
func taskChainWorkspace(ctx context.Context, r scopeRepos, taskID string) (string, bool) {
	if r.JobTemplateTask == nil || r.JobTemplatePhase == nil || r.JobTemplate == nil || taskID == "" {
		return "", false
	}
	taskResp, err := r.JobTemplateTask.ReadJobTemplateTask(ctx, &jobtemplatetaskpb.ReadJobTemplateTaskRequest{
		Data: &jobtemplatetaskpb.JobTemplateTask{Id: taskID},
	})
	if err != nil || taskResp == nil || len(taskResp.GetData()) == 0 {
		return "", false
	}
	phaseID := taskResp.GetData()[0].GetJobTemplatePhaseId()
	if phaseID == "" {
		return "", false
	}
	phaseResp, err := r.JobTemplatePhase.ReadJobTemplatePhase(ctx, &jobtemplatephasepb.ReadJobTemplatePhaseRequest{
		Data: &jobtemplatephasepb.JobTemplatePhase{Id: phaseID},
	})
	if err != nil || phaseResp == nil || len(phaseResp.GetData()) == 0 {
		return "", false
	}
	templateID := phaseResp.GetData()[0].GetJobTemplateId()
	if templateID == "" {
		return "", false
	}
	tmplResp, err := r.JobTemplate.ReadJobTemplate(ctx, &jobtemplatepb.ReadJobTemplateRequest{
		Data: &jobtemplatepb.JobTemplate{Id: templateID},
	})
	if err != nil || tmplResp == nil || len(tmplResp.GetData()) == 0 {
		return "", false
	}
	return tmplResp.GetData()[0].GetWorkspaceId(), true
}

// requireTaskChainInWorkspace fail-closes unless the task chain resolves and the
// owning template's workspace_id equals wsID (which must be non-empty).
func requireTaskChainInWorkspace(ctx context.Context, r scopeRepos, tr ports.Translator, wsID, taskID string) error {
	ws, ok := taskChainWorkspace(ctx, r, taskID)
	if !ok {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"template_task_criteria.validation.task_chain_not_found",
			"[ERR-DEFAULT] Referenced job template task chain not found"))
	}
	if wsID == "" || ws == "" || ws != wsID {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"template_task_criteria.validation.cross_workspace",
			"[ERR-DEFAULT] Template task criteria must stay within your workspace"))
	}
	return nil
}

// criterionWorkspace reads an outcome_criteria row and returns its workspace_id
// plus a found flag. A nil repo, empty id, read error, or empty result all
// return found=false so callers fail-closed.
func criterionWorkspace(ctx context.Context, repo outcomecriteriapb.OutcomeCriteriaDomainServiceServer, criterionID string) (string, bool) {
	if repo == nil || criterionID == "" {
		return "", false
	}
	resp, err := repo.ReadOutcomeCriteria(ctx, &outcomecriteriapb.ReadOutcomeCriteriaRequest{
		Data: &outcomecriteriapb.OutcomeCriteria{Id: criterionID},
	})
	if err != nil || resp == nil || len(resp.GetData()) == 0 {
		return "", false
	}
	return resp.GetData()[0].GetWorkspaceId(), true
}

// requireCriterionInWorkspace fail-closes unless the pinned outcome_criteria
// exists AND its workspace_id equals wsID (which must be non-empty).
func requireCriterionInWorkspace(ctx context.Context, r scopeRepos, tr ports.Translator, wsID, criterionID string) error {
	ws, ok := criterionWorkspace(ctx, r.OutcomeCriteria, criterionID)
	if !ok {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"template_task_criteria.validation.criterion_not_found",
			"[ERR-DEFAULT] Referenced outcome criterion not found"))
	}
	if wsID == "" || ws == "" || ws != wsID {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"template_task_criteria.validation.cross_workspace",
			"[ERR-DEFAULT] Template task criteria must stay within your workspace"))
	}
	return nil
}

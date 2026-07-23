package job_template_phase

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	scoringschemepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/scoring_scheme"
)

// job_template_phase carries its own workspace_id, but its FKs are posted
// directly from the drawer and must be proven in-workspace (red-team HIGH #2):
//   - job_template_id  — the owning template (job_template.workspace_id, field 12)
//   - scoring_scheme_id — an optional grading scheme (scoring_scheme.workspace_id,
//     field 2)
//
// The guard resolves each and requires its workspace_id to equal the caller's
// request workspace. A nil repo, empty request workspace, read error, or empty
// result set all fail-closed.

// phaseScopeRepos bundles the FK-resolution repos.
type phaseScopeRepos struct {
	JobTemplate   jobtemplatepb.JobTemplateDomainServiceServer
	ScoringScheme scoringschemepb.ScoringSchemeDomainServiceServer
}

// requireOwningTemplateInWorkspace fail-closes unless the owning job_template
// exists AND its workspace_id equals wsID.
func requireOwningTemplateInWorkspace(ctx context.Context, repo jobtemplatepb.JobTemplateDomainServiceServer, tr ports.Translator, wsID, templateID string) error {
	if repo == nil || templateID == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"job_template_phase.validation.template_not_found",
			"[ERR-DEFAULT] Referenced job template not found"))
	}
	resp, err := repo.ReadJobTemplate(ctx, &jobtemplatepb.ReadJobTemplateRequest{
		Data: &jobtemplatepb.JobTemplate{Id: templateID},
	})
	if err != nil || resp == nil || len(resp.GetData()) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"job_template_phase.validation.template_not_found",
			"[ERR-DEFAULT] Referenced job template not found"))
	}
	ws := resp.GetData()[0].GetWorkspaceId()
	if wsID == "" || ws == "" || ws != wsID {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"job_template_phase.validation.cross_workspace",
			"[ERR-DEFAULT] Job template phase must stay within your workspace"))
	}
	return nil
}

// requireScoringSchemeInWorkspace fail-closes unless the referenced
// scoring_scheme exists AND its workspace_id equals wsID. An empty schemeID is a
// no-op (the FK is optional).
func requireScoringSchemeInWorkspace(ctx context.Context, repo scoringschemepb.ScoringSchemeDomainServiceServer, tr ports.Translator, wsID, schemeID string) error {
	if schemeID == "" {
		return nil
	}
	if repo == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"job_template_phase.validation.scoring_scheme_not_found",
			"[ERR-DEFAULT] Referenced scoring scheme not found"))
	}
	resp, err := repo.ReadScoringScheme(ctx, &scoringschemepb.ReadScoringSchemeRequest{
		Data: &scoringschemepb.ScoringScheme{Id: schemeID},
	})
	if err != nil || resp == nil || len(resp.GetData()) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"job_template_phase.validation.scoring_scheme_not_found",
			"[ERR-DEFAULT] Referenced scoring scheme not found"))
	}
	ws := resp.GetData()[0].GetWorkspaceId()
	if wsID == "" || ws == "" || ws != wsID {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"job_template_phase.validation.cross_workspace",
			"[ERR-DEFAULT] Job template phase must stay within your workspace"))
	}
	return nil
}

package job_template_relation

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
)

// job_template_relation carries NO workspace_id column of its own; its tenant
// scope is inherited from the owning job_template rows (parent/child). These
// helpers resolve that scope so every use case can fail-closed against
// cross-workspace graph links (red-team HIGH #4 — cross-workspace IDOR).

// templateWorkspace reads a job_template row and returns its workspace_id plus a
// found flag. A nil repo, empty id, read error, or empty result set all return
// found=false so callers fail-closed.
func templateWorkspace(ctx context.Context, repo jobtemplatepb.JobTemplateDomainServiceServer, templateID string) (string, bool) {
	if repo == nil || templateID == "" {
		return "", false
	}
	resp, err := repo.ReadJobTemplate(ctx, &jobtemplatepb.ReadJobTemplateRequest{
		Data: &jobtemplatepb.JobTemplate{Id: templateID},
	})
	if err != nil || resp == nil || len(resp.GetData()) == 0 {
		return "", false
	}
	return resp.GetData()[0].GetWorkspaceId(), true
}

// requireTemplateInWorkspace fail-closes unless the referenced job_template
// exists AND its workspace_id equals wsID. wsID must be non-empty (an empty
// request workspace is itself a fail-closed condition — an empty==empty match
// must never grant access).
func requireTemplateInWorkspace(
	ctx context.Context,
	repo jobtemplatepb.JobTemplateDomainServiceServer,
	tr ports.Translator,
	wsID, templateID string,
) error {
	ws, ok := templateWorkspace(ctx, repo, templateID)
	if !ok {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"job_template_relation.validation.template_not_found",
			"referenced job template not found [DEFAULT]"))
	}
	if wsID == "" || ws == "" || ws != wsID {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"job_template_relation.validation.cross_workspace",
			"job template relation must stay within your workspace [DEFAULT]"))
	}
	return nil
}

package job_template_relation

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	jobtemplaterelationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_relation"
)

// DeleteJobTemplateRelationRepositories groups repository dependencies.
type DeleteJobTemplateRelationRepositories struct {
	JobTemplateRelation jobtemplaterelationpb.JobTemplateRelationDomainServiceServer
	JobTemplate         jobtemplatepb.JobTemplateDomainServiceServer
}

// DeleteJobTemplateRelationServices groups infra services.
type DeleteJobTemplateRelationServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// DeleteJobTemplateRelationUseCase removes one edge, scoped to the caller's
// workspace through the owning parent job_template (fail-closed).
type DeleteJobTemplateRelationUseCase struct {
	repositories DeleteJobTemplateRelationRepositories
	services     DeleteJobTemplateRelationServices
}

// NewDeleteJobTemplateRelationUseCase wires the use case.
func NewDeleteJobTemplateRelationUseCase(
	r DeleteJobTemplateRelationRepositories,
	s DeleteJobTemplateRelationServices,
) *DeleteJobTemplateRelationUseCase {
	return &DeleteJobTemplateRelationUseCase{repositories: r, services: s}
}

// Execute scope-checks then deletes the relation.
func (uc *DeleteJobTemplateRelationUseCase) Execute(
	ctx context.Context, req *jobtemplaterelationpb.DeleteJobTemplateRelationRequest,
) (*jobtemplaterelationpb.DeleteJobTemplateRelationResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.JobTemplateRelation, Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.GetId() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"job_template_relation.validation.id_required", "job template relation ID is required [DEFAULT]"))
	}
	if uc.repositories.JobTemplateRelation == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"job_template_relation.errors.repository_unavailable", "job template relation repository not configured [DEFAULT]"))
	}

	// Resolve the edge and confirm the caller owns it before deleting.
	resp, err := uc.repositories.JobTemplateRelation.ReadJobTemplateRelation(ctx, &jobtemplaterelationpb.ReadJobTemplateRelationRequest{
		Data: &jobtemplaterelationpb.JobTemplateRelation{Id: req.Data.GetId()},
	})
	if err != nil || resp == nil || len(resp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"job_template_relation.errors.not_found", "job template relation not found [DEFAULT]"))
	}
	// Scope BOTH ends before deleting: the caller must own the parent AND the
	// child template. A pre-existing malformed edge (parent in-workspace, child
	// foreign) is rejected rather than acted on through this scoped path
	// (red-team MED).
	wsID := contextutil.ExtractWorkspaceIDFromContext(ctx)
	if err := requireTemplateInWorkspace(ctx, uc.repositories.JobTemplate, uc.services.Translator, wsID, resp.GetData()[0].GetParentTemplateId()); err != nil {
		return nil, err
	}
	// A non-empty FOREIGN child blocks the scoped delete; an absent child carries
	// no foreign name.
	if childID := resp.GetData()[0].GetChildTemplateId(); childID != "" {
		if err := requireTemplateInWorkspace(ctx, uc.repositories.JobTemplate, uc.services.Translator, wsID, childID); err != nil {
			return nil, err
		}
	}

	return uc.repositories.JobTemplateRelation.DeleteJobTemplateRelation(ctx, req)
}

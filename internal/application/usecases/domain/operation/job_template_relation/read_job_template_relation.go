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

// ReadJobTemplateRelationRepositories groups repository dependencies.
type ReadJobTemplateRelationRepositories struct {
	JobTemplateRelation jobtemplaterelationpb.JobTemplateRelationDomainServiceServer
	JobTemplate         jobtemplatepb.JobTemplateDomainServiceServer
}

// ReadJobTemplateRelationServices groups infra services.
type ReadJobTemplateRelationServices struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ReadJobTemplateRelationUseCase reads one edge, scoped to the caller's
// workspace through the owning parent job_template (fail-closed).
type ReadJobTemplateRelationUseCase struct {
	repositories ReadJobTemplateRelationRepositories
	services     ReadJobTemplateRelationServices
}

// NewReadJobTemplateRelationUseCase wires the use case.
func NewReadJobTemplateRelationUseCase(
	r ReadJobTemplateRelationRepositories,
	s ReadJobTemplateRelationServices,
) *ReadJobTemplateRelationUseCase {
	return &ReadJobTemplateRelationUseCase{repositories: r, services: s}
}

// Execute reads and scope-checks the relation.
func (uc *ReadJobTemplateRelationUseCase) Execute(
	ctx context.Context, req *jobtemplaterelationpb.ReadJobTemplateRelationRequest,
) (*jobtemplaterelationpb.ReadJobTemplateRelationResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.JobTemplateRelation, Action: entityid.ActionRead,
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

	resp, err := uc.repositories.JobTemplateRelation.ReadJobTemplateRelation(ctx, req)
	if err != nil || resp == nil || len(resp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"job_template_relation.errors.not_found", "job template relation not found [DEFAULT]"))
	}

	// Scope BOTH ends: a pre-existing malformed edge (parent in-workspace, child
	// foreign) must not reveal the foreign child through the Spawn Graph
	// (red-team MED — read anchored only the parent).
	wsID := contextutil.ExtractWorkspaceIDFromContext(ctx)
	if err := requireTemplateInWorkspace(ctx, uc.repositories.JobTemplate, uc.services.Translator, wsID, resp.GetData()[0].GetParentTemplateId()); err != nil {
		return nil, err
	}
	// A non-empty FOREIGN child is a leak; an absent child carries no foreign name.
	if childID := resp.GetData()[0].GetChildTemplateId(); childID != "" {
		if err := requireTemplateInWorkspace(ctx, uc.repositories.JobTemplate, uc.services.Translator, wsID, childID); err != nil {
			return nil, err
		}
	}
	return resp, nil
}

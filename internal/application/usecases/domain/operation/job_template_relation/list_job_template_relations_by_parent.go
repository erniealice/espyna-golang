package job_template_relation

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	jobtemplaterelationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_relation"
)

// ListByParentRepositories groups all repository dependencies. JobTemplate is
// required to scope the (column-less) relation rows to the caller's workspace.
type ListByParentRepositories struct {
	JobTemplateRelation jobtemplaterelationpb.JobTemplateRelationDomainServiceServer
	JobTemplate         jobtemplatepb.JobTemplateDomainServiceServer
}

// ListByParentServices groups infra services.
type ListByParentServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListByParentUseCase wraps the proto-domain ListByParent RPC behind a Layer-7
// use case with auth-check. Phase 3 F7 closure — replaces the raw
// jobtemplaterelationpb.JobTemplateRelationDomainServiceServer leak that was
// previously exposed as a flat field on OperationUseCases.
type ListByParentUseCase struct {
	repositories ListByParentRepositories
	services     ListByParentServices
}

// NewListByParentUseCase wires the use case.
func NewListByParentUseCase(
	repositories ListByParentRepositories,
	services ListByParentServices,
) *ListByParentUseCase {
	return &ListByParentUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list operation.
func (uc *ListByParentUseCase) Execute(
	ctx context.Context, req *jobtemplaterelationpb.ListJobTemplateRelationsByParentRequest,
) (*jobtemplaterelationpb.ListJobTemplateRelationsByParentResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: "job_template_relation",
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"job_template_relation.validation.request_required", "request is required"))
	}
	if uc.repositories.JobTemplateRelation == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"job_template_relation.errors.repository_unavailable", "job template relation repository not configured"))
	}
	if req.GetParentTemplateId() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"job_template_relation.validation.parent_required", "parent template ID is required"))
	}

	// Fail-closed cross-workspace scope: the shared postgres adapter's
	// ListByParent has no workspace predicate (it is also the trusted internal
	// spawn path, which must not over-filter). The IDOR is closed HERE at the
	// use-case boundary — the untrusted UI/API entry point — by requiring the
	// owning parent job_template to belong to the caller's request workspace.
	wsID := contextutil.ExtractWorkspaceIDFromContext(ctx)
	if err := requireTemplateInWorkspace(ctx, uc.repositories.JobTemplate, uc.services.Translator, wsID, req.GetParentTemplateId()); err != nil {
		return nil, err
	}
	resp, err := uc.repositories.JobTemplateRelation.ListByParent(ctx, req)
	if err != nil || resp == nil {
		return resp, err
	}

	// Filter out any pre-existing malformed edge whose CHILD template is foreign,
	// so a cross-workspace child never leaks into the Spawn Graph (red-team MED —
	// list anchored only the parent). The parent is already confirmed
	// in-workspace above; the per-child workspace result is memoized so a wide
	// graph does not re-read the same child template repeatedly.
	rels := resp.GetJobTemplateRelations()
	if len(rels) > 0 {
		kept := make([]*jobtemplaterelationpb.JobTemplateRelation, 0, len(rels))
		childInWS := make(map[string]bool, len(rels))
		for _, rel := range rels {
			childID := rel.GetChildTemplateId()
			// An absent child carries no foreign name to leak (a real edge always
			// has a child); only a resolvable, in-workspace child — or no child at
			// all — survives. A non-empty FOREIGN child is dropped.
			if childID == "" {
				kept = append(kept, rel)
				continue
			}
			ok, seen := childInWS[childID]
			if !seen {
				ok = requireTemplateInWorkspace(ctx, uc.repositories.JobTemplate, uc.services.Translator, wsID, childID) == nil
				childInWS[childID] = ok
			}
			if ok {
				kept = append(kept, rel)
			}
		}
		resp.JobTemplateRelations = kept
	}
	return resp, nil
}

package job_template_relation

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	jobtemplaterelationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_relation"
)

// UpdateJobTemplateRelationRepositories groups repository dependencies.
type UpdateJobTemplateRelationRepositories struct {
	JobTemplateRelation jobtemplaterelationpb.JobTemplateRelationDomainServiceServer
	JobTemplate         jobtemplatepb.JobTemplateDomainServiceServer
}

// UpdateJobTemplateRelationServices groups infra services.
type UpdateJobTemplateRelationServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UpdateJobTemplateRelationUseCase mutates a spawn-graph edge. Relation FKs are
// mutable, so both the existing edge AND any newly-pointed templates are
// re-validated against the caller's workspace (fail-closed).
type UpdateJobTemplateRelationUseCase struct {
	repositories UpdateJobTemplateRelationRepositories
	services     UpdateJobTemplateRelationServices
}

// NewUpdateJobTemplateRelationUseCase wires the use case.
func NewUpdateJobTemplateRelationUseCase(
	r UpdateJobTemplateRelationRepositories,
	s UpdateJobTemplateRelationServices,
) *UpdateJobTemplateRelationUseCase {
	return &UpdateJobTemplateRelationUseCase{repositories: r, services: s}
}

// Execute validates and updates the relation.
func (uc *UpdateJobTemplateRelationUseCase) Execute(
	ctx context.Context, req *jobtemplaterelationpb.UpdateJobTemplateRelationRequest,
) (*jobtemplaterelationpb.UpdateJobTemplateRelationResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.JobTemplateRelation, Action: entityid.ActionUpdate,
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

	wsID := contextutil.ExtractWorkspaceIDFromContext(ctx)
	now := time.Now()
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	req.Data.DateModified = &ms
	req.Data.DateModifiedString = &s

	// The existing-edge read, both-end validation and the write all run INSIDE
	// the transaction so a mutable FK cannot be repointed at another workspace's
	// template through a TOCTOU window between the ownership check and the write
	// (red-team MED — update was previously fully non-transactional).
	writeFn := func(txCtx context.Context) (*jobtemplaterelationpb.UpdateJobTemplateRelationResponse, error) {
		// Load the existing edge and confirm the caller owns it (its parent
		// template must live in the caller's workspace) before allowing mutation.
		existing, err := uc.readRelation(txCtx, req.Data.GetId())
		if err != nil {
			return nil, err
		}
		if err := requireTemplateInWorkspace(txCtx, uc.repositories.JobTemplate, uc.services.Translator, wsID, existing.GetParentTemplateId()); err != nil {
			return nil, err
		}
		// The caller must also own the existing CHILD end: a pre-existing malformed
		// edge (parent in-workspace, child foreign) must not be mutable via this
		// scoped path. An absent existing child carries no foreign name.
		if existingChild := existing.GetChildTemplateId(); existingChild != "" {
			if err := requireTemplateInWorkspace(txCtx, uc.repositories.JobTemplate, uc.services.Translator, wsID, existingChild); err != nil {
				return nil, err
			}
		}

		// Re-validate the effective (post-update) parent/child templates: a mutable
		// FK must not be repointed at another workspace's template.
		effectiveParent := req.Data.GetParentTemplateId()
		if effectiveParent == "" {
			effectiveParent = existing.GetParentTemplateId()
		}
		effectiveChild := req.Data.GetChildTemplateId()
		if effectiveChild == "" {
			effectiveChild = existing.GetChildTemplateId()
		}
		if err := requireTemplateInWorkspace(txCtx, uc.repositories.JobTemplate, uc.services.Translator, wsID, effectiveParent); err != nil {
			return nil, err
		}
		if err := requireTemplateInWorkspace(txCtx, uc.repositories.JobTemplate, uc.services.Translator, wsID, effectiveChild); err != nil {
			return nil, err
		}

		return uc.repositories.JobTemplateRelation.UpdateJobTemplateRelation(txCtx, req)
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *jobtemplaterelationpb.UpdateJobTemplateRelationResponse
		if err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := writeFn(txCtx)
			if err != nil {
				return err
			}
			result = res
			return nil
		}); err != nil {
			return nil, err
		}
		return result, nil
	}
	return writeFn(ctx)
}

func (uc *UpdateJobTemplateRelationUseCase) readRelation(
	ctx context.Context, id string,
) (*jobtemplaterelationpb.JobTemplateRelation, error) {
	resp, err := uc.repositories.JobTemplateRelation.ReadJobTemplateRelation(ctx, &jobtemplaterelationpb.ReadJobTemplateRelationRequest{
		Data: &jobtemplaterelationpb.JobTemplateRelation{Id: id},
	})
	if err != nil || resp == nil || len(resp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"job_template_relation.errors.not_found", "job template relation not found [DEFAULT]"))
	}
	return resp.GetData()[0], nil
}

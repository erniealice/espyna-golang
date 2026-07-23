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

// CreateJobTemplateRelationRepositories groups repository dependencies. The
// JobTemplate repo is required for the fail-closed cross-workspace guard.
type CreateJobTemplateRelationRepositories struct {
	JobTemplateRelation jobtemplaterelationpb.JobTemplateRelationDomainServiceServer
	JobTemplate         jobtemplatepb.JobTemplateDomainServiceServer
}

// CreateJobTemplateRelationServices groups infra services.
type CreateJobTemplateRelationServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// CreateJobTemplateRelationUseCase creates a spawn-graph edge between two
// job_template rows, fail-closed against cross-workspace links.
type CreateJobTemplateRelationUseCase struct {
	repositories CreateJobTemplateRelationRepositories
	services     CreateJobTemplateRelationServices
}

// NewCreateJobTemplateRelationUseCase wires the use case.
func NewCreateJobTemplateRelationUseCase(
	r CreateJobTemplateRelationRepositories,
	s CreateJobTemplateRelationServices,
) *CreateJobTemplateRelationUseCase {
	return &CreateJobTemplateRelationUseCase{repositories: r, services: s}
}

// Execute validates and creates the relation.
func (uc *CreateJobTemplateRelationUseCase) Execute(
	ctx context.Context, req *jobtemplaterelationpb.CreateJobTemplateRelationRequest,
) (*jobtemplaterelationpb.CreateJobTemplateRelationResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.JobTemplateRelation, Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"job_template_relation.validation.data_required", "job template relation data is required [DEFAULT]"))
	}
	if uc.repositories.JobTemplateRelation == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"job_template_relation.errors.repository_unavailable", "job template relation repository not configured [DEFAULT]"))
	}
	data := req.Data
	if data.GetParentTemplateId() == "" || data.GetChildTemplateId() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"job_template_relation.validation.templates_required", "parent and child template IDs are required [DEFAULT]"))
	}

	// Fail-closed cross-workspace guard: BOTH the parent and child job_template
	// rows must exist and belong to the caller's request workspace. Validated
	// INSIDE the transaction so the parent/child reads and the edge write commit
	// atomically — no TOCTOU on a concurrent template workspace change (red-team
	// MED). enrich() is a pure in-memory step, so it runs before the transaction.
	wsID := contextutil.ExtractWorkspaceIDFromContext(ctx)
	uc.enrich(data)

	writeFn := func(txCtx context.Context) (*jobtemplaterelationpb.CreateJobTemplateRelationResponse, error) {
		if err := requireTemplateInWorkspace(txCtx, uc.repositories.JobTemplate, uc.services.Translator, wsID, data.GetParentTemplateId()); err != nil {
			return nil, err
		}
		if err := requireTemplateInWorkspace(txCtx, uc.repositories.JobTemplate, uc.services.Translator, wsID, data.GetChildTemplateId()); err != nil {
			return nil, err
		}
		return uc.repositories.JobTemplateRelation.CreateJobTemplateRelation(txCtx, req)
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *jobtemplaterelationpb.CreateJobTemplateRelationResponse
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

func (uc *CreateJobTemplateRelationUseCase) enrich(data *jobtemplaterelationpb.JobTemplateRelation) {
	now := time.Now()
	if data.Id == "" && uc.services.IDGenerator != nil {
		data.Id = uc.services.IDGenerator.GenerateID()
	}
	data.Active = true
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	data.DateCreated = &ms
	data.DateCreatedString = &s
	data.DateModified = &ms
	data.DateModifiedString = &s
}

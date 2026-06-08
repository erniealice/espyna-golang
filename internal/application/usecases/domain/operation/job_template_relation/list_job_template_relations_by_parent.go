package job_template_relation

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	jobtemplaterelationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_relation"
)

// ListByParentRepositories groups all repository dependencies.
type ListByParentRepositories struct {
	JobTemplateRelation jobtemplaterelationpb.JobTemplateRelationDomainServiceServer
}

// ListByParentServices groups infra services.
type ListByParentServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
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
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		"job_template_relation", entityid.ActionList); err != nil {
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
	return uc.repositories.JobTemplateRelation.ListByParent(ctx, req)
}

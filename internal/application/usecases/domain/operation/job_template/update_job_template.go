package job_template

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	jobcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_category"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
)

// UpdateJobTemplateRepositories groups all repository dependencies. JobCategory
// and Product anchor the fail-closed cross-workspace FK guards (red-team HIGH
// #2).
type UpdateJobTemplateRepositories struct {
	JobTemplate pb.JobTemplateDomainServiceServer
	JobCategory jobcategorypb.JobCategoryDomainServiceServer
	Product     productpb.ProductDomainServiceServer
}

// UpdateJobTemplateServices groups all business service dependencies
type UpdateJobTemplateServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UpdateJobTemplateUseCase handles the business logic for updating job templates
type UpdateJobTemplateUseCase struct {
	repositories UpdateJobTemplateRepositories
	services     UpdateJobTemplateServices
}

// NewUpdateJobTemplateUseCase creates a new UpdateJobTemplateUseCase
func NewUpdateJobTemplateUseCase(
	repositories UpdateJobTemplateRepositories,
	services UpdateJobTemplateServices,
) *UpdateJobTemplateUseCase {
	return &UpdateJobTemplateUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update job template operation
func (uc *UpdateJobTemplateUseCase) Execute(ctx context.Context, req *pb.UpdateJobTemplateRequest) (*pb.UpdateJobTemplateResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: "job_template",
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	uc.enrichData(req.Data)

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Fail-closed cross-workspace FK guard on the submitted (effective) FKs: a
	// mutable job_category_id / output_product_id must not be repointed at another
	// workspace's row. Empty (omitted) FKs are skipped — the persisted value was
	// already validated on the create/prior update.
	wsID := contextutil.ExtractWorkspaceIDFromContext(ctx)
	if err := requireTemplateFKsInWorkspace(ctx, uc.fkScopeRepos(), uc.services.Translator, wsID, req.Data.GetJobCategoryId(), req.Data.GetOutputProductId()); err != nil {
		return nil, err
	}

	// Call repository
	_, err := uc.repositories.JobTemplate.UpdateJobTemplate(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_template.errors.update_failed", "job template update failed [DEFAULT]"))
	}

	return &pb.UpdateJobTemplateResponse{
		Success: true,
		Data:    []*pb.JobTemplate{req.Data},
	}, nil
}

// fkScopeRepos bundles the FK-resolution repos for the cross-workspace guard.
func (uc *UpdateJobTemplateUseCase) fkScopeRepos() templateFKScopeRepos {
	return templateFKScopeRepos{
		JobCategory: uc.repositories.JobCategory,
		Product:     uc.repositories.Product,
	}
}

// validateInput validates the input request
func (uc *UpdateJobTemplateUseCase) validateInput(ctx context.Context, req *pb.UpdateJobTemplateRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_template.validation.request_required", "request is required"))
	}
	if req.Data == nil || req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_template.validation.id_required", "job template ID is required"))
	}
	return nil
}

// enrichData adds audit information for updates
func (uc *UpdateJobTemplateUseCase) enrichData(data *pb.JobTemplate) {
	now := time.Now()
	dm := now.UnixMilli()
	dms := now.Format(time.RFC3339)
	data.DateModified = &dm
	data.DateModifiedString = &dms
}

// validateBusinessRules enforces business constraints
func (uc *UpdateJobTemplateUseCase) validateBusinessRules(ctx context.Context, data *pb.JobTemplate) error {
	if strings.TrimSpace(data.Name) == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_template.validation.name_required", "job template name is required"))
	}
	if len(data.Name) > 200 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_template.validation.name_too_long", "job template name cannot exceed 200 characters"))
	}
	return nil
}

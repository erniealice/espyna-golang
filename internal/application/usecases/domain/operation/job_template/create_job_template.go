package job_template

import (
	"context"
	"errors"
	"fmt"
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

// CreateJobTemplateRepositories groups all repository dependencies. JobCategory
// and Product anchor the fail-closed cross-workspace FK guards (red-team HIGH
// #2 — job_category_id / output_product_id posted directly from the drawer).
type CreateJobTemplateRepositories struct {
	JobTemplate pb.JobTemplateDomainServiceServer
	JobCategory jobcategorypb.JobCategoryDomainServiceServer
	Product     productpb.ProductDomainServiceServer
}

// CreateJobTemplateServices groups all business service dependencies
type CreateJobTemplateServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// CreateJobTemplateUseCase handles the business logic for creating job templates
type CreateJobTemplateUseCase struct {
	repositories CreateJobTemplateRepositories
	services     CreateJobTemplateServices
}

// NewCreateJobTemplateUseCase creates use case with grouped dependencies
func NewCreateJobTemplateUseCase(
	repositories CreateJobTemplateRepositories,
	services CreateJobTemplateServices,
) *CreateJobTemplateUseCase {
	return &CreateJobTemplateUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create job template operation
func (uc *CreateJobTemplateUseCase) Execute(ctx context.Context, req *pb.CreateJobTemplateRequest) (*pb.CreateJobTemplateResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: "job_template",
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}

	// Check for transaction support and route accordingly
	if uc.services.Transactor != nil {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

// executeWithTransaction performs the operation within a transaction
func (uc *CreateJobTemplateUseCase) executeWithTransaction(ctx context.Context, req *pb.CreateJobTemplateRequest) (*pb.CreateJobTemplateResponse, error) {
	var result *pb.CreateJobTemplateResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore performs the core create operation
func (uc *CreateJobTemplateUseCase) executeCore(ctx context.Context, req *pb.CreateJobTemplateRequest) (*pb.CreateJobTemplateResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichData(req.Data); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Fail-closed cross-workspace FK guard: every non-empty FK must resolve to
	// the caller's request workspace.
	wsID := contextutil.ExtractWorkspaceIDFromContext(ctx)
	if err := requireTemplateFKsInWorkspace(ctx, uc.fkScopeRepos(), uc.services.Translator, wsID, req.Data.GetJobCategoryId(), req.Data.GetOutputProductId()); err != nil {
		return nil, err
	}

	// Call repository
	response, err := uc.repositories.JobTemplate.CreateJobTemplate(ctx, req)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// fkScopeRepos bundles the FK-resolution repos for the cross-workspace guard.
func (uc *CreateJobTemplateUseCase) fkScopeRepos() templateFKScopeRepos {
	return templateFKScopeRepos{
		JobCategory: uc.repositories.JobCategory,
		Product:     uc.repositories.Product,
	}
}

// validateInput validates the input request
func (uc *CreateJobTemplateUseCase) validateInput(ctx context.Context, req *pb.CreateJobTemplateRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_template.validation.request_required", "request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_template.validation.data_required", "job template data is required [DEFAULT]"))
	}
	return nil
}

// enrichData adds generated fields and audit information
func (uc *CreateJobTemplateUseCase) enrichData(data *pb.JobTemplate) error {
	now := time.Now()

	// Always generate a new ID, overriding any passed ID
	if uc.services.IDGenerator != nil {
		data.Id = uc.services.IDGenerator.GenerateID()
	} else {
		data.Id = fmt.Sprintf("job_template-%d", now.UnixNano())
	}

	// Set audit fields
	dc := now.UnixMilli()
	dcs := now.Format(time.RFC3339)
	dm := now.UnixMilli()
	dms := now.Format(time.RFC3339)
	data.DateCreated = &dc
	data.DateCreatedString = &dcs
	data.DateModified = &dm
	data.DateModifiedString = &dms
	data.Active = true

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateJobTemplateUseCase) validateBusinessRules(ctx context.Context, data *pb.JobTemplate) error {
	if strings.TrimSpace(data.Name) == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_template.validation.name_required", "job template name is required [DEFAULT]"))
	}
	if len(data.Name) > 200 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_template.validation.name_too_long", "job template name cannot exceed 200 characters [DEFAULT]"))
	}
	return nil
}

package job_template

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
)

// CreateJobTemplateRepositories groups all repository dependencies
type CreateJobTemplateRepositories struct {
	JobTemplate pb.JobTemplateDomainServiceServer
}

// CreateJobTemplateServices groups all business service dependencies
type CreateJobTemplateServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"job_template", ports.ActionCreate); err != nil {
		return nil, err
	}

	// Check for transaction support and route accordingly
	if uc.services.TransactionService != nil {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

// executeWithTransaction performs the operation within a transaction
func (uc *CreateJobTemplateUseCase) executeWithTransaction(ctx context.Context, req *pb.CreateJobTemplateRequest) (*pb.CreateJobTemplateResponse, error) {
	var result *pb.CreateJobTemplateResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
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

	// Call repository
	response, err := uc.repositories.JobTemplate.CreateJobTemplate(ctx, req)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// validateInput validates the input request
func (uc *CreateJobTemplateUseCase) validateInput(ctx context.Context, req *pb.CreateJobTemplateRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template.validation.request_required", "request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template.validation.data_required", "job template data is required [DEFAULT]"))
	}
	return nil
}

// enrichData adds generated fields and audit information
func (uc *CreateJobTemplateUseCase) enrichData(data *pb.JobTemplate) error {
	now := time.Now()

	// Always generate a new ID, overriding any passed ID
	if uc.services.IDService != nil {
		data.Id = uc.services.IDService.GenerateID()
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
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template.validation.name_required", "job template name is required [DEFAULT]"))
	}
	if len(data.Name) > 200 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template.validation.name_too_long", "job template name cannot exceed 200 characters [DEFAULT]"))
	}
	return nil
}

package job_template

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
)

// DeleteJobTemplateRepositories groups all repository dependencies
type DeleteJobTemplateRepositories struct {
	JobTemplate pb.JobTemplateDomainServiceServer
}

// DeleteJobTemplateServices groups all business service dependencies
type DeleteJobTemplateServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteJobTemplateUseCase handles the business logic for deleting job templates
type DeleteJobTemplateUseCase struct {
	repositories DeleteJobTemplateRepositories
	services     DeleteJobTemplateServices
}

// NewDeleteJobTemplateUseCase creates a new DeleteJobTemplateUseCase
func NewDeleteJobTemplateUseCase(
	repositories DeleteJobTemplateRepositories,
	services DeleteJobTemplateServices,
) *DeleteJobTemplateUseCase {
	return &DeleteJobTemplateUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete job template operation
func (uc *DeleteJobTemplateUseCase) Execute(ctx context.Context, req *pb.DeleteJobTemplateRequest) (*pb.DeleteJobTemplateResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"job_template", ports.ActionDelete); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	result, err := uc.repositories.JobTemplate.DeleteJobTemplate(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template.errors.deletion_failed", "job template deletion failed [DEFAULT]"))
	}

	return result, nil
}

// validateInput validates the input request
func (uc *DeleteJobTemplateUseCase) validateInput(ctx context.Context, req *pb.DeleteJobTemplateRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template.validation.request_required", "request is required"))
	}
	if req.Data == nil || req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template.validation.id_required", "job template ID is required"))
	}
	return nil
}

package job_template

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
)

// ListJobTemplatesRepositories groups all repository dependencies
type ListJobTemplatesRepositories struct {
	JobTemplate pb.JobTemplateDomainServiceServer
}

// ListJobTemplatesServices groups all business service dependencies
type ListJobTemplatesServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListJobTemplatesUseCase handles the business logic for listing job templates
type ListJobTemplatesUseCase struct {
	repositories ListJobTemplatesRepositories
	services     ListJobTemplatesServices
}

// NewListJobTemplatesUseCase creates a new ListJobTemplatesUseCase
func NewListJobTemplatesUseCase(
	repositories ListJobTemplatesRepositories,
	services ListJobTemplatesServices,
) *ListJobTemplatesUseCase {
	return &ListJobTemplatesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list job templates operation
func (uc *ListJobTemplatesUseCase) Execute(ctx context.Context, req *pb.ListJobTemplatesRequest) (*pb.ListJobTemplatesResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"job_template", ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template.validation.request_required", "request is required"))
	}

	// Call repository
	result, err := uc.repositories.JobTemplate.ListJobTemplates(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template.errors.list_failed", "job template listing failed [DEFAULT]"))
	}

	return result, nil
}

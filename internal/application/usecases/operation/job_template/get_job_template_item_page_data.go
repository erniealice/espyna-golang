package job_template

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
)

// GetJobTemplateItemPageDataRepositories groups all repository dependencies
type GetJobTemplateItemPageDataRepositories struct {
	JobTemplate pb.JobTemplateDomainServiceServer
}

// GetJobTemplateItemPageDataServices groups all business service dependencies
type GetJobTemplateItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetJobTemplateItemPageDataUseCase handles the business logic for getting job template item page data
type GetJobTemplateItemPageDataUseCase struct {
	repositories GetJobTemplateItemPageDataRepositories
	services     GetJobTemplateItemPageDataServices
}

// NewGetJobTemplateItemPageDataUseCase creates use case with grouped dependencies
func NewGetJobTemplateItemPageDataUseCase(
	repositories GetJobTemplateItemPageDataRepositories,
	services GetJobTemplateItemPageDataServices,
) *GetJobTemplateItemPageDataUseCase {
	return &GetJobTemplateItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get job template item page data operation
func (uc *GetJobTemplateItemPageDataUseCase) Execute(ctx context.Context, req *pb.GetJobTemplateItemPageDataRequest) (*pb.GetJobTemplateItemPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"job_template", ports.ActionRead); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template.validation.request_required", "Request is required for job template item page data"))
	}

	if req.JobTemplateId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template.validation.id_required", "Job template ID is required"))
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes within a transaction
func (uc *GetJobTemplateItemPageDataUseCase) executeWithTransaction(ctx context.Context, req *pb.GetJobTemplateItemPageDataRequest) (*pb.GetJobTemplateItemPageDataResponse, error) {
	var result *pb.GetJobTemplateItemPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "job_template.errors.get_item_page_data_failed", "[ERR-DEFAULT] Failed to load job template details")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic
func (uc *GetJobTemplateItemPageDataUseCase) executeCore(ctx context.Context, req *pb.GetJobTemplateItemPageDataRequest) (*pb.GetJobTemplateItemPageDataResponse, error) {
	return uc.repositories.JobTemplate.GetJobTemplateItemPageData(ctx, req)
}

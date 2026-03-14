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

// GetJobTemplateListPageDataRepositories groups all repository dependencies
type GetJobTemplateListPageDataRepositories struct {
	JobTemplate pb.JobTemplateDomainServiceServer
}

// GetJobTemplateListPageDataServices groups all business service dependencies
type GetJobTemplateListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetJobTemplateListPageDataUseCase handles the business logic for getting job template list page data
type GetJobTemplateListPageDataUseCase struct {
	repositories GetJobTemplateListPageDataRepositories
	services     GetJobTemplateListPageDataServices
}

// NewGetJobTemplateListPageDataUseCase creates use case with grouped dependencies
func NewGetJobTemplateListPageDataUseCase(
	repositories GetJobTemplateListPageDataRepositories,
	services GetJobTemplateListPageDataServices,
) *GetJobTemplateListPageDataUseCase {
	return &GetJobTemplateListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get job template list page data operation
func (uc *GetJobTemplateListPageDataUseCase) Execute(ctx context.Context, req *pb.GetJobTemplateListPageDataRequest) (*pb.GetJobTemplateListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"job_template", ports.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template.validation.request_required", "Request is required for job template list page data"))
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes within a transaction
func (uc *GetJobTemplateListPageDataUseCase) executeWithTransaction(ctx context.Context, req *pb.GetJobTemplateListPageDataRequest) (*pb.GetJobTemplateListPageDataResponse, error) {
	var result *pb.GetJobTemplateListPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "job_template.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load job template list")
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
func (uc *GetJobTemplateListPageDataUseCase) executeCore(ctx context.Context, req *pb.GetJobTemplateListPageDataRequest) (*pb.GetJobTemplateListPageDataResponse, error) {
	return uc.repositories.JobTemplate.GetJobTemplateListPageData(ctx, req)
}

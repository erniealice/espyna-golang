package template_task_criteria

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/template_task_criteria"
)

type GetTemplateTaskCriteriaListPageDataRepositories struct {
	TemplateTaskCriteria pb.TemplateTaskCriteriaDomainServiceServer
}

type GetTemplateTaskCriteriaListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetTemplateTaskCriteriaListPageDataUseCase handles the business logic for getting template task criteria list page data
type GetTemplateTaskCriteriaListPageDataUseCase struct {
	repositories GetTemplateTaskCriteriaListPageDataRepositories
	services     GetTemplateTaskCriteriaListPageDataServices
}

// NewGetTemplateTaskCriteriaListPageDataUseCase creates a new GetTemplateTaskCriteriaListPageDataUseCase
func NewGetTemplateTaskCriteriaListPageDataUseCase(
	repositories GetTemplateTaskCriteriaListPageDataRepositories,
	services GetTemplateTaskCriteriaListPageDataServices,
) *GetTemplateTaskCriteriaListPageDataUseCase {
	return &GetTemplateTaskCriteriaListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get template task criteria list page data operation
func (uc *GetTemplateTaskCriteriaListPageDataUseCase) Execute(
	ctx context.Context,
	req *pb.GetTemplateTaskCriteriaListPageDataRequest,
) (*pb.GetTemplateTaskCriteriaListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityTemplateTaskCriteria, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes list page data retrieval within a transaction
func (uc *GetTemplateTaskCriteriaListPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *pb.GetTemplateTaskCriteriaListPageDataRequest,
) (*pb.GetTemplateTaskCriteriaListPageDataResponse, error) {
	var result *pb.GetTemplateTaskCriteriaListPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"template_task_criteria.errors.list_page_data_failed",
				"template task criteria list page data retrieval failed: %w",
			), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for getting template task criteria list page data
func (uc *GetTemplateTaskCriteriaListPageDataUseCase) executeCore(
	ctx context.Context,
	req *pb.GetTemplateTaskCriteriaListPageDataRequest,
) (*pb.GetTemplateTaskCriteriaListPageDataResponse, error) {
	resp, err := uc.repositories.TemplateTaskCriteria.GetTemplateTaskCriteriaListPageData(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"template_task_criteria.errors.list_page_data_failed",
			"failed to retrieve template task criteria list page data: %w",
		), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *GetTemplateTaskCriteriaListPageDataUseCase) validateInput(
	ctx context.Context,
	req *pb.GetTemplateTaskCriteriaListPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"template_task_criteria.validation.request_required",
			"request is required",
		))
	}

	if req.Pagination != nil {
		if err := uc.validatePagination(ctx, req.Pagination); err != nil {
			return err
		}
	}

	return nil
}

// validatePagination validates pagination parameters
func (uc *GetTemplateTaskCriteriaListPageDataUseCase) validatePagination(
	ctx context.Context,
	pagination *commonpb.PaginationRequest,
) error {
	if pagination.Limit < 0 || pagination.Limit > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"template_task_criteria.validation.invalid_limit",
			"pagination limit must be between 1 and 100",
		))
	}

	return nil
}

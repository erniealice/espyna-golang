package phase_outcome_summary

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/phase_outcome_summary"
)

type GetPhaseOutcomeSummaryListPageDataRepositories struct {
	PhaseOutcomeSummary pb.PhaseOutcomeSummaryDomainServiceServer
}

type GetPhaseOutcomeSummaryListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetPhaseOutcomeSummaryListPageDataUseCase handles the business logic for getting phase outcome summary list page data
type GetPhaseOutcomeSummaryListPageDataUseCase struct {
	repositories GetPhaseOutcomeSummaryListPageDataRepositories
	services     GetPhaseOutcomeSummaryListPageDataServices
}

// NewGetPhaseOutcomeSummaryListPageDataUseCase creates a new GetPhaseOutcomeSummaryListPageDataUseCase
func NewGetPhaseOutcomeSummaryListPageDataUseCase(
	repositories GetPhaseOutcomeSummaryListPageDataRepositories,
	services GetPhaseOutcomeSummaryListPageDataServices,
) *GetPhaseOutcomeSummaryListPageDataUseCase {
	return &GetPhaseOutcomeSummaryListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get phase outcome summary list page data operation
func (uc *GetPhaseOutcomeSummaryListPageDataUseCase) Execute(
	ctx context.Context,
	req *pb.GetPhaseOutcomeSummaryListPageDataRequest,
) (*pb.GetPhaseOutcomeSummaryListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.PhaseOutcomeSummary, entityid.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes list page data retrieval within a transaction
func (uc *GetPhaseOutcomeSummaryListPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *pb.GetPhaseOutcomeSummaryListPageDataRequest,
) (*pb.GetPhaseOutcomeSummaryListPageDataResponse, error) {
	var result *pb.GetPhaseOutcomeSummaryListPageDataResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.Translator,
				"phase_outcome_summary.errors.list_page_data_failed",
				"phase outcome summary list page data retrieval failed: %w",
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

// executeCore contains the core business logic for getting phase outcome summary list page data
func (uc *GetPhaseOutcomeSummaryListPageDataUseCase) executeCore(
	ctx context.Context,
	req *pb.GetPhaseOutcomeSummaryListPageDataRequest,
) (*pb.GetPhaseOutcomeSummaryListPageDataResponse, error) {
	resp, err := uc.repositories.PhaseOutcomeSummary.GetPhaseOutcomeSummaryListPageData(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"phase_outcome_summary.errors.list_page_data_failed",
			"failed to retrieve phase outcome summary list page data: %w",
		), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *GetPhaseOutcomeSummaryListPageDataUseCase) validateInput(
	ctx context.Context,
	req *pb.GetPhaseOutcomeSummaryListPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"phase_outcome_summary.validation.request_required",
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
func (uc *GetPhaseOutcomeSummaryListPageDataUseCase) validatePagination(
	ctx context.Context,
	pagination *commonpb.PaginationRequest,
) error {
	if pagination.Limit < 0 || pagination.Limit > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"phase_outcome_summary.validation.invalid_limit",
			"pagination limit must be between 1 and 100",
		))
	}

	return nil
}

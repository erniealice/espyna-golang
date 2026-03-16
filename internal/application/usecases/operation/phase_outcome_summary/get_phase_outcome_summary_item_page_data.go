package phase_outcome_summary

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/phase_outcome_summary"
)

type GetPhaseOutcomeSummaryItemPageDataRepositories struct {
	PhaseOutcomeSummary pb.PhaseOutcomeSummaryDomainServiceServer
}

type GetPhaseOutcomeSummaryItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetPhaseOutcomeSummaryItemPageDataUseCase handles the business logic for getting phase outcome summary item page data
type GetPhaseOutcomeSummaryItemPageDataUseCase struct {
	repositories GetPhaseOutcomeSummaryItemPageDataRepositories
	services     GetPhaseOutcomeSummaryItemPageDataServices
}

// NewGetPhaseOutcomeSummaryItemPageDataUseCase creates a new GetPhaseOutcomeSummaryItemPageDataUseCase
func NewGetPhaseOutcomeSummaryItemPageDataUseCase(
	repositories GetPhaseOutcomeSummaryItemPageDataRepositories,
	services GetPhaseOutcomeSummaryItemPageDataServices,
) *GetPhaseOutcomeSummaryItemPageDataUseCase {
	return &GetPhaseOutcomeSummaryItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get phase outcome summary item page data operation
func (uc *GetPhaseOutcomeSummaryItemPageDataUseCase) Execute(
	ctx context.Context,
	req *pb.GetPhaseOutcomeSummaryItemPageDataRequest,
) (*pb.GetPhaseOutcomeSummaryItemPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPhaseOutcomeSummary, ports.ActionList); err != nil {
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

// executeWithTransaction executes item page data retrieval within a transaction
func (uc *GetPhaseOutcomeSummaryItemPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *pb.GetPhaseOutcomeSummaryItemPageDataRequest,
) (*pb.GetPhaseOutcomeSummaryItemPageDataResponse, error) {
	var result *pb.GetPhaseOutcomeSummaryItemPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"phase_outcome_summary.errors.item_page_data_failed",
				"phase outcome summary item page data retrieval failed: %w",
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

// executeCore contains the core business logic for getting phase outcome summary item page data
func (uc *GetPhaseOutcomeSummaryItemPageDataUseCase) executeCore(
	ctx context.Context,
	req *pb.GetPhaseOutcomeSummaryItemPageDataRequest,
) (*pb.GetPhaseOutcomeSummaryItemPageDataResponse, error) {
	readReq := &pb.ReadPhaseOutcomeSummaryRequest{
		Data: &pb.PhaseOutcomeSummary{
			Id: req.PhaseOutcomeSummaryId,
		},
	}

	readResp, err := uc.repositories.PhaseOutcomeSummary.ReadPhaseOutcomeSummary(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"phase_outcome_summary.errors.read_failed",
			"failed to retrieve phase outcome summary: %w",
		), err)
	}

	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"phase_outcome_summary.errors.not_found",
			"phase outcome summary not found",
		))
	}

	item := readResp.Data[0]

	return &pb.GetPhaseOutcomeSummaryItemPageDataResponse{
		PhaseOutcomeSummary: item,
		Success:             true,
	}, nil
}

// validateInput validates the input request
func (uc *GetPhaseOutcomeSummaryItemPageDataUseCase) validateInput(
	ctx context.Context,
	req *pb.GetPhaseOutcomeSummaryItemPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"phase_outcome_summary.validation.request_required",
			"request is required",
		))
	}

	if req.PhaseOutcomeSummaryId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"phase_outcome_summary.validation.id_required",
			"phase outcome summary ID is required",
		))
	}

	return nil
}

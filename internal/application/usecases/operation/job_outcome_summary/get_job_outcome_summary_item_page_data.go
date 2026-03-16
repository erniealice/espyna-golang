package job_outcome_summary

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary"
)

type GetJobOutcomeSummaryItemPageDataRepositories struct {
	JobOutcomeSummary pb.JobOutcomeSummaryDomainServiceServer
}

type GetJobOutcomeSummaryItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetJobOutcomeSummaryItemPageDataUseCase handles the business logic for getting job outcome summary item page data
type GetJobOutcomeSummaryItemPageDataUseCase struct {
	repositories GetJobOutcomeSummaryItemPageDataRepositories
	services     GetJobOutcomeSummaryItemPageDataServices
}

// NewGetJobOutcomeSummaryItemPageDataUseCase creates a new GetJobOutcomeSummaryItemPageDataUseCase
func NewGetJobOutcomeSummaryItemPageDataUseCase(
	repositories GetJobOutcomeSummaryItemPageDataRepositories,
	services GetJobOutcomeSummaryItemPageDataServices,
) *GetJobOutcomeSummaryItemPageDataUseCase {
	return &GetJobOutcomeSummaryItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get job outcome summary item page data operation
func (uc *GetJobOutcomeSummaryItemPageDataUseCase) Execute(
	ctx context.Context,
	req *pb.GetJobOutcomeSummaryItemPageDataRequest,
) (*pb.GetJobOutcomeSummaryItemPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityJobOutcomeSummary, ports.ActionList); err != nil {
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
func (uc *GetJobOutcomeSummaryItemPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *pb.GetJobOutcomeSummaryItemPageDataRequest,
) (*pb.GetJobOutcomeSummaryItemPageDataResponse, error) {
	var result *pb.GetJobOutcomeSummaryItemPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"job_outcome_summary.errors.item_page_data_failed",
				"job outcome summary item page data retrieval failed: %w",
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

// executeCore contains the core business logic for getting job outcome summary item page data
func (uc *GetJobOutcomeSummaryItemPageDataUseCase) executeCore(
	ctx context.Context,
	req *pb.GetJobOutcomeSummaryItemPageDataRequest,
) (*pb.GetJobOutcomeSummaryItemPageDataResponse, error) {
	readReq := &pb.ReadJobOutcomeSummaryRequest{
		Data: &pb.JobOutcomeSummary{
			Id: req.JobOutcomeSummaryId,
		},
	}

	readResp, err := uc.repositories.JobOutcomeSummary.ReadJobOutcomeSummary(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"job_outcome_summary.errors.read_failed",
			"failed to retrieve job outcome summary: %w",
		), err)
	}

	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"job_outcome_summary.errors.not_found",
			"job outcome summary not found",
		))
	}

	item := readResp.Data[0]

	return &pb.GetJobOutcomeSummaryItemPageDataResponse{
		JobOutcomeSummary: item,
		Success:           true,
	}, nil
}

// validateInput validates the input request
func (uc *GetJobOutcomeSummaryItemPageDataUseCase) validateInput(
	ctx context.Context,
	req *pb.GetJobOutcomeSummaryItemPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"job_outcome_summary.validation.request_required",
			"request is required",
		))
	}

	if req.JobOutcomeSummaryId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"job_outcome_summary.validation.id_required",
			"job outcome summary ID is required",
		))
	}

	return nil
}

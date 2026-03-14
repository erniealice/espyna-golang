package job_template_phase

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
)

type GetJobTemplatePhaseItemPageDataRepositories struct {
	JobTemplatePhase pb.JobTemplatePhaseDomainServiceServer
}

type GetJobTemplatePhaseItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetJobTemplatePhaseItemPageDataUseCase handles the business logic for getting job template phase item page data
type GetJobTemplatePhaseItemPageDataUseCase struct {
	repositories GetJobTemplatePhaseItemPageDataRepositories
	services     GetJobTemplatePhaseItemPageDataServices
}

// NewGetJobTemplatePhaseItemPageDataUseCase creates a new GetJobTemplatePhaseItemPageDataUseCase
func NewGetJobTemplatePhaseItemPageDataUseCase(
	repositories GetJobTemplatePhaseItemPageDataRepositories,
	services GetJobTemplatePhaseItemPageDataServices,
) *GetJobTemplatePhaseItemPageDataUseCase {
	return &GetJobTemplatePhaseItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get job template phase item page data operation
func (uc *GetJobTemplatePhaseItemPageDataUseCase) Execute(
	ctx context.Context,
	req *pb.GetJobTemplatePhaseItemPageDataRequest,
) (*pb.GetJobTemplatePhaseItemPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityJobTemplatePhase, ports.ActionList); err != nil {
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
func (uc *GetJobTemplatePhaseItemPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *pb.GetJobTemplatePhaseItemPageDataRequest,
) (*pb.GetJobTemplatePhaseItemPageDataResponse, error) {
	var result *pb.GetJobTemplatePhaseItemPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"job_template_phase.errors.item_page_data_failed",
				"job template phase item page data retrieval failed: %w",
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

// executeCore contains the core business logic for getting job template phase item page data
func (uc *GetJobTemplatePhaseItemPageDataUseCase) executeCore(
	ctx context.Context,
	req *pb.GetJobTemplatePhaseItemPageDataRequest,
) (*pb.GetJobTemplatePhaseItemPageDataResponse, error) {
	// Retrieve the entity via Read
	readReq := &pb.ReadJobTemplatePhaseRequest{
		Data: &pb.JobTemplatePhase{
			Id: req.JobTemplatePhaseId,
		},
	}

	readResp, err := uc.repositories.JobTemplatePhase.ReadJobTemplatePhase(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"job_template_phase.errors.read_failed",
			"failed to retrieve job template phase: %w",
		), err)
	}

	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"job_template_phase.errors.not_found",
			"job template phase not found",
		))
	}

	phase := readResp.Data[0]

	return &pb.GetJobTemplatePhaseItemPageDataResponse{
		JobTemplatePhase: phase,
		Success:          true,
	}, nil
}

// validateInput validates the input request
func (uc *GetJobTemplatePhaseItemPageDataUseCase) validateInput(
	ctx context.Context,
	req *pb.GetJobTemplatePhaseItemPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"job_template_phase.validation.request_required",
			"request is required",
		))
	}

	if req.JobTemplatePhaseId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"job_template_phase.validation.id_required",
			"job template phase ID is required",
		))
	}

	return nil
}

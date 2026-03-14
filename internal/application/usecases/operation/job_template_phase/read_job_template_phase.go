package job_template_phase

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
)

type ReadJobTemplatePhaseRepositories struct {
	JobTemplatePhase pb.JobTemplatePhaseDomainServiceServer
}

type ReadJobTemplatePhaseServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadJobTemplatePhaseUseCase handles the business logic for reading job template phases
type ReadJobTemplatePhaseUseCase struct {
	repositories ReadJobTemplatePhaseRepositories
	services     ReadJobTemplatePhaseServices
}

// NewReadJobTemplatePhaseUseCase creates a new ReadJobTemplatePhaseUseCase
func NewReadJobTemplatePhaseUseCase(
	repositories ReadJobTemplatePhaseRepositories,
	services ReadJobTemplatePhaseServices,
) *ReadJobTemplatePhaseUseCase {
	return &ReadJobTemplatePhaseUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read job template phase operation
func (uc *ReadJobTemplatePhaseUseCase) Execute(ctx context.Context, req *pb.ReadJobTemplatePhaseRequest) (*pb.ReadJobTemplatePhaseResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityJobTemplatePhase, ports.ActionRead); err != nil {
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

// executeWithTransaction executes reading within a transaction
func (uc *ReadJobTemplatePhaseUseCase) executeWithTransaction(ctx context.Context, req *pb.ReadJobTemplatePhaseRequest) (*pb.ReadJobTemplatePhaseResponse, error) {
	var result *pb.ReadJobTemplatePhaseResponse

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

// executeCore contains the core business logic for reading a job template phase
func (uc *ReadJobTemplatePhaseUseCase) executeCore(ctx context.Context, req *pb.ReadJobTemplatePhaseRequest) (*pb.ReadJobTemplatePhaseResponse, error) {
	resp, err := uc.repositories.JobTemplatePhase.ReadJobTemplatePhase(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template_phase.errors.not_found", "[ERR-DEFAULT] Job template phase not found"))
	}
	if resp == nil || len(resp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template_phase.errors.not_found", "[ERR-DEFAULT] Job template phase not found"))
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ReadJobTemplatePhaseUseCase) validateInput(ctx context.Context, req *pb.ReadJobTemplatePhaseRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template_phase.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template_phase.validation.data_required", "[ERR-DEFAULT] Job template phase data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template_phase.validation.id_required", "[ERR-DEFAULT] Job template phase ID is required"))
	}
	return nil
}

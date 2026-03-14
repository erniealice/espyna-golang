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

type ListJobTemplatePhasesRepositories struct {
	JobTemplatePhase pb.JobTemplatePhaseDomainServiceServer
}

type ListJobTemplatePhasesServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListJobTemplatePhasesUseCase handles the business logic for listing job template phases
type ListJobTemplatePhasesUseCase struct {
	repositories ListJobTemplatePhasesRepositories
	services     ListJobTemplatePhasesServices
}

// NewListJobTemplatePhasesUseCase creates a new ListJobTemplatePhasesUseCase
func NewListJobTemplatePhasesUseCase(
	repositories ListJobTemplatePhasesRepositories,
	services ListJobTemplatePhasesServices,
) *ListJobTemplatePhasesUseCase {
	return &ListJobTemplatePhasesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list job template phases operation
func (uc *ListJobTemplatePhasesUseCase) Execute(ctx context.Context, req *pb.ListJobTemplatePhasesRequest) (*pb.ListJobTemplatePhasesResponse, error) {
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

// executeWithTransaction executes listing within a transaction
func (uc *ListJobTemplatePhasesUseCase) executeWithTransaction(ctx context.Context, req *pb.ListJobTemplatePhasesRequest) (*pb.ListJobTemplatePhasesResponse, error) {
	var result *pb.ListJobTemplatePhasesResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "job_template_phase.errors.list_failed", "job template phase listing failed: %w"), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for listing job template phases
func (uc *ListJobTemplatePhasesUseCase) executeCore(ctx context.Context, req *pb.ListJobTemplatePhasesRequest) (*pb.ListJobTemplatePhasesResponse, error) {
	resp, err := uc.repositories.JobTemplatePhase.ListJobTemplatePhases(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template_phase.errors.list_failed", "job template phase listing failed: %w"), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListJobTemplatePhasesUseCase) validateInput(ctx context.Context, req *pb.ListJobTemplatePhasesRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template_phase.validation.request_required", "request is required"))
	}

	return nil
}

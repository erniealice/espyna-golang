package outcome_criteria

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
)

type GetCurrentPublishedRepositories struct {
	OutcomeCriteria pb.OutcomeCriteriaDomainServiceServer
}

type GetCurrentPublishedServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetCurrentPublishedUseCase handles the business logic for getting the current published outcome criteria
type GetCurrentPublishedUseCase struct {
	repositories GetCurrentPublishedRepositories
	services     GetCurrentPublishedServices
}

// NewGetCurrentPublishedUseCase creates a new GetCurrentPublishedUseCase
func NewGetCurrentPublishedUseCase(
	repositories GetCurrentPublishedRepositories,
	services GetCurrentPublishedServices,
) *GetCurrentPublishedUseCase {
	return &GetCurrentPublishedUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get current published operation
func (uc *GetCurrentPublishedUseCase) Execute(ctx context.Context, req *pb.GetCurrentPublishedOutcomeCriteriaRequest) (*pb.GetCurrentPublishedOutcomeCriteriaResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.OutcomeCriteria, entityid.ActionRead); err != nil {
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

// executeWithTransaction executes get current published within a transaction
func (uc *GetCurrentPublishedUseCase) executeWithTransaction(ctx context.Context, req *pb.GetCurrentPublishedOutcomeCriteriaRequest) (*pb.GetCurrentPublishedOutcomeCriteriaResponse, error) {
	var result *pb.GetCurrentPublishedOutcomeCriteriaResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "outcome_criteria.errors.get_current_published_failed", "get current published outcome criteria failed: %w"), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for getting the current published outcome criteria
func (uc *GetCurrentPublishedUseCase) executeCore(ctx context.Context, req *pb.GetCurrentPublishedOutcomeCriteriaRequest) (*pb.GetCurrentPublishedOutcomeCriteriaResponse, error) {
	resp, err := uc.repositories.OutcomeCriteria.GetCurrentPublished(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.errors.get_current_published_failed", "failed to get current published outcome criteria: %w"), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *GetCurrentPublishedUseCase) validateInput(ctx context.Context, req *pb.GetCurrentPublishedOutcomeCriteriaRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.validation.request_required", "request is required"))
	}

	return nil
}

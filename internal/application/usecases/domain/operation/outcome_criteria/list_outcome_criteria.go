package outcome_criteria

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
)

type ListOutcomeCriteriaRepositories struct {
	OutcomeCriteria pb.OutcomeCriteriaDomainServiceServer
}

type ListOutcomeCriteriaServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListOutcomeCriteriaUseCase handles the business logic for listing outcome criteria
type ListOutcomeCriteriaUseCase struct {
	repositories ListOutcomeCriteriaRepositories
	services     ListOutcomeCriteriaServices
}

// NewListOutcomeCriteriaUseCase creates a new ListOutcomeCriteriaUseCase
func NewListOutcomeCriteriaUseCase(
	repositories ListOutcomeCriteriaRepositories,
	services ListOutcomeCriteriaServices,
) *ListOutcomeCriteriaUseCase {
	return &ListOutcomeCriteriaUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list outcome criteria operation
func (uc *ListOutcomeCriteriaUseCase) Execute(ctx context.Context, req *pb.ListOutcomeCriteriasRequest) (*pb.ListOutcomeCriteriasResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityOutcomeCriteria, ports.ActionList); err != nil {
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
func (uc *ListOutcomeCriteriaUseCase) executeWithTransaction(ctx context.Context, req *pb.ListOutcomeCriteriasRequest) (*pb.ListOutcomeCriteriasResponse, error) {
	var result *pb.ListOutcomeCriteriasResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "outcome_criteria.errors.list_failed", "outcome criteria listing failed: %w"), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for listing outcome criteria
func (uc *ListOutcomeCriteriaUseCase) executeCore(ctx context.Context, req *pb.ListOutcomeCriteriasRequest) (*pb.ListOutcomeCriteriasResponse, error) {
	resp, err := uc.repositories.OutcomeCriteria.ListOutcomeCriterias(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "outcome_criteria.errors.list_failed", "outcome criteria listing failed: %w"), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListOutcomeCriteriaUseCase) validateInput(ctx context.Context, req *pb.ListOutcomeCriteriasRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "outcome_criteria.validation.request_required", "request is required"))
	}

	return nil
}

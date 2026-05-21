package outcome_criteria

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
)

type ListByScopeRepositories struct {
	OutcomeCriteria pb.OutcomeCriteriaDomainServiceServer
}

type ListByScopeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListByScopeUseCase handles the business logic for listing outcome criteria by scope
type ListByScopeUseCase struct {
	repositories ListByScopeRepositories
	services     ListByScopeServices
}

// NewListByScopeUseCase creates a new ListByScopeUseCase
func NewListByScopeUseCase(
	repositories ListByScopeRepositories,
	services ListByScopeServices,
) *ListByScopeUseCase {
	return &ListByScopeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list by scope operation
func (uc *ListByScopeUseCase) Execute(ctx context.Context, req *pb.ListOutcomeCriteriasByScopeRequest) (*pb.ListOutcomeCriteriasByScopeResponse, error) {
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

// executeWithTransaction executes list by scope within a transaction
func (uc *ListByScopeUseCase) executeWithTransaction(ctx context.Context, req *pb.ListOutcomeCriteriasByScopeRequest) (*pb.ListOutcomeCriteriasByScopeResponse, error) {
	var result *pb.ListOutcomeCriteriasByScopeResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "outcome_criteria.errors.list_by_scope_failed", "outcome criteria listing by scope failed: %w"), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for listing outcome criteria by scope
func (uc *ListByScopeUseCase) executeCore(ctx context.Context, req *pb.ListOutcomeCriteriasByScopeRequest) (*pb.ListOutcomeCriteriasByScopeResponse, error) {
	resp, err := uc.repositories.OutcomeCriteria.ListByScope(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "outcome_criteria.errors.list_by_scope_failed", "failed to list outcome criteria by scope: %w"), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListByScopeUseCase) validateInput(ctx context.Context, req *pb.ListOutcomeCriteriasByScopeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "outcome_criteria.validation.request_required", "request is required"))
	}
	if req.Scope == enumspb.CriteriaScope_CRITERIA_SCOPE_UNSPECIFIED {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "outcome_criteria.validation.scope_required", "scope is required"))
	}

	return nil
}

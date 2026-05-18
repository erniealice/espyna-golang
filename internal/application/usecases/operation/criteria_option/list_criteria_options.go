package criteria_option

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/criteria_option"
)

type ListCriteriaOptionsRepositories struct {
	CriteriaOption pb.CriteriaOptionDomainServiceServer
}

type ListCriteriaOptionsServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListCriteriaOptionsUseCase handles the business logic for listing criteria options
type ListCriteriaOptionsUseCase struct {
	repositories ListCriteriaOptionsRepositories
	services     ListCriteriaOptionsServices
}

// NewListCriteriaOptionsUseCase creates a new ListCriteriaOptionsUseCase
func NewListCriteriaOptionsUseCase(
	repositories ListCriteriaOptionsRepositories,
	services ListCriteriaOptionsServices,
) *ListCriteriaOptionsUseCase {
	return &ListCriteriaOptionsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list criteria options operation
func (uc *ListCriteriaOptionsUseCase) Execute(ctx context.Context, req *pb.ListCriteriaOptionsRequest) (*pb.ListCriteriaOptionsResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityCriteriaOption, ports.ActionList); err != nil {
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
func (uc *ListCriteriaOptionsUseCase) executeWithTransaction(ctx context.Context, req *pb.ListCriteriaOptionsRequest) (*pb.ListCriteriaOptionsResponse, error) {
	var result *pb.ListCriteriaOptionsResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "criteria_option.errors.list_failed", "criteria option listing failed: %w"), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for listing criteria options
func (uc *ListCriteriaOptionsUseCase) executeCore(ctx context.Context, req *pb.ListCriteriaOptionsRequest) (*pb.ListCriteriaOptionsResponse, error) {
	resp, err := uc.repositories.CriteriaOption.ListCriteriaOptions(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "criteria_option.errors.list_failed", "criteria option listing failed: %w"), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListCriteriaOptionsUseCase) validateInput(ctx context.Context, req *pb.ListCriteriaOptionsRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "criteria_option.validation.request_required", "request is required"))
	}

	return nil
}

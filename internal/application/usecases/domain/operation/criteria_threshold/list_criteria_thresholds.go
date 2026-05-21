package criteria_threshold

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/criteria_threshold"
)

type ListCriteriaThresholdsRepositories struct {
	CriteriaThreshold pb.CriteriaThresholdDomainServiceServer
}

type ListCriteriaThresholdsServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListCriteriaThresholdsUseCase handles the business logic for listing criteria thresholds
type ListCriteriaThresholdsUseCase struct {
	repositories ListCriteriaThresholdsRepositories
	services     ListCriteriaThresholdsServices
}

// NewListCriteriaThresholdsUseCase creates a new ListCriteriaThresholdsUseCase
func NewListCriteriaThresholdsUseCase(
	repositories ListCriteriaThresholdsRepositories,
	services ListCriteriaThresholdsServices,
) *ListCriteriaThresholdsUseCase {
	return &ListCriteriaThresholdsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list criteria thresholds operation
func (uc *ListCriteriaThresholdsUseCase) Execute(ctx context.Context, req *pb.ListCriteriaThresholdsRequest) (*pb.ListCriteriaThresholdsResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityCriteriaThreshold, ports.ActionList); err != nil {
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
func (uc *ListCriteriaThresholdsUseCase) executeWithTransaction(ctx context.Context, req *pb.ListCriteriaThresholdsRequest) (*pb.ListCriteriaThresholdsResponse, error) {
	var result *pb.ListCriteriaThresholdsResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "criteria_threshold.errors.list_failed", "criteria threshold listing failed: %w"), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for listing criteria thresholds
func (uc *ListCriteriaThresholdsUseCase) executeCore(ctx context.Context, req *pb.ListCriteriaThresholdsRequest) (*pb.ListCriteriaThresholdsResponse, error) {
	resp, err := uc.repositories.CriteriaThreshold.ListCriteriaThresholds(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "criteria_threshold.errors.list_failed", "criteria threshold listing failed: %w"), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListCriteriaThresholdsUseCase) validateInput(ctx context.Context, req *pb.ListCriteriaThresholdsRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "criteria_threshold.validation.request_required", "request is required"))
	}

	return nil
}

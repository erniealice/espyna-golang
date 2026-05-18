package criteria_threshold

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/criteria_threshold"
)

type ListByCriteriaRepositories struct {
	CriteriaThreshold pb.CriteriaThresholdDomainServiceServer
}

type ListByCriteriaServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListByCriteriaUseCase handles the business logic for listing thresholds by criteria
type ListByCriteriaUseCase struct {
	repositories ListByCriteriaRepositories
	services     ListByCriteriaServices
}

// NewListByCriteriaUseCase creates a new ListByCriteriaUseCase
func NewListByCriteriaUseCase(
	repositories ListByCriteriaRepositories,
	services ListByCriteriaServices,
) *ListByCriteriaUseCase {
	return &ListByCriteriaUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list by criteria operation
func (uc *ListByCriteriaUseCase) Execute(ctx context.Context, req *pb.ListCriteriaThresholdsByCriteriaRequest) (*pb.ListCriteriaThresholdsByCriteriaResponse, error) {
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

// executeWithTransaction executes list by criteria within a transaction
func (uc *ListByCriteriaUseCase) executeWithTransaction(ctx context.Context, req *pb.ListCriteriaThresholdsByCriteriaRequest) (*pb.ListCriteriaThresholdsByCriteriaResponse, error) {
	var result *pb.ListCriteriaThresholdsByCriteriaResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "criteria_threshold.errors.list_by_criteria_failed", "criteria threshold listing by criteria failed: %w"), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for listing thresholds by criteria
func (uc *ListByCriteriaUseCase) executeCore(ctx context.Context, req *pb.ListCriteriaThresholdsByCriteriaRequest) (*pb.ListCriteriaThresholdsByCriteriaResponse, error) {
	resp, err := uc.repositories.CriteriaThreshold.ListByCriteria(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "criteria_threshold.errors.list_by_criteria_failed", "failed to list criteria thresholds by criteria: %w"), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListByCriteriaUseCase) validateInput(ctx context.Context, req *pb.ListCriteriaThresholdsByCriteriaRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "criteria_threshold.validation.request_required", "request is required"))
	}
	if req.OutcomeCriteriaId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "criteria_threshold.validation.criteria_id_required", "outcome criteria ID is required"))
	}

	return nil
}

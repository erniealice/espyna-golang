package criteria_threshold

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/criteria_threshold"
)

type DeleteCriteriaThresholdRepositories struct {
	CriteriaThreshold pb.CriteriaThresholdDomainServiceServer
}

type DeleteCriteriaThresholdServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteCriteriaThresholdUseCase handles the business logic for deleting criteria thresholds
type DeleteCriteriaThresholdUseCase struct {
	repositories DeleteCriteriaThresholdRepositories
	services     DeleteCriteriaThresholdServices
}

// NewDeleteCriteriaThresholdUseCase creates a new DeleteCriteriaThresholdUseCase
func NewDeleteCriteriaThresholdUseCase(
	repositories DeleteCriteriaThresholdRepositories,
	services DeleteCriteriaThresholdServices,
) *DeleteCriteriaThresholdUseCase {
	return &DeleteCriteriaThresholdUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete criteria threshold operation
func (uc *DeleteCriteriaThresholdUseCase) Execute(ctx context.Context, req *pb.DeleteCriteriaThresholdRequest) (*pb.DeleteCriteriaThresholdResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityCriteriaThreshold, ports.ActionDelete); err != nil {
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

// executeWithTransaction executes deletion within a transaction
func (uc *DeleteCriteriaThresholdUseCase) executeWithTransaction(ctx context.Context, req *pb.DeleteCriteriaThresholdRequest) (*pb.DeleteCriteriaThresholdResponse, error) {
	var result *pb.DeleteCriteriaThresholdResponse

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

// executeCore contains the core business logic for deleting a criteria threshold
func (uc *DeleteCriteriaThresholdUseCase) executeCore(ctx context.Context, req *pb.DeleteCriteriaThresholdRequest) (*pb.DeleteCriteriaThresholdResponse, error) {
	// First, check if the entity exists
	_, err := uc.repositories.CriteriaThreshold.ReadCriteriaThreshold(ctx, &pb.ReadCriteriaThresholdRequest{
		Data: &pb.CriteriaThreshold{Id: req.Data.Id},
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "criteria_threshold.errors.not_found", "[ERR-DEFAULT] Criteria threshold not found"))
	}

	resp, err := uc.repositories.CriteriaThreshold.DeleteCriteriaThreshold(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "criteria_threshold.errors.deletion_failed", "[ERR-DEFAULT] Criteria threshold deletion failed"))
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteCriteriaThresholdUseCase) validateInput(ctx context.Context, req *pb.DeleteCriteriaThresholdRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "criteria_threshold.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "criteria_threshold.validation.data_required", "[ERR-DEFAULT] Criteria threshold data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "criteria_threshold.validation.id_required", "[ERR-DEFAULT] Criteria threshold ID is required"))
	}
	return nil
}

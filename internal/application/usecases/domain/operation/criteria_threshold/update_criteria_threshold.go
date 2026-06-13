package criteria_threshold

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/criteria_threshold"
)

type UpdateCriteriaThresholdRepositories struct {
	CriteriaThreshold pb.CriteriaThresholdDomainServiceServer
}

type UpdateCriteriaThresholdServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UpdateCriteriaThresholdUseCase handles the business logic for updating criteria thresholds
type UpdateCriteriaThresholdUseCase struct {
	repositories UpdateCriteriaThresholdRepositories
	services     UpdateCriteriaThresholdServices
}

// NewUpdateCriteriaThresholdUseCase creates a new UpdateCriteriaThresholdUseCase
func NewUpdateCriteriaThresholdUseCase(
	repositories UpdateCriteriaThresholdRepositories,
	services UpdateCriteriaThresholdServices,
) *UpdateCriteriaThresholdUseCase {
	return &UpdateCriteriaThresholdUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update criteria threshold operation
func (uc *UpdateCriteriaThresholdUseCase) Execute(ctx context.Context, req *pb.UpdateCriteriaThresholdRequest) (*pb.UpdateCriteriaThresholdResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.CriteriaThreshold,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Business enrichment
	enrichedData := uc.applyBusinessLogic(req.Data)

	// Use transaction service if available
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req, enrichedData)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req, enrichedData)
}

// executeWithTransaction executes update within a transaction
func (uc *UpdateCriteriaThresholdUseCase) executeWithTransaction(ctx context.Context, req *pb.UpdateCriteriaThresholdRequest, enrichedData *pb.CriteriaThreshold) (*pb.UpdateCriteriaThresholdResponse, error) {
	var result *pb.UpdateCriteriaThresholdResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req, enrichedData)
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

// executeCore contains the core business logic for updating a criteria threshold
func (uc *UpdateCriteriaThresholdUseCase) executeCore(ctx context.Context, req *pb.UpdateCriteriaThresholdRequest, enrichedData *pb.CriteriaThreshold) (*pb.UpdateCriteriaThresholdResponse, error) {
	// First, check if the entity exists
	_, err := uc.repositories.CriteriaThreshold.ReadCriteriaThreshold(ctx, &pb.ReadCriteriaThresholdRequest{
		Data: &pb.CriteriaThreshold{Id: req.Data.Id},
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "criteria_threshold.errors.not_found", "[ERR-DEFAULT] Criteria threshold not found"))
	}

	resp, err := uc.repositories.CriteriaThreshold.UpdateCriteriaThreshold(ctx, &pb.UpdateCriteriaThresholdRequest{
		Data: enrichedData,
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "criteria_threshold.errors.update_failed", "[ERR-DEFAULT] Criteria threshold update failed"))
	}
	return resp, nil
}

// applyBusinessLogic applies business rules and returns enriched data
func (uc *UpdateCriteriaThresholdUseCase) applyBusinessLogic(data *pb.CriteriaThreshold) *pb.CriteriaThreshold {
	now := time.Now()

	// Business logic: Update modification audit fields
	data.DateModified = &[]int64{now.UnixMilli()}[0]
	data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return data
}

// validateInput validates the input request
func (uc *UpdateCriteriaThresholdUseCase) validateInput(ctx context.Context, req *pb.UpdateCriteriaThresholdRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "criteria_threshold.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "criteria_threshold.validation.data_required", "[ERR-DEFAULT] Criteria threshold data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "criteria_threshold.validation.id_required", "[ERR-DEFAULT] Criteria threshold ID is required"))
	}
	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateCriteriaThresholdUseCase) validateBusinessRules(ctx context.Context, data *pb.CriteriaThreshold) error {
	if data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "criteria_threshold.validation.data_required", "[ERR-DEFAULT] Criteria threshold data is required"))
	}
	if data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "criteria_threshold.validation.id_required", "[ERR-DEFAULT] Criteria threshold ID is required"))
	}

	return nil
}

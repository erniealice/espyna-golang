package criteria_threshold

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/criteria_threshold"
)

type CreateCriteriaThresholdRepositories struct {
	CriteriaThreshold pb.CriteriaThresholdDomainServiceServer
}

type CreateCriteriaThresholdServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// CreateCriteriaThresholdUseCase handles the business logic for creating criteria thresholds
type CreateCriteriaThresholdUseCase struct {
	repositories CreateCriteriaThresholdRepositories
	services     CreateCriteriaThresholdServices
}

// NewCreateCriteriaThresholdUseCase creates a new CreateCriteriaThresholdUseCase
func NewCreateCriteriaThresholdUseCase(
	repositories CreateCriteriaThresholdRepositories,
	services CreateCriteriaThresholdServices,
) *CreateCriteriaThresholdUseCase {
	return &CreateCriteriaThresholdUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create criteria threshold operation
func (uc *CreateCriteriaThresholdUseCase) Execute(ctx context.Context, req *pb.CreateCriteriaThresholdRequest) (*pb.CreateCriteriaThresholdResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.CriteriaThreshold, entityid.ActionCreate); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "criteria_threshold.validation.data_required", "[ERR-DEFAULT] Criteria threshold data is required"))
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

// executeWithTransaction executes creation within a transaction
func (uc *CreateCriteriaThresholdUseCase) executeWithTransaction(ctx context.Context, req *pb.CreateCriteriaThresholdRequest, enrichedData *pb.CriteriaThreshold) (*pb.CreateCriteriaThresholdResponse, error) {
	var result *pb.CreateCriteriaThresholdResponse
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

// executeCore contains the core business logic for creating a criteria threshold
func (uc *CreateCriteriaThresholdUseCase) executeCore(ctx context.Context, req *pb.CreateCriteriaThresholdRequest, enrichedData *pb.CriteriaThreshold) (*pb.CreateCriteriaThresholdResponse, error) {
	resp, err := uc.repositories.CriteriaThreshold.CreateCriteriaThreshold(ctx, &pb.CreateCriteriaThresholdRequest{
		Data: enrichedData,
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "criteria_threshold.errors.creation_failed", "[ERR-DEFAULT] Criteria threshold creation failed"))
	}
	return resp, nil
}

// applyBusinessLogic applies business rules and returns enriched data
func (uc *CreateCriteriaThresholdUseCase) applyBusinessLogic(data *pb.CriteriaThreshold) *pb.CriteriaThreshold {
	now := time.Now()

	// Business logic: Generate ID if not provided
	if data.Id == "" {
		data.Id = uc.services.IDGenerator.GenerateID()
	}

	// Business logic: Set creation audit fields
	data.DateCreated = &[]int64{now.UnixMilli()}[0]
	data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	data.DateModified = &[]int64{now.UnixMilli()}[0]
	data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return data
}

// validateBusinessRules enforces business constraints
func (uc *CreateCriteriaThresholdUseCase) validateBusinessRules(ctx context.Context, data *pb.CriteriaThreshold) error {
	if data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "criteria_threshold.validation.data_required", "[ERR-DEFAULT] Criteria threshold data is required"))
	}
	if data.OutcomeCriteriaId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "criteria_threshold.validation.criteria_id_required", "[ERR-DEFAULT] Outcome criteria ID is required"))
	}

	return nil
}

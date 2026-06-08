package criteria_threshold

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/criteria_threshold"
)

type ReadCriteriaThresholdRepositories struct {
	CriteriaThreshold pb.CriteriaThresholdDomainServiceServer
}

type ReadCriteriaThresholdServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ReadCriteriaThresholdUseCase handles the business logic for reading criteria thresholds
type ReadCriteriaThresholdUseCase struct {
	repositories ReadCriteriaThresholdRepositories
	services     ReadCriteriaThresholdServices
}

// NewReadCriteriaThresholdUseCase creates a new ReadCriteriaThresholdUseCase
func NewReadCriteriaThresholdUseCase(
	repositories ReadCriteriaThresholdRepositories,
	services ReadCriteriaThresholdServices,
) *ReadCriteriaThresholdUseCase {
	return &ReadCriteriaThresholdUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read criteria threshold operation
func (uc *ReadCriteriaThresholdUseCase) Execute(ctx context.Context, req *pb.ReadCriteriaThresholdRequest) (*pb.ReadCriteriaThresholdResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.CriteriaThreshold, entityid.ActionRead); err != nil {
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

// executeWithTransaction executes reading within a transaction
func (uc *ReadCriteriaThresholdUseCase) executeWithTransaction(ctx context.Context, req *pb.ReadCriteriaThresholdRequest) (*pb.ReadCriteriaThresholdResponse, error) {
	var result *pb.ReadCriteriaThresholdResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
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

// executeCore contains the core business logic for reading a criteria threshold
func (uc *ReadCriteriaThresholdUseCase) executeCore(ctx context.Context, req *pb.ReadCriteriaThresholdRequest) (*pb.ReadCriteriaThresholdResponse, error) {
	resp, err := uc.repositories.CriteriaThreshold.ReadCriteriaThreshold(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "criteria_threshold.errors.not_found", "[ERR-DEFAULT] Criteria threshold not found"))
	}
	if resp == nil || len(resp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "criteria_threshold.errors.not_found", "[ERR-DEFAULT] Criteria threshold not found"))
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ReadCriteriaThresholdUseCase) validateInput(ctx context.Context, req *pb.ReadCriteriaThresholdRequest) error {
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

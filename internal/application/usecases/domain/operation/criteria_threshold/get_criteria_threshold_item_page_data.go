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

type GetCriteriaThresholdItemPageDataRepositories struct {
	CriteriaThreshold pb.CriteriaThresholdDomainServiceServer
}

type GetCriteriaThresholdItemPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetCriteriaThresholdItemPageDataUseCase handles the business logic for getting criteria threshold item page data
type GetCriteriaThresholdItemPageDataUseCase struct {
	repositories GetCriteriaThresholdItemPageDataRepositories
	services     GetCriteriaThresholdItemPageDataServices
}

// NewGetCriteriaThresholdItemPageDataUseCase creates a new GetCriteriaThresholdItemPageDataUseCase
func NewGetCriteriaThresholdItemPageDataUseCase(
	repositories GetCriteriaThresholdItemPageDataRepositories,
	services GetCriteriaThresholdItemPageDataServices,
) *GetCriteriaThresholdItemPageDataUseCase {
	return &GetCriteriaThresholdItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get criteria threshold item page data operation
func (uc *GetCriteriaThresholdItemPageDataUseCase) Execute(
	ctx context.Context,
	req *pb.GetCriteriaThresholdItemPageDataRequest,
) (*pb.GetCriteriaThresholdItemPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		ports.EntityCriteriaThreshold, ports.ActionList); err != nil {
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

// executeWithTransaction executes item page data retrieval within a transaction
func (uc *GetCriteriaThresholdItemPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *pb.GetCriteriaThresholdItemPageDataRequest,
) (*pb.GetCriteriaThresholdItemPageDataResponse, error) {
	var result *pb.GetCriteriaThresholdItemPageDataResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.Translator,
				"criteria_threshold.errors.item_page_data_failed",
				"criteria threshold item page data retrieval failed: %w",
			), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for getting criteria threshold item page data
func (uc *GetCriteriaThresholdItemPageDataUseCase) executeCore(
	ctx context.Context,
	req *pb.GetCriteriaThresholdItemPageDataRequest,
) (*pb.GetCriteriaThresholdItemPageDataResponse, error) {
	readReq := &pb.ReadCriteriaThresholdRequest{
		Data: &pb.CriteriaThreshold{
			Id: req.CriteriaThresholdId,
		},
	}

	readResp, err := uc.repositories.CriteriaThreshold.ReadCriteriaThreshold(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"criteria_threshold.errors.read_failed",
			"failed to retrieve criteria threshold: %w",
		), err)
	}

	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"criteria_threshold.errors.not_found",
			"criteria threshold not found",
		))
	}

	item := readResp.Data[0]

	return &pb.GetCriteriaThresholdItemPageDataResponse{
		CriteriaThreshold: item,
		Success:           true,
	}, nil
}

// validateInput validates the input request
func (uc *GetCriteriaThresholdItemPageDataUseCase) validateInput(
	ctx context.Context,
	req *pb.GetCriteriaThresholdItemPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"criteria_threshold.validation.request_required",
			"request is required",
		))
	}

	if req.CriteriaThresholdId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"criteria_threshold.validation.id_required",
			"criteria threshold ID is required",
		))
	}

	return nil
}

package outcome_criteria

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
)

type GetOutcomeCriteriaItemPageDataRepositories struct {
	OutcomeCriteria pb.OutcomeCriteriaDomainServiceServer
}

type GetOutcomeCriteriaItemPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetOutcomeCriteriaItemPageDataUseCase handles the business logic for getting outcome criteria item page data
type GetOutcomeCriteriaItemPageDataUseCase struct {
	repositories GetOutcomeCriteriaItemPageDataRepositories
	services     GetOutcomeCriteriaItemPageDataServices
}

// NewGetOutcomeCriteriaItemPageDataUseCase creates a new GetOutcomeCriteriaItemPageDataUseCase
func NewGetOutcomeCriteriaItemPageDataUseCase(
	repositories GetOutcomeCriteriaItemPageDataRepositories,
	services GetOutcomeCriteriaItemPageDataServices,
) *GetOutcomeCriteriaItemPageDataUseCase {
	return &GetOutcomeCriteriaItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get outcome criteria item page data operation
func (uc *GetOutcomeCriteriaItemPageDataUseCase) Execute(
	ctx context.Context,
	req *pb.GetOutcomeCriteriaItemPageDataRequest,
) (*pb.GetOutcomeCriteriaItemPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.OutcomeCriteria, entityid.ActionList); err != nil {
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
func (uc *GetOutcomeCriteriaItemPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *pb.GetOutcomeCriteriaItemPageDataRequest,
) (*pb.GetOutcomeCriteriaItemPageDataResponse, error) {
	var result *pb.GetOutcomeCriteriaItemPageDataResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.Translator,
				"outcome_criteria.errors.item_page_data_failed",
				"outcome criteria item page data retrieval failed: %w",
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

// executeCore contains the core business logic for getting outcome criteria item page data
func (uc *GetOutcomeCriteriaItemPageDataUseCase) executeCore(
	ctx context.Context,
	req *pb.GetOutcomeCriteriaItemPageDataRequest,
) (*pb.GetOutcomeCriteriaItemPageDataResponse, error) {
	readReq := &pb.ReadOutcomeCriteriaRequest{
		Data: &pb.OutcomeCriteria{
			Id: req.OutcomeCriteriaId,
		},
	}

	readResp, err := uc.repositories.OutcomeCriteria.ReadOutcomeCriteria(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"outcome_criteria.errors.read_failed",
			"failed to retrieve outcome criteria: %w",
		), err)
	}

	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"outcome_criteria.errors.not_found",
			"outcome criteria not found",
		))
	}

	item := readResp.Data[0]

	return &pb.GetOutcomeCriteriaItemPageDataResponse{
		OutcomeCriteria: item,
		Success:         true,
	}, nil
}

// validateInput validates the input request
func (uc *GetOutcomeCriteriaItemPageDataUseCase) validateInput(
	ctx context.Context,
	req *pb.GetOutcomeCriteriaItemPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"outcome_criteria.validation.request_required",
			"request is required",
		))
	}

	if req.OutcomeCriteriaId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"outcome_criteria.validation.id_required",
			"outcome criteria ID is required",
		))
	}

	return nil
}

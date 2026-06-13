package criteria_option

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/criteria_option"
)

type GetCriteriaOptionItemPageDataRepositories struct {
	CriteriaOption pb.CriteriaOptionDomainServiceServer
}

type GetCriteriaOptionItemPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// GetCriteriaOptionItemPageDataUseCase handles the business logic for getting criteria option item page data
type GetCriteriaOptionItemPageDataUseCase struct {
	repositories GetCriteriaOptionItemPageDataRepositories
	services     GetCriteriaOptionItemPageDataServices
}

// NewGetCriteriaOptionItemPageDataUseCase creates a new GetCriteriaOptionItemPageDataUseCase
func NewGetCriteriaOptionItemPageDataUseCase(
	repositories GetCriteriaOptionItemPageDataRepositories,
	services GetCriteriaOptionItemPageDataServices,
) *GetCriteriaOptionItemPageDataUseCase {
	return &GetCriteriaOptionItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get criteria option item page data operation
func (uc *GetCriteriaOptionItemPageDataUseCase) Execute(
	ctx context.Context,
	req *pb.GetCriteriaOptionItemPageDataRequest,
) (*pb.GetCriteriaOptionItemPageDataResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.CriteriaOption,
		Action: entityid.ActionList,
	}); err != nil {
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
func (uc *GetCriteriaOptionItemPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *pb.GetCriteriaOptionItemPageDataRequest,
) (*pb.GetCriteriaOptionItemPageDataResponse, error) {
	var result *pb.GetCriteriaOptionItemPageDataResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.Translator,
				"criteria_option.errors.item_page_data_failed",
				"criteria option item page data retrieval failed: %w",
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

// executeCore contains the core business logic for getting criteria option item page data
func (uc *GetCriteriaOptionItemPageDataUseCase) executeCore(
	ctx context.Context,
	req *pb.GetCriteriaOptionItemPageDataRequest,
) (*pb.GetCriteriaOptionItemPageDataResponse, error) {
	readReq := &pb.ReadCriteriaOptionRequest{
		Data: &pb.CriteriaOption{
			Id: req.CriteriaOptionId,
		},
	}

	readResp, err := uc.repositories.CriteriaOption.ReadCriteriaOption(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"criteria_option.errors.read_failed",
			"failed to retrieve criteria option: %w",
		), err)
	}

	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"criteria_option.errors.not_found",
			"criteria option not found",
		))
	}

	item := readResp.Data[0]

	return &pb.GetCriteriaOptionItemPageDataResponse{
		CriteriaOption: item,
		Success:        true,
	}, nil
}

// validateInput validates the input request
func (uc *GetCriteriaOptionItemPageDataUseCase) validateInput(
	ctx context.Context,
	req *pb.GetCriteriaOptionItemPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"criteria_option.validation.request_required",
			"request is required",
		))
	}

	if req.CriteriaOptionId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"criteria_option.validation.id_required",
			"criteria option ID is required",
		))
	}

	return nil
}

package criteria_option

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/criteria_option"
)

type DeleteCriteriaOptionRepositories struct {
	CriteriaOption pb.CriteriaOptionDomainServiceServer
}

type DeleteCriteriaOptionServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// DeleteCriteriaOptionUseCase handles the business logic for deleting criteria options
type DeleteCriteriaOptionUseCase struct {
	repositories DeleteCriteriaOptionRepositories
	services     DeleteCriteriaOptionServices
}

// NewDeleteCriteriaOptionUseCase creates a new DeleteCriteriaOptionUseCase
func NewDeleteCriteriaOptionUseCase(
	repositories DeleteCriteriaOptionRepositories,
	services DeleteCriteriaOptionServices,
) *DeleteCriteriaOptionUseCase {
	return &DeleteCriteriaOptionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete criteria option operation
func (uc *DeleteCriteriaOptionUseCase) Execute(ctx context.Context, req *pb.DeleteCriteriaOptionRequest) (*pb.DeleteCriteriaOptionResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.CriteriaOption, entityid.ActionDelete); err != nil {
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

// executeWithTransaction executes deletion within a transaction
func (uc *DeleteCriteriaOptionUseCase) executeWithTransaction(ctx context.Context, req *pb.DeleteCriteriaOptionRequest) (*pb.DeleteCriteriaOptionResponse, error) {
	var result *pb.DeleteCriteriaOptionResponse

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

// executeCore contains the core business logic for deleting a criteria option
func (uc *DeleteCriteriaOptionUseCase) executeCore(ctx context.Context, req *pb.DeleteCriteriaOptionRequest) (*pb.DeleteCriteriaOptionResponse, error) {
	_, err := uc.repositories.CriteriaOption.ReadCriteriaOption(ctx, &pb.ReadCriteriaOptionRequest{
		Data: &pb.CriteriaOption{Id: req.Data.Id},
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "criteria_option.errors.not_found", "[ERR-DEFAULT] Criteria option not found"))
	}

	resp, err := uc.repositories.CriteriaOption.DeleteCriteriaOption(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "criteria_option.errors.deletion_failed", "[ERR-DEFAULT] Criteria option deletion failed"))
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteCriteriaOptionUseCase) validateInput(ctx context.Context, req *pb.DeleteCriteriaOptionRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "criteria_option.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "criteria_option.validation.data_required", "[ERR-DEFAULT] Criteria option data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "criteria_option.validation.id_required", "[ERR-DEFAULT] Criteria option ID is required"))
	}
	return nil
}

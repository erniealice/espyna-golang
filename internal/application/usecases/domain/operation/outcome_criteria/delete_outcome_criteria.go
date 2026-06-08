package outcome_criteria

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
)

type DeleteOutcomeCriteriaRepositories struct {
	OutcomeCriteria pb.OutcomeCriteriaDomainServiceServer
}

type DeleteOutcomeCriteriaServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// DeleteOutcomeCriteriaUseCase handles the business logic for deleting outcome criteria
type DeleteOutcomeCriteriaUseCase struct {
	repositories DeleteOutcomeCriteriaRepositories
	services     DeleteOutcomeCriteriaServices
}

// NewDeleteOutcomeCriteriaUseCase creates a new DeleteOutcomeCriteriaUseCase
func NewDeleteOutcomeCriteriaUseCase(
	repositories DeleteOutcomeCriteriaRepositories,
	services DeleteOutcomeCriteriaServices,
) *DeleteOutcomeCriteriaUseCase {
	return &DeleteOutcomeCriteriaUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete outcome criteria operation
func (uc *DeleteOutcomeCriteriaUseCase) Execute(ctx context.Context, req *pb.DeleteOutcomeCriteriaRequest) (*pb.DeleteOutcomeCriteriaResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.OutcomeCriteria, entityid.ActionDelete); err != nil {
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
func (uc *DeleteOutcomeCriteriaUseCase) executeWithTransaction(ctx context.Context, req *pb.DeleteOutcomeCriteriaRequest) (*pb.DeleteOutcomeCriteriaResponse, error) {
	var result *pb.DeleteOutcomeCriteriaResponse

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

// executeCore contains the core business logic for deleting an outcome criteria
func (uc *DeleteOutcomeCriteriaUseCase) executeCore(ctx context.Context, req *pb.DeleteOutcomeCriteriaRequest) (*pb.DeleteOutcomeCriteriaResponse, error) {
	// First, check if the entity exists
	_, err := uc.repositories.OutcomeCriteria.ReadOutcomeCriteria(ctx, &pb.ReadOutcomeCriteriaRequest{
		Data: &pb.OutcomeCriteria{Id: req.Data.Id},
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.errors.not_found", "[ERR-DEFAULT] Outcome criteria not found"))
	}

	resp, err := uc.repositories.OutcomeCriteria.DeleteOutcomeCriteria(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.errors.deletion_failed", "[ERR-DEFAULT] Outcome criteria deletion failed"))
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteOutcomeCriteriaUseCase) validateInput(ctx context.Context, req *pb.DeleteOutcomeCriteriaRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.validation.data_required", "[ERR-DEFAULT] Outcome criteria data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.validation.id_required", "[ERR-DEFAULT] Outcome criteria ID is required"))
	}
	return nil
}

package outcome_criteria

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
)

type UpdateOutcomeCriteriaRepositories struct {
	OutcomeCriteria pb.OutcomeCriteriaDomainServiceServer
}

type UpdateOutcomeCriteriaServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// UpdateOutcomeCriteriaUseCase handles the business logic for updating outcome criteria
type UpdateOutcomeCriteriaUseCase struct {
	repositories UpdateOutcomeCriteriaRepositories
	services     UpdateOutcomeCriteriaServices
}

// NewUpdateOutcomeCriteriaUseCase creates a new UpdateOutcomeCriteriaUseCase
func NewUpdateOutcomeCriteriaUseCase(
	repositories UpdateOutcomeCriteriaRepositories,
	services UpdateOutcomeCriteriaServices,
) *UpdateOutcomeCriteriaUseCase {
	return &UpdateOutcomeCriteriaUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update outcome criteria operation
func (uc *UpdateOutcomeCriteriaUseCase) Execute(ctx context.Context, req *pb.UpdateOutcomeCriteriaRequest) (*pb.UpdateOutcomeCriteriaResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		ports.EntityOutcomeCriteria, ports.ActionUpdate); err != nil {
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
func (uc *UpdateOutcomeCriteriaUseCase) executeWithTransaction(ctx context.Context, req *pb.UpdateOutcomeCriteriaRequest, enrichedData *pb.OutcomeCriteria) (*pb.UpdateOutcomeCriteriaResponse, error) {
	var result *pb.UpdateOutcomeCriteriaResponse

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

// executeCore contains the core business logic for updating an outcome criteria
func (uc *UpdateOutcomeCriteriaUseCase) executeCore(ctx context.Context, req *pb.UpdateOutcomeCriteriaRequest, enrichedData *pb.OutcomeCriteria) (*pb.UpdateOutcomeCriteriaResponse, error) {
	// First, check if the entity exists
	_, err := uc.repositories.OutcomeCriteria.ReadOutcomeCriteria(ctx, &pb.ReadOutcomeCriteriaRequest{
		Data: &pb.OutcomeCriteria{Id: req.Data.Id},
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.errors.not_found", "[ERR-DEFAULT] Outcome criteria not found"))
	}

	resp, err := uc.repositories.OutcomeCriteria.UpdateOutcomeCriteria(ctx, &pb.UpdateOutcomeCriteriaRequest{
		Data: enrichedData,
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.errors.update_failed", "[ERR-DEFAULT] Outcome criteria update failed"))
	}
	return resp, nil
}

// applyBusinessLogic applies business rules and returns enriched data
func (uc *UpdateOutcomeCriteriaUseCase) applyBusinessLogic(data *pb.OutcomeCriteria) *pb.OutcomeCriteria {
	now := time.Now()

	// Business logic: Update modification audit fields
	data.DateModified = &[]int64{now.UnixMilli()}[0]
	data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return data
}

// validateInput validates the input request
func (uc *UpdateOutcomeCriteriaUseCase) validateInput(ctx context.Context, req *pb.UpdateOutcomeCriteriaRequest) error {
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

// validateBusinessRules enforces business constraints
func (uc *UpdateOutcomeCriteriaUseCase) validateBusinessRules(ctx context.Context, data *pb.OutcomeCriteria) error {
	if data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.validation.data_required", "[ERR-DEFAULT] Outcome criteria data is required"))
	}
	if data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.validation.id_required", "[ERR-DEFAULT] Outcome criteria ID is required"))
	}
	// Validate Name only if provided (partial update support)
	if data.Name != "" && len(data.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.validation.name_too_long", "[ERR-DEFAULT] Outcome criteria name is too long"))
	}

	return nil
}

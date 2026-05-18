package outcome_criteria

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
)

type CreateOutcomeCriteriaRepositories struct {
	OutcomeCriteria pb.OutcomeCriteriaDomainServiceServer
}

type CreateOutcomeCriteriaServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateOutcomeCriteriaUseCase handles the business logic for creating outcome criteria
type CreateOutcomeCriteriaUseCase struct {
	repositories CreateOutcomeCriteriaRepositories
	services     CreateOutcomeCriteriaServices
}

// NewCreateOutcomeCriteriaUseCase creates a new CreateOutcomeCriteriaUseCase
func NewCreateOutcomeCriteriaUseCase(
	repositories CreateOutcomeCriteriaRepositories,
	services CreateOutcomeCriteriaServices,
) *CreateOutcomeCriteriaUseCase {
	return &CreateOutcomeCriteriaUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create outcome criteria operation
func (uc *CreateOutcomeCriteriaUseCase) Execute(ctx context.Context, req *pb.CreateOutcomeCriteriaRequest) (*pb.CreateOutcomeCriteriaResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityOutcomeCriteria, ports.ActionCreate); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "outcome_criteria.validation.data_required", "[ERR-DEFAULT] Outcome criteria data is required"))
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Business enrichment
	enrichedData := uc.applyBusinessLogic(req.Data)

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req, enrichedData)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req, enrichedData)
}

// executeWithTransaction executes creation within a transaction
func (uc *CreateOutcomeCriteriaUseCase) executeWithTransaction(ctx context.Context, req *pb.CreateOutcomeCriteriaRequest, enrichedData *pb.OutcomeCriteria) (*pb.CreateOutcomeCriteriaResponse, error) {
	var result *pb.CreateOutcomeCriteriaResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
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

// executeCore contains the core business logic for creating an outcome criteria
func (uc *CreateOutcomeCriteriaUseCase) executeCore(ctx context.Context, req *pb.CreateOutcomeCriteriaRequest, enrichedData *pb.OutcomeCriteria) (*pb.CreateOutcomeCriteriaResponse, error) {
	resp, err := uc.repositories.OutcomeCriteria.CreateOutcomeCriteria(ctx, &pb.CreateOutcomeCriteriaRequest{
		Data: enrichedData,
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "outcome_criteria.errors.creation_failed", "[ERR-DEFAULT] Outcome criteria creation failed"))
	}
	return resp, nil
}

// applyBusinessLogic applies business rules and returns enriched data
func (uc *CreateOutcomeCriteriaUseCase) applyBusinessLogic(data *pb.OutcomeCriteria) *pb.OutcomeCriteria {
	now := time.Now()

	// Business logic: Generate ID if not provided
	if data.Id == "" {
		data.Id = uc.services.IDService.GenerateID()
	}

	// Business logic: Set active status for new criteria
	data.Active = true

	// Business logic: Set creation audit fields
	data.DateCreated = &[]int64{now.UnixMilli()}[0]
	data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	data.DateModified = &[]int64{now.UnixMilli()}[0]
	data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return data
}

// validateBusinessRules enforces business constraints
func (uc *CreateOutcomeCriteriaUseCase) validateBusinessRules(ctx context.Context, data *pb.OutcomeCriteria) error {
	if data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "outcome_criteria.validation.data_required", "[ERR-DEFAULT] Outcome criteria data is required"))
	}
	if data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "outcome_criteria.validation.name_required", "[ERR-DEFAULT] Outcome criteria name is required"))
	}
	if len(data.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "outcome_criteria.validation.name_too_long", "[ERR-DEFAULT] Outcome criteria name is too long"))
	}

	return nil
}

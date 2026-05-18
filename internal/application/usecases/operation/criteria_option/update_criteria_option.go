package criteria_option

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/criteria_option"
)

type UpdateCriteriaOptionRepositories struct {
	CriteriaOption pb.CriteriaOptionDomainServiceServer
}

type UpdateCriteriaOptionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateCriteriaOptionUseCase handles the business logic for updating criteria options
type UpdateCriteriaOptionUseCase struct {
	repositories UpdateCriteriaOptionRepositories
	services     UpdateCriteriaOptionServices
}

// NewUpdateCriteriaOptionUseCase creates a new UpdateCriteriaOptionUseCase
func NewUpdateCriteriaOptionUseCase(
	repositories UpdateCriteriaOptionRepositories,
	services UpdateCriteriaOptionServices,
) *UpdateCriteriaOptionUseCase {
	return &UpdateCriteriaOptionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update criteria option operation
func (uc *UpdateCriteriaOptionUseCase) Execute(ctx context.Context, req *pb.UpdateCriteriaOptionRequest) (*pb.UpdateCriteriaOptionResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityCriteriaOption, ports.ActionUpdate); err != nil {
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
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req, enrichedData)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req, enrichedData)
}

// executeWithTransaction executes update within a transaction
func (uc *UpdateCriteriaOptionUseCase) executeWithTransaction(ctx context.Context, req *pb.UpdateCriteriaOptionRequest, enrichedData *pb.CriteriaOption) (*pb.UpdateCriteriaOptionResponse, error) {
	var result *pb.UpdateCriteriaOptionResponse

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

// executeCore contains the core business logic for updating a criteria option
func (uc *UpdateCriteriaOptionUseCase) executeCore(ctx context.Context, req *pb.UpdateCriteriaOptionRequest, enrichedData *pb.CriteriaOption) (*pb.UpdateCriteriaOptionResponse, error) {
	_, err := uc.repositories.CriteriaOption.ReadCriteriaOption(ctx, &pb.ReadCriteriaOptionRequest{
		Data: &pb.CriteriaOption{Id: req.Data.Id},
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "criteria_option.errors.not_found", "[ERR-DEFAULT] Criteria option not found"))
	}

	resp, err := uc.repositories.CriteriaOption.UpdateCriteriaOption(ctx, &pb.UpdateCriteriaOptionRequest{
		Data: enrichedData,
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "criteria_option.errors.update_failed", "[ERR-DEFAULT] Criteria option update failed"))
	}
	return resp, nil
}

// applyBusinessLogic applies business rules and returns enriched data
func (uc *UpdateCriteriaOptionUseCase) applyBusinessLogic(data *pb.CriteriaOption) *pb.CriteriaOption {
	now := time.Now()
	data.DateModified = &[]int64{now.UnixMilli()}[0]
	data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	return data
}

// validateInput validates the input request
func (uc *UpdateCriteriaOptionUseCase) validateInput(ctx context.Context, req *pb.UpdateCriteriaOptionRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "criteria_option.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "criteria_option.validation.data_required", "[ERR-DEFAULT] Criteria option data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "criteria_option.validation.id_required", "[ERR-DEFAULT] Criteria option ID is required"))
	}
	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateCriteriaOptionUseCase) validateBusinessRules(ctx context.Context, data *pb.CriteriaOption) error {
	if data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "criteria_option.validation.data_required", "[ERR-DEFAULT] Criteria option data is required"))
	}
	if data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "criteria_option.validation.id_required", "[ERR-DEFAULT] Criteria option ID is required"))
	}
	return nil
}

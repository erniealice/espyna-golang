package prepayment

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	prepaymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/prepayment"
)

const entityPrepayment = "prepayment"

// CreatePrepaymentRepositories groups all repository dependencies
type CreatePrepaymentRepositories struct {
	Prepayment prepaymentpb.PrepaymentDomainServiceServer
}

// CreatePrepaymentServices groups all business service dependencies
type CreatePrepaymentServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreatePrepaymentUseCase handles the business logic for creating prepayments
type CreatePrepaymentUseCase struct {
	repositories CreatePrepaymentRepositories
	services     CreatePrepaymentServices
}

// NewCreatePrepaymentUseCase creates use case with grouped dependencies
func NewCreatePrepaymentUseCase(
	repositories CreatePrepaymentRepositories,
	services CreatePrepaymentServices,
) *CreatePrepaymentUseCase {
	return &CreatePrepaymentUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create prepayment operation
func (uc *CreatePrepaymentUseCase) Execute(ctx context.Context, req *prepaymentpb.CreatePrepaymentRequest) (*prepaymentpb.CreatePrepaymentResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityPrepayment, ports.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *prepaymentpb.CreatePrepaymentResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "prepayment.errors.creation_failed", "Prepayment creation failed [DEFAULT]")
				return fmt.Errorf("%s: %w", translatedError, err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	return uc.executeCore(ctx, req)
}

func (uc *CreatePrepaymentUseCase) executeCore(ctx context.Context, req *prepaymentpb.CreatePrepaymentRequest) (*prepaymentpb.CreatePrepaymentResponse, error) {
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	if err := uc.enrichPrepaymentData(req.Data); err != nil {
		return nil, err
	}

	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	if uc.repositories.Prepayment == nil {
		return nil, errors.New("prepayment repository is not available")
	}
	return uc.repositories.Prepayment.CreatePrepayment(ctx, req)
}

func (uc *CreatePrepaymentUseCase) validateInput(ctx context.Context, req *prepaymentpb.CreatePrepaymentRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "prepayment.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "prepayment.validation.data_required", "[ERR-DEFAULT] Prepayment data is required"))
	}

	req.Data.Description = strings.TrimSpace(req.Data.Description)
	if req.Data.Description == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "prepayment.validation.description_required", "[ERR-DEFAULT] Description is required"))
	}
	if req.Data.TotalAmount <= 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "prepayment.validation.total_amount_required", "[ERR-DEFAULT] Total amount must be greater than zero"))
	}
	if req.Data.AmortizationMonths <= 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "prepayment.validation.amortization_months_required", "[ERR-DEFAULT] Amortization months must be greater than zero"))
	}
	return nil
}

func (uc *CreatePrepaymentUseCase) enrichPrepaymentData(p *prepaymentpb.Prepayment) error {
	now := time.Now()
	if p.Id == "" {
		p.Id = uc.services.IDService.GenerateID()
	}
	p.DateCreated = &[]int64{now.UnixMilli()}[0]
	p.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	p.DateModified = &[]int64{now.UnixMilli()}[0]
	p.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	p.Active = true
	// Remaining amount starts equal to total amount
	if p.RemainingAmount == 0 {
		p.RemainingAmount = p.TotalAmount
	}
	return nil
}

func (uc *CreatePrepaymentUseCase) validateBusinessRules(ctx context.Context, p *prepaymentpb.Prepayment) error {
	if len(p.Description) > 500 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "prepayment.validation.description_too_long", "[ERR-DEFAULT] Description must not exceed 500 characters"))
	}
	if p.AmortizationMonths > 360 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "prepayment.validation.amortization_months_too_long", "[ERR-DEFAULT] Amortization period cannot exceed 360 months"))
	}
	return nil
}

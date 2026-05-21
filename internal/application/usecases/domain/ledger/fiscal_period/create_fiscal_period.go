package fiscalperiod

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	fiscalperiodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/fiscal_period"
)

const entityFiscalPeriod = "fiscal_period"

// CreateFiscalPeriodRepositories groups all repository dependencies
type CreateFiscalPeriodRepositories struct {
	FiscalPeriod fiscalperiodpb.FiscalPeriodDomainServiceServer // Primary entity repository
}

// CreateFiscalPeriodServices groups all business service dependencies
type CreateFiscalPeriodServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// CreateFiscalPeriodUseCase handles the business logic for creating fiscal periods
type CreateFiscalPeriodUseCase struct {
	repositories CreateFiscalPeriodRepositories
	services     CreateFiscalPeriodServices
}

// NewCreateFiscalPeriodUseCase creates use case with grouped dependencies
func NewCreateFiscalPeriodUseCase(
	repositories CreateFiscalPeriodRepositories,
	services CreateFiscalPeriodServices,
) *CreateFiscalPeriodUseCase {
	return &CreateFiscalPeriodUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create fiscal period operation
func (uc *CreateFiscalPeriodUseCase) Execute(ctx context.Context, req *fiscalperiodpb.CreateFiscalPeriodRequest) (*fiscalperiodpb.CreateFiscalPeriodResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityFiscalPeriod, ports.ActionCreate); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes fiscal period creation within a transaction
func (uc *CreateFiscalPeriodUseCase) executeWithTransaction(ctx context.Context, req *fiscalperiodpb.CreateFiscalPeriodRequest) (*fiscalperiodpb.CreateFiscalPeriodResponse, error) {
	var result *fiscalperiodpb.CreateFiscalPeriodResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "fiscal_period.errors.creation_failed", "Fiscal period creation failed [DEFAULT]")
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

// executeCore contains the core business logic
func (uc *CreateFiscalPeriodUseCase) executeCore(ctx context.Context, req *fiscalperiodpb.CreateFiscalPeriodRequest) (*fiscalperiodpb.CreateFiscalPeriodResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichFiscalPeriodData(req.Data); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	if uc.repositories.FiscalPeriod == nil {
		return nil, errors.New("fiscal period repository is not available")
	}
	return uc.repositories.FiscalPeriod.CreateFiscalPeriod(ctx, req)
}

// validateInput validates the input request
func (uc *CreateFiscalPeriodUseCase) validateInput(ctx context.Context, req *fiscalperiodpb.CreateFiscalPeriodRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fiscal_period.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fiscal_period.validation.data_required", "[ERR-DEFAULT] Fiscal period data is required"))
	}

	// Trim leading and trailing spaces
	req.Data.Name = strings.TrimSpace(req.Data.Name)

	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fiscal_period.validation.name_required", "[ERR-DEFAULT] Name is required"))
	}

	// Dates are required
	if req.Data.StartDate == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fiscal_period.validation.start_date_required", "[ERR-DEFAULT] Start date is required"))
	}
	if req.Data.EndDate == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fiscal_period.validation.end_date_required", "[ERR-DEFAULT] End date is required"))
	}

	return nil
}

// enrichFiscalPeriodData adds generated fields and audit information
func (uc *CreateFiscalPeriodUseCase) enrichFiscalPeriodData(period *fiscalperiodpb.FiscalPeriod) error {
	now := time.Now()

	// Generate Fiscal Period ID if not provided
	if period.Id == "" {
		period.Id = uc.services.IDGenerator.GenerateID()
	}

	// Set status to OPEN if not set
	if period.Status == fiscalperiodpb.FiscalPeriodStatus_FISCAL_PERIOD_STATUS_UNSPECIFIED {
		period.Status = fiscalperiodpb.FiscalPeriodStatus_FISCAL_PERIOD_STATUS_OPEN
	}

	// Set audit fields
	period.DateCreated = &[]int64{now.UnixMilli()}[0]
	period.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	period.DateModified = &[]int64{now.UnixMilli()}[0]
	period.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	period.Active = true

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateFiscalPeriodUseCase) validateBusinessRules(ctx context.Context, period *fiscalperiodpb.FiscalPeriod) error {
	// Validate name length
	if len(period.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fiscal_period.validation.name_too_long", "[ERR-DEFAULT] Name must not exceed 100 characters"))
	}

	// Start date must be before end date
	if period.StartDate >= period.EndDate {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fiscal_period.validation.invalid_date_range", "[ERR-DEFAULT] Start date must be before end date"))
	}

	// Fiscal year must be a reasonable value
	if period.FiscalYear < 1900 || period.FiscalYear > 2100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fiscal_period.validation.invalid_fiscal_year", "[ERR-DEFAULT] Fiscal year must be between 1900 and 2100"))
	}

	// Period number must be 1–12 (monthly periods within a fiscal year)
	if period.PeriodNumber != 0 && (period.PeriodNumber < 1 || period.PeriodNumber > 12) {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fiscal_period.validation.invalid_period_number", "[ERR-DEFAULT] Period number must be between 1 and 12"))
	}

	return nil
}

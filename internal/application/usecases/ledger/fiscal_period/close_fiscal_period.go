package fiscalperiod

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	fiscalperiodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/fiscal_period"
)

// CloseFiscalPeriodRepositories groups all repository dependencies
type CloseFiscalPeriodRepositories struct {
	FiscalPeriod fiscalperiodpb.FiscalPeriodDomainServiceServer // Primary entity repository
}

// CloseFiscalPeriodServices groups all business service dependencies
type CloseFiscalPeriodServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// CloseFiscalPeriodUseCase handles the business logic for closing fiscal periods.
// The repository is responsible for:
//   - Verifying the period is currently OPEN
//   - Verifying no DRAFT journal entries exist for this period (all must be POSTED or REVERSED)
//   - Setting status=CLOSED and recording closed_by / closed_at audit fields
type CloseFiscalPeriodUseCase struct {
	repositories CloseFiscalPeriodRepositories
	services     CloseFiscalPeriodServices
}

// NewCloseFiscalPeriodUseCase creates use case with grouped dependencies
func NewCloseFiscalPeriodUseCase(
	repositories CloseFiscalPeriodRepositories,
	services CloseFiscalPeriodServices,
) *CloseFiscalPeriodUseCase {
	return &CloseFiscalPeriodUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the close fiscal period operation
func (uc *CloseFiscalPeriodUseCase) Execute(ctx context.Context, req *fiscalperiodpb.CloseFiscalPeriodRequest) (*fiscalperiodpb.CloseFiscalPeriodResponse, error) {
	// Authorization check — closing requires update-level access
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityFiscalPeriod, ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "fiscal_period.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes closing within a transaction
func (uc *CloseFiscalPeriodUseCase) executeWithTransaction(ctx context.Context, req *fiscalperiodpb.CloseFiscalPeriodRequest) (*fiscalperiodpb.CloseFiscalPeriodResponse, error) {
	var result *fiscalperiodpb.CloseFiscalPeriodResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "fiscal_period.errors.close_failed", "Fiscal period close failed [DEFAULT]")
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

// executeCore contains the core business logic for closing
func (uc *CloseFiscalPeriodUseCase) executeCore(ctx context.Context, req *fiscalperiodpb.CloseFiscalPeriodRequest) (*fiscalperiodpb.CloseFiscalPeriodResponse, error) {
	// Call repository
	if uc.repositories.FiscalPeriod == nil {
		return nil, errors.New("fiscal period repository is not available")
	}
	resp, err := uc.repositories.FiscalPeriod.CloseFiscalPeriod(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "fiscal_period.errors.close_failed", "[ERR-DEFAULT] Fiscal period close failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *CloseFiscalPeriodUseCase) validateInput(ctx context.Context, req *fiscalperiodpb.CloseFiscalPeriodRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "fiscal_period.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.FiscalPeriodId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "fiscal_period.validation.id_required", "[ERR-DEFAULT] Fiscal period ID is required"))
	}
	if req.ClosedBy == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "fiscal_period.validation.closed_by_required", "[ERR-DEFAULT] Closed by (user ID) is required"))
	}
	return nil
}

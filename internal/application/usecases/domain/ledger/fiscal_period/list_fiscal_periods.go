package fiscalperiod

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	fiscalperiodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/fiscal_period"
)

// ListFiscalPeriodsRepositories groups all repository dependencies
type ListFiscalPeriodsRepositories struct {
	FiscalPeriod fiscalperiodpb.FiscalPeriodDomainServiceServer // Primary entity repository
}

// ListFiscalPeriodsServices groups all business service dependencies
type ListFiscalPeriodsServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ListFiscalPeriodsUseCase handles the business logic for listing fiscal periods
type ListFiscalPeriodsUseCase struct {
	repositories ListFiscalPeriodsRepositories
	services     ListFiscalPeriodsServices
}

// NewListFiscalPeriodsUseCase creates use case with grouped dependencies
func NewListFiscalPeriodsUseCase(
	repositories ListFiscalPeriodsRepositories,
	services ListFiscalPeriodsServices,
) *ListFiscalPeriodsUseCase {
	return &ListFiscalPeriodsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list fiscal periods operation
func (uc *ListFiscalPeriodsUseCase) Execute(ctx context.Context, req *fiscalperiodpb.ListFiscalPeriodsRequest) (*fiscalperiodpb.ListFiscalPeriodsResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityFiscalPeriod, entityid.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fiscal_period.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fiscal_period.errors.business_rule_validation_failed", "[ERR-DEFAULT] Business rule validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	if uc.repositories.FiscalPeriod == nil {
		return nil, errors.New("fiscal period repository is not available")
	}
	resp, err := uc.repositories.FiscalPeriod.ListFiscalPeriods(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fiscal_period.errors.list_failed", "[ERR-DEFAULT] Failed to list fiscal periods")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ListFiscalPeriodsUseCase) validateInput(ctx context.Context, req *fiscalperiodpb.ListFiscalPeriodsRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fiscal_period.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for listing
func (uc *ListFiscalPeriodsUseCase) validateBusinessRules(ctx context.Context, req *fiscalperiodpb.ListFiscalPeriodsRequest) error {
	// No additional business rules for listing fiscal periods
	return nil
}

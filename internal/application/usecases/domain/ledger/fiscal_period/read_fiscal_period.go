package fiscalperiod

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	fiscalperiodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/fiscal_period"
)

// ReadFiscalPeriodRepositories groups all repository dependencies
type ReadFiscalPeriodRepositories struct {
	FiscalPeriod fiscalperiodpb.FiscalPeriodDomainServiceServer // Primary entity repository
}

// ReadFiscalPeriodServices groups all business service dependencies
type ReadFiscalPeriodServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ReadFiscalPeriodUseCase handles the business logic for reading fiscal periods
type ReadFiscalPeriodUseCase struct {
	repositories ReadFiscalPeriodRepositories
	services     ReadFiscalPeriodServices
}

// NewReadFiscalPeriodUseCase creates use case with grouped dependencies
func NewReadFiscalPeriodUseCase(
	repositories ReadFiscalPeriodRepositories,
	services ReadFiscalPeriodServices,
) *ReadFiscalPeriodUseCase {
	return &ReadFiscalPeriodUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read fiscal period operation
func (uc *ReadFiscalPeriodUseCase) Execute(ctx context.Context, req *fiscalperiodpb.ReadFiscalPeriodRequest) (*fiscalperiodpb.ReadFiscalPeriodResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityFiscalPeriod, ports.ActionRead); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	if uc.repositories.FiscalPeriod == nil {
		return nil, errors.New("fiscal period repository is not available")
	}
	resp, err := uc.repositories.FiscalPeriod.ReadFiscalPeriod(ctx, req)
	if err != nil {
		return nil, err
	}

	// Return response as-is (even if empty data for not found case)
	return resp, nil
}

// validateInput validates the input request
func (uc *ReadFiscalPeriodUseCase) validateInput(ctx context.Context, req *fiscalperiodpb.ReadFiscalPeriodRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fiscal_period.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fiscal_period.validation.data_required", "[ERR-DEFAULT] Data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fiscal_period.validation.id_required", "[ERR-DEFAULT] ID is required"))
	}
	return nil
}

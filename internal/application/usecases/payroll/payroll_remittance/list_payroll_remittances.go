package payrollremittance

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	payrollremittancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/payroll_remittance"
)

// ListPayrollRemittancesRepositories groups all repository dependencies.
type ListPayrollRemittancesRepositories struct {
	PayrollRemittance payrollremittancepb.PayrollRemittanceDomainServiceServer
}

// newListPayrollRemittancesRepositories casts the generic Repositories to this use case's repos.
func newListPayrollRemittancesRepositories(r Repositories) ListPayrollRemittancesRepositories {
	return ListPayrollRemittancesRepositories{PayrollRemittance: r.PayrollRemittance}
}

// ListPayrollRemittancesServices groups all business service dependencies.
type ListPayrollRemittancesServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListPayrollRemittancesUseCase handles the business logic for listing payroll remittances.
type ListPayrollRemittancesUseCase struct {
	repositories ListPayrollRemittancesRepositories
	services     ListPayrollRemittancesServices
}

// NewListPayrollRemittancesUseCase creates the use case with grouped dependencies.
func NewListPayrollRemittancesUseCase(
	repositories ListPayrollRemittancesRepositories,
	services ListPayrollRemittancesServices,
) *ListPayrollRemittancesUseCase {
	return &ListPayrollRemittancesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list payroll remittances operation.
func (uc *ListPayrollRemittancesUseCase) Execute(ctx context.Context, req *payrollremittancepb.ListPayrollRemittancesRequest) (*payrollremittancepb.ListPayrollRemittancesResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityPayrollRemittance, ports.ActionList); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payroll_remittance.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	if uc.repositories.PayrollRemittance == nil {
		return nil, errors.New("payroll remittance repository is not available")
	}

	resp, err := uc.repositories.PayrollRemittance.ListPayrollRemittances(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payroll_remittance.errors.list_failed", "[ERR-DEFAULT] Failed to list payroll remittances")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

func (uc *ListPayrollRemittancesUseCase) validateInput(ctx context.Context, req *payrollremittancepb.ListPayrollRemittancesRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payroll_remittance.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	return nil
}

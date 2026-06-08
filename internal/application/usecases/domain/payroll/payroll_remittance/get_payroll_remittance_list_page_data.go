package payrollremittance

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	payrollremittancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/payroll_remittance"
)

// GetPayrollRemittanceListPageDataRepositories groups all repository dependencies.
type GetPayrollRemittanceListPageDataRepositories struct {
	PayrollRemittance payrollremittancepb.PayrollRemittanceDomainServiceServer
}

// newGetPayrollRemittanceListPageDataRepositories casts the generic Repositories to this use case's repos.
func newGetPayrollRemittanceListPageDataRepositories(r Repositories) GetPayrollRemittanceListPageDataRepositories {
	return GetPayrollRemittanceListPageDataRepositories{PayrollRemittance: r.PayrollRemittance}
}

// GetPayrollRemittanceListPageDataServices groups all business service dependencies.
type GetPayrollRemittanceListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetPayrollRemittanceListPageDataUseCase handles the business logic for getting payroll remittance
// list page data with pagination, filtering, sorting, and search.
type GetPayrollRemittanceListPageDataUseCase struct {
	repositories GetPayrollRemittanceListPageDataRepositories
	services     GetPayrollRemittanceListPageDataServices
}

// NewGetPayrollRemittanceListPageDataUseCase creates the use case with grouped dependencies.
func NewGetPayrollRemittanceListPageDataUseCase(
	repositories GetPayrollRemittanceListPageDataRepositories,
	services GetPayrollRemittanceListPageDataServices,
) *GetPayrollRemittanceListPageDataUseCase {
	return &GetPayrollRemittanceListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get payroll remittance list page data operation.
func (uc *GetPayrollRemittanceListPageDataUseCase) Execute(ctx context.Context, req *payrollremittancepb.GetPayrollRemittanceListPageDataRequest) (*payrollremittancepb.GetPayrollRemittanceListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityPayrollRemittance, entityid.ActionList); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "payroll_remittance.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "payroll_remittance.errors.business_rule_validation_failed", "[ERR-DEFAULT] Business rule validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	if uc.repositories.PayrollRemittance == nil {
		return nil, errors.New("payroll remittance repository is not available")
	}

	resp, err := uc.repositories.PayrollRemittance.GetPayrollRemittanceListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "payroll_remittance.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load payroll remittance list")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

func (uc *GetPayrollRemittanceListPageDataUseCase) validateInput(ctx context.Context, req *payrollremittancepb.GetPayrollRemittanceListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "payroll_remittance.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	if req.Pagination != nil {
		if req.Pagination.Limit > 0 && (req.Pagination.Limit < 1 || req.Pagination.Limit > 100) {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "payroll_remittance.validation.invalid_pagination_limit", "[ERR-DEFAULT] Invalid pagination limit"))
		}
	}

	if req.Search != nil && req.Search.Query != "" {
		if len(req.Search.Query) > 100 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "payroll_remittance.validation.search_query_too_long", "[ERR-DEFAULT] Search query is too long"))
		}
	}

	return nil
}

func (uc *GetPayrollRemittanceListPageDataUseCase) validateBusinessRules(ctx context.Context, req *payrollremittancepb.GetPayrollRemittanceListPageDataRequest) error {
	// No additional business rules for getting payroll remittance list page data
	return nil
}

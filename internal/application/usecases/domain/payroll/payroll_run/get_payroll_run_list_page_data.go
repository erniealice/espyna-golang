package payrollrun

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	payrollrunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/payroll_run"
)

// GetPayrollRunListPageDataRepositories groups all repository dependencies.
type GetPayrollRunListPageDataRepositories struct {
	PayrollRun payrollrunpb.PayrollRunDomainServiceServer
}

// newGetPayrollRunListPageDataRepositories casts the generic Repositories to this use case's repos.
func newGetPayrollRunListPageDataRepositories(r Repositories) GetPayrollRunListPageDataRepositories {
	return GetPayrollRunListPageDataRepositories{PayrollRun: r.PayrollRun}
}

// GetPayrollRunListPageDataServices groups all business service dependencies.
type GetPayrollRunListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// GetPayrollRunListPageDataUseCase handles the business logic for getting payroll run
// list page data with pagination, filtering, sorting, and search.
type GetPayrollRunListPageDataUseCase struct {
	repositories GetPayrollRunListPageDataRepositories
	services     GetPayrollRunListPageDataServices
}

// NewGetPayrollRunListPageDataUseCase creates the use case with grouped dependencies.
func NewGetPayrollRunListPageDataUseCase(
	repositories GetPayrollRunListPageDataRepositories,
	services GetPayrollRunListPageDataServices,
) *GetPayrollRunListPageDataUseCase {
	return &GetPayrollRunListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get payroll run list page data operation.
func (uc *GetPayrollRunListPageDataUseCase) Execute(ctx context.Context, req *payrollrunpb.GetPayrollRunListPageDataRequest) (*payrollrunpb.GetPayrollRunListPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityPayrollRun,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "payroll_run.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "payroll_run.errors.business_rule_validation_failed", "[ERR-DEFAULT] Business rule validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	if uc.repositories.PayrollRun == nil {
		return nil, errors.New("payroll run repository is not available")
	}

	resp, err := uc.repositories.PayrollRun.GetPayrollRunListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "payroll_run.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load payroll run list")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

func (uc *GetPayrollRunListPageDataUseCase) validateInput(ctx context.Context, req *payrollrunpb.GetPayrollRunListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "payroll_run.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	if req.Pagination != nil {
		if req.Pagination.Limit > 0 && (req.Pagination.Limit < 1 || req.Pagination.Limit > 100) {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "payroll_run.validation.invalid_pagination_limit", "[ERR-DEFAULT] Invalid pagination limit"))
		}
	}

	if req.Search != nil && req.Search.Query != "" {
		if len(req.Search.Query) > 100 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "payroll_run.validation.search_query_too_long", "[ERR-DEFAULT] Search query is too long"))
		}
	}

	return nil
}

func (uc *GetPayrollRunListPageDataUseCase) validateBusinessRules(ctx context.Context, req *payrollrunpb.GetPayrollRunListPageDataRequest) error {
	// No additional business rules for getting payroll run list page data
	return nil
}

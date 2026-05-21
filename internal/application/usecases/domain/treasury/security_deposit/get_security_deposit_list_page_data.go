package securitydeposit

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	securitydepositpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/security_deposit"
)

// GetSecurityDepositListPageDataRepositories groups all repository dependencies
type GetSecurityDepositListPageDataRepositories struct {
	SecurityDeposit securitydepositpb.SecurityDepositDomainServiceServer
}

// GetSecurityDepositListPageDataServices groups all business service dependencies
type GetSecurityDepositListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetSecurityDepositListPageDataUseCase handles fetching paginated, searchable security deposit list data
type GetSecurityDepositListPageDataUseCase struct {
	repositories GetSecurityDepositListPageDataRepositories
	services     GetSecurityDepositListPageDataServices
}

// NewGetSecurityDepositListPageDataUseCase creates use case with grouped dependencies
func NewGetSecurityDepositListPageDataUseCase(
	repositories GetSecurityDepositListPageDataRepositories,
	services GetSecurityDepositListPageDataServices,
) *GetSecurityDepositListPageDataUseCase {
	return &GetSecurityDepositListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get security deposit list page data operation
func (uc *GetSecurityDepositListPageDataUseCase) Execute(ctx context.Context, req *securitydepositpb.GetSecurityDepositListPageDataRequest) (*securitydepositpb.GetSecurityDepositListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entitySecurityDeposit, ports.ActionList); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "security_deposit.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	if uc.repositories.SecurityDeposit == nil {
		return nil, errors.New("security deposit repository is not available")
	}
	resp, err := uc.repositories.SecurityDeposit.GetSecurityDepositListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "security_deposit.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load security deposit list")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

func (uc *GetSecurityDepositListPageDataUseCase) validateInput(ctx context.Context, req *securitydepositpb.GetSecurityDepositListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "security_deposit.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Pagination != nil && req.Pagination.Limit > 0 && req.Pagination.Limit > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "security_deposit.validation.invalid_pagination_limit", "[ERR-DEFAULT] Invalid pagination limit"))
	}
	if req.Search != nil && len(req.Search.Query) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "security_deposit.validation.search_query_too_long", "[ERR-DEFAULT] Search query is too long"))
	}
	return nil
}

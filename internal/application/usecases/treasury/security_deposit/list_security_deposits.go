package securitydeposit

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	securitydepositpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/security_deposit"
)

// ListSecurityDepositsRepositories groups all repository dependencies
type ListSecurityDepositsRepositories struct {
	SecurityDeposit securitydepositpb.SecurityDepositDomainServiceServer
}

// ListSecurityDepositsServices groups all business service dependencies
type ListSecurityDepositsServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListSecurityDepositsUseCase handles the business logic for listing security deposits
type ListSecurityDepositsUseCase struct {
	repositories ListSecurityDepositsRepositories
	services     ListSecurityDepositsServices
}

// NewListSecurityDepositsUseCase creates a new ListSecurityDepositsUseCase
func NewListSecurityDepositsUseCase(
	repositories ListSecurityDepositsRepositories,
	services ListSecurityDepositsServices,
) *ListSecurityDepositsUseCase {
	return &ListSecurityDepositsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list security deposits operation
func (uc *ListSecurityDepositsUseCase) Execute(ctx context.Context, req *securitydepositpb.ListSecurityDepositsRequest) (*securitydepositpb.ListSecurityDepositsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entitySecurityDeposit, ports.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "security_deposit.validation.request_required", "Request is required [DEFAULT]"))
	}

	if uc.repositories.SecurityDeposit == nil {
		return nil, errors.New("security deposit repository is not available")
	}
	return uc.repositories.SecurityDeposit.ListSecurityDeposits(ctx, req)
}

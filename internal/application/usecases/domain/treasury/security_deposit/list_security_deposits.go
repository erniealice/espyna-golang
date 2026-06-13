package securitydeposit

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	securitydepositpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/security_deposit"
)

// ListSecurityDepositsRepositories groups all repository dependencies
type ListSecurityDepositsRepositories struct {
	SecurityDeposit securitydepositpb.SecurityDepositDomainServiceServer
}

// ListSecurityDepositsServices groups all business service dependencies
type ListSecurityDepositsServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entitySecurityDeposit,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "security_deposit.validation.request_required", "Request is required [DEFAULT]"))
	}

	if uc.repositories.SecurityDeposit == nil {
		return nil, errors.New("security deposit repository is not available")
	}
	return uc.repositories.SecurityDeposit.ListSecurityDeposits(ctx, req)
}

package account

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	accountpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/account"
)

// ReadAccountRepositories groups all repository dependencies
type ReadAccountRepositories struct {
	Account accountpb.AccountDomainServiceServer // Primary entity repository
}

// ReadAccountServices groups all business service dependencies
type ReadAccountServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ReadAccountUseCase handles the business logic for reading accounts
type ReadAccountUseCase struct {
	repositories ReadAccountRepositories
	services     ReadAccountServices
}

// NewReadAccountUseCase creates use case with grouped dependencies
func NewReadAccountUseCase(
	repositories ReadAccountRepositories,
	services ReadAccountServices,
) *ReadAccountUseCase {
	return &ReadAccountUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read account operation
func (uc *ReadAccountUseCase) Execute(ctx context.Context, req *accountpb.ReadAccountRequest) (*accountpb.ReadAccountResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityAccount,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	if uc.repositories.Account == nil {
		return nil, errors.New("account repository is not available")
	}
	resp, err := uc.repositories.Account.ReadAccount(ctx, req)
	if err != nil {
		return nil, err
	}

	// Return response as-is (even if empty data for not found case)
	return resp, nil
}

// validateInput validates the input request
func (uc *ReadAccountUseCase) validateInput(ctx context.Context, req *accountpb.ReadAccountRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "account.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "account.validation.data_required", "[ERR-DEFAULT] Data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "account.validation.id_required", "[ERR-DEFAULT] ID is required"))
	}
	return nil
}

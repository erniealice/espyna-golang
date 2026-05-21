package equityaccount

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	equityaccountpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/equity_account"
)

const entityEquityAccount = "equity_account"

// CreateEquityAccountRepositories groups all repository dependencies.
type CreateEquityAccountRepositories struct {
	EquityAccount equityaccountpb.EquityAccountDomainServiceServer
}

// CreateEquityAccountServices groups all business service dependencies.
type CreateEquityAccountServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateEquityAccountUseCase handles the business logic for creating equity accounts.
type CreateEquityAccountUseCase struct {
	repositories CreateEquityAccountRepositories
	services     CreateEquityAccountServices
}

// NewCreateEquityAccountUseCase creates the use case with grouped dependencies.
func NewCreateEquityAccountUseCase(
	repositories CreateEquityAccountRepositories,
	services CreateEquityAccountServices,
) *CreateEquityAccountUseCase {
	return &CreateEquityAccountUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create equity account operation.
func (uc *CreateEquityAccountUseCase) Execute(ctx context.Context, req *equityaccountpb.CreateEquityAccountRequest) (*equityaccountpb.CreateEquityAccountResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityEquityAccount, ports.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	return uc.executeCore(ctx, req)
}

func (uc *CreateEquityAccountUseCase) executeWithTransaction(ctx context.Context, req *equityaccountpb.CreateEquityAccountRequest) (*equityaccountpb.CreateEquityAccountResponse, error) {
	var result *equityaccountpb.CreateEquityAccountResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "equity_account.errors.creation_failed", "Equity account creation failed [DEFAULT]")
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

func (uc *CreateEquityAccountUseCase) executeCore(ctx context.Context, req *equityaccountpb.CreateEquityAccountRequest) (*equityaccountpb.CreateEquityAccountResponse, error) {
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	if err := uc.enrichData(req.Data); err != nil {
		return nil, err
	}

	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	if uc.repositories.EquityAccount == nil {
		return nil, errors.New("equity_account repository is not available")
	}
	return uc.repositories.EquityAccount.CreateEquityAccount(ctx, req)
}

func (uc *CreateEquityAccountUseCase) validateInput(ctx context.Context, req *equityaccountpb.CreateEquityAccountRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "equity_account.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "equity_account.validation.data_required", "[ERR-DEFAULT] Equity account data is required"))
	}

	req.Data.Name = strings.TrimSpace(req.Data.Name)

	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "equity_account.validation.name_required", "[ERR-DEFAULT] Name is required"))
	}
	return nil
}

func (uc *CreateEquityAccountUseCase) enrichData(account *equityaccountpb.EquityAccount) error {
	now := time.Now()

	if account.Id == "" {
		account.Id = uc.services.IDService.GenerateID()
	}

	account.DateCreated = &[]int64{now.UnixMilli()}[0]
	account.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	account.DateModified = &[]int64{now.UnixMilli()}[0]
	account.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	account.Active = true

	return nil
}

func (uc *CreateEquityAccountUseCase) validateBusinessRules(ctx context.Context, account *equityaccountpb.EquityAccount) error {
	if len(account.Name) > 200 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "equity_account.validation.name_too_long", "[ERR-DEFAULT] Name must not exceed 200 characters"))
	}
	return nil
}

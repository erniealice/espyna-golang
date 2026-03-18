package equitytransaction

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	equitytransactionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/equity_transaction"
)

const entityEquityTransaction = "equity_transaction"

// CreateEquityTransactionRepositories groups all repository dependencies.
type CreateEquityTransactionRepositories struct {
	EquityTransaction equitytransactionpb.EquityTransactionDomainServiceServer
}

// CreateEquityTransactionServices groups all business service dependencies.
type CreateEquityTransactionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateEquityTransactionUseCase handles the business logic for creating equity transactions.
//
// Business rule: the transaction type determines the double-entry journal entry generated:
//   - CONTRIBUTION  → DR Cash / CR Owner's Capital
//   - WITHDRAWAL    → DR Owner's Draw / CR Cash
//   - DISTRIBUTION  → DR Retained Earnings / CR Cash
//   - TRANSFER      → DR Source Equity Account / CR Destination Equity Account
//
// The caller (HTTP handler) is responsible for creating the corresponding JournalEntry
// and linking journal_entry_id on the EquityTransaction before passing the request.
type CreateEquityTransactionUseCase struct {
	repositories CreateEquityTransactionRepositories
	services     CreateEquityTransactionServices
}

// NewCreateEquityTransactionUseCase creates the use case with grouped dependencies.
func NewCreateEquityTransactionUseCase(
	repositories CreateEquityTransactionRepositories,
	services CreateEquityTransactionServices,
) *CreateEquityTransactionUseCase {
	return &CreateEquityTransactionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create equity transaction operation.
func (uc *CreateEquityTransactionUseCase) Execute(ctx context.Context, req *equitytransactionpb.CreateEquityTransactionRequest) (*equitytransactionpb.CreateEquityTransactionResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityEquityTransaction, ports.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	return uc.executeCore(ctx, req)
}

func (uc *CreateEquityTransactionUseCase) executeWithTransaction(ctx context.Context, req *equitytransactionpb.CreateEquityTransactionRequest) (*equitytransactionpb.CreateEquityTransactionResponse, error) {
	var result *equitytransactionpb.CreateEquityTransactionResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "equity_transaction.errors.creation_failed", "Equity transaction creation failed [DEFAULT]")
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

func (uc *CreateEquityTransactionUseCase) executeCore(ctx context.Context, req *equitytransactionpb.CreateEquityTransactionRequest) (*equitytransactionpb.CreateEquityTransactionResponse, error) {
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	if err := uc.enrichData(req.Data); err != nil {
		return nil, err
	}

	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	if uc.repositories.EquityTransaction == nil {
		return nil, errors.New("equity_transaction repository is not available")
	}
	return uc.repositories.EquityTransaction.CreateEquityTransaction(ctx, req)
}

func (uc *CreateEquityTransactionUseCase) validateInput(ctx context.Context, req *equitytransactionpb.CreateEquityTransactionRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "equity_transaction.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "equity_transaction.validation.data_required", "[ERR-DEFAULT] Equity transaction data is required"))
	}
	if req.Data.EquityAccountId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "equity_transaction.validation.account_required", "[ERR-DEFAULT] Equity account ID is required"))
	}
	if req.Data.Amount <= 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "equity_transaction.validation.amount_positive", "[ERR-DEFAULT] Amount must be greater than zero"))
	}
	if req.Data.TransactionType == equitytransactionpb.EquityTransactionType_EQUITY_TRANSACTION_TYPE_UNSPECIFIED {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "equity_transaction.validation.type_required", "[ERR-DEFAULT] Transaction type is required"))
	}
	return nil
}

func (uc *CreateEquityTransactionUseCase) enrichData(txn *equitytransactionpb.EquityTransaction) error {
	now := time.Now()

	if txn.Id == "" {
		txn.Id = uc.services.IDService.GenerateID()
	}

	// Set transaction date to now if not provided
	if txn.TransactionDate == 0 {
		txn.TransactionDate = now.UnixMilli()
	}
	dateStr := time.UnixMilli(txn.TransactionDate).Format("2006-01-02")
	txn.TransactionDateString = &dateStr

	txn.DateCreated = &[]int64{now.UnixMilli()}[0]
	txn.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

func (uc *CreateEquityTransactionUseCase) validateBusinessRules(ctx context.Context, txn *equitytransactionpb.EquityTransaction) error {
	if txn.Description != nil && len(*txn.Description) > 500 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "equity_transaction.validation.description_too_long", "[ERR-DEFAULT] Description must not exceed 500 characters"))
	}
	return nil
}

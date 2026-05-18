// Package fund_transaction contains use cases for the FundTransaction entity in the funding domain.
//
// FundTransaction is an append-only event log for all money movements on a Fund.
// workspace_id is nullable — fund-global events (OPENING_BALANCE, LIMIT_INCREASE,
// LIMIT_DECREASE) have no workspace attribution; workspace-originated events
// (DRAW, SETTLEMENT, CASH_IN, CASH_OUT, TRANSFER_*) carry workspace_id.
//
// Append-only semantics: only status transitions are permitted via Update
// (DRAFT → POSTED → VOIDED). All other mutations violate the event-sourcing contract.
// Corrections must be made by inserting a *_REVERSAL row with reverses_id set.
// This constraint will be enforced in FS-E projection use cases.
package fund_transaction

import (
	"context"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	fundtransactionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/funding/fund_transaction"
)

const entityFundTransaction = "fund_transaction"

// FundTransactionRepositories groups repository dependencies.
type FundTransactionRepositories struct {
	FundTransaction fundtransactionpb.FundTransactionDomainServiceServer
}

// FundTransactionServices groups service dependencies.
type FundTransactionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all fund_transaction use cases.
type UseCases struct {
	Create *CreateFundTransactionUseCase
	Read   *ReadFundTransactionUseCase
	Update *UpdateFundTransactionUseCase
	Delete *DeleteFundTransactionUseCase
	List   *ListFundTransactionsUseCase
}

// NewUseCases creates a new collection of fund_transaction use cases.
func NewUseCases(repos FundTransactionRepositories, services FundTransactionServices) *UseCases {
	return &UseCases{
		Create: &CreateFundTransactionUseCase{repo: repos.FundTransaction, services: services},
		Read:   &ReadFundTransactionUseCase{repo: repos.FundTransaction, services: services},
		Update: &UpdateFundTransactionUseCase{repo: repos.FundTransaction, services: services},
		Delete: &DeleteFundTransactionUseCase{repo: repos.FundTransaction, services: services},
		List:   &ListFundTransactionsUseCase{repo: repos.FundTransaction, services: services},
	}
}

// CreateFundTransactionUseCase handles creating a fund_transaction.
type CreateFundTransactionUseCase struct {
	repo     fundtransactionpb.FundTransactionDomainServiceServer
	services FundTransactionServices
}

func (uc *CreateFundTransactionUseCase) Execute(ctx context.Context, req *fundtransactionpb.CreateFundTransactionRequest) (*fundtransactionpb.CreateFundTransactionResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityFundTransaction, ports.ActionCreate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("fund_transaction data is required")
	}
	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDService.GenerateID()
	}
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.Active = true
	// posted_at records server insertion time (separate from effective_at which is caller-supplied).
	if req.Data.PostedAt == 0 {
		req.Data.PostedAt = now.UnixMilli()
	}
	return uc.repo.CreateFundTransaction(ctx, req)
}

// ReadFundTransactionUseCase handles reading a fund_transaction.
type ReadFundTransactionUseCase struct {
	repo     fundtransactionpb.FundTransactionDomainServiceServer
	services FundTransactionServices
}

func (uc *ReadFundTransactionUseCase) Execute(ctx context.Context, req *fundtransactionpb.ReadFundTransactionRequest) (*fundtransactionpb.ReadFundTransactionResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityFundTransaction, ports.ActionRead); err != nil {
		return nil, err
	}
	return uc.repo.ReadFundTransaction(ctx, req)
}

// UpdateFundTransactionUseCase handles status transitions on a fund_transaction.
//
// IMPORTANT: FundTransaction is append-only (architecture.md §3.10). This use
// case permits only status transitions (DRAFT → POSTED → VOIDED). Full mutation
// enforcement will be implemented in FS-E use cases. For now callers are expected
// to only pass a status change — other field mutations are silently accepted at the
// adapter layer but are considered a contract violation.
type UpdateFundTransactionUseCase struct {
	repo     fundtransactionpb.FundTransactionDomainServiceServer
	services FundTransactionServices
}

func (uc *UpdateFundTransactionUseCase) Execute(ctx context.Context, req *fundtransactionpb.UpdateFundTransactionRequest) (*fundtransactionpb.UpdateFundTransactionResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityFundTransaction, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("fund_transaction ID is required")
	}
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	return uc.repo.UpdateFundTransaction(ctx, req)
}

// DeleteFundTransactionUseCase handles deleting a fund_transaction.
type DeleteFundTransactionUseCase struct {
	repo     fundtransactionpb.FundTransactionDomainServiceServer
	services FundTransactionServices
}

func (uc *DeleteFundTransactionUseCase) Execute(ctx context.Context, req *fundtransactionpb.DeleteFundTransactionRequest) (*fundtransactionpb.DeleteFundTransactionResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityFundTransaction, ports.ActionDelete); err != nil {
		return nil, err
	}
	return uc.repo.DeleteFundTransaction(ctx, req)
}

// ListFundTransactionsUseCase handles listing fund_transactions.
type ListFundTransactionsUseCase struct {
	repo     fundtransactionpb.FundTransactionDomainServiceServer
	services FundTransactionServices
}

func (uc *ListFundTransactionsUseCase) Execute(ctx context.Context, req *fundtransactionpb.ListFundTransactionsRequest) (*fundtransactionpb.ListFundTransactionsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityFundTransaction, ports.ActionRead); err != nil {
		return nil, err
	}
	return uc.repo.ListFundTransactions(ctx, req)
}

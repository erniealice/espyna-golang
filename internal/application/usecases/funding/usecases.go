// Package funding contains use cases for the funding domain.
// Three entities model cross-workspace shared funding sources:
//   - Fund (global — no workspace_id)
//   - FundAllocation (workspace-scoped junction)
//   - FundTransaction (append-only event log; workspace_id nullable)
package funding

import (
	fundUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/funding/fund"
	fundAllocationUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/funding/fund_allocation"
	fundTransactionUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/funding/fund_transaction"

	// Application ports
	"github.com/erniealice/espyna-golang/internal/application/ports"

	// Protobuf domain services
	fundpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/funding/fund"
	fundallocationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/funding/fund_allocation"
	fundtransactionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/funding/fund_transaction"
)

// FundingRepositories contains all funding domain repositories.
type FundingRepositories struct {
	Fund            fundpb.FundDomainServiceServer
	FundAllocation  fundallocationpb.FundAllocationDomainServiceServer
	FundTransaction fundtransactionpb.FundTransactionDomainServiceServer
}

// FundingUseCases contains all funding-related use cases.
type FundingUseCases struct {
	Fund            *fundUseCases.UseCases
	FundAllocation  *fundAllocationUseCases.UseCases
	FundTransaction *fundTransactionUseCases.UseCases
}

// NewUseCases creates all funding use cases with proper constructor injection.
func NewUseCases(
	repos FundingRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idService ports.IDService,
) *FundingUseCases {
	svcFund := fundUseCases.FundServices{
		AuthorizationService: authSvc,
		TransactionService:   txSvc,
		TranslationService:   i18nSvc,
		IDService:            idService,
	}
	svcAlloc := fundAllocationUseCases.FundAllocationServices{
		AuthorizationService: authSvc,
		TransactionService:   txSvc,
		TranslationService:   i18nSvc,
		IDService:            idService,
	}
	svcTx := fundTransactionUseCases.FundTransactionServices{
		AuthorizationService: authSvc,
		TransactionService:   txSvc,
		TranslationService:   i18nSvc,
		IDService:            idService,
	}

	return &FundingUseCases{
		Fund: fundUseCases.NewUseCases(
			fundUseCases.FundRepositories{Fund: repos.Fund},
			svcFund,
		),
		FundAllocation: fundAllocationUseCases.NewUseCases(
			fundAllocationUseCases.FundAllocationRepositories{FundAllocation: repos.FundAllocation},
			svcAlloc,
		),
		FundTransaction: fundTransactionUseCases.NewUseCases(
			fundTransactionUseCases.FundTransactionRepositories{FundTransaction: repos.FundTransaction},
			svcTx,
		),
	}
}

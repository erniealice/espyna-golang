// Package funding contains use cases for the funding domain.
// Three entities model cross-workspace shared funding sources:
//   - Fund (global — no workspace_id)
//   - FundAllocation (workspace-scoped junction)
//   - FundTransaction (append-only event log; workspace_id nullable)
package funding

import (
	fundUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/funding/fund"
	fundAllocationUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/funding/fund_allocation"
	fundTransactionUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/funding/fund_transaction"

	// Application ports
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"

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
	authSvc ports.Authorizer,
	txSvc ports.Transactor,
	i18nSvc ports.Translator,
	idService ports.IDGenerator,
	actionGate *actiongate.ActionGatekeeper,
) *FundingUseCases {
	svcFund := fundUseCases.FundServices{
		Authorizer:       authSvc,
		Transactor:       txSvc,
		Translator:       i18nSvc,
		IDGenerator:      idService,
		ActionGatekeeper: actionGate,
	}
	svcAlloc := fundAllocationUseCases.FundAllocationServices{
		Authorizer:       authSvc,
		Transactor:       txSvc,
		Translator:       i18nSvc,
		IDGenerator:      idService,
		ActionGatekeeper: actionGate,
	}
	svcTx := fundTransactionUseCases.FundTransactionServices{
		Authorizer:       authSvc,
		Transactor:       txSvc,
		Translator:       i18nSvc,
		IDGenerator:      idService,
		ActionGatekeeper: actionGate,
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

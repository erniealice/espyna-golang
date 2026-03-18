package domain

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"

	// Protobuf domain services - Treasury domain
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
	disbursementpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement"
	loanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/loan"
	loanpaymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/loan_payment"
	pettycashfundpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/petty_cash_fund"
	pettycashreplenishmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/petty_cash_replenishment"
	pettycashvoucherpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/petty_cash_voucher"
	securitydepositpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/security_deposit"
)

// TreasuryRepositories contains all treasury domain repositories
type TreasuryRepositories struct {
	// Existing treasury repositories
	Collection   collectionpb.CollectionDomainServiceServer
	Disbursement disbursementpb.DisbursementDomainServiceServer

	// Loans & Petty Cash repositories
	Loan                   loanpb.LoanDomainServiceServer
	LoanPayment            loanpaymentpb.LoanPaymentDomainServiceServer
	SecurityDeposit        securitydepositpb.SecurityDepositDomainServiceServer
	PettyCashFund          pettycashfundpb.PettyCashFundDomainServiceServer
	PettyCashVoucher       pettycashvoucherpb.PettyCashVoucherDomainServiceServer
	PettyCashReplenishment pettycashreplenishmentpb.PettyCashReplenishmentDomainServiceServer
}

// NewTreasuryRepositories creates and returns a new set of TreasuryRepositories.
// Individual repository failures are logged but do not prevent other repositories
// from being created (graceful degradation per-repository).
func NewTreasuryRepositories(dbProvider contracts.Provider, tableConfig *registry.TableConfig) (*TreasuryRepositories, error) {
	if dbProvider == nil {
		return nil, fmt.Errorf("database provider not initialized")
	}

	repoCreator, ok := dbProvider.(contracts.RepositoryProvider)
	if !ok {
		return nil, fmt.Errorf("database provider doesn't implement contracts.RepositoryProvider interface")
	}

	conn := repoCreator.GetConnection()
	repos := &TreasuryRepositories{}
	var skipped []string

	// Helper: try to create a repository, log and skip on failure
	tryCreate := func(entity string) interface{} {
		repo, err := repoCreator.CreateRepository(entity, conn, tableConfig.TableName(entity))
		if err != nil {
			skipped = append(skipped, entity)
			return nil
		}
		return repo
	}

	// Existing treasury repositories
	if r := tryCreate(entityid.TreasuryCollection); r != nil {
		repos.Collection = r.(collectionpb.CollectionDomainServiceServer)
	}
	if r := tryCreate(entityid.TreasuryDisbursement); r != nil {
		repos.Disbursement = r.(disbursementpb.DisbursementDomainServiceServer)
	}

	// Loans & Petty Cash repositories
	if r := tryCreate(entityid.Loan); r != nil {
		repos.Loan = r.(loanpb.LoanDomainServiceServer)
	}
	if r := tryCreate(entityid.LoanPayment); r != nil {
		repos.LoanPayment = r.(loanpaymentpb.LoanPaymentDomainServiceServer)
	}
	if r := tryCreate(entityid.SecurityDeposit); r != nil {
		repos.SecurityDeposit = r.(securitydepositpb.SecurityDepositDomainServiceServer)
	}
	if r := tryCreate(entityid.PettyCashFund); r != nil {
		repos.PettyCashFund = r.(pettycashfundpb.PettyCashFundDomainServiceServer)
	}
	if r := tryCreate(entityid.PettyCashVoucher); r != nil {
		repos.PettyCashVoucher = r.(pettycashvoucherpb.PettyCashVoucherDomainServiceServer)
	}
	if r := tryCreate(entityid.PettyCashReplenishment); r != nil {
		repos.PettyCashReplenishment = r.(pettycashreplenishmentpb.PettyCashReplenishmentDomainServiceServer)
	}

	if len(skipped) > 0 {
		fmt.Printf("⚠️  Treasury repos skipped (no adapter registered): %v\n", skipped)
	}

	return repos, nil
}

package domain

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"

	// Protobuf domain services - Treasury domain
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
	disbursementpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement"
	disbursementschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement_schedule"
	loanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/loan"
	loanpaymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/loan_payment"
	pettycashfundpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/petty_cash_fund"
	pettycashreplenishmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/petty_cash_replenishment"
	pettycashvoucherpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/petty_cash_voucher"
	securitydepositpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/security_deposit"
	withholdingcertificatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/withholding_certificate"

	// Cross-domain references required by Plan B Phase 2 advance use cases
	// (AmortizeAdvanceCollection emits a Revenue row; AmortizeAdvanceDisbursement
	// emits an ExpenseRecognition row). Populated by the composition layer from
	// the Revenue + Expenditure provider blocks.
	expenserecognitionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"

	// Plan B Phase 7 — MILESTONE recognize use cases (selling + buying) anchor
	// on BillingEvent / SupplierBillingEvent rows + their junction tables.
	supplierbillingeventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_billing_event"
	billingeventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billing_event"
	collectionbillingeventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_billing_event"
	disbursementsupplierbillingeventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement_supplier_billing_event"
)

// TreasuryRepositories contains all treasury domain repositories
type TreasuryRepositories struct {
	// Existing treasury repositories
	Collection           collectionpb.CollectionDomainServiceServer
	Disbursement         disbursementpb.DisbursementDomainServiceServer
	DisbursementSchedule disbursementschedulepb.DisbursementScheduleDomainServiceServer

	// Loans & Petty Cash repositories
	Loan                   loanpb.LoanDomainServiceServer
	LoanPayment            loanpaymentpb.LoanPaymentDomainServiceServer
	SecurityDeposit        securitydepositpb.SecurityDepositDomainServiceServer
	PettyCashFund          pettycashfundpb.PettyCashFundDomainServiceServer
	PettyCashVoucher       pettycashvoucherpb.PettyCashVoucherDomainServiceServer
	PettyCashReplenishment pettycashreplenishmentpb.PettyCashReplenishmentDomainServiceServer

	// Tax extension
	WithholdingCertificate withholdingcertificatepb.WithholdingCertificateDomainServiceServer

	// Cross-domain repositories (populated by the composition layer post-construction).
	Revenue            revenuepb.RevenueDomainServiceServer
	ExpenseRecognition expenserecognitionpb.ExpenseRecognitionDomainServiceServer

	// Plan B Phase 7 — MILESTONE recognize repositories. Created via the
	// registry when the matching adapters are registered for the active
	// build; otherwise nil-safe (the use case construction guards on nil).
	BillingEvent                     billingeventpb.BillingEventDomainServiceServer
	SupplierBillingEvent             supplierbillingeventpb.SupplierBillingEventDomainServiceServer
	CollectionBillingEvent           collectionbillingeventpb.CollectionBillingEventDomainServiceServer
	DisbursementSupplierBillingEvent disbursementsupplierbillingeventpb.DisbursementSupplierBillingEventDomainServiceServer
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
	if r := tryCreate(entityid.DisbursementSchedule); r != nil {
		repos.DisbursementSchedule = r.(disbursementschedulepb.DisbursementScheduleDomainServiceServer)
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
	if r := tryCreate(entityid.WithholdingCertificate); r != nil {
		repos.WithholdingCertificate = r.(withholdingcertificatepb.WithholdingCertificateDomainServiceServer)
	}

	// Plan B Phase 7 — MILESTONE junctions + supplier_billing_event.
	// BillingEvent itself comes from the subscription domain and is wired
	// post-construction by the composition layer when available.
	if r := tryCreate(entityid.SupplierBillingEvent); r != nil {
		repos.SupplierBillingEvent = r.(supplierbillingeventpb.SupplierBillingEventDomainServiceServer)
	}
	if r := tryCreate(entityid.CollectionBillingEvent); r != nil {
		repos.CollectionBillingEvent = r.(collectionbillingeventpb.CollectionBillingEventDomainServiceServer)
	}
	if r := tryCreate(entityid.DisbursementSupplierBillingEvent); r != nil {
		repos.DisbursementSupplierBillingEvent = r.(disbursementsupplierbillingeventpb.DisbursementSupplierBillingEventDomainServiceServer)
	}

	if len(skipped) > 0 {
		fmt.Printf("⚠️  Treasury repos skipped (no adapter registered): %v\n", skipped)
	}

	return repos, nil
}

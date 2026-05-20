package treasury

import (
	collectionUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/treasury/collection"
	disbursementUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/treasury/disbursement"
	disbursementscheduleUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/treasury/disbursement_schedule"
	pettyCashUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/treasury/petty_cash"
	securityDepositUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/treasury/security_deposit"
	withholdingCertificateUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/treasury/withholding_certificate"

	"github.com/erniealice/espyna-golang/internal/application/ports"

	// Protobuf domain services for treasury repositories
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

	// Cross-domain repositories required by advance use cases.
	expenserecognitionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"

	// MILESTONE recognize use cases anchor on BillingEvent / SupplierBillingEvent
	// rows + their junction tables.
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

	// Cross-domain repositories required by advance use cases
	// (AmortizeAdvanceCollection emits a Revenue row; AmortizeAdvanceDisbursement
	// emits an ExpenseRecognition row). Populated by the composition layer.
	Revenue            revenuepb.RevenueDomainServiceServer
	ExpenseRecognition expenserecognitionpb.ExpenseRecognitionDomainServiceServer

	// MILESTONE recognize use cases (selling + buying) require the BillingEvent /
	// SupplierBillingEvent reads + their junction tables. Optional; nil-safe.
	BillingEvent                             billingeventpb.BillingEventDomainServiceServer
	SupplierBillingEvent                     supplierbillingeventpb.SupplierBillingEventDomainServiceServer
	CollectionBillingEvent           collectionbillingeventpb.CollectionBillingEventDomainServiceServer
	DisbursementSupplierBillingEvent disbursementsupplierbillingeventpb.DisbursementSupplierBillingEventDomainServiceServer
}

// TreasuryUseCases contains all treasury-related use cases.
//
// 20260518-hexagonal-strict-adherence Phase 1.C — the 11 advance-related use
// cases formerly carried as flat top-level fields have been folded back into
// the entity sub-aggregates (.Collection and .Disbursement). F6 closes here.
// The cross-entity GetAdvancesDashboard use case is replaced by two entity-
// side ListAdvancesForDashboard use cases (F5).
type TreasuryUseCases struct {
	Collection             *collectionUseCases.UseCases
	Disbursement           *disbursementUseCases.UseCases
	DisbursementSchedule   *disbursementscheduleUseCases.UseCases
	SecurityDeposit        *securityDepositUseCases.UseCases
	PettyCash              *pettyCashUseCases.UseCases
	WithholdingCertificate *withholdingCertificateUseCases.UseCases

	// LoanDashboard + CashDashboard fields retired 2026-05-21 (Wave B P1.C.5
	// unified Treasury candidate) — both surfaces now live under
	// `service.Dashboard.Treasury.Loan` and `service.Dashboard.Treasury.Cash`.
	// The `usecases/treasury/dashboard/` + `usecases/treasury/collection/
	// dashboard/` packages are retired in the same commit per Q-SDM-
	// DASHBOARD-DOWNSTREAM and Q-SDM-DASHBOARD-COUNT.
}

// NewUseCases creates all treasury use cases with proper constructor injection.
func NewUseCases(
	repos TreasuryRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idService ports.IDService,
) *TreasuryUseCases {
	collectionUC := collectionUseCases.NewUseCases(
		collectionUseCases.CollectionRepositories{
			Collection: repos.Collection,
		},
		collectionUseCases.CollectionServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	disbursementUC := disbursementUseCases.NewUseCases(
		disbursementUseCases.DisbursementRepositories{
			Disbursement: repos.Disbursement,
		},
		disbursementUseCases.DisbursementServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	var disbursementScheduleUC *disbursementscheduleUseCases.UseCases
	if repos.DisbursementSchedule != nil {
		disbursementScheduleUC = disbursementscheduleUseCases.NewUseCases(
			disbursementscheduleUseCases.DisbursementScheduleRepositories{
				DisbursementSchedule: repos.DisbursementSchedule,
			},
			disbursementscheduleUseCases.DisbursementScheduleServices{
				AuthorizationService: authSvc,
				TransactionService:   txSvc,
				TranslationService:   i18nSvc,
				IDService:            idService,
			},
		)
	}

	securityDepositUC := securityDepositUseCases.NewUseCases(
		securityDepositUseCases.SecurityDepositRepositories{
			SecurityDeposit: repos.SecurityDeposit,
		},
		securityDepositUseCases.SecurityDepositServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	pettyCashUC := pettyCashUseCases.NewUseCases(
		pettyCashUseCases.PettyCashRepositories{
			PettyCashFund: repos.PettyCashFund,
		},
		pettyCashUseCases.PettyCashServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	// Loan + Cash dashboard wiring retired 2026-05-21 (Wave B P1.C.5 unified
	// Treasury candidate) — type-assertion + factory wiring now lives in the
	// service-layer initializer at `internal/composition/core/initializers/
	// service.go` (search "Wave B P1.C.5 Treasury").

	withholdingCertificateUC := withholdingCertificateUseCases.NewUseCases(
		withholdingCertificateUseCases.WithholdingCertificateRepositories{
			WithholdingCertificate: repos.WithholdingCertificate,
		},
		withholdingCertificateUseCases.WithholdingCertificateServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	// --------------------------------------------------------------------------
	// 20260518-hexagonal-strict-adherence Phase 1.C — advance use cases are now
	// nested under .Collection / .Disbursement sub-aggregates (F6).
	//
	// Q1-B (LOCKED): each advance workflow routes its terminal Update through
	// the wrapping Update use case (belt-and-suspenders — BURN_DOWN guard now
	// lives at the wrapping use case layer). The wrapping Update is itself
	// transaction-aware (see C-iv-pre); when called from inside an advance
	// workflow's own ExecuteInTransaction the wrapper short-circuits to
	// executeCore so no nested independent tx is started.
	// --------------------------------------------------------------------------

	if repos.Collection != nil && collectionUC != nil {
		collectionUC.AmortizeAdvance = collectionUseCases.NewAmortizeAdvanceCollectionUseCase(
			collectionUseCases.AmortizeAdvanceCollectionRepositories{
				TreasuryCollection: repos.Collection,
				Revenue:            repos.Revenue,
			},
			collectionUseCases.AmortizeAdvanceCollectionServices{
				AuthorizationService: authSvc,
				TransactionService:   txSvc,
				TranslationService:   i18nSvc,
				IDService:            idService,
			},
			collectionUC.UpdateCollection,
		)

		collectionUC.SettleUnscheduledAdvance = collectionUseCases.NewSettleUnscheduledAdvanceUseCase(
			collectionUseCases.SettleUnscheduledAdvanceRepositories{
				TreasuryCollection: repos.Collection,
			},
			collectionUseCases.SettleUnscheduledAdvanceServices{
				AuthorizationService: authSvc,
				TransactionService:   txSvc,
				TranslationService:   i18nSvc,
			},
			collectionUC.UpdateCollection,
		)

		collectionUC.RefundUnscheduledAdvance = collectionUseCases.NewRefundUnscheduledAdvanceUseCase(
			collectionUseCases.RefundUnscheduledAdvanceRepositories{
				TreasuryCollection: repos.Collection,
			},
			collectionUseCases.RefundUnscheduledAdvanceServices{
				AuthorizationService: authSvc,
				TransactionService:   txSvc,
				TranslationService:   i18nSvc,
			},
			collectionUC.UpdateCollection,
		)

		collectionUC.CancelAdvance = collectionUseCases.NewCancelAdvanceUseCase(
			collectionUseCases.CancelAdvanceRepositories{
				TreasuryCollection: repos.Collection,
			},
			collectionUseCases.CancelAdvanceServices{
				AuthorizationService: authSvc,
				TransactionService:   txSvc,
				TranslationService:   i18nSvc,
			},
			collectionUC.UpdateCollection,
		)

		if repos.Revenue != nil && repos.BillingEvent != nil && repos.CollectionBillingEvent != nil {
			collectionUC.RecognizeMilestoneAdvance = collectionUseCases.NewRecognizeMilestoneAdvanceCollectionUseCase(
				collectionUseCases.RecognizeMilestoneAdvanceCollectionRepositories{
					TreasuryCollection:             repos.Collection,
					Revenue:                        repos.Revenue,
					BillingEvent:                   repos.BillingEvent,
					CollectionBillingEvent: repos.CollectionBillingEvent,
				},
				collectionUseCases.RecognizeMilestoneAdvanceCollectionServices{
					AuthorizationService: authSvc,
					TransactionService:   txSvc,
					TranslationService:   i18nSvc,
					IDService:            idService,
				},
				collectionUC.UpdateCollection,
			)
		}

		collectionUC.ListAdvancesForDashboard = collectionUseCases.NewListAdvanceCollectionsForDashboardUseCase(
			collectionUseCases.ListAdvanceCollectionsForDashboardRepositories{
				Collection: repos.Collection,
			},
			collectionUseCases.ListAdvanceCollectionsForDashboardServices{
				AuthorizationService: authSvc,
				TranslationService:   i18nSvc,
			},
		)
	}

	if repos.Disbursement != nil && disbursementUC != nil {
		disbursementUC.AmortizeAdvance = disbursementUseCases.NewAmortizeAdvanceDisbursementUseCase(
			disbursementUseCases.AmortizeAdvanceDisbursementRepositories{
				TreasuryDisbursement: repos.Disbursement,
				ExpenseRecognition:   repos.ExpenseRecognition,
			},
			disbursementUseCases.AmortizeAdvanceDisbursementServices{
				AuthorizationService: authSvc,
				TransactionService:   txSvc,
				TranslationService:   i18nSvc,
				IDService:            idService,
			},
			disbursementUC.UpdateDisbursement,
		)

		disbursementUC.SettleUnscheduledAdvance = disbursementUseCases.NewSettleUnscheduledAdvanceUseCase(
			disbursementUseCases.SettleUnscheduledAdvanceRepositories{
				TreasuryDisbursement: repos.Disbursement,
			},
			disbursementUseCases.SettleUnscheduledAdvanceServices{
				AuthorizationService: authSvc,
				TransactionService:   txSvc,
				TranslationService:   i18nSvc,
			},
			disbursementUC.UpdateDisbursement,
		)

		disbursementUC.RefundUnscheduledAdvance = disbursementUseCases.NewRefundUnscheduledAdvanceUseCase(
			disbursementUseCases.RefundUnscheduledAdvanceRepositories{
				TreasuryDisbursement: repos.Disbursement,
			},
			disbursementUseCases.RefundUnscheduledAdvanceServices{
				AuthorizationService: authSvc,
				TransactionService:   txSvc,
				TranslationService:   i18nSvc,
			},
			disbursementUC.UpdateDisbursement,
		)

		disbursementUC.CancelAdvance = disbursementUseCases.NewCancelAdvanceUseCase(
			disbursementUseCases.CancelAdvanceRepositories{
				TreasuryDisbursement: repos.Disbursement,
			},
			disbursementUseCases.CancelAdvanceServices{
				AuthorizationService: authSvc,
				TransactionService:   txSvc,
				TranslationService:   i18nSvc,
			},
			disbursementUC.UpdateDisbursement,
		)

		if repos.ExpenseRecognition != nil && repos.SupplierBillingEvent != nil && repos.DisbursementSupplierBillingEvent != nil {
			disbursementUC.RecognizeMilestoneAdvance = disbursementUseCases.NewRecognizeMilestoneAdvanceDisbursementUseCase(
				disbursementUseCases.RecognizeMilestoneAdvanceDisbursementRepositories{
					TreasuryDisbursement:                     repos.Disbursement,
					ExpenseRecognition:                       repos.ExpenseRecognition,
					SupplierBillingEvent:                     repos.SupplierBillingEvent,
					DisbursementSupplierBillingEvent: repos.DisbursementSupplierBillingEvent,
				},
				disbursementUseCases.RecognizeMilestoneAdvanceDisbursementServices{
					AuthorizationService: authSvc,
					TransactionService:   txSvc,
					TranslationService:   i18nSvc,
					IDService:            idService,
				},
				disbursementUC.UpdateDisbursement,
			)
		}

		disbursementUC.ListAdvancesForDashboard = disbursementUseCases.NewListAdvanceDisbursementsForDashboardUseCase(
			disbursementUseCases.ListAdvanceDisbursementsForDashboardRepositories{
				Disbursement: repos.Disbursement,
			},
			disbursementUseCases.ListAdvanceDisbursementsForDashboardServices{
				AuthorizationService: authSvc,
				TranslationService:   i18nSvc,
			},
		)
	}

	return &TreasuryUseCases{
		Collection:             collectionUC,
		Disbursement:           disbursementUC,
		DisbursementSchedule:   disbursementScheduleUC,
		SecurityDeposit:        securityDepositUC,
		PettyCash:              pettyCashUC,
		WithholdingCertificate: withholdingCertificateUC,
	}
}

package treasury

import (
	// Collection use cases
	collectionUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/treasury/collection"
	// Disbursement use cases
	disbursementUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/treasury/disbursement"
	// DisbursementSchedule use cases
	disbursementscheduleUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/treasury/disbursement_schedule"
	// PettyCash use cases
	pettyCashUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/treasury/petty_cash"
	// SecurityDeposit use cases
	securityDepositUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/treasury/security_deposit"
	// TreasuryCollection (advance Plan B Phase 2) use cases
	treasurycollectionUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/treasury/treasury_collection"
	// TreasuryDisbursement (advance Plan B Phase 2) use cases
	treasurydisbursementUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/treasury/treasury_disbursement"
	// WithholdingCertificate use cases
	withholdingCertificateUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/treasury/withholding_certificate"

	// Application ports
	"github.com/erniealice/espyna-golang/internal/application/ports"

	// Dashboard use cases
	cashdashboard "github.com/erniealice/espyna-golang/internal/application/usecases/treasury/collection/dashboard"
	loandashboard "github.com/erniealice/espyna-golang/internal/application/usecases/treasury/dashboard"

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

	// Cross-domain repositories required by Plan B Phase 2 advance use cases.
	expenserecognitionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"

	// Plan B Phase 7 — MILESTONE recognize use cases anchor on BillingEvent /
	// SupplierBillingEvent rows + their junction tables.
	supplierbillingeventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_billing_event"
	billingeventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billing_event"
	treasurycollectionbillingeventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/treasury_collection_billing_event"
	treasurydisbursementsupplierbillingeventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/treasury_disbursement_supplier_billing_event"
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

	// Cross-domain repositories required by Plan B Phase 2 advance use cases
	// (AmortizeAdvanceCollection emits a Revenue row; AmortizeAdvanceDisbursement
	// emits an ExpenseRecognition row). Populated by the composition layer.
	Revenue            revenuepb.RevenueDomainServiceServer
	ExpenseRecognition expenserecognitionpb.ExpenseRecognitionDomainServiceServer

	// Plan B Phase 7 — MILESTONE recognize use cases (selling + buying)
	// require the BillingEvent / SupplierBillingEvent reads + their junction
	// tables. Optional; when nil, the corresponding RecognizeMilestoneAdvance
	// use case construction is skipped (nil-safe).
	BillingEvent                             billingeventpb.BillingEventDomainServiceServer
	SupplierBillingEvent                     supplierbillingeventpb.SupplierBillingEventDomainServiceServer
	TreasuryCollectionBillingEvent           treasurycollectionbillingeventpb.TreasuryCollectionBillingEventDomainServiceServer
	TreasuryDisbursementSupplierBillingEvent treasurydisbursementsupplierbillingeventpb.TreasuryDisbursementSupplierBillingEventDomainServiceServer
}

// TreasuryUseCases contains all treasury-related use cases
type TreasuryUseCases struct {
	Collection             *collectionUseCases.UseCases
	Disbursement           *disbursementUseCases.UseCases
	DisbursementSchedule   *disbursementscheduleUseCases.UseCases
	SecurityDeposit        *securityDepositUseCases.UseCases
	PettyCash              *pettyCashUseCases.UseCases
	WithholdingCertificate *withholdingCertificateUseCases.UseCases
	// Loans — use cases to be created in future iterations
	// Loan, LoanPayment, PettyCashVoucher, PettyCashReplenishment

	// Dashboard use cases (nil when postgres build tag is inactive).
	LoanDashboard *loandashboard.GetLoanDashboardPageDataUseCase
	CashDashboard *cashdashboard.GetCashDashboardPageDataUseCase

	// 20260517-advance-cash-events Plan B Phase 2 — advance use cases (BOTH sides).
	// Constructed unconditionally; nil-safe via the existing repo nil-guards.
	AmortizeAdvanceCollection           *treasurycollectionUseCases.AmortizeAdvanceCollectionUseCase
	AmortizeAdvanceDisbursement         *treasurydisbursementUseCases.AmortizeAdvanceDisbursementUseCase
	SettleUnscheduledAdvanceCollection  *treasurycollectionUseCases.SettleUnscheduledAdvanceUseCase
	SettleUnscheduledAdvanceDisbursement *treasurydisbursementUseCases.SettleUnscheduledAdvanceUseCase
	RefundUnscheduledAdvanceCollection  *treasurycollectionUseCases.RefundUnscheduledAdvanceUseCase
	RefundUnscheduledAdvanceDisbursement *treasurydisbursementUseCases.RefundUnscheduledAdvanceUseCase
	CancelAdvanceCollection             *treasurycollectionUseCases.CancelAdvanceUseCase
	CancelAdvanceDisbursement           *treasurydisbursementUseCases.CancelAdvanceUseCase
	// 20260517-advance-cash-events Plan B Phase 7 — MILESTONE recognize use
	// cases (selling + buying). Nil when the BillingEvent / SupplierBillingEvent
	// repositories are not wired (e.g., mock/firestore deployments without
	// these adapters registered).
	RecognizeMilestoneAdvanceCollection   *treasurycollectionUseCases.RecognizeMilestoneAdvanceCollectionUseCase
	RecognizeMilestoneAdvanceDisbursement *treasurydisbursementUseCases.RecognizeMilestoneAdvanceDisbursementUseCase
	GetAdvancesDashboard                  *GetAdvancesDashboardUseCase
}

// NewUseCases creates all treasury use cases with proper constructor injection
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

	// Wire loan dashboard via type assertions on loan repos.
	var loanDash *loandashboard.GetLoanDashboardPageDataUseCase
	if repos.Loan != nil && repos.LoanPayment != nil {
		loanQ, lOK := repos.Loan.(loandashboard.LoanDashboardQueries)
		pmtQ, pOK := repos.LoanPayment.(loandashboard.LoanPaymentDashboardQueries)
		if lOK && pOK {
			loanDash = loandashboard.NewGetLoanDashboardPageDataUseCase(loanQ, pmtQ)
		}
	}

	// Wire cash dashboard via type assertion on collection repo.
	var cashDash *cashdashboard.GetCashDashboardPageDataUseCase
	if repos.Collection != nil {
		if collQ, ok := repos.Collection.(cashdashboard.CollectionDashboardQueries); ok {
			cashDash = cashdashboard.NewGetCashDashboardPageDataUseCase(collQ)
		}
	}

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

	// 20260517-advance-cash-events Plan B Phase 2 — wire advance use cases (BOTH sides).
	var amortizeAdvCol *treasurycollectionUseCases.AmortizeAdvanceCollectionUseCase
	if repos.Collection != nil {
		amortizeAdvCol = treasurycollectionUseCases.NewAmortizeAdvanceCollectionUseCase(
			treasurycollectionUseCases.AmortizeAdvanceCollectionRepositories{
				TreasuryCollection: repos.Collection,
				Revenue:            repos.Revenue,
			},
			treasurycollectionUseCases.AmortizeAdvanceCollectionServices{
				AuthorizationService: authSvc,
				TransactionService:   txSvc,
				TranslationService:   i18nSvc,
				IDService:            idService,
			},
		)
	}

	var amortizeAdvDis *treasurydisbursementUseCases.AmortizeAdvanceDisbursementUseCase
	if repos.Disbursement != nil {
		amortizeAdvDis = treasurydisbursementUseCases.NewAmortizeAdvanceDisbursementUseCase(
			treasurydisbursementUseCases.AmortizeAdvanceDisbursementRepositories{
				TreasuryDisbursement: repos.Disbursement,
				ExpenseRecognition:   repos.ExpenseRecognition,
			},
			treasurydisbursementUseCases.AmortizeAdvanceDisbursementServices{
				AuthorizationService: authSvc,
				TransactionService:   txSvc,
				TranslationService:   i18nSvc,
				IDService:            idService,
			},
		)
	}

	var settleUnschedCol *treasurycollectionUseCases.SettleUnscheduledAdvanceUseCase
	if repos.Collection != nil {
		settleUnschedCol = treasurycollectionUseCases.NewSettleUnscheduledAdvanceUseCase(
			treasurycollectionUseCases.SettleUnscheduledAdvanceRepositories{
				TreasuryCollection: repos.Collection,
			},
			treasurycollectionUseCases.SettleUnscheduledAdvanceServices{
				AuthorizationService: authSvc,
				TransactionService:   txSvc,
				TranslationService:   i18nSvc,
			},
		)
	}

	var settleUnschedDis *treasurydisbursementUseCases.SettleUnscheduledAdvanceUseCase
	if repos.Disbursement != nil {
		settleUnschedDis = treasurydisbursementUseCases.NewSettleUnscheduledAdvanceUseCase(
			treasurydisbursementUseCases.SettleUnscheduledAdvanceRepositories{
				TreasuryDisbursement: repos.Disbursement,
			},
			treasurydisbursementUseCases.SettleUnscheduledAdvanceServices{
				AuthorizationService: authSvc,
				TransactionService:   txSvc,
				TranslationService:   i18nSvc,
			},
		)
	}

	var refundUnschedCol *treasurycollectionUseCases.RefundUnscheduledAdvanceUseCase
	if repos.Collection != nil {
		refundUnschedCol = treasurycollectionUseCases.NewRefundUnscheduledAdvanceUseCase(
			treasurycollectionUseCases.RefundUnscheduledAdvanceRepositories{
				TreasuryCollection: repos.Collection,
			},
			treasurycollectionUseCases.RefundUnscheduledAdvanceServices{
				AuthorizationService: authSvc,
				TransactionService:   txSvc,
				TranslationService:   i18nSvc,
			},
		)
	}

	var refundUnschedDis *treasurydisbursementUseCases.RefundUnscheduledAdvanceUseCase
	if repos.Disbursement != nil {
		refundUnschedDis = treasurydisbursementUseCases.NewRefundUnscheduledAdvanceUseCase(
			treasurydisbursementUseCases.RefundUnscheduledAdvanceRepositories{
				TreasuryDisbursement: repos.Disbursement,
			},
			treasurydisbursementUseCases.RefundUnscheduledAdvanceServices{
				AuthorizationService: authSvc,
				TransactionService:   txSvc,
				TranslationService:   i18nSvc,
			},
		)
	}

	var cancelAdvCol *treasurycollectionUseCases.CancelAdvanceUseCase
	if repos.Collection != nil {
		cancelAdvCol = treasurycollectionUseCases.NewCancelAdvanceUseCase(
			treasurycollectionUseCases.CancelAdvanceRepositories{
				TreasuryCollection: repos.Collection,
			},
			treasurycollectionUseCases.CancelAdvanceServices{
				AuthorizationService: authSvc,
				TransactionService:   txSvc,
				TranslationService:   i18nSvc,
			},
		)
	}

	var cancelAdvDis *treasurydisbursementUseCases.CancelAdvanceUseCase
	if repos.Disbursement != nil {
		cancelAdvDis = treasurydisbursementUseCases.NewCancelAdvanceUseCase(
			treasurydisbursementUseCases.CancelAdvanceRepositories{
				TreasuryDisbursement: repos.Disbursement,
			},
			treasurydisbursementUseCases.CancelAdvanceServices{
				AuthorizationService: authSvc,
				TransactionService:   txSvc,
				TranslationService:   i18nSvc,
			},
		)
	}

	// 20260517-advance-cash-events Plan B Phase 7 — MILESTONE recognize use
	// cases. Nil-safe when the junction tables / billing event repositories
	// are not yet registered for the active build (mock + firestore paths).
	var recognizeMilestoneCol *treasurycollectionUseCases.RecognizeMilestoneAdvanceCollectionUseCase
	if repos.Collection != nil && repos.Revenue != nil && repos.BillingEvent != nil && repos.TreasuryCollectionBillingEvent != nil {
		recognizeMilestoneCol = treasurycollectionUseCases.NewRecognizeMilestoneAdvanceCollectionUseCase(
			treasurycollectionUseCases.RecognizeMilestoneAdvanceCollectionRepositories{
				TreasuryCollection:             repos.Collection,
				Revenue:                        repos.Revenue,
				BillingEvent:                   repos.BillingEvent,
				TreasuryCollectionBillingEvent: repos.TreasuryCollectionBillingEvent,
			},
			treasurycollectionUseCases.RecognizeMilestoneAdvanceCollectionServices{
				AuthorizationService: authSvc,
				TransactionService:   txSvc,
				TranslationService:   i18nSvc,
				IDService:            idService,
			},
		)
	}

	var recognizeMilestoneDis *treasurydisbursementUseCases.RecognizeMilestoneAdvanceDisbursementUseCase
	if repos.Disbursement != nil && repos.ExpenseRecognition != nil && repos.SupplierBillingEvent != nil && repos.TreasuryDisbursementSupplierBillingEvent != nil {
		recognizeMilestoneDis = treasurydisbursementUseCases.NewRecognizeMilestoneAdvanceDisbursementUseCase(
			treasurydisbursementUseCases.RecognizeMilestoneAdvanceDisbursementRepositories{
				TreasuryDisbursement:                     repos.Disbursement,
				ExpenseRecognition:                       repos.ExpenseRecognition,
				SupplierBillingEvent:                     repos.SupplierBillingEvent,
				TreasuryDisbursementSupplierBillingEvent: repos.TreasuryDisbursementSupplierBillingEvent,
			},
			treasurydisbursementUseCases.RecognizeMilestoneAdvanceDisbursementServices{
				AuthorizationService: authSvc,
				TransactionService:   txSvc,
				TranslationService:   i18nSvc,
				IDService:            idService,
			},
		)
	}

	advancesDash := NewGetAdvancesDashboardUseCase(
		GetAdvancesDashboardRepositories{
			TreasuryCollection:   repos.Collection,
			TreasuryDisbursement: repos.Disbursement,
		},
		GetAdvancesDashboardServices{
			AuthorizationService: authSvc,
			TranslationService:   i18nSvc,
		},
	)

	return &TreasuryUseCases{
		Collection:             collectionUC,
		Disbursement:           disbursementUC,
		DisbursementSchedule:   disbursementScheduleUC,
		SecurityDeposit:        securityDepositUC,
		PettyCash:              pettyCashUC,
		WithholdingCertificate: withholdingCertificateUC,
		LoanDashboard:          loanDash,
		CashDashboard:          cashDash,

		AmortizeAdvanceCollection:            amortizeAdvCol,
		AmortizeAdvanceDisbursement:          amortizeAdvDis,
		SettleUnscheduledAdvanceCollection:   settleUnschedCol,
		SettleUnscheduledAdvanceDisbursement: settleUnschedDis,
		RefundUnscheduledAdvanceCollection:   refundUnschedCol,
		RefundUnscheduledAdvanceDisbursement: refundUnschedDis,
		CancelAdvanceCollection:              cancelAdvCol,
		CancelAdvanceDisbursement:            cancelAdvDis,
		RecognizeMilestoneAdvanceCollection:   recognizeMilestoneCol,
		RecognizeMilestoneAdvanceDisbursement: recognizeMilestoneDis,
		GetAdvancesDashboard:                 advancesDash,
	}
}

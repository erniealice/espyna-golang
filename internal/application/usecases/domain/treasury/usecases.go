package treasury

import (
	collectionUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/treasury/collection"
	collectionMethodUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/treasury/collection_method"
	collectionMethodEligibilityRuleUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/treasury/collection_method_eligibility_rule"
	collectionMethodGrantUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/treasury/collection_method_grant"
	disbursementUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/treasury/disbursement"
	disbursementMethodUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/treasury/disbursement_method"
	disbursementscheduleUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/treasury/disbursement_schedule"
	pettyCashUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/treasury/petty_cash"
	securityDepositUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/treasury/security_deposit"
	withholdingCertificateUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/treasury/withholding_certificate"

	"github.com/erniealice/espyna-golang/internal/application/ports"

	// Protobuf domain services for treasury repositories
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
	collectionmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method"
	collectionmethodeligibilityrulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method_eligibility_rule"
	collectionmethodgrantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method_grant"
	disbursementpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement"
	disbursementmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement_method"
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

	// Treasury-domain-rebuild Stage 1 — method management templates.
	CollectionMethod   collectionmethodpb.CollectionMethodDomainServiceServer
	DisbursementMethod disbursementmethodpb.DisbursementMethodDomainServiceServer

	// Treasury-domain-rebuild Stage 2 — collection-method eligibility rule.
	CollectionMethodEligibilityRule collectionmethodeligibilityrulepb.CollectionMethodEligibilityRuleDomainServiceServer

	// Treasury-domain-rebuild Stage 3 — collection-method audience grant (CONFIG).
	CollectionMethodGrant collectionmethodgrantpb.CollectionMethodGrantDomainServiceServer

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
	BillingEvent                     billingeventpb.BillingEventDomainServiceServer
	SupplierBillingEvent             supplierbillingeventpb.SupplierBillingEventDomainServiceServer
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
	Collection                      *collectionUseCases.UseCases
	CollectionMethod                *collectionMethodUseCases.UseCases
	CollectionMethodEligibilityRule *collectionMethodEligibilityRuleUseCases.UseCases
	CollectionMethodGrant           *collectionMethodGrantUseCases.UseCases
	Disbursement                    *disbursementUseCases.UseCases
	DisbursementMethod              *disbursementMethodUseCases.UseCases
	DisbursementSchedule            *disbursementscheduleUseCases.UseCases
	SecurityDeposit                 *securityDepositUseCases.UseCases
	PettyCash                       *pettyCashUseCases.UseCases
	WithholdingCertificate          *withholdingCertificateUseCases.UseCases

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
	authSvc ports.Authorizer,
	txSvc ports.Transactor,
	i18nSvc ports.Translator,
	idService ports.IDGenerator,
) *TreasuryUseCases {
	collectionUC := collectionUseCases.NewUseCases(
		collectionUseCases.CollectionRepositories{
			Collection: repos.Collection,
		},
		collectionUseCases.CollectionServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	disbursementUC := disbursementUseCases.NewUseCases(
		disbursementUseCases.DisbursementRepositories{
			Disbursement: repos.Disbursement,
		},
		disbursementUseCases.DisbursementServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	// Treasury-domain-rebuild Stage 1 — method management templates. Nil-safe:
	// the use cases are constructed unconditionally (their internal repository
	// guards handle a nil repo), so the view-closure wiring always resolves.
	collectionMethodUC := collectionMethodUseCases.NewUseCases(
		collectionMethodUseCases.CollectionMethodRepositories{
			CollectionMethod: repos.CollectionMethod,
		},
		collectionMethodUseCases.CollectionMethodServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	// Treasury-domain-rebuild Stage 2 — collection-method eligibility rule. Nil-safe
	// (the use cases guard on a nil repo), so the view-closure wiring always resolves.
	collectionMethodEligibilityRuleUC := collectionMethodEligibilityRuleUseCases.NewUseCases(
		collectionMethodEligibilityRuleUseCases.CollectionMethodEligibilityRuleRepositories{
			CollectionMethodEligibilityRule: repos.CollectionMethodEligibilityRule,
		},
		collectionMethodEligibilityRuleUseCases.CollectionMethodEligibilityRuleServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	// Treasury-domain-rebuild Stage 3 — collection-method audience grant (CONFIG).
	// Nil-safe. The CollectionMethod template repo is injected so the audience-mode
	// guardrail (create + bulk_grant) can resolve the method's audience_mode.
	collectionMethodGrantUC := collectionMethodGrantUseCases.NewUseCases(
		collectionMethodGrantUseCases.CollectionMethodGrantRepositories{
			CollectionMethodGrant: repos.CollectionMethodGrant,
			CollectionMethod:      repos.CollectionMethod,
		},
		collectionMethodGrantUseCases.CollectionMethodGrantServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	disbursementMethodUC := disbursementMethodUseCases.NewUseCases(
		disbursementMethodUseCases.DisbursementMethodRepositories{
			DisbursementMethod: repos.DisbursementMethod,
		},
		disbursementMethodUseCases.DisbursementMethodServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	var disbursementScheduleUC *disbursementscheduleUseCases.UseCases
	if repos.DisbursementSchedule != nil {
		disbursementScheduleUC = disbursementscheduleUseCases.NewUseCases(
			disbursementscheduleUseCases.DisbursementScheduleRepositories{
				DisbursementSchedule: repos.DisbursementSchedule,
			},
			disbursementscheduleUseCases.DisbursementScheduleServices{
				Authorizer:  authSvc,
				Transactor:  txSvc,
				Translator:  i18nSvc,
				IDGenerator: idService,
			},
		)
	}

	securityDepositUC := securityDepositUseCases.NewUseCases(
		securityDepositUseCases.SecurityDepositRepositories{
			SecurityDeposit: repos.SecurityDeposit,
		},
		securityDepositUseCases.SecurityDepositServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	pettyCashUC := pettyCashUseCases.NewUseCases(
		pettyCashUseCases.PettyCashRepositories{
			PettyCashFund: repos.PettyCashFund,
		},
		pettyCashUseCases.PettyCashServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
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
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
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
				Authorizer:  authSvc,
				Transactor:  txSvc,
				Translator:  i18nSvc,
				IDGenerator: idService,
			},
			collectionUC.UpdateCollection,
		)

		collectionUC.SettleUnscheduledAdvance = collectionUseCases.NewSettleUnscheduledAdvanceUseCase(
			collectionUseCases.SettleUnscheduledAdvanceRepositories{
				TreasuryCollection: repos.Collection,
			},
			collectionUseCases.SettleUnscheduledAdvanceServices{
				Authorizer: authSvc,
				Transactor: txSvc,
				Translator: i18nSvc,
			},
			collectionUC.UpdateCollection,
		)

		collectionUC.RefundUnscheduledAdvance = collectionUseCases.NewRefundUnscheduledAdvanceUseCase(
			collectionUseCases.RefundUnscheduledAdvanceRepositories{
				TreasuryCollection: repos.Collection,
			},
			collectionUseCases.RefundUnscheduledAdvanceServices{
				Authorizer: authSvc,
				Transactor: txSvc,
				Translator: i18nSvc,
			},
			collectionUC.UpdateCollection,
		)

		collectionUC.CancelAdvance = collectionUseCases.NewCancelAdvanceUseCase(
			collectionUseCases.CancelAdvanceRepositories{
				TreasuryCollection: repos.Collection,
			},
			collectionUseCases.CancelAdvanceServices{
				Authorizer: authSvc,
				Transactor: txSvc,
				Translator: i18nSvc,
			},
			collectionUC.UpdateCollection,
		)

		if repos.Revenue != nil && repos.BillingEvent != nil && repos.CollectionBillingEvent != nil {
			collectionUC.RecognizeMilestoneAdvance = collectionUseCases.NewRecognizeMilestoneAdvanceCollectionUseCase(
				collectionUseCases.RecognizeMilestoneAdvanceCollectionRepositories{
					TreasuryCollection:     repos.Collection,
					Revenue:                repos.Revenue,
					BillingEvent:           repos.BillingEvent,
					CollectionBillingEvent: repos.CollectionBillingEvent,
				},
				collectionUseCases.RecognizeMilestoneAdvanceCollectionServices{
					Authorizer:  authSvc,
					Transactor:  txSvc,
					Translator:  i18nSvc,
					IDGenerator: idService,
				},
				collectionUC.UpdateCollection,
			)
		}

		collectionUC.ListAdvancesForDashboard = collectionUseCases.NewListAdvanceCollectionsForDashboardUseCase(
			collectionUseCases.ListAdvanceCollectionsForDashboardRepositories{
				Collection: repos.Collection,
			},
			collectionUseCases.ListAdvanceCollectionsForDashboardServices{
				Authorizer: authSvc,
				Translator: i18nSvc,
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
				Authorizer:  authSvc,
				Transactor:  txSvc,
				Translator:  i18nSvc,
				IDGenerator: idService,
			},
			disbursementUC.UpdateDisbursement,
		)

		disbursementUC.SettleUnscheduledAdvance = disbursementUseCases.NewSettleUnscheduledAdvanceUseCase(
			disbursementUseCases.SettleUnscheduledAdvanceRepositories{
				TreasuryDisbursement: repos.Disbursement,
			},
			disbursementUseCases.SettleUnscheduledAdvanceServices{
				Authorizer: authSvc,
				Transactor: txSvc,
				Translator: i18nSvc,
			},
			disbursementUC.UpdateDisbursement,
		)

		disbursementUC.RefundUnscheduledAdvance = disbursementUseCases.NewRefundUnscheduledAdvanceUseCase(
			disbursementUseCases.RefundUnscheduledAdvanceRepositories{
				TreasuryDisbursement: repos.Disbursement,
			},
			disbursementUseCases.RefundUnscheduledAdvanceServices{
				Authorizer: authSvc,
				Transactor: txSvc,
				Translator: i18nSvc,
			},
			disbursementUC.UpdateDisbursement,
		)

		disbursementUC.CancelAdvance = disbursementUseCases.NewCancelAdvanceUseCase(
			disbursementUseCases.CancelAdvanceRepositories{
				TreasuryDisbursement: repos.Disbursement,
			},
			disbursementUseCases.CancelAdvanceServices{
				Authorizer: authSvc,
				Transactor: txSvc,
				Translator: i18nSvc,
			},
			disbursementUC.UpdateDisbursement,
		)

		if repos.ExpenseRecognition != nil && repos.SupplierBillingEvent != nil && repos.DisbursementSupplierBillingEvent != nil {
			disbursementUC.RecognizeMilestoneAdvance = disbursementUseCases.NewRecognizeMilestoneAdvanceDisbursementUseCase(
				disbursementUseCases.RecognizeMilestoneAdvanceDisbursementRepositories{
					TreasuryDisbursement:             repos.Disbursement,
					ExpenseRecognition:               repos.ExpenseRecognition,
					SupplierBillingEvent:             repos.SupplierBillingEvent,
					DisbursementSupplierBillingEvent: repos.DisbursementSupplierBillingEvent,
				},
				disbursementUseCases.RecognizeMilestoneAdvanceDisbursementServices{
					Authorizer:  authSvc,
					Transactor:  txSvc,
					Translator:  i18nSvc,
					IDGenerator: idService,
				},
				disbursementUC.UpdateDisbursement,
			)
		}

		disbursementUC.ListAdvancesForDashboard = disbursementUseCases.NewListAdvanceDisbursementsForDashboardUseCase(
			disbursementUseCases.ListAdvanceDisbursementsForDashboardRepositories{
				Disbursement: repos.Disbursement,
			},
			disbursementUseCases.ListAdvanceDisbursementsForDashboardServices{
				Authorizer: authSvc,
				Translator: i18nSvc,
			},
		)
	}

	return &TreasuryUseCases{
		Collection:                      collectionUC,
		CollectionMethod:                collectionMethodUC,
		CollectionMethodEligibilityRule: collectionMethodEligibilityRuleUC,
		CollectionMethodGrant:           collectionMethodGrantUC,
		Disbursement:                    disbursementUC,
		DisbursementMethod:              disbursementMethodUC,
		DisbursementSchedule:            disbursementScheduleUC,
		SecurityDeposit:                 securityDepositUC,
		PettyCash:                       pettyCashUC,
		WithholdingCertificate:          withholdingCertificateUC,
	}
}

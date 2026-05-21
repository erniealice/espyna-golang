package expenditure

import (
	// Expenditure use cases
	accruedExpenseUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/expenditure/accrued_expense"

	accruedExpenseSettlementUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/expenditure/accrued_expense_settlement"
	expenditureUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/expenditure/expenditure"
	expenditureAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/expenditure/expenditure_attribute"
	expenditureCategoryUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/expenditure/expenditure_category"
	expenditureLineItemUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/expenditure/expenditure_line_item"
	expenseRecognitionUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/expenditure/expense_recognition"
	expenseRecognitionLineUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/expenditure/expense_recognition_line"
	expenseRecognitionRunUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/expenditure/expense_recognition_run"
	prepaymentUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/expenditure/prepayment"
	procurementRequestUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/expenditure/procurement_request"
	procurementRequestLineUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/expenditure/procurement_request_line"
	purchaseOrderUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/expenditure/purchase_order"
	purchaseOrderLineItemUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/expenditure/purchase_order_line_item"
	supplierContractUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/expenditure/supplier_contract"
	supplierContractLineUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/expenditure/supplier_contract_line"
	supplierContractPriceScheduleUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/expenditure/supplier_contract_price_schedule"
	supplierContractPriceScheduleLineUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/expenditure/supplier_contract_price_schedule_line"

	// Cross-domain (treasury): AmortizeAdvanceDisbursement is composed into the
	// GenerateExpenseRun engine. Plan B Phase 2.
	treasurydisbursementUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/treasury/disbursement"

	// Application ports
	"github.com/erniealice/espyna-golang/internal/application/ports"

	// Protobuf domain services - Entity domain (cross-domain dependency)
	paymenttermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/payment_term"

	// Protobuf domain services for expenditure repositories
	accruedexpensepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/accrued_expense"
	expenditurepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure"
	expenditureattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_attribute"
	expenditurecategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_category"
	expenditurelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_line_item"
	expenserecognitionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition"
	expenserecognitionlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition_line"
	expenserecognitionrunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition_run"
	prepaymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/prepayment"
	procurementrequestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request"
	procurementrequestlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request_line"
	purchaseorderpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/purchase_order"
	purchaseorderlineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/purchase_order_line_item"
	suppliercontractpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract"
	suppliercontractlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_line"
	scpspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_price_schedule"
	scpslpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_price_schedule_line"

	// Cross-domain: procurement domain for SupplierSubscription workspace validation
	// + CostPlan + SupplierProductCostPlan (RecognizeExpenseFromSupplierSubscription).
	costplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_plan"
	supplierproductcostplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_product_cost_plan"
	suppliersubscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_subscription"

	// Cross-domain: treasury for ListExpenseRunCandidates advance enumeration.
	disbursementpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement"
)

// ExpenditureRepositories contains all expenditure domain repositories
type ExpenditureRepositories struct {
	Expenditure            expenditurepb.ExpenditureDomainServiceServer
	ExpenditureLineItem    expenditurelineitempb.ExpenditureLineItemDomainServiceServer
	ExpenditureCategory    expenditurecategorypb.ExpenditureCategoryDomainServiceServer
	ExpenditureAttribute   expenditureattributepb.ExpenditureAttributeDomainServiceServer
	Prepayment             prepaymentpb.PrepaymentDomainServiceServer
	PurchaseOrder          purchaseorderpb.PurchaseOrderDomainServiceServer
	PurchaseOrderLineItem  purchaseorderlineitempb.PurchaseOrderLineItemDomainServiceServer
	SupplierContract       suppliercontractpb.SupplierContractDomainServiceServer
	SupplierContractLine   suppliercontractlinepb.SupplierContractLineDomainServiceServer
	ProcurementRequest     procurementrequestpb.ProcurementRequestDomainServiceServer
	ProcurementRequestLine procurementrequestlinepb.ProcurementRequestLineDomainServiceServer
	// SPS Wave 2 repositories
	SupplierContractPriceSchedule     scpspb.SupplierContractPriceScheduleDomainServiceServer
	SupplierContractPriceScheduleLine scpslpb.SupplierContractPriceScheduleLineDomainServiceServer
	ExpenseRecognition                expenserecognitionpb.ExpenseRecognitionDomainServiceServer
	ExpenseRecognitionLine            expenserecognitionlinepb.ExpenseRecognitionLineDomainServiceServer
	AccruedExpense                    accruedexpensepb.AccruedExpenseDomainServiceServer
	AccruedExpenseSettlement          accruedexpensepb.AccruedExpenseSettlementDomainServiceServer
	// Cross-domain dependency: payment term lookup for due date computation
	PaymentTerm paymenttermpb.PaymentTermDomainServiceServer
	// Cross-domain dependency: supplier subscription workspace validation on RecognizeFromExpenditure
	SupplierSubscription suppliersubscriptionpb.SupplierSubscriptionDomainServiceServer
	// Cross-domain (Plan A Phase 2): CostPlan + SupplierProductCostPlan are
	// required by RecognizeExpenseFromSupplierSubscription; TreasuryDisbursement
	// is required by ListExpenseRunCandidates (advance enumeration).
	CostPlan                costplanpb.CostPlanDomainServiceServer
	SupplierProductCostPlan supplierproductcostplanpb.SupplierProductCostPlanDomainServiceServer
	TreasuryDisbursement    disbursementpb.DisbursementDomainServiceServer
	// In-domain (Plan A Phase 4): ExpenseRecognitionRun repo (CRUD + Attempt RPCs).
	ExpenseRecognitionRun expenserecognitionrunpb.ExpenseRecognitionRunDomainServiceServer
}

// ExpenditureUseCases contains all expenditure-related use cases.
//
// 20260518-hexagonal-strict-adherence Phase 3 F6 closure — the three flat
// advance fields have been folded into entity sub-aggregates:
//   - RecognizeExpenseFromSupplierSubscription → .ExpenseRecognition.RecognizeFromSupplierSubscription
//   - ListExpenseRunCandidates + GenerateExpenseRun → .ExpenseRecognitionRun.*
type ExpenditureUseCases struct {
	Expenditure            *expenditureUseCases.UseCases
	ExpenditureLineItem    *expenditureLineItemUseCases.UseCases
	ExpenditureCategory    *expenditureCategoryUseCases.UseCases
	ExpenditureAttribute   *expenditureAttributeUseCases.UseCases
	Prepayment             *prepaymentUseCases.UseCases
	PurchaseOrder          *purchaseOrderUseCases.UseCases
	PurchaseOrderLineItem  *purchaseOrderLineItemUseCases.UseCases
	SupplierContract       *supplierContractUseCases.UseCases
	SupplierContractLine   *supplierContractLineUseCases.UseCases
	ProcurementRequest     *procurementRequestUseCases.UseCases
	ProcurementRequestLine *procurementRequestLineUseCases.UseCases
	// SPS Wave 2 use cases
	SupplierContractPriceSchedule     *supplierContractPriceScheduleUseCases.UseCases
	SupplierContractPriceScheduleLine *supplierContractPriceScheduleLineUseCases.UseCases
	ExpenseRecognition                *expenseRecognitionUseCases.UseCases
	ExpenseRecognitionLine            *expenseRecognitionLineUseCases.UseCases
	ExpenseRecognitionRun             *expenseRecognitionRunUseCases.UseCases
	AccruedExpense                    *accruedExpenseUseCases.UseCases
	AccruedExpenseSettlement          *accruedExpenseSettlementUseCases.UseCases

	// Dashboard field retired 2026-05-21 (Wave C P1.C.8 Expenditure) — the
	// dashboard now lives under `service.Dashboard.Expenditure` per Q-SDM-
	// DASHBOARD-DOWNSTREAM. The `usecases/expenditure/dashboard/` package is
	// retired in the same commit; the repository composition relocated to
	// `usecases/service/dashboard/expenditure/`.
}

// NewUseCases creates all expenditure use cases with proper constructor injection.
//
// amortizeAdvDis is the cross-domain (treasury) AmortizeAdvanceDisbursement
// use case composed into GenerateExpenseRun (Plan A Phase 4). May be nil; the
// run engine logs an "amortizer_unavailable" attempt outcome in that case.
func NewUseCases(
	repos ExpenditureRepositories,
	authSvc ports.Authorizer,
	txSvc ports.Transactor,
	i18nSvc ports.Translator,
	idService ports.IDGenerator,
	amortizeAdvDis *treasurydisbursementUseCases.AmortizeAdvanceDisbursementUseCase,
) *ExpenditureUseCases {
	expenditureUC := expenditureUseCases.NewUseCases(
		expenditureUseCases.ExpenditureRepositories{
			Expenditure: repos.Expenditure,
			PaymentTerm: repos.PaymentTerm,
		},
		expenditureUseCases.ExpenditureServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	expenditureLineItemUC := expenditureLineItemUseCases.NewUseCases(
		expenditureLineItemUseCases.ExpenditureLineItemRepositories{
			ExpenditureLineItem: repos.ExpenditureLineItem,
		},
		expenditureLineItemUseCases.ExpenditureLineItemServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	expenditureCategoryUC := expenditureCategoryUseCases.NewUseCases(
		expenditureCategoryUseCases.ExpenditureCategoryRepositories{
			ExpenditureCategory: repos.ExpenditureCategory,
		},
		expenditureCategoryUseCases.ExpenditureCategoryServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	expenditureAttributeUC := expenditureAttributeUseCases.NewUseCases(
		expenditureAttributeUseCases.ExpenditureAttributeRepositories{
			ExpenditureAttribute: repos.ExpenditureAttribute,
		},
		expenditureAttributeUseCases.ExpenditureAttributeServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	prepaymentUC := prepaymentUseCases.NewUseCases(
		prepaymentUseCases.PrepaymentRepositories{
			Prepayment: repos.Prepayment,
		},
		prepaymentUseCases.PrepaymentServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	purchaseOrderUC := purchaseOrderUseCases.NewUseCases(
		purchaseOrderUseCases.PurchaseOrderRepositories{
			PurchaseOrder: repos.PurchaseOrder,
			PaymentTerm:   repos.PaymentTerm,
		},
		purchaseOrderUseCases.PurchaseOrderServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	purchaseOrderLineItemUC := purchaseOrderLineItemUseCases.NewUseCases(
		purchaseOrderLineItemUseCases.PurchaseOrderLineItemRepositories{
			PurchaseOrderLineItem: repos.PurchaseOrderLineItem,
		},
		purchaseOrderLineItemUseCases.PurchaseOrderLineItemServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	supplierContractUC := supplierContractUseCases.NewUseCases(
		supplierContractUseCases.SupplierContractRepositories{
			SupplierContract: repos.SupplierContract,
		},
		supplierContractUseCases.SupplierContractServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	supplierContractLineUC := supplierContractLineUseCases.NewUseCases(
		supplierContractLineUseCases.SupplierContractLineRepositories{
			SupplierContractLine: repos.SupplierContractLine,
		},
		supplierContractLineUseCases.SupplierContractLineServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	procurementRequestUC := procurementRequestUseCases.NewUseCases(
		procurementRequestUseCases.ProcurementRequestRepositories{
			ProcurementRequest: repos.ProcurementRequest,
		},
		procurementRequestUseCases.ProcurementRequestServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	procurementRequestLineUC := procurementRequestLineUseCases.NewUseCases(
		procurementRequestLineUseCases.ProcurementRequestLineRepositories{
			ProcurementRequestLine: repos.ProcurementRequestLine,
		},
		procurementRequestLineUseCases.ProcurementRequestLineServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	supplierContractPriceScheduleUC := supplierContractPriceScheduleUseCases.NewUseCases(
		supplierContractPriceScheduleUseCases.SupplierContractPriceScheduleRepositories{
			SupplierContractPriceSchedule: repos.SupplierContractPriceSchedule,
		},
		supplierContractPriceScheduleUseCases.SupplierContractPriceScheduleServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	supplierContractPriceScheduleLineUC := supplierContractPriceScheduleLineUseCases.NewUseCases(
		supplierContractPriceScheduleLineUseCases.SupplierContractPriceScheduleLineRepositories{
			SupplierContractPriceScheduleLine: repos.SupplierContractPriceScheduleLine,
		},
		supplierContractPriceScheduleLineUseCases.SupplierContractPriceScheduleLineServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	expenseRecognitionUC := expenseRecognitionUseCases.NewUseCases(
		expenseRecognitionUseCases.ExpenseRecognitionRepositories{
			ExpenseRecognition:     repos.ExpenseRecognition,
			ExpenseRecognitionLine: repos.ExpenseRecognitionLine,
			Expenditure:            repos.Expenditure,
			ExpenditureLineItem:    repos.ExpenditureLineItem,
			SupplierSubscription:   repos.SupplierSubscription,
		},
		expenseRecognitionUseCases.ExpenseRecognitionServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	expenseRecognitionLineUC := expenseRecognitionLineUseCases.NewUseCases(
		expenseRecognitionLineUseCases.ExpenseRecognitionLineRepositories{
			ExpenseRecognitionLine: repos.ExpenseRecognitionLine,
		},
		expenseRecognitionLineUseCases.ExpenseRecognitionLineServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	accruedExpenseUC := accruedExpenseUseCases.NewUseCases(
		accruedExpenseUseCases.AccruedExpenseRepositories{
			AccruedExpense:           repos.AccruedExpense,
			AccruedExpenseSettlement: repos.AccruedExpenseSettlement,
		},
		accruedExpenseUseCases.AccruedExpenseServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	accruedExpenseSettlementUC := accruedExpenseSettlementUseCases.NewUseCases(
		accruedExpenseSettlementUseCases.AccruedExpenseSettlementRepositories{
			AccruedExpenseSettlement: repos.AccruedExpenseSettlement,
		},
		accruedExpenseSettlementUseCases.AccruedExpenseSettlementServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	// Expenditure dashboard wiring retired 2026-05-21 (Wave C P1.C.8) —
	// type-assertion + factory wiring now lives in the service-layer
	// initializer at `internal/composition/core/initializers/service.go`
	// (search "Wave C P1.C.8 Expenditure").

	// 20260517-expense-run Plan A Phase 2 — RecognizeExpenseFromSupplierSubscription.
	recognizeFromSupplierSub := expenseRecognitionUseCases.NewRecognizeExpenseFromSupplierSubscriptionUseCase(
		expenseRecognitionUseCases.RecognizeExpenseFromSupplierSubscriptionRepositories{
			ExpenseRecognition:      repos.ExpenseRecognition,
			ExpenseRecognitionLine:  repos.ExpenseRecognitionLine,
			Expenditure:             repos.Expenditure,
			ExpenditureLineItem:     repos.ExpenditureLineItem,
			SupplierSubscription:    repos.SupplierSubscription,
			CostPlan:                repos.CostPlan,
			SupplierProductCostPlan: repos.SupplierProductCostPlan,
		},
		expenseRecognitionUseCases.RecognizeExpenseFromSupplierSubscriptionServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	// 20260517-expense-run Plan A Phase 2 — ListExpenseRunCandidates.
	listRunCandidates := expenseRecognitionRunUseCases.NewListExpenseRunCandidatesUseCase(
		expenseRecognitionRunUseCases.ListExpenseRunCandidatesRepositories{
			SupplierSubscription: repos.SupplierSubscription,
			CostPlan:             repos.CostPlan,
			ExpenseRecognition:   repos.ExpenseRecognition,
			TreasuryDisbursement: repos.TreasuryDisbursement,
			Expenditure:          repos.Expenditure,
		},
		expenseRecognitionRunUseCases.ListExpenseRunCandidatesServices{
			Authorizer: authSvc,
			Translator: i18nSvc,
		},
	)

	// 20260517-expense-run Plan A Phase 4 — GenerateExpenseRun (composes the
	// two inner use cases plus the cross-domain AmortizeAdvanceDisbursement).
	//
	// The Attempt writer port is satisfied by repos.ExpenseRecognitionRun when
	// it implements CreateExpenseRecognitionRunAttempt (the Phase-0 proto
	// declares it on the same DomainServiceServer interface, so we pass the
	// run repo through directly). When the postgres adapter isn't registered,
	// the run engine still returns the in-memory attempt rows in the response.
	var attemptWriter expenseRecognitionRunUseCases.ExpenseRecognitionRunAttemptWriter
	if repos.ExpenseRecognitionRun != nil {
		// The proto DomainServiceServer already declares
		// CreateExpenseRecognitionRunAttempt, so it satisfies the interface.
		attemptWriter = repos.ExpenseRecognitionRun
	}
	generateExpenseRun := expenseRecognitionRunUseCases.NewGenerateExpenseRunUseCase(
		expenseRecognitionRunUseCases.GenerateExpenseRunRepositories{
			ExpenseRecognition:    repos.ExpenseRecognition,
			ExpenseRecognitionRun: repos.ExpenseRecognitionRun,
			AttemptWriter:         attemptWriter,
		},
		expenseRecognitionRunUseCases.GenerateExpenseRunServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
		recognizeFromSupplierSub,
		amortizeAdvDis,
	)

	// Phase 3 F6 closure — nest RecognizeFromSupplierSubscription under the
	// ExpenseRecognition sub-aggregate.
	if expenseRecognitionUC != nil {
		expenseRecognitionUC.RecognizeFromSupplierSubscription = recognizeFromSupplierSub
	}

	// Phase 3 F6 closure — nest ListExpenseRunCandidates + GenerateExpenseRun
	// under the new ExpenseRecognitionRun sub-aggregate. The sub-aggregate's
	// NewUseCases returns an empty shell because both use cases need
	// cross-domain dependencies that are wired here.
	expenseRecognitionRunUC := expenseRecognitionRunUseCases.NewUseCases(
		expenseRecognitionRunUseCases.ExpenseRecognitionRunRepositories{
			ExpenseRecognitionRun: repos.ExpenseRecognitionRun,
		},
		expenseRecognitionRunUseCases.ExpenseRecognitionRunServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)
	if expenseRecognitionRunUC != nil {
		expenseRecognitionRunUC.ListExpenseRunCandidates = listRunCandidates
		expenseRecognitionRunUC.GenerateExpenseRun = generateExpenseRun
	}

	return &ExpenditureUseCases{
		Expenditure:                       expenditureUC,
		ExpenditureLineItem:               expenditureLineItemUC,
		ExpenditureCategory:               expenditureCategoryUC,
		ExpenditureAttribute:              expenditureAttributeUC,
		Prepayment:                        prepaymentUC,
		PurchaseOrder:                     purchaseOrderUC,
		PurchaseOrderLineItem:             purchaseOrderLineItemUC,
		SupplierContract:                  supplierContractUC,
		SupplierContractLine:              supplierContractLineUC,
		ProcurementRequest:                procurementRequestUC,
		ProcurementRequestLine:            procurementRequestLineUC,
		SupplierContractPriceSchedule:     supplierContractPriceScheduleUC,
		SupplierContractPriceScheduleLine: supplierContractPriceScheduleLineUC,
		ExpenseRecognition:                expenseRecognitionUC,
		ExpenseRecognitionLine:            expenseRecognitionLineUC,
		ExpenseRecognitionRun:             expenseRecognitionRunUC,
		AccruedExpense:                    accruedExpenseUC,
		AccruedExpenseSettlement:          accruedExpenseSettlementUC,
	}
}

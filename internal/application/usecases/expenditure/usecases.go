package expenditure

import (
	// Expenditure use cases
	accruedExpenseUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/accrued_expense"

	// Dashboard use case (purchase + expense share one use case)
	accruedExpenseSettlementUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/accrued_expense_settlement"
	expendituredashboard "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/dashboard"
	expenditureUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/expenditure"
	expenditureAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/expenditure_attribute"
	expenditureCategoryUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/expenditure_category"
	expenditureLineItemUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/expenditure_line_item"
	expenseRecognitionUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/expense_recognition"
	expenseRecognitionLineUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/expense_recognition_line"
	expenseRecognitionRunUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/expense_recognition_run"
	prepaymentUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/prepayment"
	procurementRequestUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/procurement_request"
	procurementRequestLineUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/procurement_request_line"
	purchaseOrderUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/purchase_order"
	purchaseOrderLineItemUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/purchase_order_line_item"
	supplierContractUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/supplier_contract"
	supplierContractLineUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/supplier_contract_line"
	supplierContractPriceScheduleUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/supplier_contract_price_schedule"
	supplierContractPriceScheduleLineUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/supplier_contract_price_schedule_line"
	// Cross-domain (treasury): AmortizeAdvanceDisbursement is composed into the
	// GenerateExpenseRun engine. Plan B Phase 2.
	treasurydisbursementUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/treasury/treasury_disbursement"

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

// ExpenditureUseCases contains all expenditure-related use cases
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
	AccruedExpense                    *accruedExpenseUseCases.UseCases
	AccruedExpenseSettlement          *accruedExpenseSettlementUseCases.UseCases

	// Dashboard use case — shared for both purchase and expense surfaces
	// (the request carries the Kind discriminator). Nil when postgres build
	// tag is inactive.
	Dashboard *expendituredashboard.GetExpenditureDashboardPageDataUseCase

	// 20260517-expense-run Plan A Phase 2 + Phase 4 — buying-side recognition.
	// Constructed in NewUseCases when the required cross-domain repos are present.
	// Nil-safe consumers; surface views degrade to disabled buttons + helpful
	// tooltips when unwired.
	RecognizeExpenseFromSupplierSubscription *expenseRecognitionUseCases.RecognizeExpenseFromSupplierSubscriptionUseCase
	ListExpenseRunCandidates                 *expenseRecognitionRunUseCases.ListExpenseRunCandidatesUseCase
	GenerateExpenseRun                       *expenseRecognitionRunUseCases.GenerateExpenseRunUseCase
}

// NewUseCases creates all expenditure use cases with proper constructor injection.
//
// amortizeAdvDis is the cross-domain (treasury) AmortizeAdvanceDisbursement
// use case composed into GenerateExpenseRun (Plan A Phase 4). May be nil; the
// run engine logs an "amortizer_unavailable" attempt outcome in that case.
func NewUseCases(
	repos ExpenditureRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idService ports.IDService,
	amortizeAdvDis *treasurydisbursementUseCases.AmortizeAdvanceDisbursementUseCase,
) *ExpenditureUseCases {
	expenditureUC := expenditureUseCases.NewUseCases(
		expenditureUseCases.ExpenditureRepositories{
			Expenditure: repos.Expenditure,
			PaymentTerm: repos.PaymentTerm,
		},
		expenditureUseCases.ExpenditureServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	expenditureLineItemUC := expenditureLineItemUseCases.NewUseCases(
		expenditureLineItemUseCases.ExpenditureLineItemRepositories{
			ExpenditureLineItem: repos.ExpenditureLineItem,
		},
		expenditureLineItemUseCases.ExpenditureLineItemServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	expenditureCategoryUC := expenditureCategoryUseCases.NewUseCases(
		expenditureCategoryUseCases.ExpenditureCategoryRepositories{
			ExpenditureCategory: repos.ExpenditureCategory,
		},
		expenditureCategoryUseCases.ExpenditureCategoryServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	expenditureAttributeUC := expenditureAttributeUseCases.NewUseCases(
		expenditureAttributeUseCases.ExpenditureAttributeRepositories{
			ExpenditureAttribute: repos.ExpenditureAttribute,
		},
		expenditureAttributeUseCases.ExpenditureAttributeServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	prepaymentUC := prepaymentUseCases.NewUseCases(
		prepaymentUseCases.PrepaymentRepositories{
			Prepayment: repos.Prepayment,
		},
		prepaymentUseCases.PrepaymentServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	purchaseOrderUC := purchaseOrderUseCases.NewUseCases(
		purchaseOrderUseCases.PurchaseOrderRepositories{
			PurchaseOrder: repos.PurchaseOrder,
			PaymentTerm:   repos.PaymentTerm,
		},
		purchaseOrderUseCases.PurchaseOrderServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	purchaseOrderLineItemUC := purchaseOrderLineItemUseCases.NewUseCases(
		purchaseOrderLineItemUseCases.PurchaseOrderLineItemRepositories{
			PurchaseOrderLineItem: repos.PurchaseOrderLineItem,
		},
		purchaseOrderLineItemUseCases.PurchaseOrderLineItemServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	supplierContractUC := supplierContractUseCases.NewUseCases(
		supplierContractUseCases.SupplierContractRepositories{
			SupplierContract: repos.SupplierContract,
		},
		supplierContractUseCases.SupplierContractServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	supplierContractLineUC := supplierContractLineUseCases.NewUseCases(
		supplierContractLineUseCases.SupplierContractLineRepositories{
			SupplierContractLine: repos.SupplierContractLine,
		},
		supplierContractLineUseCases.SupplierContractLineServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	procurementRequestUC := procurementRequestUseCases.NewUseCases(
		procurementRequestUseCases.ProcurementRequestRepositories{
			ProcurementRequest: repos.ProcurementRequest,
		},
		procurementRequestUseCases.ProcurementRequestServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	procurementRequestLineUC := procurementRequestLineUseCases.NewUseCases(
		procurementRequestLineUseCases.ProcurementRequestLineRepositories{
			ProcurementRequestLine: repos.ProcurementRequestLine,
		},
		procurementRequestLineUseCases.ProcurementRequestLineServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	supplierContractPriceScheduleUC := supplierContractPriceScheduleUseCases.NewUseCases(
		supplierContractPriceScheduleUseCases.SupplierContractPriceScheduleRepositories{
			SupplierContractPriceSchedule: repos.SupplierContractPriceSchedule,
		},
		supplierContractPriceScheduleUseCases.SupplierContractPriceScheduleServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	supplierContractPriceScheduleLineUC := supplierContractPriceScheduleLineUseCases.NewUseCases(
		supplierContractPriceScheduleLineUseCases.SupplierContractPriceScheduleLineRepositories{
			SupplierContractPriceScheduleLine: repos.SupplierContractPriceScheduleLine,
		},
		supplierContractPriceScheduleLineUseCases.SupplierContractPriceScheduleLineServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
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
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	expenseRecognitionLineUC := expenseRecognitionLineUseCases.NewUseCases(
		expenseRecognitionLineUseCases.ExpenseRecognitionLineRepositories{
			ExpenseRecognitionLine: repos.ExpenseRecognitionLine,
		},
		expenseRecognitionLineUseCases.ExpenseRecognitionLineServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	accruedExpenseUC := accruedExpenseUseCases.NewUseCases(
		accruedExpenseUseCases.AccruedExpenseRepositories{
			AccruedExpense:           repos.AccruedExpense,
			AccruedExpenseSettlement: repos.AccruedExpenseSettlement,
		},
		accruedExpenseUseCases.AccruedExpenseServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	accruedExpenseSettlementUC := accruedExpenseSettlementUseCases.NewUseCases(
		accruedExpenseSettlementUseCases.AccruedExpenseSettlementRepositories{
			AccruedExpenseSettlement: repos.AccruedExpenseSettlement,
		},
		accruedExpenseSettlementUseCases.AccruedExpenseSettlementServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	// Wire expenditure dashboard via type assertion on expenditure repo.
	var expenditureDash *expendituredashboard.GetExpenditureDashboardPageDataUseCase
	if repos.Expenditure != nil {
		if eq, ok := repos.Expenditure.(expendituredashboard.ExpenditureDashboardQueries); ok {
			expenditureDash = expendituredashboard.NewGetExpenditureDashboardPageDataUseCase(eq)
		}
	}

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
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
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
			AuthorizationService: authSvc,
			TranslationService:   i18nSvc,
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
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
		recognizeFromSupplierSub,
		amortizeAdvDis,
	)

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
		AccruedExpense:                    accruedExpenseUC,
		AccruedExpenseSettlement:          accruedExpenseSettlementUC,
		Dashboard:                         expenditureDash,

		RecognizeExpenseFromSupplierSubscription: recognizeFromSupplierSub,
		ListExpenseRunCandidates:                 listRunCandidates,
		GenerateExpenseRun:                       generateExpenseRun,
	}
}

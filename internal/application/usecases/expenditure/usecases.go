package expenditure

import (
	// Expenditure use cases
	accruedExpenseUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/accrued_expense"
	accruedExpenseSettlementUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/accrued_expense_settlement"
	expenditureUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/expenditure"
	expenditureAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/expenditure_attribute"
	expenditureCategoryUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/expenditure_category"
	expenditureLineItemUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/expenditure_line_item"
	expenseRecognitionUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/expense_recognition"
	expenseRecognitionLineUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/expense_recognition_line"
	prepaymentUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/prepayment"
	procurementRequestUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/procurement_request"
	procurementRequestLineUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/procurement_request_line"
	purchaseOrderUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/purchase_order"
	purchaseOrderLineItemUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/purchase_order_line_item"
	supplierContractUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/supplier_contract"
	supplierContractLineUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/supplier_contract_line"
	supplierContractPriceScheduleUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/supplier_contract_price_schedule"
	supplierContractPriceScheduleLineUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/supplier_contract_price_schedule_line"

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
	prepaymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/prepayment"
	procurementrequestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request"
	procurementrequestlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request_line"
	purchaseorderpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/purchase_order"
	purchaseorderlineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/purchase_order_line_item"
	suppliercontractpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract"
	suppliercontractlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_line"
	scpspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_price_schedule"
	scpslpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_price_schedule_line"
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
}

// NewUseCases creates all expenditure use cases with proper constructor injection
func NewUseCases(
	repos ExpenditureRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idService ports.IDService,
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
			ExpenseRecognition: repos.ExpenseRecognition,
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
	}
}

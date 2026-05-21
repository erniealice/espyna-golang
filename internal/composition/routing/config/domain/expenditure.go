package domain

import (
	"fmt"

	accruedexpensepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/accrued_expense"
	expenditurepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure"
	expenditureattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_attribute"
	expenditurecategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_category"
	expenditurelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_line_item"
	expenserecognitionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition"
	expenserecognitionlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition_line"
	procurementrequestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request"
	procurementrequestlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request_line"
	suppliercontractpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract"
	suppliercontractlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_line"
	scpspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_price_schedule"
	scpslpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_price_schedule_line"

	expenditureuc "github.com/erniealice/espyna-golang/internal/application/usecases/domain/expenditure"
	"github.com/erniealice/espyna-golang/internal/composition/contracts"
)

// ConfigureExpenditureDomain configures routes for the Expenditure domain with use cases injected directly
func ConfigureExpenditureDomain(expenditureUseCases *expenditureuc.ExpenditureUseCases) contracts.DomainRouteConfiguration {
	if expenditureUseCases == nil {
		fmt.Printf("  Expenditure use cases is NIL\n")
		return contracts.DomainRouteConfiguration{
			Domain:  "expenditure",
			Prefix:  "/expenditure",
			Enabled: false,
			Routes:  []contracts.RouteConfiguration{},
		}
	}

	fmt.Printf("  Expenditure use cases are properly initialized!\n")

	routes := []contracts.RouteConfiguration{}

	// Expenditure entity routes
	if expenditureUseCases.Expenditure != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/expenditure/create",
			Handler: contracts.NewGenericHandler(expenditureUseCases.Expenditure.CreateExpenditure, &expenditurepb.CreateExpenditureRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/expenditure/read",
			Handler: contracts.NewGenericHandler(expenditureUseCases.Expenditure.ReadExpenditure, &expenditurepb.ReadExpenditureRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/expenditure/update",
			Handler: contracts.NewGenericHandler(expenditureUseCases.Expenditure.UpdateExpenditure, &expenditurepb.UpdateExpenditureRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/expenditure/delete",
			Handler: contracts.NewGenericHandler(expenditureUseCases.Expenditure.DeleteExpenditure, &expenditurepb.DeleteExpenditureRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/expenditure/list",
			Handler: contracts.NewGenericHandler(expenditureUseCases.Expenditure.ListExpenditures, &expenditurepb.ListExpendituresRequest{}),
		})
	}

	// Expenditure Line Item entity routes
	if expenditureUseCases.ExpenditureLineItem != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/expenditure-line-item/create",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ExpenditureLineItem.CreateExpenditureLineItem, &expenditurelineitempb.CreateExpenditureLineItemRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/expenditure-line-item/read",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ExpenditureLineItem.ReadExpenditureLineItem, &expenditurelineitempb.ReadExpenditureLineItemRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/expenditure-line-item/update",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ExpenditureLineItem.UpdateExpenditureLineItem, &expenditurelineitempb.UpdateExpenditureLineItemRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/expenditure-line-item/delete",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ExpenditureLineItem.DeleteExpenditureLineItem, &expenditurelineitempb.DeleteExpenditureLineItemRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/expenditure-line-item/list",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ExpenditureLineItem.ListExpenditureLineItems, &expenditurelineitempb.ListExpenditureLineItemsRequest{}),
		})
	}

	// Expenditure Category entity routes
	if expenditureUseCases.ExpenditureCategory != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/expenditure-category/create",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ExpenditureCategory.CreateExpenditureCategory, &expenditurecategorypb.CreateExpenditureCategoryRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/expenditure-category/read",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ExpenditureCategory.ReadExpenditureCategory, &expenditurecategorypb.ReadExpenditureCategoryRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/expenditure-category/update",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ExpenditureCategory.UpdateExpenditureCategory, &expenditurecategorypb.UpdateExpenditureCategoryRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/expenditure-category/delete",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ExpenditureCategory.DeleteExpenditureCategory, &expenditurecategorypb.DeleteExpenditureCategoryRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/expenditure-category/list",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ExpenditureCategory.ListExpenditureCategories, &expenditurecategorypb.ListExpenditureCategoriesRequest{}),
		})
	}

	// Expenditure Attribute entity routes
	if expenditureUseCases.ExpenditureAttribute != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/expenditure-attribute/create",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ExpenditureAttribute.CreateExpenditureAttribute, &expenditureattributepb.CreateExpenditureAttributeRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/expenditure-attribute/read",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ExpenditureAttribute.ReadExpenditureAttribute, &expenditureattributepb.ReadExpenditureAttributeRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/expenditure-attribute/update",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ExpenditureAttribute.UpdateExpenditureAttribute, &expenditureattributepb.UpdateExpenditureAttributeRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/expenditure-attribute/delete",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ExpenditureAttribute.DeleteExpenditureAttribute, &expenditureattributepb.DeleteExpenditureAttributeRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/expenditure-attribute/list",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ExpenditureAttribute.ListExpenditureAttributes, &expenditureattributepb.ListExpenditureAttributesRequest{}),
		})
	}

	// Supplier Contract entity routes
	if expenditureUseCases.SupplierContract != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/supplier-contract/create",
			Handler: contracts.NewGenericHandler(expenditureUseCases.SupplierContract.CreateSupplierContract, &suppliercontractpb.CreateSupplierContractRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/supplier-contract/read",
			Handler: contracts.NewGenericHandler(expenditureUseCases.SupplierContract.ReadSupplierContract, &suppliercontractpb.ReadSupplierContractRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/supplier-contract/update",
			Handler: contracts.NewGenericHandler(expenditureUseCases.SupplierContract.UpdateSupplierContract, &suppliercontractpb.UpdateSupplierContractRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/supplier-contract/delete",
			Handler: contracts.NewGenericHandler(expenditureUseCases.SupplierContract.DeleteSupplierContract, &suppliercontractpb.DeleteSupplierContractRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/supplier-contract/list",
			Handler: contracts.NewGenericHandler(expenditureUseCases.SupplierContract.ListSupplierContracts, &suppliercontractpb.ListSupplierContractsRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/supplier-contract/get-list-page-data",
			Handler: contracts.NewGenericHandler(expenditureUseCases.SupplierContract.GetSupplierContractListPageData, &suppliercontractpb.GetSupplierContractListPageDataRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/supplier-contract/get-item-page-data",
			Handler: contracts.NewGenericHandler(expenditureUseCases.SupplierContract.GetSupplierContractItemPageData, &suppliercontractpb.GetSupplierContractItemPageDataRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/supplier-contract/approve",
			Handler: contracts.NewGenericHandler(expenditureUseCases.SupplierContract.ApproveSupplierContract, &suppliercontractpb.ApproveSupplierContractRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/supplier-contract/terminate",
			Handler: contracts.NewGenericHandler(expenditureUseCases.SupplierContract.TerminateSupplierContract, &suppliercontractpb.TerminateSupplierContractRequest{}),
		})
	}

	// Supplier Contract Line entity routes
	if expenditureUseCases.SupplierContractLine != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/supplier-contract-line/create",
			Handler: contracts.NewGenericHandler(expenditureUseCases.SupplierContractLine.CreateSupplierContractLine, &suppliercontractlinepb.CreateSupplierContractLineRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/supplier-contract-line/read",
			Handler: contracts.NewGenericHandler(expenditureUseCases.SupplierContractLine.ReadSupplierContractLine, &suppliercontractlinepb.ReadSupplierContractLineRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/supplier-contract-line/update",
			Handler: contracts.NewGenericHandler(expenditureUseCases.SupplierContractLine.UpdateSupplierContractLine, &suppliercontractlinepb.UpdateSupplierContractLineRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/supplier-contract-line/delete",
			Handler: contracts.NewGenericHandler(expenditureUseCases.SupplierContractLine.DeleteSupplierContractLine, &suppliercontractlinepb.DeleteSupplierContractLineRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/supplier-contract-line/list",
			Handler: contracts.NewGenericHandler(expenditureUseCases.SupplierContractLine.ListSupplierContractLines, &suppliercontractlinepb.ListSupplierContractLinesRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/supplier-contract-line/get-list-page-data",
			Handler: contracts.NewGenericHandler(expenditureUseCases.SupplierContractLine.GetSupplierContractLineListPageData, &suppliercontractlinepb.GetSupplierContractLineListPageDataRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/supplier-contract-line/get-item-page-data",
			Handler: contracts.NewGenericHandler(expenditureUseCases.SupplierContractLine.GetSupplierContractLineItemPageData, &suppliercontractlinepb.GetSupplierContractLineItemPageDataRequest{}),
		})
	}

	// Procurement Request entity routes
	if expenditureUseCases.ProcurementRequest != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/procurement-request/create",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ProcurementRequest.CreateProcurementRequest, &procurementrequestpb.CreateProcurementRequestRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/procurement-request/read",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ProcurementRequest.ReadProcurementRequest, &procurementrequestpb.ReadProcurementRequestRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/procurement-request/update",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ProcurementRequest.UpdateProcurementRequest, &procurementrequestpb.UpdateProcurementRequestRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/procurement-request/delete",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ProcurementRequest.DeleteProcurementRequest, &procurementrequestpb.DeleteProcurementRequestRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/procurement-request/list",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ProcurementRequest.ListProcurementRequests, &procurementrequestpb.ListProcurementRequestsRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/procurement-request/get-list-page-data",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ProcurementRequest.GetProcurementRequestListPageData, &procurementrequestpb.GetProcurementRequestListPageDataRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/procurement-request/get-item-page-data",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ProcurementRequest.GetProcurementRequestItemPageData, &procurementrequestpb.GetProcurementRequestItemPageDataRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/procurement-request/submit",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ProcurementRequest.SubmitProcurementRequest, &procurementrequestpb.SubmitProcurementRequestRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/procurement-request/approve",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ProcurementRequest.ApproveProcurementRequest, &procurementrequestpb.ApproveProcurementRequestRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/procurement-request/reject",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ProcurementRequest.RejectProcurementRequest, &procurementrequestpb.RejectProcurementRequestRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/procurement-request/spawn-purchase-order",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ProcurementRequest.SpawnPurchaseOrder, &procurementrequestpb.SpawnPurchaseOrderRequest{}),
		})
	}

	// Procurement Request Line entity routes
	if expenditureUseCases.ProcurementRequestLine != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/procurement-request-line/create",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ProcurementRequestLine.CreateProcurementRequestLine, &procurementrequestlinepb.CreateProcurementRequestLineRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/procurement-request-line/read",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ProcurementRequestLine.ReadProcurementRequestLine, &procurementrequestlinepb.ReadProcurementRequestLineRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/procurement-request-line/update",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ProcurementRequestLine.UpdateProcurementRequestLine, &procurementrequestlinepb.UpdateProcurementRequestLineRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/procurement-request-line/delete",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ProcurementRequestLine.DeleteProcurementRequestLine, &procurementrequestlinepb.DeleteProcurementRequestLineRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/procurement-request-line/list",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ProcurementRequestLine.ListProcurementRequestLines, &procurementrequestlinepb.ListProcurementRequestLinesRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/procurement-request-line/get-list-page-data",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ProcurementRequestLine.GetProcurementRequestLineListPageData, &procurementrequestlinepb.GetProcurementRequestLineListPageDataRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/procurement-request-line/get-item-page-data",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ProcurementRequestLine.GetProcurementRequestLineItemPageData, &procurementrequestlinepb.GetProcurementRequestLineItemPageDataRequest{}),
		})
	}

	// SPS Wave 2 (2026-04-30): Supplier Contract Price Schedule
	if expenditureUseCases.SupplierContractPriceSchedule != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/supplier-contract-price-schedule/create",
			Handler: contracts.NewGenericHandler(expenditureUseCases.SupplierContractPriceSchedule.CreateSupplierContractPriceSchedule, &scpspb.CreateSupplierContractPriceScheduleRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/supplier-contract-price-schedule/read",
			Handler: contracts.NewGenericHandler(expenditureUseCases.SupplierContractPriceSchedule.ReadSupplierContractPriceSchedule, &scpspb.ReadSupplierContractPriceScheduleRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/supplier-contract-price-schedule/update",
			Handler: contracts.NewGenericHandler(expenditureUseCases.SupplierContractPriceSchedule.UpdateSupplierContractPriceSchedule, &scpspb.UpdateSupplierContractPriceScheduleRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/supplier-contract-price-schedule/delete",
			Handler: contracts.NewGenericHandler(expenditureUseCases.SupplierContractPriceSchedule.DeleteSupplierContractPriceSchedule, &scpspb.DeleteSupplierContractPriceScheduleRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/supplier-contract-price-schedule/list",
			Handler: contracts.NewGenericHandler(expenditureUseCases.SupplierContractPriceSchedule.ListSupplierContractPriceSchedules, &scpspb.ListSupplierContractPriceSchedulesRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/supplier-contract-price-schedule/activate",
			Handler: contracts.NewGenericHandler(expenditureUseCases.SupplierContractPriceSchedule.ActivateSupplierContractPriceSchedule, &scpspb.ActivateSupplierContractPriceScheduleRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/supplier-contract-price-schedule/supersede",
			Handler: contracts.NewGenericHandler(expenditureUseCases.SupplierContractPriceSchedule.SupersedeSupplierContractPriceSchedule, &scpspb.SupersedeSupplierContractPriceScheduleRequest{}),
		})
	}

	// Supplier Contract Price Schedule Line
	if expenditureUseCases.SupplierContractPriceScheduleLine != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/supplier-contract-price-schedule-line/create",
			Handler: contracts.NewGenericHandler(expenditureUseCases.SupplierContractPriceScheduleLine.CreateSupplierContractPriceScheduleLine, &scpslpb.CreateSupplierContractPriceScheduleLineRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/supplier-contract-price-schedule-line/read",
			Handler: contracts.NewGenericHandler(expenditureUseCases.SupplierContractPriceScheduleLine.ReadSupplierContractPriceScheduleLine, &scpslpb.ReadSupplierContractPriceScheduleLineRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/supplier-contract-price-schedule-line/update",
			Handler: contracts.NewGenericHandler(expenditureUseCases.SupplierContractPriceScheduleLine.UpdateSupplierContractPriceScheduleLine, &scpslpb.UpdateSupplierContractPriceScheduleLineRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/supplier-contract-price-schedule-line/delete",
			Handler: contracts.NewGenericHandler(expenditureUseCases.SupplierContractPriceScheduleLine.DeleteSupplierContractPriceScheduleLine, &scpslpb.DeleteSupplierContractPriceScheduleLineRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/supplier-contract-price-schedule-line/list",
			Handler: contracts.NewGenericHandler(expenditureUseCases.SupplierContractPriceScheduleLine.ListSupplierContractPriceScheduleLines, &scpslpb.ListSupplierContractPriceScheduleLinesRequest{}),
		})
	}

	// Expense Recognition
	if expenditureUseCases.ExpenseRecognition != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/expense-recognition/create",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ExpenseRecognition.CreateExpenseRecognition, &expenserecognitionpb.CreateExpenseRecognitionRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/expense-recognition/read",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ExpenseRecognition.ReadExpenseRecognition, &expenserecognitionpb.ReadExpenseRecognitionRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/expense-recognition/update",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ExpenseRecognition.UpdateExpenseRecognition, &expenserecognitionpb.UpdateExpenseRecognitionRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/expense-recognition/delete",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ExpenseRecognition.DeleteExpenseRecognition, &expenserecognitionpb.DeleteExpenseRecognitionRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/expense-recognition/list",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ExpenseRecognition.ListExpenseRecognitions, &expenserecognitionpb.ListExpenseRecognitionsRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/expense-recognition/recognize-from-expenditure",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ExpenseRecognition.RecognizeFromExpenditure, &expenserecognitionpb.RecognizeFromExpenditureRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/expense-recognition/recognize-from-contract",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ExpenseRecognition.RecognizeFromContract, &expenserecognitionpb.RecognizeFromContractRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/expense-recognition/reverse",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ExpenseRecognition.ReverseExpenseRecognition, &expenserecognitionpb.ReverseExpenseRecognitionRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/expense-recognition/get-unrecognized-expenditures",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ExpenseRecognition.GetUnrecognizedExpenditures, &expenserecognitionpb.GetUnrecognizedExpendituresRequest{}),
		})
	}

	// Expense Recognition Line
	if expenditureUseCases.ExpenseRecognitionLine != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/expense-recognition-line/create",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ExpenseRecognitionLine.CreateExpenseRecognitionLine, &expenserecognitionlinepb.CreateExpenseRecognitionLineRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/expense-recognition-line/read",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ExpenseRecognitionLine.ReadExpenseRecognitionLine, &expenserecognitionlinepb.ReadExpenseRecognitionLineRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/expense-recognition-line/update",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ExpenseRecognitionLine.UpdateExpenseRecognitionLine, &expenserecognitionlinepb.UpdateExpenseRecognitionLineRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/expense-recognition-line/delete",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ExpenseRecognitionLine.DeleteExpenseRecognitionLine, &expenserecognitionlinepb.DeleteExpenseRecognitionLineRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/expense-recognition-line/list",
			Handler: contracts.NewGenericHandler(expenditureUseCases.ExpenseRecognitionLine.ListExpenseRecognitionLines, &expenserecognitionlinepb.ListExpenseRecognitionLinesRequest{}),
		})
	}

	// Accrued Expense
	if expenditureUseCases.AccruedExpense != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/accrued-expense/create",
			Handler: contracts.NewGenericHandler(expenditureUseCases.AccruedExpense.CreateAccruedExpense, &accruedexpensepb.CreateAccruedExpenseRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/accrued-expense/read",
			Handler: contracts.NewGenericHandler(expenditureUseCases.AccruedExpense.ReadAccruedExpense, &accruedexpensepb.ReadAccruedExpenseRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/accrued-expense/update",
			Handler: contracts.NewGenericHandler(expenditureUseCases.AccruedExpense.UpdateAccruedExpense, &accruedexpensepb.UpdateAccruedExpenseRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/accrued-expense/delete",
			Handler: contracts.NewGenericHandler(expenditureUseCases.AccruedExpense.DeleteAccruedExpense, &accruedexpensepb.DeleteAccruedExpenseRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/accrued-expense/list",
			Handler: contracts.NewGenericHandler(expenditureUseCases.AccruedExpense.ListAccruedExpenses, &accruedexpensepb.ListAccruedExpensesRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/accrued-expense/accrue-from-contract",
			Handler: contracts.NewGenericHandler(expenditureUseCases.AccruedExpense.AccrueFromContract, &accruedexpensepb.AccrueFromContractRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/accrued-expense/reverse-accrual",
			Handler: contracts.NewGenericHandler(expenditureUseCases.AccruedExpense.ReverseAccrual, &accruedexpensepb.ReverseAccrualRequest{}),
		})
		// SPS Wave 2 (2026-04-30 supplier-pricing-symmetry) — SettleAccrual
		// is the SOLE writer of AccruedExpense.settled_amount /
		// remaining_amount / (PARTIAL|SETTLED) status. The Execute shim on
		// SettleAccrualUseCase delegates to SettleAccrual().
		if expenditureUseCases.AccruedExpense.SettleAccrual != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/expenditure/accrued-expense/settle-accrual",
				Handler: contracts.NewGenericHandler(expenditureUseCases.AccruedExpense.SettleAccrual, &accruedexpensepb.SettleAccrualRequest{}),
			})
		}
	}

	// Accrued Expense Settlement (HIGH-3 join table)
	if expenditureUseCases.AccruedExpenseSettlement != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/accrued-expense-settlement/create",
			Handler: contracts.NewGenericHandler(expenditureUseCases.AccruedExpenseSettlement.CreateAccruedExpenseSettlement, &accruedexpensepb.CreateAccruedExpenseSettlementRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/accrued-expense-settlement/read",
			Handler: contracts.NewGenericHandler(expenditureUseCases.AccruedExpenseSettlement.ReadAccruedExpenseSettlement, &accruedexpensepb.ReadAccruedExpenseSettlementRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/accrued-expense-settlement/update",
			Handler: contracts.NewGenericHandler(expenditureUseCases.AccruedExpenseSettlement.UpdateAccruedExpenseSettlement, &accruedexpensepb.UpdateAccruedExpenseSettlementRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/accrued-expense-settlement/delete",
			Handler: contracts.NewGenericHandler(expenditureUseCases.AccruedExpenseSettlement.DeleteAccruedExpenseSettlement, &accruedexpensepb.DeleteAccruedExpenseSettlementRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/expenditure/accrued-expense-settlement/list",
			Handler: contracts.NewGenericHandler(expenditureUseCases.AccruedExpenseSettlement.ListAccruedExpenseSettlements, &accruedexpensepb.ListAccruedExpenseSettlementsRequest{}),
		})
	}

	return contracts.DomainRouteConfiguration{
		Domain:  "expenditure",
		Prefix:  "/expenditure",
		Enabled: true,
		Routes:  routes,
	}
}

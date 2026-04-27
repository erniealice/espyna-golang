package domain

import (
	"fmt"

	expenditurepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure"
	expenditureattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_attribute"
	expenditurecategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_category"
	expenditurelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_line_item"
	procurementrequestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request"
	procurementrequestlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request_line"
	suppliercontractpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract"
	suppliercontractlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_line"

	expenditureuc "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure"
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

	return contracts.DomainRouteConfiguration{
		Domain:  "expenditure",
		Prefix:  "/expenditure",
		Enabled: true,
		Routes:  routes,
	}
}

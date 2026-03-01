package domain

import (
	"fmt"

	expenditurepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure"
	expenditureattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_attribute"
	expenditurecategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_category"
	expenditurelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_line_item"

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

	return contracts.DomainRouteConfiguration{
		Domain:  "expenditure",
		Prefix:  "/expenditure",
		Enabled: true,
		Routes:  routes,
	}
}

package domain

import (
	"fmt"

	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	revenueattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_attribute"
	revenuecategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_category"
	revenuelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_line_item"

	revenueuc "github.com/erniealice/espyna-golang/internal/application/usecases/revenue"
	"github.com/erniealice/espyna-golang/internal/composition/contracts"
)

// ConfigureRevenueDomain configures routes for the Revenue domain with use cases injected directly
func ConfigureRevenueDomain(revenueUseCases *revenueuc.RevenueUseCases) contracts.DomainRouteConfiguration {
	if revenueUseCases == nil {
		fmt.Printf("  Revenue use cases is NIL\n")
		return contracts.DomainRouteConfiguration{
			Domain:  "revenue",
			Prefix:  "/revenue",
			Enabled: false,
			Routes:  []contracts.RouteConfiguration{},
		}
	}

	fmt.Printf("  Revenue use cases are properly initialized!\n")

	routes := []contracts.RouteConfiguration{}

	// Revenue entity routes
	if revenueUseCases.Revenue != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/revenue/revenue/create",
			Handler: contracts.NewGenericHandler(revenueUseCases.Revenue.CreateRevenue, &revenuepb.CreateRevenueRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/revenue/revenue/read",
			Handler: contracts.NewGenericHandler(revenueUseCases.Revenue.ReadRevenue, &revenuepb.ReadRevenueRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/revenue/revenue/update",
			Handler: contracts.NewGenericHandler(revenueUseCases.Revenue.UpdateRevenue, &revenuepb.UpdateRevenueRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/revenue/revenue/delete",
			Handler: contracts.NewGenericHandler(revenueUseCases.Revenue.DeleteRevenue, &revenuepb.DeleteRevenueRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/revenue/revenue/list",
			Handler: contracts.NewGenericHandler(revenueUseCases.Revenue.ListRevenues, &revenuepb.ListRevenuesRequest{}),
		})
	}

	// Revenue Line Item entity routes
	if revenueUseCases.RevenueLineItem != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/revenue/revenue-line-item/create",
			Handler: contracts.NewGenericHandler(revenueUseCases.RevenueLineItem.CreateRevenueLineItem, &revenuelineitempb.CreateRevenueLineItemRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/revenue/revenue-line-item/read",
			Handler: contracts.NewGenericHandler(revenueUseCases.RevenueLineItem.ReadRevenueLineItem, &revenuelineitempb.ReadRevenueLineItemRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/revenue/revenue-line-item/update",
			Handler: contracts.NewGenericHandler(revenueUseCases.RevenueLineItem.UpdateRevenueLineItem, &revenuelineitempb.UpdateRevenueLineItemRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/revenue/revenue-line-item/delete",
			Handler: contracts.NewGenericHandler(revenueUseCases.RevenueLineItem.DeleteRevenueLineItem, &revenuelineitempb.DeleteRevenueLineItemRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/revenue/revenue-line-item/list",
			Handler: contracts.NewGenericHandler(revenueUseCases.RevenueLineItem.ListRevenueLineItems, &revenuelineitempb.ListRevenueLineItemsRequest{}),
		})
	}

	// Revenue Category entity routes
	if revenueUseCases.RevenueCategory != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/revenue/revenue-category/create",
			Handler: contracts.NewGenericHandler(revenueUseCases.RevenueCategory.CreateRevenueCategory, &revenuecategorypb.CreateRevenueCategoryRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/revenue/revenue-category/read",
			Handler: contracts.NewGenericHandler(revenueUseCases.RevenueCategory.ReadRevenueCategory, &revenuecategorypb.ReadRevenueCategoryRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/revenue/revenue-category/update",
			Handler: contracts.NewGenericHandler(revenueUseCases.RevenueCategory.UpdateRevenueCategory, &revenuecategorypb.UpdateRevenueCategoryRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/revenue/revenue-category/delete",
			Handler: contracts.NewGenericHandler(revenueUseCases.RevenueCategory.DeleteRevenueCategory, &revenuecategorypb.DeleteRevenueCategoryRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/revenue/revenue-category/list",
			Handler: contracts.NewGenericHandler(revenueUseCases.RevenueCategory.ListRevenueCategories, &revenuecategorypb.ListRevenueCategoriesRequest{}),
		})
	}

	// Revenue Attribute entity routes
	if revenueUseCases.RevenueAttribute != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/revenue/revenue-attribute/create",
			Handler: contracts.NewGenericHandler(revenueUseCases.RevenueAttribute.CreateRevenueAttribute, &revenueattributepb.CreateRevenueAttributeRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/revenue/revenue-attribute/read",
			Handler: contracts.NewGenericHandler(revenueUseCases.RevenueAttribute.ReadRevenueAttribute, &revenueattributepb.ReadRevenueAttributeRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/revenue/revenue-attribute/update",
			Handler: contracts.NewGenericHandler(revenueUseCases.RevenueAttribute.UpdateRevenueAttribute, &revenueattributepb.UpdateRevenueAttributeRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/revenue/revenue-attribute/delete",
			Handler: contracts.NewGenericHandler(revenueUseCases.RevenueAttribute.DeleteRevenueAttribute, &revenueattributepb.DeleteRevenueAttributeRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/revenue/revenue-attribute/list",
			Handler: contracts.NewGenericHandler(revenueUseCases.RevenueAttribute.ListRevenueAttributes, &revenueattributepb.ListRevenueAttributesRequest{}),
		})
	}

	return contracts.DomainRouteConfiguration{
		Domain:  "revenue",
		Prefix:  "/revenue",
		Enabled: true,
		Routes:  routes,
	}
}

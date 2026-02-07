package domain

import (
	"fmt"

	commonuc "leapfor.xyz/espyna/internal/application/usecases/common"
	"leapfor.xyz/espyna/internal/composition/contracts"
	attributepb "leapfor.xyz/esqyma/golang/v1/domain/common"
	categorypb "leapfor.xyz/esqyma/golang/v1/domain/common"
)

// ConfigureCommonDomain configures routes for the Common domain with use cases injected directly
func ConfigureCommonDomain(commonUseCases *commonuc.CommonUseCases) contracts.DomainRouteConfiguration {
	// Handle nil use cases gracefully for backward compatibility
	if commonUseCases == nil {
		fmt.Printf("WARNING: Common use cases is NIL\n")
		return contracts.DomainRouteConfiguration{
			Domain:  "common",
			Prefix:  "/common",
			Enabled: false,                            // Disable until use cases are properly initialized
			Routes:  []contracts.RouteConfiguration{}, // No routes without use cases
		}
	}

	fmt.Printf("SUCCESS: Common use cases are properly initialized!\n")

	// Initialize routes array
	routes := []contracts.RouteConfiguration{}

	// Attribute module routes
	if commonUseCases.Attribute != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/common/attribute/create",
			Handler: contracts.NewGenericHandler(commonUseCases.Attribute.CreateAttribute, &attributepb.CreateAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/common/attribute/read",
			Handler: contracts.NewGenericHandler(commonUseCases.Attribute.ReadAttribute, &attributepb.ReadAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/common/attribute/update",
			Handler: contracts.NewGenericHandler(commonUseCases.Attribute.UpdateAttribute, &attributepb.UpdateAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/common/attribute/delete",
			Handler: contracts.NewGenericHandler(commonUseCases.Attribute.DeleteAttribute, &attributepb.DeleteAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/common/attribute/list",
			Handler: contracts.NewGenericHandler(commonUseCases.Attribute.ListAttributes, &attributepb.ListAttributesRequest{}),
		})
	}

	// Category module routes
	if commonUseCases.Category != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/common/category/create",
			Handler: contracts.NewGenericHandler(commonUseCases.Category.CreateCategory, &categorypb.CreateCategoryRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/common/category/read",
			Handler: contracts.NewGenericHandler(commonUseCases.Category.ReadCategory, &categorypb.ReadCategoryRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/common/category/update",
			Handler: contracts.NewGenericHandler(commonUseCases.Category.UpdateCategory, &categorypb.UpdateCategoryRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/common/category/delete",
			Handler: contracts.NewGenericHandler(commonUseCases.Category.DeleteCategory, &categorypb.DeleteCategoryRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/common/category/list",
			Handler: contracts.NewGenericHandler(commonUseCases.Category.ListCategories, &categorypb.ListCategoriesRequest{}),
		})
	}

	return contracts.DomainRouteConfiguration{
		Domain:  "common",
		Prefix:  "/common",
		Enabled: true,
		Routes:  routes,
	}
}

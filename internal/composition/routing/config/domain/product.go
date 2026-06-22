package domain

import (
	"fmt"

	lineworkspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/line_workspace_user"
	plangrouppb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/plan_group"
	plangroupplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/plan_group_plan"
	priceproductpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_product"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
	productattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_attribute"
	productlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_line"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	productplanstaffpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan_staff"
	resourcepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/resource"

	productuc "github.com/erniealice/espyna-golang/internal/application/usecases/domain/product"
	"github.com/erniealice/espyna-golang/internal/composition/contracts"
)

// ConfigureProductDomain configures routes for the Product domain with use cases injected directly
func ConfigureProductDomain(productUseCases *productuc.ProductUseCases) contracts.DomainRouteConfiguration {
	// Handle nil use cases gracefully for backward compatibility
	if productUseCases == nil {
		fmt.Printf("⚠️  Product use cases is NIL\n")
		return contracts.DomainRouteConfiguration{
			Domain:  "product",
			Prefix:  "/product",
			Enabled: false,                            // Disable until use cases are properly initialized
			Routes:  []contracts.RouteConfiguration{}, // No routes without use cases
		}
	}

	fmt.Printf("✅ Product use cases are properly initialized!\n")

	// Initialize routes array
	routes := []contracts.RouteConfiguration{}

	// Price Product module routes
	if productUseCases.PriceProduct != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/price-product/create",
			Handler: contracts.NewGenericHandler(productUseCases.PriceProduct.CreatePriceProduct, &priceproductpb.CreatePriceProductRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/price-product/read",
			Handler: contracts.NewGenericHandler(productUseCases.PriceProduct.ReadPriceProduct, &priceproductpb.ReadPriceProductRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/price-product/update",
			Handler: contracts.NewGenericHandler(productUseCases.PriceProduct.UpdatePriceProduct, &priceproductpb.UpdatePriceProductRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/price-product/delete",
			Handler: contracts.NewGenericHandler(productUseCases.PriceProduct.DeletePriceProduct, &priceproductpb.DeletePriceProductRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/price-product/list",
			Handler: contracts.NewGenericHandler(productUseCases.PriceProduct.ListPriceProducts, &priceproductpb.ListPriceProductsRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/price-product/get-list-page-data",
			Handler: contracts.NewGenericHandler(productUseCases.PriceProduct.GetPriceProductListPageData, &priceproductpb.GetPriceProductListPageDataRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/price-product/get-item-page-data",
			Handler: contracts.NewGenericHandler(productUseCases.PriceProduct.GetPriceProductItemPageData, &priceproductpb.GetPriceProductItemPageDataRequest{}),
		})
	}

	// Product module routes
	if productUseCases.Product != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product/create",
			Handler: contracts.NewGenericHandler(productUseCases.Product.CreateProduct, &productpb.CreateProductRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product/read",
			Handler: contracts.NewGenericHandler(productUseCases.Product.ReadProduct, &productpb.ReadProductRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product/update",
			Handler: contracts.NewGenericHandler(productUseCases.Product.UpdateProduct, &productpb.UpdateProductRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product/delete",
			Handler: contracts.NewGenericHandler(productUseCases.Product.DeleteProduct, &productpb.DeleteProductRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product/list",
			Handler: contracts.NewGenericHandler(productUseCases.Product.ListProducts, &productpb.ListProductsRequest{}),
		})

		// Note: Product page data methods are not implemented yet - commented out for now
		// routes = append(routes, contracts.RouteConfiguration{
		// 	Method:  "POST",
		// 	Path:    "/api/product/product/get-list-page-data",
		// 	Handler: contracts.NewGenericHandler(productUseCases.Product.GetListPageData, &productpb.GetListPageDataRequest{}),
		// })

		// routes = append(routes, contracts.RouteConfiguration{
		// 	Method:  "POST",
		// 	Path:    "/api/product/product/get-item-page-data",
		// 	Handler: contracts.NewGenericHandler(productUseCases.Product.GetItemPageData, &productpb.GetItemPageDataRequest{}),
		// })
	}

	// Product Attribute module routes
	if productUseCases.ProductAttribute != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-attribute/create",
			Handler: contracts.NewGenericHandler(productUseCases.ProductAttribute.CreateProductAttribute, &productattributepb.CreateProductAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-attribute/read",
			Handler: contracts.NewGenericHandler(productUseCases.ProductAttribute.ReadProductAttribute, &productattributepb.ReadProductAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-attribute/update",
			Handler: contracts.NewGenericHandler(productUseCases.ProductAttribute.UpdateProductAttribute, &productattributepb.UpdateProductAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-attribute/delete",
			Handler: contracts.NewGenericHandler(productUseCases.ProductAttribute.DeleteProductAttribute, &productattributepb.DeleteProductAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-attribute/list",
			Handler: contracts.NewGenericHandler(productUseCases.ProductAttribute.ListProductAttributes, &productattributepb.ListProductAttributesRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-attribute/get-list-page-data",
			Handler: contracts.NewGenericHandler(productUseCases.ProductAttribute.GetProductAttributeListPageData, &productattributepb.GetProductAttributeListPageDataRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-attribute/get-item-page-data",
			Handler: contracts.NewGenericHandler(productUseCases.ProductAttribute.GetProductAttributeItemPageData, &productattributepb.GetProductAttributeItemPageDataRequest{}),
		})
	}

	// Product Line module routes
	if productUseCases.ProductLine != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-line/create",
			Handler: contracts.NewGenericHandler(productUseCases.ProductLine.CreateProductLine, &productlinepb.CreateProductLineRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-line/read",
			Handler: contracts.NewGenericHandler(productUseCases.ProductLine.ReadProductLine, &productlinepb.ReadProductLineRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-line/update",
			Handler: contracts.NewGenericHandler(productUseCases.ProductLine.UpdateProductLine, &productlinepb.UpdateProductLineRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-line/delete",
			Handler: contracts.NewGenericHandler(productUseCases.ProductLine.DeleteProductLine, &productlinepb.DeleteProductLineRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-line/list",
			Handler: contracts.NewGenericHandler(productUseCases.ProductLine.ListProductLines, &productlinepb.ListProductLinesRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-line/get-list-page-data",
			Handler: contracts.NewGenericHandler(productUseCases.ProductLine.GetProductLineListPageData, &productlinepb.GetProductLineListPageDataRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-line/get-item-page-data",
			Handler: contracts.NewGenericHandler(productUseCases.ProductLine.GetProductLineItemPageData, &productlinepb.GetProductLineItemPageDataRequest{}),
		})
	}

	// Product Plan module routes
	if productUseCases.ProductPlan != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-plan/create",
			Handler: contracts.NewGenericHandler(productUseCases.ProductPlan.CreateProductPlan, &productplanpb.CreateProductPlanRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-plan/read",
			Handler: contracts.NewGenericHandler(productUseCases.ProductPlan.ReadProductPlan, &productplanpb.ReadProductPlanRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-plan/update",
			Handler: contracts.NewGenericHandler(productUseCases.ProductPlan.UpdateProductPlan, &productplanpb.UpdateProductPlanRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-plan/delete",
			Handler: contracts.NewGenericHandler(productUseCases.ProductPlan.DeleteProductPlan, &productplanpb.DeleteProductPlanRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-plan/list",
			Handler: contracts.NewGenericHandler(productUseCases.ProductPlan.ListProductPlans, &productplanpb.ListProductPlansRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-plan/get-list-page-data",
			Handler: contracts.NewGenericHandler(productUseCases.ProductPlan.GetProductPlanListPageData, &productplanpb.GetProductPlanListPageDataRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-plan/get-item-page-data",
			Handler: contracts.NewGenericHandler(productUseCases.ProductPlan.GetProductPlanItemPageData, &productplanpb.GetProductPlanItemPageDataRequest{}),
		})
	}

	// Plan Group module routes
	if productUseCases.PlanGroup != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/plan-group/create",
			Handler: contracts.NewGenericHandler(productUseCases.PlanGroup.CreatePlanGroup, &plangrouppb.CreatePlanGroupRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/plan-group/read",
			Handler: contracts.NewGenericHandler(productUseCases.PlanGroup.ReadPlanGroup, &plangrouppb.ReadPlanGroupRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/plan-group/update",
			Handler: contracts.NewGenericHandler(productUseCases.PlanGroup.UpdatePlanGroup, &plangrouppb.UpdatePlanGroupRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/plan-group/delete",
			Handler: contracts.NewGenericHandler(productUseCases.PlanGroup.DeletePlanGroup, &plangrouppb.DeletePlanGroupRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/plan-group/list",
			Handler: contracts.NewGenericHandler(productUseCases.PlanGroup.ListPlanGroups, &plangrouppb.ListPlanGroupsRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/plan-group/get-list-page-data",
			Handler: contracts.NewGenericHandler(productUseCases.PlanGroup.GetPlanGroupListPageData, &plangrouppb.GetPlanGroupListPageDataRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/plan-group/get-item-page-data",
			Handler: contracts.NewGenericHandler(productUseCases.PlanGroup.GetPlanGroupItemPageData, &plangrouppb.GetPlanGroupItemPageDataRequest{}),
		})
	}

	// Plan Group Plan module routes
	if productUseCases.PlanGroupPlan != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/plan-group-plan/create",
			Handler: contracts.NewGenericHandler(productUseCases.PlanGroupPlan.CreatePlanGroupPlan, &plangroupplanpb.CreatePlanGroupPlanRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/plan-group-plan/read",
			Handler: contracts.NewGenericHandler(productUseCases.PlanGroupPlan.ReadPlanGroupPlan, &plangroupplanpb.ReadPlanGroupPlanRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/plan-group-plan/update",
			Handler: contracts.NewGenericHandler(productUseCases.PlanGroupPlan.UpdatePlanGroupPlan, &plangroupplanpb.UpdatePlanGroupPlanRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/plan-group-plan/delete",
			Handler: contracts.NewGenericHandler(productUseCases.PlanGroupPlan.DeletePlanGroupPlan, &plangroupplanpb.DeletePlanGroupPlanRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/plan-group-plan/list",
			Handler: contracts.NewGenericHandler(productUseCases.PlanGroupPlan.ListPlanGroupPlans, &plangroupplanpb.ListPlanGroupPlansRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/plan-group-plan/get-list-page-data",
			Handler: contracts.NewGenericHandler(productUseCases.PlanGroupPlan.GetPlanGroupPlanListPageData, &plangroupplanpb.GetPlanGroupPlanListPageDataRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/plan-group-plan/get-item-page-data",
			Handler: contracts.NewGenericHandler(productUseCases.PlanGroupPlan.GetPlanGroupPlanItemPageData, &plangroupplanpb.GetPlanGroupPlanItemPageDataRequest{}),
		})
	}

	// Product Plan Staff module routes
	if productUseCases.ProductPlanStaff != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-plan-staff/create",
			Handler: contracts.NewGenericHandler(productUseCases.ProductPlanStaff.CreateProductPlanStaff, &productplanstaffpb.CreateProductPlanStaffRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-plan-staff/read",
			Handler: contracts.NewGenericHandler(productUseCases.ProductPlanStaff.ReadProductPlanStaff, &productplanstaffpb.ReadProductPlanStaffRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-plan-staff/update",
			Handler: contracts.NewGenericHandler(productUseCases.ProductPlanStaff.UpdateProductPlanStaff, &productplanstaffpb.UpdateProductPlanStaffRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-plan-staff/delete",
			Handler: contracts.NewGenericHandler(productUseCases.ProductPlanStaff.DeleteProductPlanStaff, &productplanstaffpb.DeleteProductPlanStaffRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-plan-staff/list",
			Handler: contracts.NewGenericHandler(productUseCases.ProductPlanStaff.ListProductPlanStaffs, &productplanstaffpb.ListProductPlanStaffsRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-plan-staff/get-list-page-data",
			Handler: contracts.NewGenericHandler(productUseCases.ProductPlanStaff.GetProductPlanStaffListPageData, &productplanstaffpb.GetProductPlanStaffListPageDataRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-plan-staff/get-item-page-data",
			Handler: contracts.NewGenericHandler(productUseCases.ProductPlanStaff.GetProductPlanStaffItemPageData, &productplanstaffpb.GetProductPlanStaffItemPageDataRequest{}),
		})
	}

	// Line Workspace User module routes
	if productUseCases.LineWorkspaceUser != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/line-workspace-user/create",
			Handler: contracts.NewGenericHandler(productUseCases.LineWorkspaceUser.CreateLineWorkspaceUser, &lineworkspaceuserpb.CreateLineWorkspaceUserRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/line-workspace-user/read",
			Handler: contracts.NewGenericHandler(productUseCases.LineWorkspaceUser.ReadLineWorkspaceUser, &lineworkspaceuserpb.ReadLineWorkspaceUserRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/line-workspace-user/update",
			Handler: contracts.NewGenericHandler(productUseCases.LineWorkspaceUser.UpdateLineWorkspaceUser, &lineworkspaceuserpb.UpdateLineWorkspaceUserRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/line-workspace-user/delete",
			Handler: contracts.NewGenericHandler(productUseCases.LineWorkspaceUser.DeleteLineWorkspaceUser, &lineworkspaceuserpb.DeleteLineWorkspaceUserRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/line-workspace-user/list",
			Handler: contracts.NewGenericHandler(productUseCases.LineWorkspaceUser.ListLineWorkspaceUsers, &lineworkspaceuserpb.ListLineWorkspaceUsersRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/line-workspace-user/get-list-page-data",
			Handler: contracts.NewGenericHandler(productUseCases.LineWorkspaceUser.GetLineWorkspaceUserListPageData, &lineworkspaceuserpb.GetLineWorkspaceUserListPageDataRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/line-workspace-user/get-item-page-data",
			Handler: contracts.NewGenericHandler(productUseCases.LineWorkspaceUser.GetLineWorkspaceUserItemPageData, &lineworkspaceuserpb.GetLineWorkspaceUserItemPageDataRequest{}),
		})
	}

	// Resource module routes
	if productUseCases.Resource != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/resource/create",
			Handler: contracts.NewGenericHandler(productUseCases.Resource.CreateResource, &resourcepb.CreateResourceRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/resource/read",
			Handler: contracts.NewGenericHandler(productUseCases.Resource.ReadResource, &resourcepb.ReadResourceRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/resource/update",
			Handler: contracts.NewGenericHandler(productUseCases.Resource.UpdateResource, &resourcepb.UpdateResourceRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/resource/delete",
			Handler: contracts.NewGenericHandler(productUseCases.Resource.DeleteResource, &resourcepb.DeleteResourceRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/resource/list",
			Handler: contracts.NewGenericHandler(productUseCases.Resource.ListResources, &resourcepb.ListResourcesRequest{}),
		})

		// Note: Resource page data methods are not implemented yet - commented out for now
		// routes = append(routes, contracts.RouteConfiguration{
		// 	Method:  "POST",
		// 	Path:    "/api/product/resource/get-list-page-data",
		// 	Handler: contracts.NewGenericHandler(productUseCases.Resource.GetListPageData, &resourcepb.GetListPageDataRequest{}),
		// })

		// routes = append(routes, contracts.RouteConfiguration{
		// 	Method:  "POST",
		// 	Path:    "/api/product/resource/get-item-page-data",
		// 	Handler: contracts.NewGenericHandler(productUseCases.Resource.GetItemPageData, &resourcepb.GetItemPageDataRequest{}),
		// })
	}

	return contracts.DomainRouteConfiguration{
		Domain:  "product",
		Prefix:  "/product",
		Enabled: true,
		Routes:  routes,
	}
}

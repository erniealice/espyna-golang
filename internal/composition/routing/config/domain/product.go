package domain

import (
	"fmt"

	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection"
	collectionattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection_attribute"
	collectionplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection_plan"
	priceproductpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_product"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
	productattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_attribute"
	productcollectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_collection"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	resourcepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/resource"

	productuc "github.com/erniealice/espyna-golang/internal/application/usecases/product"
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

	// Collection module routes
	if productUseCases.Collection != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/collection/create",
			Handler: contracts.NewGenericHandler(productUseCases.Collection.CreateCollection, &collectionpb.CreateCollectionRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/collection/read",
			Handler: contracts.NewGenericHandler(productUseCases.Collection.ReadCollection, &collectionpb.ReadCollectionRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/collection/update",
			Handler: contracts.NewGenericHandler(productUseCases.Collection.UpdateCollection, &collectionpb.UpdateCollectionRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/collection/delete",
			Handler: contracts.NewGenericHandler(productUseCases.Collection.DeleteCollection, &collectionpb.DeleteCollectionRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/collection/list",
			Handler: contracts.NewGenericHandler(productUseCases.Collection.ListCollections, &collectionpb.ListCollectionsRequest{}),
		})

		// Note: Collection page data methods are not implemented yet - commented out for now
		// routes = append(routes, contracts.RouteConfiguration{
		// 	Method:  "POST",
		// 	Path:    "/api/product/collection/get-list-page-data",
		// 	Handler: contracts.NewGenericHandler(productUseCases.Collection.GetListPageData, &collectionpb.GetListPageDataRequest{}),
		// })

		// routes = append(routes, contracts.RouteConfiguration{
		// 	Method:  "POST",
		// 	Path:    "/api/product/collection/get-item-page-data",
		// 	Handler: contracts.NewGenericHandler(productUseCases.Collection.GetItemPageData, &collectionpb.GetItemPageDataRequest{}),
		// })
	}

	// Collection Attribute module routes
	if productUseCases.CollectionAttribute != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/collection-attribute/create",
			Handler: contracts.NewGenericHandler(productUseCases.CollectionAttribute.CreateCollectionAttribute, &collectionattributepb.CreateCollectionAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/collection-attribute/read",
			Handler: contracts.NewGenericHandler(productUseCases.CollectionAttribute.ReadCollectionAttribute, &collectionattributepb.ReadCollectionAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/collection-attribute/update",
			Handler: contracts.NewGenericHandler(productUseCases.CollectionAttribute.UpdateCollectionAttribute, &collectionattributepb.UpdateCollectionAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/collection-attribute/delete",
			Handler: contracts.NewGenericHandler(productUseCases.CollectionAttribute.DeleteCollectionAttribute, &collectionattributepb.DeleteCollectionAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/collection-attribute/list",
			Handler: contracts.NewGenericHandler(productUseCases.CollectionAttribute.ListCollectionAttributes, &collectionattributepb.ListCollectionAttributesRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/collection-attribute/get-list-page-data",
			Handler: contracts.NewGenericHandler(productUseCases.CollectionAttribute.GetCollectionAttributeListPageData, &collectionattributepb.GetCollectionAttributeListPageDataRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/collection-attribute/get-item-page-data",
			Handler: contracts.NewGenericHandler(productUseCases.CollectionAttribute.GetCollectionAttributeItemPageData, &collectionattributepb.GetCollectionAttributeItemPageDataRequest{}),
		})
	}

	// Collection Plan module routes
	if productUseCases.CollectionPlan != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/collection-plan/create",
			Handler: contracts.NewGenericHandler(productUseCases.CollectionPlan.CreateCollectionPlan, &collectionplanpb.CreateCollectionPlanRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/collection-plan/read",
			Handler: contracts.NewGenericHandler(productUseCases.CollectionPlan.ReadCollectionPlan, &collectionplanpb.ReadCollectionPlanRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/collection-plan/update",
			Handler: contracts.NewGenericHandler(productUseCases.CollectionPlan.UpdateCollectionPlan, &collectionplanpb.UpdateCollectionPlanRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/collection-plan/delete",
			Handler: contracts.NewGenericHandler(productUseCases.CollectionPlan.DeleteCollectionPlan, &collectionplanpb.DeleteCollectionPlanRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/collection-plan/list",
			Handler: contracts.NewGenericHandler(productUseCases.CollectionPlan.ListCollectionPlans, &collectionplanpb.ListCollectionPlansRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/collection-plan/get-list-page-data",
			Handler: contracts.NewGenericHandler(productUseCases.CollectionPlan.GetCollectionPlanListPageData, &collectionplanpb.GetCollectionPlanListPageDataRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/collection-plan/get-item-page-data",
			Handler: contracts.NewGenericHandler(productUseCases.CollectionPlan.GetCollectionPlanItemPageData, &collectionplanpb.GetCollectionPlanItemPageDataRequest{}),
		})
	}

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

	// Product Collection module routes
	if productUseCases.ProductCollection != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-collection/create",
			Handler: contracts.NewGenericHandler(productUseCases.ProductCollection.CreateProductCollection, &productcollectionpb.CreateProductCollectionRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-collection/read",
			Handler: contracts.NewGenericHandler(productUseCases.ProductCollection.ReadProductCollection, &productcollectionpb.ReadProductCollectionRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-collection/update",
			Handler: contracts.NewGenericHandler(productUseCases.ProductCollection.UpdateProductCollection, &productcollectionpb.UpdateProductCollectionRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-collection/delete",
			Handler: contracts.NewGenericHandler(productUseCases.ProductCollection.DeleteProductCollection, &productcollectionpb.DeleteProductCollectionRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-collection/list",
			Handler: contracts.NewGenericHandler(productUseCases.ProductCollection.ListProductCollections, &productcollectionpb.ListProductCollectionsRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-collection/get-list-page-data",
			Handler: contracts.NewGenericHandler(productUseCases.ProductCollection.GetProductCollectionListPageData, &productcollectionpb.GetProductCollectionListPageDataRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/product/product-collection/get-item-page-data",
			Handler: contracts.NewGenericHandler(productUseCases.ProductCollection.GetProductCollectionItemPageData, &productcollectionpb.GetProductCollectionItemPageDataRequest{}),
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

package domain

import (
	"leapfor.xyz/espyna/internal/application/ports"
	"leapfor.xyz/espyna/internal/application/usecases"
	"leapfor.xyz/espyna/internal/orchestration/workflow/executor"
)

// RegisterProductUseCases registers all product domain use cases with the registry.
func RegisterProductUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Product == nil {
		return
	}

	registerCollectionUseCases(useCases, register)
	registerCollectionPlanUseCases(useCases, register)
	registerPriceProductUseCases(useCases, register)
	registerProductCoreUseCases(useCases, register)
	registerProductCollectionUseCases(useCases, register)
	registerProductPlanUseCases(useCases, register)
	registerResourceUseCases(useCases, register)
}

func registerCollectionUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Product.Collection == nil {
		return
	}
	if useCases.Product.Collection.CreateCollection != nil {
		register("product.collection.create", executor.New(useCases.Product.Collection.CreateCollection.Execute))
	}
	if useCases.Product.Collection.ReadCollection != nil {
		register("product.collection.read", executor.New(useCases.Product.Collection.ReadCollection.Execute))
	}
	if useCases.Product.Collection.UpdateCollection != nil {
		register("product.collection.update", executor.New(useCases.Product.Collection.UpdateCollection.Execute))
	}
	if useCases.Product.Collection.DeleteCollection != nil {
		register("product.collection.delete", executor.New(useCases.Product.Collection.DeleteCollection.Execute))
	}
	if useCases.Product.Collection.ListCollections != nil {
		register("product.collection.list", executor.New(useCases.Product.Collection.ListCollections.Execute))
	}
}

func registerCollectionPlanUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Product.CollectionPlan == nil {
		return
	}
	if useCases.Product.CollectionPlan.CreateCollectionPlan != nil {
		register("product.collection_plan.create", executor.New(useCases.Product.CollectionPlan.CreateCollectionPlan.Execute))
	}
	if useCases.Product.CollectionPlan.ReadCollectionPlan != nil {
		register("product.collection_plan.read", executor.New(useCases.Product.CollectionPlan.ReadCollectionPlan.Execute))
	}
	if useCases.Product.CollectionPlan.UpdateCollectionPlan != nil {
		register("product.collection_plan.update", executor.New(useCases.Product.CollectionPlan.UpdateCollectionPlan.Execute))
	}
	if useCases.Product.CollectionPlan.DeleteCollectionPlan != nil {
		register("product.collection_plan.delete", executor.New(useCases.Product.CollectionPlan.DeleteCollectionPlan.Execute))
	}
	if useCases.Product.CollectionPlan.ListCollectionPlans != nil {
		register("product.collection_plan.list", executor.New(useCases.Product.CollectionPlan.ListCollectionPlans.Execute))
	}
}

func registerPriceProductUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Product.PriceProduct == nil {
		return
	}
	if useCases.Product.PriceProduct.CreatePriceProduct != nil {
		register("product.price_product.create", executor.New(useCases.Product.PriceProduct.CreatePriceProduct.Execute))
	}
	if useCases.Product.PriceProduct.ReadPriceProduct != nil {
		register("product.price_product.read", executor.New(useCases.Product.PriceProduct.ReadPriceProduct.Execute))
	}
	if useCases.Product.PriceProduct.UpdatePriceProduct != nil {
		register("product.price_product.update", executor.New(useCases.Product.PriceProduct.UpdatePriceProduct.Execute))
	}
	if useCases.Product.PriceProduct.DeletePriceProduct != nil {
		register("product.price_product.delete", executor.New(useCases.Product.PriceProduct.DeletePriceProduct.Execute))
	}
	if useCases.Product.PriceProduct.ListPriceProducts != nil {
		register("product.price_product.list", executor.New(useCases.Product.PriceProduct.ListPriceProducts.Execute))
	}
}

func registerProductCoreUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Product.Product == nil {
		return
	}
	if useCases.Product.Product.CreateProduct != nil {
		register("product.product.create", executor.New(useCases.Product.Product.CreateProduct.Execute))
	}
	if useCases.Product.Product.ReadProduct != nil {
		register("product.product.read", executor.New(useCases.Product.Product.ReadProduct.Execute))
	}
	if useCases.Product.Product.UpdateProduct != nil {
		register("product.product.update", executor.New(useCases.Product.Product.UpdateProduct.Execute))
	}
	if useCases.Product.Product.DeleteProduct != nil {
		register("product.product.delete", executor.New(useCases.Product.Product.DeleteProduct.Execute))
	}
	if useCases.Product.Product.ListProducts != nil {
		register("product.product.list", executor.New(useCases.Product.Product.ListProducts.Execute))
	}
}

func registerProductCollectionUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Product.ProductCollection == nil {
		return
	}
	if useCases.Product.ProductCollection.CreateProductCollection != nil {
		register("product.product_collection.create", executor.New(useCases.Product.ProductCollection.CreateProductCollection.Execute))
	}
	if useCases.Product.ProductCollection.ReadProductCollection != nil {
		register("product.product_collection.read", executor.New(useCases.Product.ProductCollection.ReadProductCollection.Execute))
	}
	if useCases.Product.ProductCollection.UpdateProductCollection != nil {
		register("product.product_collection.update", executor.New(useCases.Product.ProductCollection.UpdateProductCollection.Execute))
	}
	if useCases.Product.ProductCollection.DeleteProductCollection != nil {
		register("product.product_collection.delete", executor.New(useCases.Product.ProductCollection.DeleteProductCollection.Execute))
	}
	if useCases.Product.ProductCollection.ListProductCollections != nil {
		register("product.product_collection.list", executor.New(useCases.Product.ProductCollection.ListProductCollections.Execute))
	}
}

func registerProductPlanUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Product.ProductPlan == nil {
		return
	}
	if useCases.Product.ProductPlan.CreateProductPlan != nil {
		register("product.product_plan.create", executor.New(useCases.Product.ProductPlan.CreateProductPlan.Execute))
	}
	if useCases.Product.ProductPlan.ReadProductPlan != nil {
		register("product.product_plan.read", executor.New(useCases.Product.ProductPlan.ReadProductPlan.Execute))
	}
	if useCases.Product.ProductPlan.UpdateProductPlan != nil {
		register("product.product_plan.update", executor.New(useCases.Product.ProductPlan.UpdateProductPlan.Execute))
	}
	if useCases.Product.ProductPlan.DeleteProductPlan != nil {
		register("product.product_plan.delete", executor.New(useCases.Product.ProductPlan.DeleteProductPlan.Execute))
	}
	if useCases.Product.ProductPlan.ListProductPlans != nil {
		register("product.product_plan.list", executor.New(useCases.Product.ProductPlan.ListProductPlans.Execute))
	}
}

func registerResourceUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Product.Resource == nil {
		return
	}
	if useCases.Product.Resource.CreateResource != nil {
		register("product.resource.create", executor.New(useCases.Product.Resource.CreateResource.Execute))
	}
	if useCases.Product.Resource.ReadResource != nil {
		register("product.resource.read", executor.New(useCases.Product.Resource.ReadResource.Execute))
	}
	if useCases.Product.Resource.UpdateResource != nil {
		register("product.resource.update", executor.New(useCases.Product.Resource.UpdateResource.Execute))
	}
	if useCases.Product.Resource.DeleteResource != nil {
		register("product.resource.delete", executor.New(useCases.Product.Resource.DeleteResource.Execute))
	}
	if useCases.Product.Resource.ListResources != nil {
		register("product.resource.list", executor.New(useCases.Product.Resource.ListResources.Execute))
	}
}

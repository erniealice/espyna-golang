package domain

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases"
	"github.com/erniealice/espyna-golang/internal/orchestration/workflow/executor"
)

// RegisterProductUseCases registers all product domain use cases with the registry.
func RegisterProductUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Product == nil {
		return
	}

	registerPriceProductUseCases(useCases, register)
	registerProductCoreUseCases(useCases, register)
	registerProductLineUseCases(useCases, register)
	registerProductPlanUseCases(useCases, register)
	registerResourceUseCases(useCases, register)
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

func registerProductLineUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Product.ProductLine == nil {
		return
	}
	if useCases.Product.ProductLine.CreateProductLine != nil {
		register("product.product_line.create", executor.New(useCases.Product.ProductLine.CreateProductLine.Execute))
	}
	if useCases.Product.ProductLine.ReadProductLine != nil {
		register("product.product_line.read", executor.New(useCases.Product.ProductLine.ReadProductLine.Execute))
	}
	if useCases.Product.ProductLine.UpdateProductLine != nil {
		register("product.product_line.update", executor.New(useCases.Product.ProductLine.UpdateProductLine.Execute))
	}
	if useCases.Product.ProductLine.DeleteProductLine != nil {
		register("product.product_line.delete", executor.New(useCases.Product.ProductLine.DeleteProductLine.Execute))
	}
	if useCases.Product.ProductLine.ListProductLines != nil {
		register("product.product_line.list", executor.New(useCases.Product.ProductLine.ListProductLines.Execute))
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

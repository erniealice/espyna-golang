package product_price_plan

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	productpriceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/product_price_plan"
)

// ProductPricePlanRepositories groups all repository dependencies for product price plan use cases
type ProductPricePlanRepositories struct {
	ProductPricePlan productpriceplanpb.ProductPricePlanDomainServiceServer // Primary entity repository
	PricePlan        priceplanpb.PricePlanDomainServiceServer               // Entity reference dependency
	ProductPlan      productplanpb.ProductPlanDomainServiceServer           // Entity reference dependency (Model D)
}

// ProductPricePlanServices groups all business service dependencies for product price plan use cases
type ProductPricePlanServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// ReadProductPricePlanRepositories groups all repository dependencies
type ReadProductPricePlanRepositories struct {
	ProductPricePlan productpriceplanpb.ProductPricePlanDomainServiceServer
}

// ReadProductPricePlanServices groups all business service dependencies
type ReadProductPricePlanServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// DeleteProductPricePlanRepositories groups all repository dependencies
type DeleteProductPricePlanRepositories struct {
	ProductPricePlan productpriceplanpb.ProductPricePlanDomainServiceServer
}

// DeleteProductPricePlanServices groups all business service dependencies
type DeleteProductPricePlanServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListProductPricePlansRepositories groups all repository dependencies
type ListProductPricePlansRepositories struct {
	ProductPricePlan productpriceplanpb.ProductPricePlanDomainServiceServer
}

// ListProductPricePlansServices groups all business service dependencies
type ListProductPricePlansServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UseCases contains all product price plan-related use cases
type UseCases struct {
	CreateProductPricePlan          *CreateProductPricePlanUseCase
	ReadProductPricePlan            *ReadProductPricePlanUseCase
	UpdateProductPricePlan          *UpdateProductPricePlanUseCase
	DeleteProductPricePlan          *DeleteProductPricePlanUseCase
	ListProductPricePlans           *ListProductPricePlansUseCase
	GetProductPricePlanListPageData *GetProductPricePlanListPageDataUseCase
	GetProductPricePlanItemPageData *GetProductPricePlanItemPageDataUseCase
}

// NewUseCases creates a new collection of product price plan use cases
func NewUseCases(
	repositories ProductPricePlanRepositories,
	services ProductPricePlanServices,
) *UseCases {
	createRepos := CreateProductPricePlanRepositories{
		ProductPricePlan: repositories.ProductPricePlan,
		PricePlan:        repositories.PricePlan,
		ProductPlan:      repositories.ProductPlan,
	}
	createServices := CreateProductPricePlanServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadProductPricePlanRepositories{
		ProductPricePlan: repositories.ProductPricePlan,
	}
	readServices := ReadProductPricePlanServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateProductPricePlanRepositories{
		ProductPricePlan: repositories.ProductPricePlan,
		PricePlan:        repositories.PricePlan,
		ProductPlan:      repositories.ProductPlan,
	}
	updateServices := UpdateProductPricePlanServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteProductPricePlanRepositories{
		ProductPricePlan: repositories.ProductPricePlan,
	}
	deleteServices := DeleteProductPricePlanServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListProductPricePlansRepositories{
		ProductPricePlan: repositories.ProductPricePlan,
	}
	listServices := ListProductPricePlansServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listPageDataRepos := GetProductPricePlanListPageDataRepositories{
		ProductPricePlan: repositories.ProductPricePlan,
	}
	listPageDataServices := GetProductPricePlanListPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	itemPageDataRepos := GetProductPricePlanItemPageDataRepositories{
		ProductPricePlan: repositories.ProductPricePlan,
	}
	itemPageDataServices := GetProductPricePlanItemPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateProductPricePlan:          NewCreateProductPricePlanUseCase(createRepos, createServices),
		ReadProductPricePlan:            NewReadProductPricePlanUseCase(readRepos, readServices),
		UpdateProductPricePlan:          NewUpdateProductPricePlanUseCase(updateRepos, updateServices),
		DeleteProductPricePlan:          NewDeleteProductPricePlanUseCase(deleteRepos, deleteServices),
		ListProductPricePlans:           NewListProductPricePlansUseCase(listRepos, listServices),
		GetProductPricePlanListPageData: NewGetProductPricePlanListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetProductPricePlanItemPageData: NewGetProductPricePlanItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
	}
}

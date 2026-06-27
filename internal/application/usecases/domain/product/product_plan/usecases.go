package product_plan

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	productvariantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_variant"
)

// ProductPlanRepositories groups all repository dependencies for product plan use cases
type ProductPlanRepositories struct {
	ProductPlan    productplanpb.ProductPlanDomainServiceServer       // Primary entity repository
	Product        productpb.ProductDomainServiceServer               // Entity reference dependency
	ProductVariant productvariantpb.ProductVariantDomainServiceServer // Entity reference dependency (Model D binary invariant)
}

// ProductPlanServices groups all business service dependencies for product plan use cases
type ProductPlanServices struct {
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor // Current: Database transactions
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// ReadProductPlanRepositories groups all repository dependencies
type ReadProductPlanRepositories struct {
	ProductPlan productplanpb.ProductPlanDomainServiceServer // Primary entity repository
}

// ReadProductPlanServices groups all business service dependencies
type ReadProductPlanServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor // Current: Database transactions
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// DeleteProductPlanRepositories groups all repository dependencies
type DeleteProductPlanRepositories struct {
	ProductPlan productplanpb.ProductPlanDomainServiceServer // Primary entity repository
}

// DeleteProductPlanServices groups all business service dependencies
type DeleteProductPlanServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor // Current: Database transactions
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListProductPlansRepositories groups all repository dependencies
type ListProductPlansRepositories struct {
	ProductPlan productplanpb.ProductPlanDomainServiceServer // Primary entity repository
}

// ListProductPlansServices groups all business service dependencies
type ListProductPlansServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor // Current: Database transactions
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UseCases contains all product plan-related use cases
type UseCases struct {
	CreateProductPlan          *CreateProductPlanUseCase
	ReadProductPlan            *ReadProductPlanUseCase
	UpdateProductPlan          *UpdateProductPlanUseCase
	DeleteProductPlan          *DeleteProductPlanUseCase
	ListProductPlans           *ListProductPlansUseCase
	GetProductPlanListPageData *GetProductPlanListPageDataUseCase
	GetProductPlanItemPageData *GetProductPlanItemPageDataUseCase
}

// NewUseCases creates a new collection of product plan use cases
func NewUseCases(
	repositories ProductPlanRepositories,
	services ProductPlanServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateProductPlanRepositories(repositories)
	createServices := CreateProductPlanServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadProductPlanRepositories{
		ProductPlan: repositories.ProductPlan,
	}
	readServices := ReadProductPlanServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateProductPlanRepositories(repositories)
	updateServices := UpdateProductPlanServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteProductPlanRepositories{
		ProductPlan: repositories.ProductPlan,
	}
	deleteServices := DeleteProductPlanServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListProductPlansRepositories{
		ProductPlan: repositories.ProductPlan,
	}
	listServices := ListProductPlansServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listPageDataRepos := GetProductPlanListPageDataRepositories{
		ProductPlan: repositories.ProductPlan,
	}
	listPageDataServices := GetProductPlanListPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	itemPageDataRepos := GetProductPlanItemPageDataRepositories{
		ProductPlan: repositories.ProductPlan,
	}
	itemPageDataServices := GetProductPlanItemPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateProductPlan:          NewCreateProductPlanUseCase(createRepos, createServices),
		ReadProductPlan:            NewReadProductPlanUseCase(readRepos, readServices),
		UpdateProductPlan:          NewUpdateProductPlanUseCase(updateRepos, updateServices),
		DeleteProductPlan:          NewDeleteProductPlanUseCase(deleteRepos, deleteServices),
		ListProductPlans:           NewListProductPlansUseCase(listRepos, listServices),
		GetProductPlanListPageData: NewGetProductPlanListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetProductPlanItemPageData: NewGetProductPlanItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
	}
}

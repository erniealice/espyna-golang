package product_plan

import (
	"leapfor.xyz/espyna/internal/application/ports"
	productpb "leapfor.xyz/esqyma/golang/v1/domain/product/product"
	productplanpb "leapfor.xyz/esqyma/golang/v1/domain/product/product_plan"
)

// ProductPlanRepositories groups all repository dependencies for product plan use cases
type ProductPlanRepositories struct {
	ProductPlan productplanpb.ProductPlanDomainServiceServer // Primary entity repository
	Product     productpb.ProductDomainServiceServer         // Entity reference dependency
}

// ProductPlanServices groups all business service dependencies for product plan use cases
type ProductPlanServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// ReadProductPlanRepositories groups all repository dependencies
type ReadProductPlanRepositories struct {
	ProductPlan productplanpb.ProductPlanDomainServiceServer // Primary entity repository
}

// ReadProductPlanServices groups all business service dependencies
type ReadProductPlanServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// DeleteProductPlanRepositories groups all repository dependencies
type DeleteProductPlanRepositories struct {
	ProductPlan productplanpb.ProductPlanDomainServiceServer // Primary entity repository
}

// DeleteProductPlanServices groups all business service dependencies
type DeleteProductPlanServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// ListProductPlansRepositories groups all repository dependencies
type ListProductPlansRepositories struct {
	ProductPlan productplanpb.ProductPlanDomainServiceServer // Primary entity repository
}

// ListProductPlansServices groups all business service dependencies
type ListProductPlansServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService // Current: Database transactions
	TranslationService   ports.TranslationService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadProductPlanRepositories{
		ProductPlan: repositories.ProductPlan,
	}
	readServices := ReadProductPlanServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	updateRepos := UpdateProductPlanRepositories(repositories)
	updateServices := UpdateProductPlanServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteProductPlanRepositories{
		ProductPlan: repositories.ProductPlan,
	}
	deleteServices := DeleteProductPlanServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	listRepos := ListProductPlansRepositories{
		ProductPlan: repositories.ProductPlan,
	}
	listServices := ListProductPlansServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	listPageDataRepos := GetProductPlanListPageDataRepositories{
		ProductPlan: repositories.ProductPlan,
	}
	listPageDataServices := GetProductPlanListPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	itemPageDataRepos := GetProductPlanItemPageDataRepositories{
		ProductPlan: repositories.ProductPlan,
	}
	itemPageDataServices := GetProductPlanItemPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
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

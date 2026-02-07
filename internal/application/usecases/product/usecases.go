package product

import (
	// Product use cases
	collectionUseCases "leapfor.xyz/espyna/internal/application/usecases/product/collection"
	collectionAttributeUseCases "leapfor.xyz/espyna/internal/application/usecases/product/collection_attribute"
	collectionPlanUseCases "leapfor.xyz/espyna/internal/application/usecases/product/collection_plan"
	priceProductUseCases "leapfor.xyz/espyna/internal/application/usecases/product/price_product"
	productUseCases "leapfor.xyz/espyna/internal/application/usecases/product/product"
	productAttributeUseCases "leapfor.xyz/espyna/internal/application/usecases/product/product_attribute"
	productCollectionUseCases "leapfor.xyz/espyna/internal/application/usecases/product/product_collection"
	productPlanUseCases "leapfor.xyz/espyna/internal/application/usecases/product/product_plan"
	resourceUseCases "leapfor.xyz/espyna/internal/application/usecases/product/resource"

	// Application ports
	"leapfor.xyz/espyna/internal/application/ports"

	// Protobuf domain services for product repositories
	collectionpb "leapfor.xyz/esqyma/golang/v1/domain/product/collection"
	collectionattributepb "leapfor.xyz/esqyma/golang/v1/domain/product/collection_attribute"
	collectionplanpb "leapfor.xyz/esqyma/golang/v1/domain/product/collection_plan"
	priceproductpb "leapfor.xyz/esqyma/golang/v1/domain/product/price_product"
	productpb "leapfor.xyz/esqyma/golang/v1/domain/product/product"
	productattributepb "leapfor.xyz/esqyma/golang/v1/domain/product/product_attribute"
	productcollectionpb "leapfor.xyz/esqyma/golang/v1/domain/product/product_collection"
	productplanpb "leapfor.xyz/esqyma/golang/v1/domain/product/product_plan"
	resourcepb "leapfor.xyz/esqyma/golang/v1/domain/product/resource"

	// Cross-domain dependencies
	attributepb "leapfor.xyz/esqyma/golang/v1/domain/common"
)

// ProductRepositories contains all product domain repositories
type ProductRepositories struct {
	Collection          collectionpb.CollectionDomainServiceServer
	CollectionAttribute collectionattributepb.CollectionAttributeDomainServiceServer
	CollectionPlan      collectionplanpb.CollectionPlanDomainServiceServer
	PriceProduct        priceproductpb.PriceProductDomainServiceServer
	Product             productpb.ProductDomainServiceServer
	ProductAttribute    productattributepb.ProductAttributeDomainServiceServer
	ProductCollection   productcollectionpb.ProductCollectionDomainServiceServer
	ProductPlan         productplanpb.ProductPlanDomainServiceServer
	Resource            resourcepb.ResourceDomainServiceServer
	// Cross-domain dependency
	Attribute attributepb.AttributeDomainServiceServer
}

// ProductUseCases contains all product-related use cases
type ProductUseCases struct {
	Collection          *collectionUseCases.UseCases
	CollectionAttribute *collectionAttributeUseCases.UseCases
	CollectionPlan      *collectionPlanUseCases.UseCases
	PriceProduct        *priceProductUseCases.UseCases
	Product             *productUseCases.UseCases
	ProductAttribute    *productAttributeUseCases.UseCases
	ProductCollection   *productCollectionUseCases.UseCases
	ProductPlan         *productPlanUseCases.UseCases
	Resource            *resourceUseCases.UseCases
}

// NewUseCases creates all product use cases with proper constructor injection
func NewUseCases(
	repos ProductRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idService ports.IDService,
) *ProductUseCases {
	// Create product use cases with proper constructors
	collectionUC := collectionUseCases.NewUseCases(
		collectionUseCases.CollectionRepositories{
			Collection: repos.Collection,
		},
		collectionUseCases.CollectionServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	collectionAttributeUC := collectionAttributeUseCases.NewUseCases(
		collectionAttributeUseCases.CollectionAttributeRepositories{
			CollectionAttribute: repos.CollectionAttribute,
			Collection:          repos.Collection,
			Attribute:           repos.Attribute,
		},
		collectionAttributeUseCases.CollectionAttributeServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	collectionPlanUC := collectionPlanUseCases.NewUseCases(
		collectionPlanUseCases.CollectionPlanRepositories{
			CollectionPlan: repos.CollectionPlan,
		},
		collectionPlanUseCases.CollectionPlanServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	priceProductUC := priceProductUseCases.NewUseCases(
		priceProductUseCases.PriceProductRepositories{
			PriceProduct: repos.PriceProduct,
		},
		priceProductUseCases.PriceProductServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	productUC := productUseCases.NewUseCases(
		productUseCases.ProductRepositories{
			Product: repos.Product,
		},
		productUseCases.ProductServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	productAttributeUC := productAttributeUseCases.NewUseCases(
		productAttributeUseCases.ProductAttributeRepositories{
			ProductAttribute: repos.ProductAttribute,
		},
		productAttributeUseCases.ProductAttributeServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	productCollectionUC := productCollectionUseCases.NewUseCases(
		productCollectionUseCases.ProductCollectionRepositories{
			ProductCollection: repos.ProductCollection,
		},
		productCollectionUseCases.ProductCollectionServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	productPlanUC := productPlanUseCases.NewUseCases(
		productPlanUseCases.ProductPlanRepositories{
			ProductPlan: repos.ProductPlan,
		},
		productPlanUseCases.ProductPlanServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	resourceUC := resourceUseCases.NewUseCases(
		resourceUseCases.ResourceRepositories{
			Resource: repos.Resource,
		},
		resourceUseCases.ResourceServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	return &ProductUseCases{
		Collection:          collectionUC,
		CollectionAttribute: collectionAttributeUC,
		CollectionPlan:      collectionPlanUC,
		PriceProduct:        priceProductUC,
		Product:             productUC,
		ProductAttribute:    productAttributeUC,
		ProductCollection:   productCollectionUC,
		ProductPlan:         productPlanUC,
		Resource:            resourceUC,
	}
}

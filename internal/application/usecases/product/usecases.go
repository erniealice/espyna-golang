package product

import (
	// Product use cases
	collectionUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/product/collection"
	collectionAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/product/collection_attribute"
	collectionPlanUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/product/collection_plan"
	priceListUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/product/price_list"
	priceProductUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/product/price_product"
	productUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/product/product"
	productAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/product/product_attribute"
	productCollectionUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/product/product_collection"
	productOptionUC "github.com/erniealice/espyna-golang/internal/application/usecases/product/product_option"
	productOptionValueUC "github.com/erniealice/espyna-golang/internal/application/usecases/product/product_option_value"
	productPlanUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/product/product_plan"
	productVariantUC "github.com/erniealice/espyna-golang/internal/application/usecases/product/product_variant"
	productVariantImageUC "github.com/erniealice/espyna-golang/internal/application/usecases/product/product_variant_image"
	productVariantOptionUC "github.com/erniealice/espyna-golang/internal/application/usecases/product/product_variant_option"
	resourceUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/product/resource"

	// Application ports
	"github.com/erniealice/espyna-golang/internal/application/ports"

	// Protobuf domain services for product repositories
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection"
	collectionattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection_attribute"
	collectionplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection_plan"
	pricelistpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_list"
	priceproductpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_product"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
	productattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_attribute"
	productcollectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_collection"
	productoptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_option"
	productoptionvaluepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_option_value"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	productvariantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_variant"
	productvariantimagepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_variant_image"
	productvariantoptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_variant_option"
	resourcepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/resource"

	// Cross-domain dependencies
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// ProductRepositories contains all product domain repositories
type ProductRepositories struct {
	Collection          collectionpb.CollectionDomainServiceServer
	CollectionAttribute collectionattributepb.CollectionAttributeDomainServiceServer
	CollectionPlan      collectionplanpb.CollectionPlanDomainServiceServer
	PriceList           pricelistpb.PriceListDomainServiceServer
	PriceProduct        priceproductpb.PriceProductDomainServiceServer
	Product             productpb.ProductDomainServiceServer
	ProductAttribute    productattributepb.ProductAttributeDomainServiceServer
	ProductCollection   productcollectionpb.ProductCollectionDomainServiceServer
	ProductOption       productoptionpb.ProductOptionDomainServiceServer
	ProductOptionValue  productoptionvaluepb.ProductOptionValueDomainServiceServer
	ProductPlan         productplanpb.ProductPlanDomainServiceServer
	ProductVariant      productvariantpb.ProductVariantDomainServiceServer
	ProductVariantImage productvariantimagepb.ProductVariantImageDomainServiceServer
	ProductVariantOption productvariantoptionpb.ProductVariantOptionDomainServiceServer
	Resource            resourcepb.ResourceDomainServiceServer
	// Cross-domain dependency
	Attribute attributepb.AttributeDomainServiceServer
}

// ProductUseCases contains all product-related use cases
type ProductUseCases struct {
	Collection          *collectionUseCases.UseCases
	CollectionAttribute *collectionAttributeUseCases.UseCases
	CollectionPlan      *collectionPlanUseCases.UseCases
	PriceList           *priceListUseCases.UseCases
	PriceProduct        *priceProductUseCases.UseCases
	Product             *productUseCases.UseCases
	ProductAttribute    *productAttributeUseCases.UseCases
	ProductCollection   *productCollectionUseCases.UseCases
	ProductOption       *productOptionUC.UseCases
	ProductOptionValue  *productOptionValueUC.UseCases
	ProductPlan         *productPlanUseCases.UseCases
	ProductVariant      *productVariantUC.UseCases
	ProductVariantImage *productVariantImageUC.UseCases
	ProductVariantOption *productVariantOptionUC.UseCases
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

	priceListUC := priceListUseCases.NewUseCases(
		priceListUseCases.PriceListRepositories{
			PriceList:    repos.PriceList,
			PriceProduct: repos.PriceProduct,
		},
		priceListUseCases.PriceListServices{
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

	productOptionUseCases := productOptionUC.NewUseCases(
		productOptionUC.ProductOptionRepositories{
			ProductOption: repos.ProductOption,
		},
		productOptionUC.ProductOptionServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	productOptionValueUseCases := productOptionValueUC.NewUseCases(
		productOptionValueUC.ProductOptionValueRepositories{
			ProductOptionValue: repos.ProductOptionValue,
		},
		productOptionValueUC.ProductOptionValueServices{
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

	productVariantUseCases := productVariantUC.NewUseCases(
		productVariantUC.ProductVariantRepositories{
			ProductVariant: repos.ProductVariant,
		},
		productVariantUC.ProductVariantServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	productVariantImageUseCases := productVariantImageUC.NewUseCases(
		productVariantImageUC.ProductVariantImageRepositories{
			ProductVariantImage: repos.ProductVariantImage,
		},
		productVariantImageUC.ProductVariantImageServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	productVariantOptionUseCases := productVariantOptionUC.NewUseCases(
		productVariantOptionUC.ProductVariantOptionRepositories{
			ProductVariantOption: repos.ProductVariantOption,
		},
		productVariantOptionUC.ProductVariantOptionServices{
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
		Collection:           collectionUC,
		CollectionAttribute:  collectionAttributeUC,
		CollectionPlan:       collectionPlanUC,
		PriceList:            priceListUC,
		PriceProduct:         priceProductUC,
		Product:              productUC,
		ProductAttribute:     productAttributeUC,
		ProductCollection:    productCollectionUC,
		ProductOption:        productOptionUseCases,
		ProductOptionValue:   productOptionValueUseCases,
		ProductPlan:          productPlanUC,
		ProductVariant:       productVariantUseCases,
		ProductVariantImage:  productVariantImageUseCases,
		ProductVariantOption: productVariantOptionUseCases,
		Resource:             resourceUC,
	}
}

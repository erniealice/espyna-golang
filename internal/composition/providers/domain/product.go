package domain

import (
	"fmt"
	"log"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"

	// Protobuf domain services - Common domain
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"

	// Protobuf domain services - Product domain
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
)

// ProductRepositories contains all product domain repositories and cross-domain dependencies
type ProductRepositories struct {
	Collection           collectionpb.CollectionDomainServiceServer
	CollectionAttribute  collectionattributepb.CollectionAttributeDomainServiceServer
	CollectionPlan       collectionplanpb.CollectionPlanDomainServiceServer
	PriceList            pricelistpb.PriceListDomainServiceServer
	PriceProduct         priceproductpb.PriceProductDomainServiceServer
	Product              productpb.ProductDomainServiceServer
	ProductAttribute     productattributepb.ProductAttributeDomainServiceServer
	ProductCollection    productcollectionpb.ProductCollectionDomainServiceServer
	ProductOption        productoptionpb.ProductOptionDomainServiceServer
	ProductOptionValue   productoptionvaluepb.ProductOptionValueDomainServiceServer
	ProductPlan          productplanpb.ProductPlanDomainServiceServer
	ProductVariant       productvariantpb.ProductVariantDomainServiceServer
	ProductVariantImage  productvariantimagepb.ProductVariantImageDomainServiceServer
	ProductVariantOption productvariantoptionpb.ProductVariantOptionDomainServiceServer
	Resource             resourcepb.ResourceDomainServiceServer
	// Cross-domain dependency from Common domain (optional — nil if adapter not available)
	Attribute attributepb.AttributeDomainServiceServer
}

// NewProductRepositories creates and returns a new set of ProductRepositories
func NewProductRepositories(dbProvider contracts.Provider, dbTableConfig *registry.DatabaseTableConfig) (*ProductRepositories, error) {
	if dbProvider == nil {
		return nil, fmt.Errorf("database provider not initialized")
	}

	repoCreator, ok := dbProvider.(contracts.RepositoryProvider)
	if !ok {
		return nil, fmt.Errorf("database provider doesn't implement contracts.RepositoryProvider interface")
	}

	conn := repoCreator.GetConnection()

	// Create each repository individually using configured table names directly from dbTableConfig
	collectionRepo, err := repoCreator.CreateRepository("collection", conn, dbTableConfig.Collection)
	if err != nil {
		return nil, fmt.Errorf("failed to create collection repository: %w", err)
	}

	collectionAttributeRepo, err := repoCreator.CreateRepository("collection_attribute", conn, dbTableConfig.CollectionAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to create collection_attribute repository: %w", err)
	}

	collectionPlanRepo, err := repoCreator.CreateRepository("collection_plan", conn, dbTableConfig.CollectionPlan)
	if err != nil {
		return nil, fmt.Errorf("failed to create collection_plan repository: %w", err)
	}

	priceListRepo, err := repoCreator.CreateRepository("price_list", conn, dbTableConfig.PriceList)
	if err != nil {
		return nil, fmt.Errorf("failed to create price_list repository: %w", err)
	}

	priceProductRepo, err := repoCreator.CreateRepository("price_product", conn, dbTableConfig.PriceProduct)
	if err != nil {
		return nil, fmt.Errorf("failed to create price_product repository: %w", err)
	}

	productRepo, err := repoCreator.CreateRepository("product", conn, dbTableConfig.Product)
	if err != nil {
		return nil, fmt.Errorf("failed to create product repository: %w", err)
	}

	productAttributeRepo, err := repoCreator.CreateRepository("product_attribute", conn, dbTableConfig.ProductAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to create product_attribute repository: %w", err)
	}

	productCollectionRepo, err := repoCreator.CreateRepository("product_collection", conn, dbTableConfig.ProductCollection)
	if err != nil {
		return nil, fmt.Errorf("failed to create product_collection repository: %w", err)
	}

	productOptionRepo, err := repoCreator.CreateRepository("product_option", conn, dbTableConfig.ProductOption)
	if err != nil {
		return nil, fmt.Errorf("failed to create product_option repository: %w", err)
	}

	productOptionValueRepo, err := repoCreator.CreateRepository("product_option_value", conn, dbTableConfig.ProductOptionValue)
	if err != nil {
		return nil, fmt.Errorf("failed to create product_option_value repository: %w", err)
	}

	productPlanRepo, err := repoCreator.CreateRepository("product_plan", conn, dbTableConfig.ProductPlan)
	if err != nil {
		return nil, fmt.Errorf("failed to create product_plan repository: %w", err)
	}

	productVariantRepo, err := repoCreator.CreateRepository("product_variant", conn, dbTableConfig.ProductVariant)
	if err != nil {
		return nil, fmt.Errorf("failed to create product_variant repository: %w", err)
	}

	productVariantImageRepo, err := repoCreator.CreateRepository("product_variant_image", conn, dbTableConfig.ProductVariantImage)
	if err != nil {
		return nil, fmt.Errorf("failed to create product_variant_image repository: %w", err)
	}

	productVariantOptionRepo, err := repoCreator.CreateRepository("product_variant_option", conn, dbTableConfig.ProductVariantOption)
	if err != nil {
		return nil, fmt.Errorf("failed to create product_variant_option repository: %w", err)
	}

	resourceRepo, err := repoCreator.CreateRepository("resource", conn, dbTableConfig.Resource)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource repository: %w", err)
	}

	// Cross-domain repository - Attribute from Common domain (optional)
	var attributeServer attributepb.AttributeDomainServiceServer
	attributeRepo, err := repoCreator.CreateRepository("attribute", conn, dbTableConfig.Attribute)
	if err != nil {
		log.Printf("  Attribute repository not available for product domain: %v", err)
	} else {
		attributeServer = attributeRepo.(attributepb.AttributeDomainServiceServer)
	}

	// Type assert each repository to its interface
	return &ProductRepositories{
		Collection:           collectionRepo.(collectionpb.CollectionDomainServiceServer),
		CollectionAttribute:  collectionAttributeRepo.(collectionattributepb.CollectionAttributeDomainServiceServer),
		CollectionPlan:       collectionPlanRepo.(collectionplanpb.CollectionPlanDomainServiceServer),
		PriceList:            priceListRepo.(pricelistpb.PriceListDomainServiceServer),
		PriceProduct:         priceProductRepo.(priceproductpb.PriceProductDomainServiceServer),
		Product:              productRepo.(productpb.ProductDomainServiceServer),
		ProductAttribute:     productAttributeRepo.(productattributepb.ProductAttributeDomainServiceServer),
		ProductCollection:    productCollectionRepo.(productcollectionpb.ProductCollectionDomainServiceServer),
		ProductOption:        productOptionRepo.(productoptionpb.ProductOptionDomainServiceServer),
		ProductOptionValue:   productOptionValueRepo.(productoptionvaluepb.ProductOptionValueDomainServiceServer),
		ProductPlan:          productPlanRepo.(productplanpb.ProductPlanDomainServiceServer),
		ProductVariant:       productVariantRepo.(productvariantpb.ProductVariantDomainServiceServer),
		ProductVariantImage:  productVariantImageRepo.(productvariantimagepb.ProductVariantImageDomainServiceServer),
		ProductVariantOption: productVariantOptionRepo.(productvariantoptionpb.ProductVariantOptionDomainServiceServer),
		Resource:             resourceRepo.(resourcepb.ResourceDomainServiceServer),
		Attribute:            attributeServer,
	}, nil
}

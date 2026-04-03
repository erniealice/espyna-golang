package domain

import (
	"fmt"
	"log"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"

	// Protobuf domain services - Common domain
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"

	// Protobuf domain services - Product domain
	linepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/line"
	pricelistpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_list"
	priceproductpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_product"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
	productattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_attribute"
	productlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_line"
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
	PriceList            pricelistpb.PriceListDomainServiceServer
	PriceProduct         priceproductpb.PriceProductDomainServiceServer
	Product              productpb.ProductDomainServiceServer
	ProductAttribute     productattributepb.ProductAttributeDomainServiceServer
	Line                 linepb.LineDomainServiceServer
	ProductLine          productlinepb.ProductLineDomainServiceServer
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
func NewProductRepositories(dbProvider contracts.Provider, tableConfig *registry.TableConfig) (*ProductRepositories, error) {
	if dbProvider == nil {
		return nil, fmt.Errorf("database provider not initialized")
	}

	repoCreator, ok := dbProvider.(contracts.RepositoryProvider)
	if !ok {
		return nil, fmt.Errorf("database provider doesn't implement contracts.RepositoryProvider interface")
	}

	conn := repoCreator.GetConnection()

	// Create each repository individually using configured table names from tableConfig
	priceListRepo, err := repoCreator.CreateRepository(entityid.PriceList, conn, tableConfig.TableName(entityid.PriceList))
	if err != nil {
		return nil, fmt.Errorf("failed to create price_list repository: %w", err)
	}

	priceProductRepo, err := repoCreator.CreateRepository(entityid.PriceProduct, conn, tableConfig.TableName(entityid.PriceProduct))
	if err != nil {
		return nil, fmt.Errorf("failed to create price_product repository: %w", err)
	}

	productRepo, err := repoCreator.CreateRepository(entityid.Product, conn, tableConfig.TableName(entityid.Product))
	if err != nil {
		return nil, fmt.Errorf("failed to create product repository: %w", err)
	}

	productAttributeRepo, err := repoCreator.CreateRepository(entityid.ProductAttribute, conn, tableConfig.TableName(entityid.ProductAttribute))
	if err != nil {
		return nil, fmt.Errorf("failed to create product_attribute repository: %w", err)
	}

	lineRepo, err := repoCreator.CreateRepository(entityid.Line, conn, tableConfig.TableName(entityid.Line))
	if err != nil {
		return nil, fmt.Errorf("failed to create line repository: %w", err)
	}

	productLineRepo, err := repoCreator.CreateRepository(entityid.ProductLine, conn, tableConfig.TableName(entityid.ProductLine))
	if err != nil {
		return nil, fmt.Errorf("failed to create product_line repository: %w", err)
	}

	productOptionRepo, err := repoCreator.CreateRepository(entityid.ProductOption, conn, tableConfig.TableName(entityid.ProductOption))
	if err != nil {
		return nil, fmt.Errorf("failed to create product_option repository: %w", err)
	}

	productOptionValueRepo, err := repoCreator.CreateRepository(entityid.ProductOptionValue, conn, tableConfig.TableName(entityid.ProductOptionValue))
	if err != nil {
		return nil, fmt.Errorf("failed to create product_option_value repository: %w", err)
	}

	productPlanRepo, err := repoCreator.CreateRepository(entityid.ProductPlan, conn, tableConfig.TableName(entityid.ProductPlan))
	if err != nil {
		return nil, fmt.Errorf("failed to create product_plan repository: %w", err)
	}

	productVariantRepo, err := repoCreator.CreateRepository(entityid.ProductVariant, conn, tableConfig.TableName(entityid.ProductVariant))
	if err != nil {
		return nil, fmt.Errorf("failed to create product_variant repository: %w", err)
	}

	productVariantImageRepo, err := repoCreator.CreateRepository(entityid.ProductVariantImage, conn, tableConfig.TableName(entityid.ProductVariantImage))
	if err != nil {
		return nil, fmt.Errorf("failed to create product_variant_image repository: %w", err)
	}

	productVariantOptionRepo, err := repoCreator.CreateRepository(entityid.ProductVariantOption, conn, tableConfig.TableName(entityid.ProductVariantOption))
	if err != nil {
		return nil, fmt.Errorf("failed to create product_variant_option repository: %w", err)
	}

	resourceRepo, err := repoCreator.CreateRepository(entityid.Resource, conn, tableConfig.TableName(entityid.Resource))
	if err != nil {
		return nil, fmt.Errorf("failed to create resource repository: %w", err)
	}

	// Cross-domain repository - Attribute from Common domain (optional)
	var attributeServer attributepb.AttributeDomainServiceServer
	attributeRepo, err := repoCreator.CreateRepository(entityid.Attribute, conn, tableConfig.TableName(entityid.Attribute))
	if err != nil {
		log.Printf("  Attribute repository not available for product domain: %v", err)
	} else {
		attributeServer = attributeRepo.(attributepb.AttributeDomainServiceServer)
	}

	// Type assert each repository to its interface
	return &ProductRepositories{
		PriceList:            priceListRepo.(pricelistpb.PriceListDomainServiceServer),
		PriceProduct:         priceProductRepo.(priceproductpb.PriceProductDomainServiceServer),
		Product:              productRepo.(productpb.ProductDomainServiceServer),
		ProductAttribute:     productAttributeRepo.(productattributepb.ProductAttributeDomainServiceServer),
		Line:                 lineRepo.(linepb.LineDomainServiceServer),
		ProductLine:          productLineRepo.(productlinepb.ProductLineDomainServiceServer),
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

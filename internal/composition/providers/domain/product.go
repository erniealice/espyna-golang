package domain

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"

	// Protobuf domain services - Common domain
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"

	// Protobuf domain services - Product domain
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection"
	collectionattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection_attribute"
	collectionplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection_plan"
	priceproductpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_product"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
	productattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_attribute"
	productcollectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_collection"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	resourcepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/resource"
)

// ProductRepositories contains all 9 product domain repositories and cross-domain dependencies
// Product domain: Collection, CollectionAttribute, CollectionPlan, PriceProduct, Product, ProductAttribute, ProductCollection, ProductPlan, Resource (9 entities)
// Cross-domain: Attribute (needed by CollectionAttribute use case)
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
	// Cross-domain dependency from Common domain
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

	productPlanRepo, err := repoCreator.CreateRepository("product_plan", conn, dbTableConfig.ProductPlan)
	if err != nil {
		return nil, fmt.Errorf("failed to create product_plan repository: %w", err)
	}

	resourceRepo, err := repoCreator.CreateRepository("resource", conn, dbTableConfig.Resource)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource repository: %w", err)
	}

	// Cross-domain repository - Attribute from Common domain
	attributeRepo, err := repoCreator.CreateRepository("attribute", conn, dbTableConfig.Attribute)
	if err != nil {
		return nil, fmt.Errorf("failed to create attribute repository: %w", err)
	}

	// Type assert each repository to its interface
	return &ProductRepositories{
		Collection:          collectionRepo.(collectionpb.CollectionDomainServiceServer),
		CollectionAttribute: collectionAttributeRepo.(collectionattributepb.CollectionAttributeDomainServiceServer),
		CollectionPlan:      collectionPlanRepo.(collectionplanpb.CollectionPlanDomainServiceServer),
		PriceProduct:        priceProductRepo.(priceproductpb.PriceProductDomainServiceServer),
		Product:             productRepo.(productpb.ProductDomainServiceServer),
		ProductAttribute:    productAttributeRepo.(productattributepb.ProductAttributeDomainServiceServer),
		ProductCollection:   productCollectionRepo.(productcollectionpb.ProductCollectionDomainServiceServer),
		ProductPlan:         productPlanRepo.(productplanpb.ProductPlanDomainServiceServer),
		Resource:            resourceRepo.(resourcepb.ResourceDomainServiceServer),
		Attribute:           attributeRepo.(attributepb.AttributeDomainServiceServer),
	}, nil
}

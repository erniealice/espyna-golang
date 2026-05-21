package product

import (
	// Product use cases
	lineUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/product/line"
	priceListUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/product/price_list"
	priceProductUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/product/price_product"
	productUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/product/product"
	productAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/product/product_attribute"
	productLineUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/product/product_line"
	productOptionUC "github.com/erniealice/espyna-golang/internal/application/usecases/domain/product/product_option"
	productOptionValueUC "github.com/erniealice/espyna-golang/internal/application/usecases/domain/product/product_option_value"
	productPlanUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/product/product_plan"
	productVariantUC "github.com/erniealice/espyna-golang/internal/application/usecases/domain/product/product_variant"
	productVariantImageUC "github.com/erniealice/espyna-golang/internal/application/usecases/domain/product/product_variant_image"
	productVariantOptionUC "github.com/erniealice/espyna-golang/internal/application/usecases/domain/product/product_variant_option"
	resourceUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/product/resource"

	// Application ports
	"github.com/erniealice/espyna-golang/internal/application/ports"

	// Protobuf domain services for product repositories
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

	// Cross-domain dependencies
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// ProductRepositories contains all product domain repositories
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
	// Cross-domain dependency
	Attribute attributepb.AttributeDomainServiceServer
}

// ProductUseCases contains all product-related use cases
type ProductUseCases struct {
	Line                 *lineUseCases.UseCases
	PriceList            *priceListUseCases.UseCases
	PriceProduct         *priceProductUseCases.UseCases
	Product              *productUseCases.UseCases
	ProductAttribute     *productAttributeUseCases.UseCases
	ProductLine          *productLineUseCases.UseCases
	ProductOption        *productOptionUC.UseCases
	ProductOptionValue   *productOptionValueUC.UseCases
	ProductPlan          *productPlanUseCases.UseCases
	ProductVariant       *productVariantUC.UseCases
	ProductVariantImage  *productVariantImageUC.UseCases
	ProductVariantOption *productVariantOptionUC.UseCases
	Resource             *resourceUseCases.UseCases

	// Dashboard field retired 2026-05-21 (Wave C P1.C.11 Product) — the
	// dashboard now lives under `service.Dashboard.Product` per Q-SDM-
	// DASHBOARD-DOWNSTREAM. The `usecases/product/dashboard/` package is
	// retired in the same commit; the repository composition relocated to
	// `usecases/service/dashboard/product/`. Source use case was
	// `GetServiceDashboardPageDataUseCase` because it filters
	// product_kind="service"; service-layer surfaces it under the
	// candidate-aligned name `GetProductDashboard`.
}

// NewUseCases creates all product use cases with proper constructor injection
func NewUseCases(
	repos ProductRepositories,
	authSvc ports.Authorizer,
	txSvc ports.Transactor,
	i18nSvc ports.Translator,
	idService ports.IDGenerator,
) *ProductUseCases {
	// Create product use cases with proper constructors
	lineUC := lineUseCases.NewUseCases(
		lineUseCases.LineRepositories{
			Line: repos.Line,
		},
		lineUseCases.LineServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	priceListUC := priceListUseCases.NewUseCases(
		priceListUseCases.PriceListRepositories{
			PriceList:    repos.PriceList,
			PriceProduct: repos.PriceProduct,
		},
		priceListUseCases.PriceListServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	priceProductUC := priceProductUseCases.NewUseCases(
		priceProductUseCases.PriceProductRepositories{
			PriceProduct: repos.PriceProduct,
		},
		priceProductUseCases.PriceProductServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	productUC := productUseCases.NewUseCases(
		productUseCases.ProductRepositories{
			Product: repos.Product,
		},
		productUseCases.ProductServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	productAttributeUC := productAttributeUseCases.NewUseCases(
		productAttributeUseCases.ProductAttributeRepositories{
			ProductAttribute: repos.ProductAttribute,
		},
		productAttributeUseCases.ProductAttributeServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	productLineUC := productLineUseCases.NewUseCases(
		productLineUseCases.ProductLineRepositories{
			ProductLine: repos.ProductLine,
			Product:     repos.Product,
			Line:        repos.Line,
		},
		productLineUseCases.ProductLineServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	productOptionUseCases := productOptionUC.NewUseCases(
		productOptionUC.ProductOptionRepositories{
			ProductOption: repos.ProductOption,
		},
		productOptionUC.ProductOptionServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	productOptionValueUseCases := productOptionValueUC.NewUseCases(
		productOptionValueUC.ProductOptionValueRepositories{
			ProductOptionValue: repos.ProductOptionValue,
		},
		productOptionValueUC.ProductOptionValueServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	productPlanUC := productPlanUseCases.NewUseCases(
		productPlanUseCases.ProductPlanRepositories{
			ProductPlan:    repos.ProductPlan,
			Product:        repos.Product,
			ProductVariant: repos.ProductVariant,
		},
		productPlanUseCases.ProductPlanServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	productVariantUseCases := productVariantUC.NewUseCases(
		productVariantUC.ProductVariantRepositories{
			ProductVariant: repos.ProductVariant,
		},
		productVariantUC.ProductVariantServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	productVariantImageUseCases := productVariantImageUC.NewUseCases(
		productVariantImageUC.ProductVariantImageRepositories{
			ProductVariantImage: repos.ProductVariantImage,
		},
		productVariantImageUC.ProductVariantImageServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	productVariantOptionUseCases := productVariantOptionUC.NewUseCases(
		productVariantOptionUC.ProductVariantOptionRepositories{
			ProductVariantOption: repos.ProductVariantOption,
		},
		productVariantOptionUC.ProductVariantOptionServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	resourceUC := resourceUseCases.NewUseCases(
		resourceUseCases.ResourceRepositories{
			Resource: repos.Resource,
		},
		resourceUseCases.ResourceServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	// Product dashboard wiring retired 2026-05-21 (Wave C P1.C.11) — the
	// type-assertion + factory wiring now lives in the service-layer
	// initializer at `internal/composition/core/initializers/service.go`
	// (search "Wave C P1.C.11 Product").

	return &ProductUseCases{
		Line:                 lineUC,
		PriceList:            priceListUC,
		PriceProduct:         priceProductUC,
		Product:              productUC,
		ProductAttribute:     productAttributeUC,
		ProductLine:          productLineUC,
		ProductOption:        productOptionUseCases,
		ProductOptionValue:   productOptionValueUseCases,
		ProductPlan:          productPlanUC,
		ProductVariant:       productVariantUseCases,
		ProductVariantImage:  productVariantImageUseCases,
		ProductVariantOption: productVariantOptionUseCases,
		Resource:             resourceUC,
	}
}

package domain

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/product"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializeProduct creates all product use cases from provider repositories
// This is composition logic - it wires infrastructure (providers) to application (use cases)
func InitializeProduct(
	repos *domain.ProductRepositories,
	authSvc ports.Authorizer,
	txSvc ports.Transactor,
	i18nSvc ports.Translator,
	idSvc ports.IDGenerator,
	actionGate *actiongate.ActionGatekeeper,
) (*product.ProductUseCases, error) {
	// Use the domain's constructor which properly handles all use case creation
	return product.NewUseCases(
		product.ProductRepositories{
			PriceList:            repos.PriceList,
			PriceProduct:         repos.PriceProduct,
			Product:              repos.Product,
			ProductAttribute:     repos.ProductAttribute,
			Line:                 repos.Line,
			ProductLine:          repos.ProductLine,
			ProductOption:        repos.ProductOption,
			ProductOptionValue:   repos.ProductOptionValue,
			ProductPlan:          repos.ProductPlan,
			ProductVariant:       repos.ProductVariant,
			ProductVariantImage:  repos.ProductVariantImage,
			ProductVariantOption: repos.ProductVariantOption,
			Resource:             repos.Resource,
			Attribute:            repos.Attribute,
		},
		authSvc,
		txSvc,
		i18nSvc,
		idSvc,
		actionGate,
	), nil
}

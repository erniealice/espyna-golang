package revenue_tax_line

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	revenuetaxlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_tax_line"
)

const entityRevenueTaxLine = "revenue_tax_line"

// RevenueTaxLineRepositories groups all repository dependencies for revenue_tax_line use cases.
type RevenueTaxLineRepositories struct {
	RevenueTaxLine revenuetaxlinepb.RevenueTaxLineDomainServiceServer
}

// RevenueTaxLineServices groups all business service dependencies.
type RevenueTaxLineServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all revenue_tax_line use cases.
type UseCases struct {
	ReadRevenueTaxLine            *ReadRevenueTaxLineUseCase
	ListRevenueTaxLines           *ListRevenueTaxLinesUseCase
	CreateRevenueTaxLine          *CreateRevenueTaxLineUseCase
	ListByRevenueRevenueTaxLine   *ListByRevenueRevenueTaxLineUseCase
	DeleteByRevenueRevenueTaxLine *DeleteByRevenueRevenueTaxLineUseCase
}

// NewUseCases creates a new collection of revenue_tax_line use cases.
func NewUseCases(repositories RevenueTaxLineRepositories, services RevenueTaxLineServices) *UseCases {
	return &UseCases{
		ReadRevenueTaxLine: NewReadRevenueTaxLineUseCase(
			ReadRevenueTaxLineRepositories{RevenueTaxLine: repositories.RevenueTaxLine},
			ReadRevenueTaxLineServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		ListRevenueTaxLines: NewListRevenueTaxLinesUseCase(
			ListRevenueTaxLinesRepositories{RevenueTaxLine: repositories.RevenueTaxLine},
			ListRevenueTaxLinesServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		CreateRevenueTaxLine: NewCreateRevenueTaxLineUseCase(
			CreateRevenueTaxLineRepositories{RevenueTaxLine: repositories.RevenueTaxLine},
			CreateRevenueTaxLineServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
				IDService:            services.IDService,
			},
		),
		ListByRevenueRevenueTaxLine: NewListByRevenueRevenueTaxLineUseCase(
			ListByRevenueRepositories{RevenueTaxLine: repositories.RevenueTaxLine},
			ListByRevenueServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		DeleteByRevenueRevenueTaxLine: NewDeleteByRevenueRevenueTaxLineUseCase(
			DeleteByRevenueRepositories{RevenueTaxLine: repositories.RevenueTaxLine},
			DeleteByRevenueServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
	}
}

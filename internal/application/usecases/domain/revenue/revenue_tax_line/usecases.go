package revenue_tax_line

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	revenuetaxlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_tax_line"
)

const entityRevenueTaxLine = "revenue_tax_line"

// RevenueTaxLineRepositories groups all repository dependencies for revenue_tax_line use cases.
type RevenueTaxLineRepositories struct {
	RevenueTaxLine revenuetaxlinepb.RevenueTaxLineDomainServiceServer
}

// RevenueTaxLineServices groups all business service dependencies.
type RevenueTaxLineServices struct {
	Authorizer  ports.Authorizer
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
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
				ActionGatekeeper: services.ActionGatekeeper,
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
		ListRevenueTaxLines: NewListRevenueTaxLinesUseCase(
			ListRevenueTaxLinesRepositories{RevenueTaxLine: repositories.RevenueTaxLine},
			ListRevenueTaxLinesServices{
				ActionGatekeeper: services.ActionGatekeeper,
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
		CreateRevenueTaxLine: NewCreateRevenueTaxLineUseCase(
			CreateRevenueTaxLineRepositories{RevenueTaxLine: repositories.RevenueTaxLine},
			CreateRevenueTaxLineServices{
				ActionGatekeeper: services.ActionGatekeeper,
				Authorizer:  services.Authorizer,
				Translator:  services.Translator,
				IDGenerator: services.IDGenerator,
			},
		),
		ListByRevenueRevenueTaxLine: NewListByRevenueRevenueTaxLineUseCase(
			ListByRevenueRepositories{RevenueTaxLine: repositories.RevenueTaxLine},
			ListByRevenueServices{
				ActionGatekeeper: services.ActionGatekeeper,
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
		DeleteByRevenueRevenueTaxLine: NewDeleteByRevenueRevenueTaxLineUseCase(
			DeleteByRevenueRepositories{RevenueTaxLine: repositories.RevenueTaxLine},
			DeleteByRevenueServices{
				ActionGatekeeper: services.ActionGatekeeper,
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
	}
}

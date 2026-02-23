package revenue

import (
	// Revenue use cases
	revenueUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/revenue/revenue"
	revenueAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/revenue/revenue_attribute"
	revenueCategoryUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/revenue/revenue_category"
	revenueLineItemUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/revenue/revenue_line_item"

	// Application ports
	"github.com/erniealice/espyna-golang/internal/application/ports"

	// Protobuf domain services for revenue repositories
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	revenueattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_attribute"
	revenuecategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_category"
	revenuelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_line_item"
)

// RevenueRepositories contains all revenue domain repositories
type RevenueRepositories struct {
	Revenue          revenuepb.RevenueDomainServiceServer
	RevenueLineItem  revenuelineitempb.RevenueLineItemDomainServiceServer
	RevenueCategory  revenuecategorypb.RevenueCategoryDomainServiceServer
	RevenueAttribute revenueattributepb.RevenueAttributeDomainServiceServer
}

// RevenueUseCases contains all revenue-related use cases
type RevenueUseCases struct {
	Revenue          *revenueUseCases.UseCases
	RevenueLineItem  *revenueLineItemUseCases.UseCases
	RevenueCategory  *revenueCategoryUseCases.UseCases
	RevenueAttribute *revenueAttributeUseCases.UseCases
}

// NewUseCases creates all revenue use cases with proper constructor injection
func NewUseCases(
	repos RevenueRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idService ports.IDService,
) *RevenueUseCases {
	revenueUC := revenueUseCases.NewUseCases(
		revenueUseCases.RevenueRepositories{
			Revenue: repos.Revenue,
		},
		revenueUseCases.RevenueServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	revenueLineItemUC := revenueLineItemUseCases.NewUseCases(
		revenueLineItemUseCases.RevenueLineItemRepositories{
			RevenueLineItem: repos.RevenueLineItem,
		},
		revenueLineItemUseCases.RevenueLineItemServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	revenueCategoryUC := revenueCategoryUseCases.NewUseCases(
		revenueCategoryUseCases.RevenueCategoryRepositories{
			RevenueCategory: repos.RevenueCategory,
		},
		revenueCategoryUseCases.RevenueCategoryServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	revenueAttributeUC := revenueAttributeUseCases.NewUseCases(
		revenueAttributeUseCases.RevenueAttributeRepositories{
			RevenueAttribute: repos.RevenueAttribute,
		},
		revenueAttributeUseCases.RevenueAttributeServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	return &RevenueUseCases{
		Revenue:          revenueUC,
		RevenueLineItem:  revenueLineItemUC,
		RevenueCategory:  revenueCategoryUC,
		RevenueAttribute: revenueAttributeUC,
	}
}

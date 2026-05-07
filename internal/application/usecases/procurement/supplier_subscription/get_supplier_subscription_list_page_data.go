package supplier_subscription

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	suppliersubscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_subscription"
)

type GetSupplierSubscriptionListPageDataRepositories struct {
	SupplierSubscription suppliersubscriptionpb.SupplierSubscriptionDomainServiceServer
}

type GetSupplierSubscriptionListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type GetSupplierSubscriptionListPageDataUseCase struct {
	repositories GetSupplierSubscriptionListPageDataRepositories
	services     GetSupplierSubscriptionListPageDataServices
}

func NewGetSupplierSubscriptionListPageDataUseCase(
	repositories GetSupplierSubscriptionListPageDataRepositories,
	services GetSupplierSubscriptionListPageDataServices,
) *GetSupplierSubscriptionListPageDataUseCase {
	return &GetSupplierSubscriptionListPageDataUseCase{repositories: repositories, services: services}
}

func (uc *GetSupplierSubscriptionListPageDataUseCase) Execute(ctx context.Context, req *suppliersubscriptionpb.GetSupplierSubscriptionListPageDataRequest) (*suppliersubscriptionpb.GetSupplierSubscriptionListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntitySupplierSubscription, ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_subscription.validation.request_required", "request is required"))
	}
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *suppliersubscriptionpb.GetSupplierSubscriptionListPageDataResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.repositories.SupplierSubscription.GetSupplierSubscriptionListPageData(txCtx, req)
			if err != nil {
				return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "supplier_subscription.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load supplier subscription list: %w"), err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return uc.repositories.SupplierSubscription.GetSupplierSubscriptionListPageData(ctx, req)
}

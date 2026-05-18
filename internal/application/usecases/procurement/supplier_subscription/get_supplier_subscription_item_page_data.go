package supplier_subscription

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	suppliersubscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_subscription"
)

type GetSupplierSubscriptionItemPageDataRepositories struct {
	SupplierSubscription suppliersubscriptionpb.SupplierSubscriptionDomainServiceServer
}

type GetSupplierSubscriptionItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type GetSupplierSubscriptionItemPageDataUseCase struct {
	repositories GetSupplierSubscriptionItemPageDataRepositories
	services     GetSupplierSubscriptionItemPageDataServices
}

func NewGetSupplierSubscriptionItemPageDataUseCase(
	repositories GetSupplierSubscriptionItemPageDataRepositories,
	services GetSupplierSubscriptionItemPageDataServices,
) *GetSupplierSubscriptionItemPageDataUseCase {
	return &GetSupplierSubscriptionItemPageDataUseCase{repositories: repositories, services: services}
}

func (uc *GetSupplierSubscriptionItemPageDataUseCase) Execute(ctx context.Context, req *suppliersubscriptionpb.GetSupplierSubscriptionItemPageDataRequest) (*suppliersubscriptionpb.GetSupplierSubscriptionItemPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntitySupplierSubscription, ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil || req.SupplierSubscriptionId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_subscription.validation.id_required", "supplier subscription ID is required"))
	}
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *suppliersubscriptionpb.GetSupplierSubscriptionItemPageDataResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.repositories.SupplierSubscription.GetSupplierSubscriptionItemPageData(txCtx, req)
			if err != nil {
				return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "supplier_subscription.errors.get_item_page_data_failed", "[ERR-DEFAULT] Failed to load supplier subscription details: %w"), err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return uc.repositories.SupplierSubscription.GetSupplierSubscriptionItemPageData(ctx, req)
}

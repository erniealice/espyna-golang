package supplier_subscription

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	suppliersubscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_subscription"
)

type GetSupplierSubscriptionItemPageDataRepositories struct {
	SupplierSubscription suppliersubscriptionpb.SupplierSubscriptionDomainServiceServer
}

type GetSupplierSubscriptionItemPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.SupplierSubscription,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.SupplierSubscriptionId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_subscription.validation.id_required", "supplier subscription ID is required"))
	}
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *suppliersubscriptionpb.GetSupplierSubscriptionItemPageDataResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.repositories.SupplierSubscription.GetSupplierSubscriptionItemPageData(txCtx, req)
			if err != nil {
				return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "supplier_subscription.errors.get_item_page_data_failed", "[ERR-DEFAULT] Failed to load supplier subscription details: %w"), err)
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

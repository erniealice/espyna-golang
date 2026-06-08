package supplier_subscription

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	suppliersubscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_subscription"
)

type ListSupplierSubscriptionsRepositories struct {
	SupplierSubscription suppliersubscriptionpb.SupplierSubscriptionDomainServiceServer
}

type ListSupplierSubscriptionsServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

type ListSupplierSubscriptionsUseCase struct {
	repositories ListSupplierSubscriptionsRepositories
	services     ListSupplierSubscriptionsServices
}

func NewListSupplierSubscriptionsUseCase(
	repositories ListSupplierSubscriptionsRepositories,
	services ListSupplierSubscriptionsServices,
) *ListSupplierSubscriptionsUseCase {
	return &ListSupplierSubscriptionsUseCase{repositories: repositories, services: services}
}

func (uc *ListSupplierSubscriptionsUseCase) Execute(ctx context.Context, req *suppliersubscriptionpb.ListSupplierSubscriptionsRequest) (*suppliersubscriptionpb.ListSupplierSubscriptionsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.SupplierSubscription, entityid.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_subscription.validation.request_required", "request is required"))
	}
	result, err := uc.repositories.SupplierSubscription.ListSupplierSubscriptions(ctx, req)
	if err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_subscription.errors.list_failed", "supplier subscription listing failed")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	return result, nil
}

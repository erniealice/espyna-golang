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

type DeleteSupplierSubscriptionRepositories struct {
	SupplierSubscription suppliersubscriptionpb.SupplierSubscriptionDomainServiceServer
}

type DeleteSupplierSubscriptionServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type DeleteSupplierSubscriptionUseCase struct {
	repositories DeleteSupplierSubscriptionRepositories
	services     DeleteSupplierSubscriptionServices
}

func NewDeleteSupplierSubscriptionUseCase(
	repositories DeleteSupplierSubscriptionRepositories,
	services DeleteSupplierSubscriptionServices,
) *DeleteSupplierSubscriptionUseCase {
	return &DeleteSupplierSubscriptionUseCase{repositories: repositories, services: services}
}

func (uc *DeleteSupplierSubscriptionUseCase) Execute(ctx context.Context, req *suppliersubscriptionpb.DeleteSupplierSubscriptionRequest) (*suppliersubscriptionpb.DeleteSupplierSubscriptionResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.SupplierSubscription,
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_subscription.validation.id_required", "supplier subscription ID is required"))
	}
	result, err := uc.repositories.SupplierSubscription.DeleteSupplierSubscription(ctx, req)
	if err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_subscription.errors.deletion_failed", "supplier subscription deletion failed")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	return result, nil
}

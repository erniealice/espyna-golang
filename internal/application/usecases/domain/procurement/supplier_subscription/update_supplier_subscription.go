package supplier_subscription

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	suppliersubscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_subscription"
)

type UpdateSupplierSubscriptionRepositories struct {
	SupplierSubscription suppliersubscriptionpb.SupplierSubscriptionDomainServiceServer
}

type UpdateSupplierSubscriptionServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type UpdateSupplierSubscriptionUseCase struct {
	repositories UpdateSupplierSubscriptionRepositories
	services     UpdateSupplierSubscriptionServices
}

func NewUpdateSupplierSubscriptionUseCase(
	repositories UpdateSupplierSubscriptionRepositories,
	services UpdateSupplierSubscriptionServices,
) *UpdateSupplierSubscriptionUseCase {
	return &UpdateSupplierSubscriptionUseCase{repositories: repositories, services: services}
}

func (uc *UpdateSupplierSubscriptionUseCase) Execute(ctx context.Context, req *suppliersubscriptionpb.UpdateSupplierSubscriptionRequest) (*suppliersubscriptionpb.UpdateSupplierSubscriptionResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.SupplierSubscription,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_subscription.validation.id_required", "supplier subscription ID is required"))
	}
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	return uc.repositories.SupplierSubscription.UpdateSupplierSubscription(ctx, req)
}

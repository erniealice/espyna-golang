package supplier_subscription

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	suppliersubscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_subscription"
)

type UpdateSupplierSubscriptionRepositories struct {
	SupplierSubscription suppliersubscriptionpb.SupplierSubscriptionDomainServiceServer
}

type UpdateSupplierSubscriptionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
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
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntitySupplierSubscription, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_subscription.validation.id_required", "supplier subscription ID is required"))
	}
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	return uc.repositories.SupplierSubscription.UpdateSupplierSubscription(ctx, req)
}

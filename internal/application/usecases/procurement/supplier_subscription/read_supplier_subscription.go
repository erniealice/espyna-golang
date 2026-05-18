package supplier_subscription

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	suppliersubscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_subscription"
)

type ReadSupplierSubscriptionRepositories struct {
	SupplierSubscription suppliersubscriptionpb.SupplierSubscriptionDomainServiceServer
}

type ReadSupplierSubscriptionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type ReadSupplierSubscriptionUseCase struct {
	repositories ReadSupplierSubscriptionRepositories
	services     ReadSupplierSubscriptionServices
}

func NewReadSupplierSubscriptionUseCase(
	repositories ReadSupplierSubscriptionRepositories,
	services ReadSupplierSubscriptionServices,
) *ReadSupplierSubscriptionUseCase {
	return &ReadSupplierSubscriptionUseCase{repositories: repositories, services: services}
}

func (uc *ReadSupplierSubscriptionUseCase) Execute(ctx context.Context, req *suppliersubscriptionpb.ReadSupplierSubscriptionRequest) (*suppliersubscriptionpb.ReadSupplierSubscriptionResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntitySupplierSubscription, ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_subscription.validation.id_required", "supplier subscription ID is required"))
	}
	return uc.repositories.SupplierSubscription.ReadSupplierSubscription(ctx, req)
}

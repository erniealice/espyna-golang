package supplier_subscription

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	suppliersubscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_subscription"
)

type ReadSupplierSubscriptionRepositories struct {
	SupplierSubscription suppliersubscriptionpb.SupplierSubscriptionDomainServiceServer
}

type ReadSupplierSubscriptionServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
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
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.SupplierSubscription, entityid.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_subscription.validation.id_required", "supplier subscription ID is required"))
	}
	return uc.repositories.SupplierSubscription.ReadSupplierSubscription(ctx, req)
}

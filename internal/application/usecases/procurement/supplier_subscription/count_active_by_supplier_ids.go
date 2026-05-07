package supplier_subscription

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	suppliersubscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_subscription"
)

// CountActiveBySupplierIdsRepositories groups repository dependencies for
// the CountActiveBySupplierIds use case.
type CountActiveBySupplierIdsRepositories struct {
	SupplierSubscription suppliersubscriptionpb.SupplierSubscriptionDomainServiceServer
}

// CountActiveBySupplierIdsServices groups service dependencies for
// the CountActiveBySupplierIds use case.
type CountActiveBySupplierIdsServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// CountActiveBySupplierIdsUseCase counts active supplier subscriptions grouped by supplier ID.
type CountActiveBySupplierIdsUseCase struct {
	repositories CountActiveBySupplierIdsRepositories
	services     CountActiveBySupplierIdsServices
}

// NewCountActiveBySupplierIdsUseCase creates a new CountActiveBySupplierIdsUseCase.
func NewCountActiveBySupplierIdsUseCase(
	repos CountActiveBySupplierIdsRepositories,
	svcs CountActiveBySupplierIdsServices,
) *CountActiveBySupplierIdsUseCase {
	return &CountActiveBySupplierIdsUseCase{
		repositories: repos,
		services:     svcs,
	}
}

// Execute performs an authorization check then delegates to the repository.
func (uc *CountActiveBySupplierIdsUseCase) Execute(ctx context.Context, req *suppliersubscriptionpb.CountActiveBySupplierIdsRequest) (*suppliersubscriptionpb.CountActiveBySupplierIdsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntitySupplierSubscription, ports.ActionList); err != nil {
		return nil, err
	}
	return uc.repositories.SupplierSubscription.CountActiveBySupplierIds(ctx, req)
}

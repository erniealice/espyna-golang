package inventory_transaction

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	inventorytransactionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_transaction"
)

// GetInventoryMovementsListPageDataRepositories groups repository dependencies.
type GetInventoryMovementsListPageDataRepositories struct {
	InventoryTransaction inventorytransactionpb.InventoryTransactionDomainServiceServer
}

// GetInventoryMovementsListPageDataServices groups service dependencies.
type GetInventoryMovementsListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// GetInventoryMovementsListPageDataUseCase returns a JOIN-enriched movement list
// (inventory_transaction ⟕ inventory_item ⟕ product_variant ⟕ product) with
// date, location, type, and full-text search filtering.
type GetInventoryMovementsListPageDataUseCase struct {
	repositories GetInventoryMovementsListPageDataRepositories
	services     GetInventoryMovementsListPageDataServices
}

// NewGetInventoryMovementsListPageDataUseCase creates a new use case.
func NewGetInventoryMovementsListPageDataUseCase(
	repos GetInventoryMovementsListPageDataRepositories,
	svcs GetInventoryMovementsListPageDataServices,
) *GetInventoryMovementsListPageDataUseCase {
	return &GetInventoryMovementsListPageDataUseCase{
		repositories: repos,
		services:     svcs,
	}
}

// Execute performs an authorization check then delegates to the repository.
func (uc *GetInventoryMovementsListPageDataUseCase) Execute(
	ctx context.Context,
	req *inventorytransactionpb.GetInventoryMovementsListPageDataRequest,
) (*inventorytransactionpb.GetInventoryMovementsListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityInventoryTransaction, ports.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_transaction.validation.request_required", "Request is required [DEFAULT]"))
	}

	return uc.repositories.InventoryTransaction.GetInventoryMovementsListPageData(ctx, req)
}

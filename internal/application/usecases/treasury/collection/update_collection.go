package collection

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
)

// UpdateCollectionRepositories groups all repository dependencies
type UpdateCollectionRepositories struct {
	Collection collectionpb.CollectionDomainServiceServer
}

// UpdateCollectionServices groups all business service dependencies
type UpdateCollectionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateCollectionUseCase handles the business logic for updating collections
type UpdateCollectionUseCase struct {
	repositories UpdateCollectionRepositories
	services     UpdateCollectionServices
}

// NewUpdateCollectionUseCase creates use case with grouped dependencies
func NewUpdateCollectionUseCase(
	repositories UpdateCollectionRepositories,
	services UpdateCollectionServices,
) *UpdateCollectionUseCase {
	return &UpdateCollectionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update collection operation.
//
// 20260518-hexagonal-strict-adherence Phase 1.C-iv-pre — BURN_DOWN guard +
// transaction-aware behavior. The BURN_DOWN check moves here from the postgres
// adapter (F4 layer-violation fix). The IsTransactionActive check supports Q1-B
// caller routing — advance workflows call this use case from inside their own
// ExecuteInTransaction(...), so starting a nested tx here would decouple
// commit/rollback from the outer scope.
func (uc *UpdateCollectionUseCase) Execute(ctx context.Context, req *collectionpb.UpdateCollectionRequest) (*collectionpb.UpdateCollectionResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityCollection, ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Plan B Phase 0 hard rule (relocated from contrib/postgres adapter): the
	// ADVANCE_KIND_BURN_DOWN advance kind is reserved for v2 and must be
	// rejected at the use case boundary so callers see a consistent error
	// regardless of which driven adapter is wired.
	if req != nil && req.Data != nil {
		if err := validateAdvanceKindNotBurnDown(req.Data.GetAdvanceKind()); err != nil {
			return nil, err
		}
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		if uc.services.TransactionService.IsTransactionActive(ctx) {
			return uc.executeCore(ctx, req)
		}
		var result *collectionpb.UpdateCollectionResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("collection update failed: %w", err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	return uc.executeCore(ctx, req)
}

func (uc *UpdateCollectionUseCase) executeCore(ctx context.Context, req *collectionpb.UpdateCollectionRequest) (*collectionpb.UpdateCollectionResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection.validation.id_required", "Collection ID is required [DEFAULT]"))
	}

	// Set date_modified
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return uc.repositories.Collection.UpdateCollection(ctx, req)
}

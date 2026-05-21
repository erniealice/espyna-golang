package collection

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
)

const entityCollection = "collection"

// CreateCollectionRepositories groups all repository dependencies
type CreateCollectionRepositories struct {
	Collection collectionpb.CollectionDomainServiceServer
}

// CreateCollectionServices groups all business service dependencies
type CreateCollectionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateCollectionUseCase handles the business logic for creating collections
type CreateCollectionUseCase struct {
	repositories CreateCollectionRepositories
	services     CreateCollectionServices
}

// NewCreateCollectionUseCase creates use case with grouped dependencies
func NewCreateCollectionUseCase(
	repositories CreateCollectionRepositories,
	services CreateCollectionServices,
) *CreateCollectionUseCase {
	return &CreateCollectionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create collection operation.
//
// 20260518-hexagonal-strict-adherence Phase 1.C-iv — BURN_DOWN guard relocated
// from the postgres adapter (F4 layer-violation fix).
func (uc *CreateCollectionUseCase) Execute(ctx context.Context, req *collectionpb.CreateCollectionRequest) (*collectionpb.CreateCollectionResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityCollection, ports.ActionCreate); err != nil {
		return nil, err
	}

	// Plan B Phase 0 hard rule: ADVANCE_KIND_BURN_DOWN is reserved for v2.
	if req != nil && req.Data != nil {
		if err := validateAdvanceKindNotBurnDown(req.Data.GetAdvanceKind()); err != nil {
			return nil, err
		}
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *collectionpb.CreateCollectionResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("collection creation failed: %w", err)
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

func (uc *CreateCollectionUseCase) executeCore(ctx context.Context, req *collectionpb.CreateCollectionRequest) (*collectionpb.CreateCollectionResponse, error) {
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection.validation.data_required", "Collection data is required [DEFAULT]"))
	}

	// Enrich with ID and audit fields
	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDService.GenerateID()
	}
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.Active = true

	return uc.repositories.Collection.CreateCollection(ctx, req)
}

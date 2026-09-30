package collection

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
	collectionapplicationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_application"
)

// DeleteCollectionRepositories groups all repository dependencies
type DeleteCollectionRepositories struct {
	Collection            collectionpb.CollectionDomainServiceServer
	CollectionApplication collectionapplicationpb.CollectionApplicationDomainServiceServer
}

// DeleteCollectionServices groups all business service dependencies
type DeleteCollectionServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// DeleteCollectionUseCase handles the business logic for deleting collections
type DeleteCollectionUseCase struct {
	repositories DeleteCollectionRepositories
	services     DeleteCollectionServices
}

// NewDeleteCollectionUseCase creates a new DeleteCollectionUseCase
func NewDeleteCollectionUseCase(
	repositories DeleteCollectionRepositories,
	services DeleteCollectionServices,
) *DeleteCollectionUseCase {
	return &DeleteCollectionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete collection operation
func (uc *DeleteCollectionUseCase) Execute(ctx context.Context, req *collectionpb.DeleteCollectionRequest) (*collectionpb.DeleteCollectionResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityCollection,
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection.validation.id_required", "Collection ID is required [DEFAULT]"))
	}

	// C25: the applied-receipt check and the soft delete share one transaction under the
	// collection row lock (an ambient transaction is joined, as UpdateCollection does).
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() && !uc.services.Transactor.IsTransactionActive(ctx) {
		var result *collectionpb.DeleteCollectionResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return err
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

func (uc *DeleteCollectionUseCase) executeCore(ctx context.Context, req *collectionpb.DeleteCollectionRequest) (*collectionpb.DeleteCollectionResponse, error) {
	if err := refuseAppliedReceipt(ctx, uc.services.Translator, uc.repositories.Collection, uc.repositories.CollectionApplication, req.Data.Id); err != nil {
		return nil, err
	}
	return uc.repositories.Collection.DeleteCollection(ctx, req)
}

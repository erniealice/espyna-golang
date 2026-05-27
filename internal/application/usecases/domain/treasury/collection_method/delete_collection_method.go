package collectionmethod

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	collectionmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method"
)

// DeleteCollectionMethodRepositories groups all repository dependencies.
type DeleteCollectionMethodRepositories struct {
	CollectionMethod collectionmethodpb.CollectionMethodDomainServiceServer
}

// DeleteCollectionMethodServices groups all business service dependencies.
type DeleteCollectionMethodServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// DeleteCollectionMethodUseCase handles the business logic for deleting collection methods.
type DeleteCollectionMethodUseCase struct {
	repositories DeleteCollectionMethodRepositories
	services     DeleteCollectionMethodServices
}

// NewDeleteCollectionMethodUseCase creates a new DeleteCollectionMethodUseCase.
func NewDeleteCollectionMethodUseCase(
	repositories DeleteCollectionMethodRepositories,
	services DeleteCollectionMethodServices,
) *DeleteCollectionMethodUseCase {
	return &DeleteCollectionMethodUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete collection method operation.
func (uc *DeleteCollectionMethodUseCase) Execute(ctx context.Context, req *collectionmethodpb.DeleteCollectionMethodRequest) (*collectionmethodpb.DeleteCollectionMethodResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityCollectionMethod, ports.ActionDelete); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.id_required", "Collection method ID is required [DEFAULT]"))
	}

	if uc.repositories.CollectionMethod == nil {
		return nil, errors.New("collection method repository is not available")
	}
	return uc.repositories.CollectionMethod.DeleteCollectionMethod(ctx, req)
}

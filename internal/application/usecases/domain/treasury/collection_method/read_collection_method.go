package collectionmethod

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	collectionmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method"
)

// ReadCollectionMethodRepositories groups all repository dependencies.
type ReadCollectionMethodRepositories struct {
	CollectionMethod collectionmethodpb.CollectionMethodDomainServiceServer
}

// ReadCollectionMethodServices groups all business service dependencies.
type ReadCollectionMethodServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ReadCollectionMethodUseCase handles the business logic for reading a collection method.
type ReadCollectionMethodUseCase struct {
	repositories ReadCollectionMethodRepositories
	services     ReadCollectionMethodServices
}

// NewReadCollectionMethodUseCase creates use case with grouped dependencies.
func NewReadCollectionMethodUseCase(
	repositories ReadCollectionMethodRepositories,
	services ReadCollectionMethodServices,
) *ReadCollectionMethodUseCase {
	return &ReadCollectionMethodUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read collection method operation.
func (uc *ReadCollectionMethodUseCase) Execute(ctx context.Context, req *collectionmethodpb.ReadCollectionMethodRequest) (*collectionmethodpb.ReadCollectionMethodResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityCollectionMethod, ports.ActionRead); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.id_required", "Collection method ID is required [DEFAULT]"))
	}

	if uc.repositories.CollectionMethod == nil {
		return nil, errors.New("collection method repository is not available")
	}
	return uc.repositories.CollectionMethod.ReadCollectionMethod(ctx, req)
}

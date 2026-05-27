package collectionmethod

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	collectionmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method"
)

// ListCollectionMethodsRepositories groups all repository dependencies.
type ListCollectionMethodsRepositories struct {
	CollectionMethod collectionmethodpb.CollectionMethodDomainServiceServer
}

// ListCollectionMethodsServices groups all business service dependencies.
type ListCollectionMethodsServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ListCollectionMethodsUseCase handles the business logic for listing collection methods.
type ListCollectionMethodsUseCase struct {
	repositories ListCollectionMethodsRepositories
	services     ListCollectionMethodsServices
}

// NewListCollectionMethodsUseCase creates a new ListCollectionMethodsUseCase.
func NewListCollectionMethodsUseCase(
	repositories ListCollectionMethodsRepositories,
	services ListCollectionMethodsServices,
) *ListCollectionMethodsUseCase {
	return &ListCollectionMethodsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list collection methods operation.
func (uc *ListCollectionMethodsUseCase) Execute(ctx context.Context, req *collectionmethodpb.ListCollectionMethodsRequest) (*collectionmethodpb.ListCollectionMethodsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityCollectionMethod, ports.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.request_required", "Request is required [DEFAULT]"))
	}

	if uc.repositories.CollectionMethod == nil {
		return nil, errors.New("collection method repository is not available")
	}
	return uc.repositories.CollectionMethod.ListCollectionMethods(ctx, req)
}

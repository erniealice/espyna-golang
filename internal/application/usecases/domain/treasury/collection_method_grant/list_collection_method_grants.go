package collectionmethodgrant

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	grantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method_grant"
)

// ListCollectionMethodGrantsRepositories groups all repository dependencies.
type ListCollectionMethodGrantsRepositories struct {
	CollectionMethodGrant grantpb.CollectionMethodGrantDomainServiceServer
}

// ListCollectionMethodGrantsServices groups all business service dependencies.
type ListCollectionMethodGrantsServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ListCollectionMethodGrantsUseCase handles the business logic for listing grants.
type ListCollectionMethodGrantsUseCase struct {
	repositories ListCollectionMethodGrantsRepositories
	services     ListCollectionMethodGrantsServices
}

// NewListCollectionMethodGrantsUseCase creates a new ListCollectionMethodGrantsUseCase.
func NewListCollectionMethodGrantsUseCase(
	repositories ListCollectionMethodGrantsRepositories,
	services ListCollectionMethodGrantsServices,
) *ListCollectionMethodGrantsUseCase {
	return &ListCollectionMethodGrantsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list grants operation.
func (uc *ListCollectionMethodGrantsUseCase) Execute(ctx context.Context, req *grantpb.ListCollectionMethodGrantsRequest) (*grantpb.ListCollectionMethodGrantsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityCollectionMethodGrant, ports.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method_grant.validation.request_required", "Request is required [DEFAULT]"))
	}

	if uc.repositories.CollectionMethodGrant == nil {
		return nil, errors.New("collection method grant repository is not available")
	}
	return uc.repositories.CollectionMethodGrant.ListCollectionMethodGrants(ctx, req)
}

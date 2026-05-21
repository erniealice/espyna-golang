package collection

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
)

// ListByClientRepositories groups repository dependencies for
// the ListByClient use case.
type ListByClientRepositories struct {
	Collection collectionpb.CollectionDomainServiceServer
}

// ListByClientServices groups service dependencies for
// the ListByClient use case.
type ListByClientServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
}

// ListByClientUseCase lists collections for a given client.
type ListByClientUseCase struct {
	repositories ListByClientRepositories
	services     ListByClientServices
}

// NewListByClientUseCase creates a new ListByClientUseCase.
func NewListByClientUseCase(
	repos ListByClientRepositories,
	svcs ListByClientServices,
) *ListByClientUseCase {
	return &ListByClientUseCase{
		repositories: repos,
		services:     svcs,
	}
}

// Execute performs an authorization check then delegates to the repository.
func (uc *ListByClientUseCase) Execute(ctx context.Context, req *collectionpb.ListByClientRequest) (*collectionpb.ListByClientResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityCollection, ports.ActionList); err != nil {
		return nil, err
	}
	return uc.repositories.Collection.ListByClient(ctx, req)
}

package collectionmethodgrant

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	grantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method_grant"
)

// ReadCollectionMethodGrantRepositories groups all repository dependencies.
type ReadCollectionMethodGrantRepositories struct {
	CollectionMethodGrant grantpb.CollectionMethodGrantDomainServiceServer
}

// ReadCollectionMethodGrantServices groups all business service dependencies.
type ReadCollectionMethodGrantServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ReadCollectionMethodGrantUseCase handles the business logic for reading a grant.
type ReadCollectionMethodGrantUseCase struct {
	repositories ReadCollectionMethodGrantRepositories
	services     ReadCollectionMethodGrantServices
}

// NewReadCollectionMethodGrantUseCase creates use case with grouped dependencies.
func NewReadCollectionMethodGrantUseCase(
	repositories ReadCollectionMethodGrantRepositories,
	services ReadCollectionMethodGrantServices,
) *ReadCollectionMethodGrantUseCase {
	return &ReadCollectionMethodGrantUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read grant operation.
func (uc *ReadCollectionMethodGrantUseCase) Execute(ctx context.Context, req *grantpb.ReadCollectionMethodGrantRequest) (*grantpb.ReadCollectionMethodGrantResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityCollectionMethodGrant, ports.ActionRead); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method_grant.validation.id_required", "Collection method grant ID is required [DEFAULT]"))
	}

	if uc.repositories.CollectionMethodGrant == nil {
		return nil, errors.New("collection method grant repository is not available")
	}
	return uc.repositories.CollectionMethodGrant.ReadCollectionMethodGrant(ctx, req)
}

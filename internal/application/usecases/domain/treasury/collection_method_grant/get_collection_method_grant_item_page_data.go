package collectionmethodgrant

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	grantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method_grant"
)

// GetCollectionMethodGrantItemPageDataRepositories groups all repository dependencies.
type GetCollectionMethodGrantItemPageDataRepositories struct {
	CollectionMethodGrant grantpb.CollectionMethodGrantDomainServiceServer
}

// GetCollectionMethodGrantItemPageDataServices groups all business service dependencies.
type GetCollectionMethodGrantItemPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetCollectionMethodGrantItemPageDataUseCase handles fetching a single enriched item.
type GetCollectionMethodGrantItemPageDataUseCase struct {
	repositories GetCollectionMethodGrantItemPageDataRepositories
	services     GetCollectionMethodGrantItemPageDataServices
}

// NewGetCollectionMethodGrantItemPageDataUseCase creates use case with grouped dependencies.
func NewGetCollectionMethodGrantItemPageDataUseCase(
	repositories GetCollectionMethodGrantItemPageDataRepositories,
	services GetCollectionMethodGrantItemPageDataServices,
) *GetCollectionMethodGrantItemPageDataUseCase {
	return &GetCollectionMethodGrantItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get grant item page data operation.
func (uc *GetCollectionMethodGrantItemPageDataUseCase) Execute(ctx context.Context, req *grantpb.GetCollectionMethodGrantItemPageDataRequest) (*grantpb.GetCollectionMethodGrantItemPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityCollectionMethodGrant, ports.ActionRead); err != nil {
		return nil, err
	}

	if req == nil || req.CollectionMethodGrantId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method_grant.validation.id_required", "Collection method grant ID is required [DEFAULT]"))
	}

	if uc.repositories.CollectionMethodGrant == nil {
		return nil, errors.New("collection method grant repository is not available")
	}
	resp, err := uc.repositories.CollectionMethodGrant.GetCollectionMethodGrantItemPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method_grant.errors.get_item_page_data_failed", "[ERR-DEFAULT] Failed to load collection method grant")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

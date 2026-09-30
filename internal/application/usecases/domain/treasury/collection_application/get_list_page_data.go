package collection_application

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	collectionapplicationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_application"
)

// GetCollectionApplicationListPageDataRepositories groups repository dependencies.
type GetCollectionApplicationListPageDataRepositories struct {
	CollectionApplication collectionapplicationpb.CollectionApplicationDomainServiceServer
}

// GetCollectionApplicationListPageDataServices groups service dependencies.
type GetCollectionApplicationListPageDataServices struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// GetCollectionApplicationListPageDataUseCase returns a page of rows with pagination (C11).
type GetCollectionApplicationListPageDataUseCase struct {
	repositories GetCollectionApplicationListPageDataRepositories
	services     GetCollectionApplicationListPageDataServices
}

// NewGetCollectionApplicationListPageDataUseCase creates the use case with grouped dependencies.
func NewGetCollectionApplicationListPageDataUseCase(r GetCollectionApplicationListPageDataRepositories, s GetCollectionApplicationListPageDataServices) *GetCollectionApplicationListPageDataUseCase {
	return &GetCollectionApplicationListPageDataUseCase{repositories: r, services: s}
}

// Execute runs the use case.
func (uc *GetCollectionApplicationListPageDataUseCase) Execute(ctx context.Context, req *collectionapplicationpb.GetCollectionApplicationListPageDataRequest) (*collectionapplicationpb.GetCollectionApplicationListPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.CollectionApplication, Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	if uc.repositories.CollectionApplication == nil {
		return nil, fmt.Errorf("collection_application: repository unavailable")
	}
	if req == nil {
		req = &collectionapplicationpb.GetCollectionApplicationListPageDataRequest{}
	}
	return uc.repositories.CollectionApplication.GetCollectionApplicationListPageData(ctx, req)
}

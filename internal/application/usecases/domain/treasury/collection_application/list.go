package collection_application

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	collectionapplicationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_application"
)

// ListCollectionApplicationsRepositories groups repository dependencies.
type ListCollectionApplicationsRepositories struct {
	CollectionApplication collectionapplicationpb.CollectionApplicationDomainServiceServer
}

// ListCollectionApplicationsServices groups service dependencies.
type ListCollectionApplicationsServices struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListCollectionApplicationsUseCase lists rows of the caller's workspace (pagination honoured by the repository).
type ListCollectionApplicationsUseCase struct {
	repositories ListCollectionApplicationsRepositories
	services     ListCollectionApplicationsServices
}

// NewListCollectionApplicationsUseCase creates the use case with grouped dependencies.
func NewListCollectionApplicationsUseCase(r ListCollectionApplicationsRepositories, s ListCollectionApplicationsServices) *ListCollectionApplicationsUseCase {
	return &ListCollectionApplicationsUseCase{repositories: r, services: s}
}

// Execute runs the use case.
func (uc *ListCollectionApplicationsUseCase) Execute(ctx context.Context, req *collectionapplicationpb.ListCollectionApplicationsRequest) (*collectionapplicationpb.ListCollectionApplicationsResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.CollectionApplication, Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	if uc.repositories.CollectionApplication == nil {
		return nil, fmt.Errorf("collection_application: repository unavailable")
	}
	if req == nil {
		req = &collectionapplicationpb.ListCollectionApplicationsRequest{}
	}
	return uc.repositories.CollectionApplication.ListCollectionApplications(ctx, req)
}

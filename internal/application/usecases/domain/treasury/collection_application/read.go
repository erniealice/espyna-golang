package collection_application

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	collectionapplicationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_application"
)

// ReadCollectionApplicationRepositories groups repository dependencies.
type ReadCollectionApplicationRepositories struct {
	CollectionApplication collectionapplicationpb.CollectionApplicationDomainServiceServer
}

// ReadCollectionApplicationServices groups service dependencies.
type ReadCollectionApplicationServices struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ReadCollectionApplicationUseCase reads one row by id (a foreign-workspace and a missing id are both "not found").
type ReadCollectionApplicationUseCase struct {
	repositories ReadCollectionApplicationRepositories
	services     ReadCollectionApplicationServices
}

// NewReadCollectionApplicationUseCase creates the use case with grouped dependencies.
func NewReadCollectionApplicationUseCase(r ReadCollectionApplicationRepositories, s ReadCollectionApplicationServices) *ReadCollectionApplicationUseCase {
	return &ReadCollectionApplicationUseCase{repositories: r, services: s}
}

// Execute runs the use case.
func (uc *ReadCollectionApplicationUseCase) Execute(ctx context.Context, req *collectionapplicationpb.ReadCollectionApplicationRequest) (*collectionapplicationpb.ReadCollectionApplicationResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.CollectionApplication, Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	if uc.repositories.CollectionApplication == nil {
		return nil, fmt.Errorf("collection_application: repository unavailable")
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection_application: id is required")
	}
	return uc.repositories.CollectionApplication.ReadCollectionApplication(ctx, req)
}

package collection_method

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	collectionmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method"
)

// ReadCollectionMethodRepositories groups all repository dependencies
type ReadCollectionMethodRepositories struct {
	CollectionMethod collectionmethodpb.CollectionMethodDomainServiceServer // Primary entity repository
}

// ReadCollectionMethodServices groups all business service dependencies
type ReadCollectionMethodServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ReadCollectionMethodUseCase handles the business logic for reading collection methods
type ReadCollectionMethodUseCase struct {
	repositories ReadCollectionMethodRepositories
	services     ReadCollectionMethodServices
}

// NewReadCollectionMethodUseCase creates use case with grouped dependencies
func NewReadCollectionMethodUseCase(
	repositories ReadCollectionMethodRepositories,
	services ReadCollectionMethodServices,
) *ReadCollectionMethodUseCase {
	return &ReadCollectionMethodUseCase{
		repositories: repositories,
		services:     services,
	}
}

func (uc *ReadCollectionMethodUseCase) Execute(ctx context.Context, req *collectionmethodpb.ReadCollectionMethodRequest) (*collectionmethodpb.ReadCollectionMethodResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.CollectionMethod, entityid.ActionRead); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.CollectionMethod.ReadCollectionMethod(ctx, req)
	if err != nil {
		return nil, err
	}

	// Return response as-is (even if empty data for not found case)
	return resp, nil
}

// validateInput validates the input request
func (uc *ReadCollectionMethodUseCase) validateInput(ctx context.Context, req *collectionmethodpb.ReadCollectionMethodRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.data_required", "[ERR-DEFAULT] Data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.id_required", "[ERR-DEFAULT] ID is required"))
	}
	return nil
}

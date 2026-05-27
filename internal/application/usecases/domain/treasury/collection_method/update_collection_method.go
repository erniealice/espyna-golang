package collectionmethod

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	collectionmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method"
)

// UpdateCollectionMethodRepositories groups all repository dependencies.
type UpdateCollectionMethodRepositories struct {
	CollectionMethod collectionmethodpb.CollectionMethodDomainServiceServer
}

// UpdateCollectionMethodServices groups all business service dependencies.
type UpdateCollectionMethodServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// UpdateCollectionMethodUseCase handles the business logic for updating collection methods.
type UpdateCollectionMethodUseCase struct {
	repositories UpdateCollectionMethodRepositories
	services     UpdateCollectionMethodServices
}

// NewUpdateCollectionMethodUseCase creates use case with grouped dependencies.
func NewUpdateCollectionMethodUseCase(
	repositories UpdateCollectionMethodRepositories,
	services UpdateCollectionMethodServices,
) *UpdateCollectionMethodUseCase {
	return &UpdateCollectionMethodUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update collection method operation.
//
// Transaction-aware (mirrors UpdateCollection): when called from inside an
// outer ExecuteInTransaction (e.g. a lifecycle transition use case fires its
// terminal write through this wrapper), it short-circuits to executeCore so no
// nested independent tx is started.
func (uc *UpdateCollectionMethodUseCase) Execute(ctx context.Context, req *collectionmethodpb.UpdateCollectionMethodRequest) (*collectionmethodpb.UpdateCollectionMethodResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityCollectionMethod, ports.ActionUpdate); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		if uc.services.Transactor.IsTransactionActive(ctx) {
			return uc.executeCore(ctx, req)
		}
		var result *collectionmethodpb.UpdateCollectionMethodResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("collection method update failed: %w", err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	return uc.executeCore(ctx, req)
}

func (uc *UpdateCollectionMethodUseCase) executeCore(ctx context.Context, req *collectionmethodpb.UpdateCollectionMethodRequest) (*collectionmethodpb.UpdateCollectionMethodResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.id_required", "Collection method ID is required [DEFAULT]"))
	}

	req.Data.Name = strings.TrimSpace(req.Data.Name)
	if req.Data.Name == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.name_required", "[ERR-DEFAULT] Name is required"))
	}

	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	if uc.repositories.CollectionMethod == nil {
		return nil, errors.New("collection method repository is not available")
	}
	return uc.repositories.CollectionMethod.UpdateCollectionMethod(ctx, req)
}

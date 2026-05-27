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

// entityCollectionMethod is the permission namespace + translation key root.
const entityCollectionMethod = "collection_method"

// CreateCollectionMethodRepositories groups all repository dependencies.
type CreateCollectionMethodRepositories struct {
	CollectionMethod collectionmethodpb.CollectionMethodDomainServiceServer
}

// CreateCollectionMethodServices groups all business service dependencies.
type CreateCollectionMethodServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// CreateCollectionMethodUseCase handles the business logic for creating collection methods.
type CreateCollectionMethodUseCase struct {
	repositories CreateCollectionMethodRepositories
	services     CreateCollectionMethodServices
}

// NewCreateCollectionMethodUseCase creates use case with grouped dependencies.
func NewCreateCollectionMethodUseCase(
	repositories CreateCollectionMethodRepositories,
	services CreateCollectionMethodServices,
) *CreateCollectionMethodUseCase {
	return &CreateCollectionMethodUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create collection method operation.
func (uc *CreateCollectionMethodUseCase) Execute(ctx context.Context, req *collectionmethodpb.CreateCollectionMethodRequest) (*collectionmethodpb.CreateCollectionMethodResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityCollectionMethod, ports.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *collectionmethodpb.CreateCollectionMethodResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "collection_method.errors.creation_failed", "Collection method creation failed [DEFAULT]")
				return fmt.Errorf("%s: %w", translatedError, err)
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

func (uc *CreateCollectionMethodUseCase) executeCore(ctx context.Context, req *collectionmethodpb.CreateCollectionMethodRequest) (*collectionmethodpb.CreateCollectionMethodResponse, error) {
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	if err := uc.enrichData(req.Data); err != nil {
		return nil, err
	}

	if uc.repositories.CollectionMethod == nil {
		return nil, errors.New("collection method repository is not available")
	}
	return uc.repositories.CollectionMethod.CreateCollectionMethod(ctx, req)
}

func (uc *CreateCollectionMethodUseCase) validateInput(ctx context.Context, req *collectionmethodpb.CreateCollectionMethodRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.data_required", "[ERR-DEFAULT] Collection method data is required"))
	}

	req.Data.Name = strings.TrimSpace(req.Data.Name)
	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.name_required", "[ERR-DEFAULT] Name is required"))
	}

	// D-1.5 Q3 STRICT: tax_effect_kind is REQUIRED on the template.
	if req.Data.GetTaxEffectKind() == collectionmethodpb.CollectionMethodTaxEffectKind_COLLECTION_METHOD_TAX_EFFECT_KIND_UNSPECIFIED {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.tax_effect_kind_required", "[ERR-DEFAULT] Tax effect kind is required"))
	}

	return nil
}

func (uc *CreateCollectionMethodUseCase) enrichData(cm *collectionmethodpb.CollectionMethod) error {
	now := time.Now()
	if cm.Id == "" {
		cm.Id = uc.services.IDGenerator.GenerateID()
	}
	cm.DateCreated = &[]int64{now.UnixMilli()}[0]
	cm.DateModified = &[]int64{now.UnixMilli()}[0]
	cm.Active = true

	// New templates begin life as DRAFT (D-1.8 lifecycle). The version state
	// mirrors DRAFT until PublishCollectionMethod promotes it.
	if cm.GetLifecycle() == collectionmethodpb.CollectionMethodLifecycle_COLLECTION_METHOD_LIFECYCLE_UNSPECIFIED {
		cm.Lifecycle = collectionmethodpb.CollectionMethodLifecycle_COLLECTION_METHOD_LIFECYCLE_DRAFT
	}
	if cm.GetVersionStatus() == collectionmethodpb.CollectionMethodVersionStatus_COLLECTION_METHOD_VERSION_STATUS_UNSPECIFIED {
		cm.VersionStatus = collectionmethodpb.CollectionMethodVersionStatus_COLLECTION_METHOD_VERSION_STATUS_DRAFT
	}
	if cm.Revision == 0 {
		cm.Revision = 1
	}
	return nil
}

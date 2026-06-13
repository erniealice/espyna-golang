package collection_method

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	collectionmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method"
)

// CreateCollectionMethodRepositories groups all repository dependencies
type CreateCollectionMethodRepositories struct {
	CollectionMethod collectionmethodpb.CollectionMethodDomainServiceServer // Primary entity repository
}

// CreateCollectionMethodServices groups all business service dependencies
type CreateCollectionMethodServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// CreateCollectionMethodUseCase handles the business logic for creating collection methods
type CreateCollectionMethodUseCase struct {
	repositories CreateCollectionMethodRepositories
	services     CreateCollectionMethodServices
}

// NewCreateCollectionMethodUseCase creates use case with grouped dependencies
func NewCreateCollectionMethodUseCase(
	repositories CreateCollectionMethodRepositories,
	services CreateCollectionMethodServices,
) *CreateCollectionMethodUseCase {
	return &CreateCollectionMethodUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateCollectionMethodUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateCollectionMethodUseCase with grouped parameters instead
func NewCreateCollectionMethodUseCaseUngrouped(collectionMethodRepo collectionmethodpb.CollectionMethodDomainServiceServer) *CreateCollectionMethodUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := CreateCollectionMethodRepositories{
		CollectionMethod: collectionMethodRepo,
	}

	services := CreateCollectionMethodServices{
		Authorizer:  nil,
		Transactor:  ports.NewNoOpTransactor(),
		Translator:  ports.NewNoOpTranslator(),
		IDGenerator: ports.NewNoOpIDGenerator(),
	}

	return NewCreateCollectionMethodUseCase(repositories, services)
}

// Execute performs the create collection method operation
func (uc *CreateCollectionMethodUseCase) Execute(ctx context.Context, req *collectionmethodpb.CreateCollectionMethodRequest) (*collectionmethodpb.CreateCollectionMethodResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.CollectionMethod,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes collection method creation within a transaction
func (uc *CreateCollectionMethodUseCase) executeWithTransaction(ctx context.Context, req *collectionmethodpb.CreateCollectionMethodRequest) (*collectionmethodpb.CreateCollectionMethodResponse, error) {
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

// executeCore contains the core business logic
func (uc *CreateCollectionMethodUseCase) executeCore(ctx context.Context, req *collectionmethodpb.CreateCollectionMethodRequest) (*collectionmethodpb.CreateCollectionMethodResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichCollectionMethodData(req.Data); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	return uc.repositories.CollectionMethod.CreateCollectionMethod(ctx, req)
}

// validateInput validates the input request
func (uc *CreateCollectionMethodUseCase) validateInput(ctx context.Context, req *collectionmethodpb.CreateCollectionMethodRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.data_required", "[ERR-DEFAULT] Collection method data is required"))
	}

	// Trim leading and trailing spaces
	req.Data.Name = strings.TrimSpace(req.Data.Name)
	if req.Data.ProviderName != nil {
		trimmed := strings.TrimSpace(*req.Data.ProviderName)
		req.Data.ProviderName = &trimmed
	}

	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.name_required", "[ERR-DEFAULT] Name is required"))
	}
	return nil
}

// enrichCollectionMethodData adds generated fields and audit information
func (uc *CreateCollectionMethodUseCase) enrichCollectionMethodData(collectionMethod *collectionmethodpb.CollectionMethod) error {
	now := time.Now()

	// Generate CollectionMethod ID if not provided
	if collectionMethod.Id == "" {
		collectionMethod.Id = uc.services.IDGenerator.GenerateID()
	}

	// Set audit fields
	collectionMethod.DateCreated = &[]int64{now.UnixMilli()}[0]
	collectionMethod.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	collectionMethod.DateModified = &[]int64{now.UnixMilli()}[0]
	collectionMethod.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	collectionMethod.Active = true

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateCollectionMethodUseCase) validateBusinessRules(ctx context.Context, collectionMethod *collectionmethodpb.CollectionMethod) error {
	// Validate name length
	if len(collectionMethod.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.name_too_long", "[ERR-DEFAULT] Name must not exceed 100 characters"))
	}

	return nil
}

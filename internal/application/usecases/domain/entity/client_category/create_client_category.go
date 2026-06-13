package client_category

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	clientcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_category"
)

// CreateClientCategoryRepositories groups all repository dependencies
type CreateClientCategoryRepositories struct {
	ClientCategory clientcategorypb.ClientCategoryDomainServiceServer // Primary entity repository
}

// CreateClientCategoryServices groups all business service dependencies
type CreateClientCategoryServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// CreateClientCategoryUseCase handles the business logic for creating client categories
type CreateClientCategoryUseCase struct {
	repositories CreateClientCategoryRepositories
	services     CreateClientCategoryServices
}

// NewCreateClientCategoryUseCase creates use case with grouped dependencies
func NewCreateClientCategoryUseCase(
	repositories CreateClientCategoryRepositories,
	services CreateClientCategoryServices,
) *CreateClientCategoryUseCase {
	return &CreateClientCategoryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateClientCategoryUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateClientCategoryUseCase with grouped parameters instead
func NewCreateClientCategoryUseCaseUngrouped(clientCategoryRepo clientcategorypb.ClientCategoryDomainServiceServer) *CreateClientCategoryUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := CreateClientCategoryRepositories{
		ClientCategory: clientCategoryRepo,
	}

	services := CreateClientCategoryServices{
		Authorizer:  nil,
		Transactor:  ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
		IDGenerator: ports.NewNoOpIDGenerator(),
	}

	return NewCreateClientCategoryUseCase(repositories, services)
}

// Execute performs the create client_category operation
func (uc *CreateClientCategoryUseCase) Execute(ctx context.Context, req *clientcategorypb.CreateClientCategoryRequest) (*clientcategorypb.CreateClientCategoryResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: "client_category",
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

// executeWithTransaction executes client_category creation within a transaction
func (uc *CreateClientCategoryUseCase) executeWithTransaction(ctx context.Context, req *clientcategorypb.CreateClientCategoryRequest) (*clientcategorypb.CreateClientCategoryResponse, error) {
	var result *clientcategorypb.CreateClientCategoryResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "client_category.errors.creation_failed", "Client category creation failed [DEFAULT]")
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
func (uc *CreateClientCategoryUseCase) executeCore(ctx context.Context, req *clientcategorypb.CreateClientCategoryRequest) (*clientcategorypb.CreateClientCategoryResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichClientCategoryData(req.Data); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	return uc.repositories.ClientCategory.CreateClientCategory(ctx, req)
}

// validateInput validates the input request
func (uc *CreateClientCategoryUseCase) validateInput(ctx context.Context, req *clientcategorypb.CreateClientCategoryRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client_category.validation.request_required", "Request is required for client categories [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client_category.validation.data_required", "Client category data is required [DEFAULT]"))
	}
	if req.Data.ClientId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client_category.validation.client_id_required", "Client ID is required for client categories [DEFAULT]"))
	}
	if req.Data.CategoryId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client_category.validation.category_id_required", "Category ID is required for client categories [DEFAULT]"))
	}
	return nil
}

// enrichClientCategoryData adds generated fields and audit information
func (uc *CreateClientCategoryUseCase) enrichClientCategoryData(category *clientcategorypb.ClientCategory) error {
	now := time.Now()

	// Generate Client Category ID if not provided
	if category.Id == "" {
		category.Id = uc.services.IDGenerator.GenerateID()
	}

	// Set audit fields
	category.DateCreated = &[]int64{now.UnixMilli()}[0]
	category.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	category.DateModified = &[]int64{now.UnixMilli()}[0]
	category.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	category.Active = true

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateClientCategoryUseCase) validateBusinessRules(ctx context.Context, category *clientcategorypb.ClientCategory) error {
	// Validate client ID format if needed
	if len(category.ClientId) < 1 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client_category.validation.client_id_invalid", "Client ID is invalid [DEFAULT]"))
	}

	// Validate category ID format if needed
	if len(category.CategoryId) < 1 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client_category.validation.category_id_invalid", "Category ID is invalid [DEFAULT]"))
	}

	return nil
}

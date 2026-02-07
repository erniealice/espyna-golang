package location

import (
	"context"
	"errors"
	"fmt"
	"time"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	locationpb "leapfor.xyz/esqyma/golang/v1/domain/entity/location"
)

// CreateLocationRepositories groups all repository dependencies
type CreateLocationRepositories struct {
	Location locationpb.LocationDomainServiceServer // Primary entity repository
}

// CreateLocationServices groups all business service dependencies
type CreateLocationServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateLocationUseCase handles the business logic for creating locations
type CreateLocationUseCase struct {
	repositories CreateLocationRepositories
	services     CreateLocationServices
}

// NewCreateLocationUseCase creates use case with grouped dependencies
func NewCreateLocationUseCase(
	repositories CreateLocationRepositories,
	services CreateLocationServices,
) *CreateLocationUseCase {
	return &CreateLocationUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateLocationUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateLocationUseCase with grouped parameters instead
func NewCreateLocationUseCaseUngrouped(locationRepo locationpb.LocationDomainServiceServer) *CreateLocationUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := CreateLocationRepositories{
		Location: locationRepo,
	}

	services := CreateLocationServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewCreateLocationUseCase(repositories, services)
}

// Execute performs the create location operation
func (uc *CreateLocationUseCase) Execute(ctx context.Context, req *locationpb.CreateLocationRequest) (*locationpb.CreateLocationResponse, error) {
	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes location creation within a transaction
func (uc *CreateLocationUseCase) executeWithTransaction(ctx context.Context, req *locationpb.CreateLocationRequest) (*locationpb.CreateLocationResponse, error) {
	var result *locationpb.CreateLocationResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "location.errors.creation_failed", "Location creation failed [DEFAULT]")
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

// executeCore contains the core business logic (moved from original Execute method)
func (uc *CreateLocationUseCase) executeCore(ctx context.Context, req *locationpb.CreateLocationRequest) (*locationpb.CreateLocationResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichLocationData(req.Data); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	return uc.repositories.Location.CreateLocation(ctx, req)
}

// validateInput validates the input request
func (uc *CreateLocationUseCase) validateInput(ctx context.Context, req *locationpb.CreateLocationRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.validation.request_required", ""))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.validation.data_required", ""))
	}
	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.validation.name_required", ""))
	}
	if req.Data.Address == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.validation.address_required", ""))
	}
	return nil
}

// enrichLocationData adds generated fields and audit information
func (uc *CreateLocationUseCase) enrichLocationData(location *locationpb.Location) error {
	now := time.Now()

	// Generate Location ID if not provided
	if location.Id == "" {
		location.Id = uc.services.IDService.GenerateID()
	}

	// Set location audit fields
	location.DateCreated = &[]int64{now.UnixMilli()}[0]
	location.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	location.DateModified = &[]int64{now.UnixMilli()}[0]
	location.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	location.Active = true

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateLocationUseCase) validateBusinessRules(ctx context.Context, location *locationpb.Location) error {
	// Validate name length
	if len(location.Name) < 2 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.validation.name_too_short", ""))
	}

	if len(location.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.validation.name_too_long", ""))
	}

	// Validate address length
	if len(location.Address) < 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.validation.address_too_short", ""))
	}

	if len(location.Address) > 500 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.validation.address_too_long", ""))
	}

	// Validate description length if provided
	if location.Description != nil && len(*location.Description) > 1000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.validation.description_too_long", ""))
	}

	return nil
}

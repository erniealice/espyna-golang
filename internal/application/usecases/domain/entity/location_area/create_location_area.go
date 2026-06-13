package location_area

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	locationareapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location_area"
)

// CreateLocationAreaRepositories groups all repository dependencies
type CreateLocationAreaRepositories struct {
	LocationArea locationareapb.LocationAreaDomainServiceServer // Primary entity repository
}

// CreateLocationAreaServices groups all business service dependencies
type CreateLocationAreaServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// CreateLocationAreaUseCase handles the business logic for creating location areas
type CreateLocationAreaUseCase struct {
	repositories CreateLocationAreaRepositories
	services     CreateLocationAreaServices
}

// NewCreateLocationAreaUseCase creates use case with grouped dependencies
func NewCreateLocationAreaUseCase(
	repositories CreateLocationAreaRepositories,
	services CreateLocationAreaServices,
) *CreateLocationAreaUseCase {
	return &CreateLocationAreaUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateLocationAreaUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateLocationAreaUseCase with grouped parameters instead
func NewCreateLocationAreaUseCaseUngrouped(locationAreaRepo locationareapb.LocationAreaDomainServiceServer) *CreateLocationAreaUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := CreateLocationAreaRepositories{
		LocationArea: locationAreaRepo,
	}

	services := CreateLocationAreaServices{
		Authorizer:  nil,
		Transactor:  ports.NewNoOpTransactor(),
		Translator:  ports.NewNoOpTranslator(),
		IDGenerator: ports.NewNoOpIDGenerator(),
	}

	return NewCreateLocationAreaUseCase(repositories, services)
}

// Execute performs the create location area operation
func (uc *CreateLocationAreaUseCase) Execute(ctx context.Context, req *locationareapb.CreateLocationAreaRequest) (*locationareapb.CreateLocationAreaResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.LocationArea, entityid.ActionCreate); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes location area creation within a transaction
func (uc *CreateLocationAreaUseCase) executeWithTransaction(ctx context.Context, req *locationareapb.CreateLocationAreaRequest) (*locationareapb.CreateLocationAreaResponse, error) {
	var result *locationareapb.CreateLocationAreaResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "location_area.errors.creation_failed", "Location area creation failed [DEFAULT]")
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
func (uc *CreateLocationAreaUseCase) executeCore(ctx context.Context, req *locationareapb.CreateLocationAreaRequest) (*locationareapb.CreateLocationAreaResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichLocationAreaData(req.Data); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	return uc.repositories.LocationArea.CreateLocationArea(ctx, req)
}

// validateInput validates the input request
func (uc *CreateLocationAreaUseCase) validateInput(ctx context.Context, req *locationareapb.CreateLocationAreaRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_area.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_area.validation.data_required", "[ERR-DEFAULT] Location area data is required"))
	}

	// Trim leading and trailing spaces
	req.Data.Name = strings.TrimSpace(req.Data.Name)
	req.Data.Description = strings.TrimSpace(req.Data.Description)

	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_area.validation.name_required", "[ERR-DEFAULT] Name is required"))
	}
	return nil
}

// enrichLocationAreaData adds generated fields and audit information
func (uc *CreateLocationAreaUseCase) enrichLocationAreaData(locationArea *locationareapb.LocationArea) error {
	now := time.Now()

	// Generate Location Area ID if not provided
	if locationArea.Id == "" {
		locationArea.Id = uc.services.IDGenerator.GenerateID()
	}

	// Set location area audit fields
	locationArea.DateCreated = &[]int64{now.UnixMilli()}[0]
	locationArea.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	locationArea.DateModified = &[]int64{now.UnixMilli()}[0]
	locationArea.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	locationArea.Active = true

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateLocationAreaUseCase) validateBusinessRules(ctx context.Context, locationArea *locationareapb.LocationArea) error {
	// Validate name length
	if len(locationArea.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_area.validation.name_too_long", "[ERR-DEFAULT] Name must not exceed 100 characters"))
	}

	// Validate description length if provided
	if len(locationArea.Description) > 1000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_area.validation.description_too_long", "[ERR-DEFAULT] Description must not exceed 1000 characters"))
	}

	return nil
}

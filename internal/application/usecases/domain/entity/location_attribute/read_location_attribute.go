package location_attribute

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	locationattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location_attribute"
)

// ReadLocationAttributeRepositories groups all repository dependencies
type ReadLocationAttributeRepositories struct {
	LocationAttribute locationattributepb.LocationAttributeDomainServiceServer // Primary entity repository
}

// ReadLocationAttributeServices groups all business service dependencies
type ReadLocationAttributeServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor // Current: Database transactions
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ReadLocationAttributeUseCase handles the business logic for reading location attributes
type ReadLocationAttributeUseCase struct {
	repositories ReadLocationAttributeRepositories
	services     ReadLocationAttributeServices
}

// NewReadLocationAttributeUseCase creates a new ReadLocationAttributeUseCase
func NewReadLocationAttributeUseCase(
	repositories ReadLocationAttributeRepositories,
	services ReadLocationAttributeServices,
) *ReadLocationAttributeUseCase {
	return &ReadLocationAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadLocationAttributeUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewReadLocationAttributeUseCase with grouped parameters instead
func NewReadLocationAttributeUseCaseUngrouped(
	locationAttributeRepo locationattributepb.LocationAttributeDomainServiceServer,
) *ReadLocationAttributeUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ReadLocationAttributeRepositories{
		LocationAttribute: locationAttributeRepo,
	}

	services := ReadLocationAttributeServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewReadLocationAttributeUseCase(repositories, services)
}

func (uc *ReadLocationAttributeUseCase) Execute(ctx context.Context, req *locationattributepb.ReadLocationAttributeRequest) (*locationattributepb.ReadLocationAttributeResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.LocationAttribute,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.LocationAttribute.ReadLocationAttribute(ctx, req)
	if err != nil {
		return nil, err
	}

	// Return response as-is (even if empty data for not found case)
	return resp, nil
}

// validateInput validates the input request
func (uc *ReadLocationAttributeUseCase) validateInput(ctx context.Context, req *locationattributepb.ReadLocationAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_attribute.validation.request_required", "Request is required for location attributes [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_attribute.validation.data_required", "Location attribute data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_attribute.validation.id_required", "Location attribute ID is required [DEFAULT]"))
	}
	return nil
}

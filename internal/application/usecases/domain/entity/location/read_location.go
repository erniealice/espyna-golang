package location

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	locationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location"
)

// ReadLocationRepositories groups all repository dependencies
type ReadLocationRepositories struct {
	Location locationpb.LocationDomainServiceServer // Primary entity repository
}

// ReadLocationServices groups all business service dependencies
type ReadLocationServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ReadLocationUseCase handles the business logic for reading locations
type ReadLocationUseCase struct {
	repositories ReadLocationRepositories
	services     ReadLocationServices
}

// NewReadLocationUseCase creates use case with grouped dependencies
func NewReadLocationUseCase(
	repositories ReadLocationRepositories,
	services ReadLocationServices,
) *ReadLocationUseCase {
	return &ReadLocationUseCase{
		repositories: repositories,
		services:     services,
	}
}

func (uc *ReadLocationUseCase) Execute(ctx context.Context, req *locationpb.ReadLocationRequest) (*locationpb.ReadLocationResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Location,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.Location.ReadLocation(ctx, req)
	if err != nil {
		return nil, err
	}

	// Return response as-is (even if empty data for not found case)
	return resp, nil
}

// validateInput validates the input request
func (uc *ReadLocationUseCase) validateInput(ctx context.Context, req *locationpb.ReadLocationRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location.validation.data_required", "[ERR-DEFAULT] Data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location.validation.id_required", "[ERR-DEFAULT] ID is required"))
	}
	return nil
}

package location_area

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	locationareapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location_area"
)

// ReadLocationAreaRepositories groups all repository dependencies
type ReadLocationAreaRepositories struct {
	LocationArea locationareapb.LocationAreaDomainServiceServer // Primary entity repository
}

// ReadLocationAreaServices groups all business service dependencies
type ReadLocationAreaServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ReadLocationAreaUseCase handles the business logic for reading location areas
type ReadLocationAreaUseCase struct {
	repositories ReadLocationAreaRepositories
	services     ReadLocationAreaServices
}

// NewReadLocationAreaUseCase creates use case with grouped dependencies
func NewReadLocationAreaUseCase(
	repositories ReadLocationAreaRepositories,
	services ReadLocationAreaServices,
) *ReadLocationAreaUseCase {
	return &ReadLocationAreaUseCase{
		repositories: repositories,
		services:     services,
	}
}

func (uc *ReadLocationAreaUseCase) Execute(ctx context.Context, req *locationareapb.ReadLocationAreaRequest) (*locationareapb.ReadLocationAreaResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.LocationArea, entityid.ActionRead); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.LocationArea.ReadLocationArea(ctx, req)
	if err != nil {
		return nil, err
	}

	// Return response as-is (even if empty data for not found case)
	return resp, nil
}

// validateInput validates the input request
func (uc *ReadLocationAreaUseCase) validateInput(ctx context.Context, req *locationareapb.ReadLocationAreaRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_area.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_area.validation.data_required", "[ERR-DEFAULT] Data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_area.validation.id_required", "[ERR-DEFAULT] ID is required"))
	}
	return nil
}

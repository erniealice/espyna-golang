package location

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	locationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location"
)

// ReadLocationRepositories groups all repository dependencies
type ReadLocationRepositories struct {
	Location locationpb.LocationDomainServiceServer // Primary entity repository
}

// ReadLocationServices groups all business service dependencies
type ReadLocationServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
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
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityLocation, ports.ActionRead); err != nil {
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
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.validation.request_required", ""))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.validation.data_required", ""))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.validation.id_required", ""))
	}
	return nil
}

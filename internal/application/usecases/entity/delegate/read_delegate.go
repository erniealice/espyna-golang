package delegate

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	delegatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate"
)

// ReadDelegateRepositories groups all repository dependencies
type ReadDelegateRepositories struct {
	Delegate delegatepb.DelegateDomainServiceServer // Primary entity repository
}

// ReadDelegateServices groups all business service dependencies
type ReadDelegateServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadDelegateUseCase handles the business logic for reading a delegate
type ReadDelegateUseCase struct {
	repositories ReadDelegateRepositories
	services     ReadDelegateServices
}

// NewReadDelegateUseCase creates use case with grouped dependencies
func NewReadDelegateUseCase(
	repositories ReadDelegateRepositories,
	services ReadDelegateServices,
) *ReadDelegateUseCase {
	return &ReadDelegateUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadDelegateUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewReadDelegateUseCase with grouped parameters instead
func NewReadDelegateUseCaseUngrouped(delegateRepo delegatepb.DelegateDomainServiceServer) *ReadDelegateUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ReadDelegateRepositories{
		Delegate: delegateRepo,
	}

	services := ReadDelegateServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewReadDelegateUseCase(repositories, services)
}

// Execute performs the read delegate operation
func (uc *ReadDelegateUseCase) Execute(ctx context.Context, req *delegatepb.ReadDelegateRequest) (*delegatepb.ReadDelegateResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityDelegate, ports.ActionRead); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.validation.request_required", "Request is required for Parent/Guardians"))
	}
	if req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.validation.data_required", "Parent/Guardian data is required"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.validation.id_required", "Delegate ID is required [DEFAULT]"))
	}

	// Call repository
	resp, err := uc.repositories.Delegate.ReadDelegate(ctx, req)
	if err != nil {
		return nil, err
	}

	// Return response from repository (includes empty results for not found)

	return resp, nil
}

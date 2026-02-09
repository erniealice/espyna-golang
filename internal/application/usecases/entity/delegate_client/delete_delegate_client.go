package delegate_client

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	delegatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate"
	delegateclientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate_client"
)

// DeleteDelegateClientRepositories groups all repository dependencies
type DeleteDelegateClientRepositories struct {
	DelegateClient delegateclientpb.DelegateClientDomainServiceServer // Primary entity repository
	Delegate       delegatepb.DelegateDomainServiceServer             // Entity reference validation
	Client         clientpb.ClientDomainServiceServer                 // Entity reference validation
}

// DeleteDelegateClientServices groups all business service dependencies
type DeleteDelegateClientServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteDelegateClientUseCase handles the business logic for deleting delegate clients
type DeleteDelegateClientUseCase struct {
	repositories DeleteDelegateClientRepositories
	services     DeleteDelegateClientServices
}

// NewDeleteDelegateClientUseCase creates use case with grouped dependencies
func NewDeleteDelegateClientUseCase(
	repositories DeleteDelegateClientRepositories,
	services DeleteDelegateClientServices,
) *DeleteDelegateClientUseCase {
	return &DeleteDelegateClientUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewDeleteDelegateClientUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewDeleteDelegateClientUseCase with grouped parameters instead
func NewDeleteDelegateClientUseCaseUngrouped(
	delegateClientRepo delegateclientpb.DelegateClientDomainServiceServer,
) *DeleteDelegateClientUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := DeleteDelegateClientRepositories{
		DelegateClient: delegateClientRepo,
		Delegate:       nil,
		Client:         nil,
	}

	services := DeleteDelegateClientServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewDeleteDelegateClientUseCase(repositories, services)
}

// Execute performs the delete delegate client operation
func (uc *DeleteDelegateClientUseCase) Execute(ctx context.Context, req *delegateclientpb.DeleteDelegateClientRequest) (*delegateclientpb.DeleteDelegateClientResponse, error) {
	// Input validation
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.validation.request_required", "Request is required for Delegate-Client relationships [DEFAULT]"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.validation.id_required", "Delegate-Client relationship ID is required [DEFAULT]"))
	}

	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityDelegateClient, ports.ActionDelete); err != nil {
		return nil, err
	}

	// Business logic pre-processing
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.DelegateClient.DeleteDelegateClient(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.errors.deletion_failed", "Delegate-Client relationship deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteDelegateClientUseCase) validateInput(ctx context.Context, req *delegateclientpb.DeleteDelegateClientRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.validation.request_required", "Request is required for Delegate-Client relationships [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.validation.data_required", "Delegate-Client relationship data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.validation.id_required", "Delegate-Client relationship ID is required [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints before deletion
func (uc *DeleteDelegateClientUseCase) validateBusinessRules(ctx context.Context, delegateClient *delegateclientpb.DelegateClient) error {
	// Add any business rules that should prevent deletion
	// For example, check if there are any dependent records
	// or if the delegate-client relationship is currently active in some workflow

	// Currently no business rules prevent deletion
	return nil
}

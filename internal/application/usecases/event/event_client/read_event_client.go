package eventclient

import (
	"context"
	"errors"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	clientpb "leapfor.xyz/esqyma/golang/v1/domain/entity/client"
	eventpb "leapfor.xyz/esqyma/golang/v1/domain/event/event"
	eventclientpb "leapfor.xyz/esqyma/golang/v1/domain/event/event_client"
)

// ReadEventClientRepositories groups all repository dependencies
type ReadEventClientRepositories struct {
	EventClient eventclientpb.EventClientDomainServiceServer // Primary entity repository
	Event       eventpb.EventDomainServiceServer             // Entity reference validation
	Client      clientpb.ClientDomainServiceServer           // Entity reference validation
}

// ReadEventClientServices groups all business service dependencies
type ReadEventClientServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// ReadEventClientUseCase handles the business logic for reading event client associations
type ReadEventClientUseCase struct {
	repositories ReadEventClientRepositories
	services     ReadEventClientServices
}

// NewReadEventClientUseCase creates use case with grouped dependencies
func NewReadEventClientUseCase(
	repositories ReadEventClientRepositories,
	services ReadEventClientServices,
) *ReadEventClientUseCase {
	return &ReadEventClientUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadEventClientUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewReadEventClientUseCase with grouped parameters instead
func NewReadEventClientUseCaseUngrouped(
	eventClientRepo eventclientpb.EventClientDomainServiceServer,
) *ReadEventClientUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ReadEventClientRepositories{
		EventClient: eventClientRepo,
		Event:       nil,
		Client:      nil,
	}

	services := ReadEventClientServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return &ReadEventClientUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read event client operation
func (uc *ReadEventClientUseCase) Execute(ctx context.Context, req *eventclientpb.ReadEventClientRequest) (*eventclientpb.ReadEventClientResponse, error) {
	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_client.validation.request_required", ""))
	}

	if req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_client.validation.data_required", ""))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_client.validation.id_required", ""))
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_client.errors.authorization_failed", "Authorization failed for schedule enrollment")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityEventClient, ports.ActionRead)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_client.errors.authorization_failed", "Authorization failed for schedule enrollment")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_client.errors.authorization_failed", "Authorization failed for schedule enrollment")
		return nil, errors.New(translatedError)
	}

	// Call repository
	return uc.repositories.EventClient.ReadEventClient(ctx, req)
}

package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
)

// UpdateClientRepositories groups all repository dependencies
type UpdateClientRepositories struct {
	Client clientpb.ClientDomainServiceServer // Primary entity repository
}

// UpdateClientServices groups all business service dependencies
type UpdateClientServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateClientUseCase handles the business logic for updating a client
type UpdateClientUseCase struct {
	repositories UpdateClientRepositories
	services     UpdateClientServices
}

// NewUpdateClientUseCase creates use case with grouped dependencies
func NewUpdateClientUseCase(
	repositories UpdateClientRepositories,
	services UpdateClientServices,
) *UpdateClientUseCase {
	return &UpdateClientUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdateClientUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewUpdateClientUseCase with grouped parameters instead
func NewUpdateClientUseCaseUngrouped(clientRepo clientpb.ClientDomainServiceServer) *UpdateClientUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := UpdateClientRepositories{
		Client: clientRepo,
	}

	services := UpdateClientServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUpdateClientUseCase(repositories, services)
}

// Execute performs the update client operation
func (uc *UpdateClientUseCase) Execute(ctx context.Context, req *clientpb.UpdateClientRequest) (*clientpb.UpdateClientResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityClient, ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client.validation.request_required", "Request is required for clients [DEFAULT]"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client.validation.id_required", "Client ID is required [DEFAULT]"))
	}

	// Business logic validation
	if req.Data.User != nil && req.Data.User.EmailAddress == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client.validation.email_required", "Client email is required [DEFAULT]"))
	}

	// 2026-05-03 — Preserve active flags when the payload does not carry
	// them. The drawer form no longer exposes an "active" toggle (it's
	// derived from status), so unmarshalled requests arrive with Active
	// at proto3 zero. We can't distinguish that from an explicit false,
	// so we conservatively copy the existing value: lifecycle changes go
	// through the SetClientStatus closure (raw SQL update), not through
	// this use case. Skipped if the existing read fails — the repository
	// will reject malformed requests downstream.
	if !req.Data.Active {
		if readResp, readErr := uc.repositories.Client.ReadClient(ctx, &clientpb.ReadClientRequest{
			Data: &clientpb.Client{Id: req.Data.Id},
		}); readErr == nil && readResp != nil && len(readResp.GetData()) > 0 {
			existing := readResp.GetData()[0]
			req.Data.Active = existing.GetActive()
			// Same treatment for the embedded representative user — the
			// representative tab also dropped its active toggle.
			if req.Data.User != nil && !req.Data.User.Active {
				if eu := existing.GetUser(); eu != nil {
					req.Data.User.Active = eu.GetActive()
				}
			}
		}
	}

	// Call repository
	resp, err := uc.repositories.Client.UpdateClient(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client.errors.update_failed", "Client update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

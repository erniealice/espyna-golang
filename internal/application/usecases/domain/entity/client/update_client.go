package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
)

// UpdateClientRepositories groups all repository dependencies
type UpdateClientRepositories struct {
	Client clientpb.ClientDomainServiceServer // Primary entity repository
	// Attribute overlay repos (Q-GSE-10): sync client_attribute rows in the same
	// transaction as the client update. Optional (nil-safe).
	Attributes AttributeRepositories
}

// UpdateClientServices groups all business service dependencies
type UpdateClientServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator      ports.IDGenerator
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
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
	}

	return NewUpdateClientUseCase(repositories, services)
}

// Execute performs the update client operation
func (uc *UpdateClientUseCase) Execute(ctx context.Context, req *clientpb.UpdateClientRequest) (*clientpb.UpdateClientResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Client,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client.validation.request_required", "Request is required for clients [DEFAULT]"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client.validation.id_required", "Client ID is required [DEFAULT]"))
	}

	// Business logic validation
	if req.Data.User != nil && req.Data.User.EmailAddress == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client.validation.email_required", "Client email is required [DEFAULT]"))
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

	// Attribute overlay (Q-GSE-10). Sync the client_attribute rows ONLY when the
	// drawer marked the section present — an update whose payload omits the section
	// (attributes_present=false) must NEVER wipe a client's existing attributes.
	// A blank optional value inside a present section clears (deletes) that row.
	syncAttrs := req.GetAttributesPresent()
	if syncAttrs && !uc.repositories.Attributes.enabled() {
		// Reject fail-closed: the section was submitted but the server cannot validate it.
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client.validation.attributes_unavailable", "Client attributes cannot be validated [DEFAULT]"))
	}

	if syncAttrs {
		// Resolve + validate before any write; roll the whole update back on failure.
		resolvedAttrs, verr := resolveAndValidateAttributes(ctx, uc.repositories.Attributes, req.GetAttributes())
		if verr != nil {
			return nil, verr
		}
		clientID := req.Data.Id
		run := func(txCtx context.Context) (*clientpb.UpdateClientResponse, error) {
			resp, err := uc.repositories.Client.UpdateClient(txCtx, req)
			if err != nil {
				translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "client.errors.update_failed", "Client update failed [DEFAULT]")
				return nil, fmt.Errorf("%s: %w", translatedError, err)
			}
			if serr := syncClientAttributes(txCtx, uc.repositories.Attributes, uc.services.IDGenerator, clientID, resolvedAttrs); serr != nil {
				return nil, serr
			}
			return resp, nil
		}

		if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
			var result *clientpb.UpdateClientResponse
			if err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
				r, err := run(txCtx)
				if err != nil {
					return err
				}
				result = r
				return nil
			}); err != nil {
				return nil, err
			}
			return result, nil
		}
		return run(ctx)
	}

	// Call repository (no attribute section present — unchanged path).
	resp, err := uc.repositories.Client.UpdateClient(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client.errors.update_failed", "Client update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

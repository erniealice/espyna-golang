package supplier

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	supplierpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier"
)

// UpdateSupplierRepositories groups all repository dependencies
type UpdateSupplierRepositories struct {
	Supplier supplierpb.SupplierDomainServiceServer // Primary entity repository
}

// UpdateSupplierServices groups all business service dependencies
type UpdateSupplierServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UpdateSupplierUseCase handles the business logic for updating a supplier
type UpdateSupplierUseCase struct {
	repositories UpdateSupplierRepositories
	services     UpdateSupplierServices
}

// NewUpdateSupplierUseCase creates use case with grouped dependencies
func NewUpdateSupplierUseCase(
	repositories UpdateSupplierRepositories,
	services UpdateSupplierServices,
) *UpdateSupplierUseCase {
	return &UpdateSupplierUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdateSupplierUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewUpdateSupplierUseCase with grouped parameters instead
func NewUpdateSupplierUseCaseUngrouped(supplierRepo supplierpb.SupplierDomainServiceServer) *UpdateSupplierUseCase {
	repositories := UpdateSupplierRepositories{
		Supplier: supplierRepo,
	}

	services := UpdateSupplierServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
	}

	return NewUpdateSupplierUseCase(repositories, services)
}

// Execute performs the update supplier operation
func (uc *UpdateSupplierUseCase) Execute(ctx context.Context, req *supplierpb.UpdateSupplierRequest) (*supplierpb.UpdateSupplierResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: "supplier",
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier.validation.request_required", "Request is required for suppliers [DEFAULT]"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier.validation.id_required", "Supplier ID is required [DEFAULT]"))
	}

	// Business logic validation
	if req.Data.Name == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier.validation.name_required", "Supplier name is required [DEFAULT]"))
	}

	// 2026-05-03 — Preserve active flags when the payload does not carry
	// them. The drawer form no longer exposes an "active" toggle (it's
	// derived from status), so unmarshalled requests arrive with Active
	// at proto3 zero. We can't distinguish that from an explicit false,
	// so we conservatively copy the existing value: lifecycle changes go
	// through the SetSupplierStatus closure (raw SQL update), not through
	// this use case. Skipped if the existing read fails — the repository
	// will reject malformed requests downstream.
	if !req.Data.Active {
		if readResp, readErr := uc.repositories.Supplier.ReadSupplier(ctx, &supplierpb.ReadSupplierRequest{
			Data: &supplierpb.Supplier{Id: req.Data.Id},
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
	resp, err := uc.repositories.Supplier.UpdateSupplier(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier.errors.update_failed", "Supplier update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

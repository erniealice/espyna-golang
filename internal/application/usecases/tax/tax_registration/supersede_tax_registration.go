package tax_registration

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	taxregistrationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_registration"
	taxregistrationkindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_registration_kind"
)

// SupersedeTaxRegistrationRepositories groups repository dependencies.
type SupersedeTaxRegistrationRepositories struct {
	TaxRegistration     taxregistrationpb.TaxRegistrationDomainServiceServer
	TaxRegistrationKind taxregistrationkindpb.TaxRegistrationKindDomainServiceServer
}

// SupersedeTaxRegistrationServices groups service dependencies.
type SupersedeTaxRegistrationServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// SupersedeTaxRegistrationRequest is the input for superseding a tax_registration.
type SupersedeTaxRegistrationRequest struct {
	// PriorID is the ID of the currently ACTIVE row to supersede.
	PriorID string
	// New row data — must include TaxRegistrationKindId, RegistrationNumber, EffectiveFrom, etc.
	// Id, ComputePathSnapshot, PartyRoleSnapshot, SupersedesId, Status are set by the use case.
	NewData *taxregistrationpb.TaxRegistration
}

// SupersedeTaxRegistrationUseCase implements immutable-row + self-FK supersession:
//  1. Reads the ACTIVE prior row and validates it.
//  2. Stamps the prior row: status=SUPERSEDED, effective_to=NewData.EffectiveFrom.
//  3. Reads the kind to copy compute_path/party_role snapshots.
//  4. INSERTs a new ACTIVE row with supersedes_id = priorID.
//
// This use case maps to the "update" route name for CRUD permission compatibility
// (the route still uses the update permission but calls Supersede semantics).
type SupersedeTaxRegistrationUseCase struct {
	repositories SupersedeTaxRegistrationRepositories
	services     SupersedeTaxRegistrationServices
}

// NewSupersedeTaxRegistrationUseCase creates a new SupersedeTaxRegistrationUseCase.
func NewSupersedeTaxRegistrationUseCase(repositories SupersedeTaxRegistrationRepositories, services SupersedeTaxRegistrationServices) *SupersedeTaxRegistrationUseCase {
	return &SupersedeTaxRegistrationUseCase{repositories: repositories, services: services}
}

// Execute performs the supersede operation.
func (uc *SupersedeTaxRegistrationUseCase) Execute(ctx context.Context, req *SupersedeTaxRegistrationRequest) (*taxregistrationpb.TaxRegistration, error) {
	// Supersede is an "update" action in CRUD permission terms.
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityTaxRegistration, ports.ActionUpdate); err != nil {
		return nil, err
	}

	if req == nil || req.PriorID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"tax_registration.validation.prior_id_required", "Prior Tax Registration ID is required [DEFAULT]"))
	}
	if req.NewData == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"tax_registration.validation.data_required", "New Tax Registration data is required [DEFAULT]"))
	}
	if req.NewData.TaxRegistrationKindId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"tax_registration.validation.kind_id_required", "Tax Registration Kind is required [DEFAULT]"))
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *taxregistrationpb.TaxRegistration
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("tax_registration supersession failed: %w", err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return uc.executeCore(ctx, req)
}

func (uc *SupersedeTaxRegistrationUseCase) executeCore(ctx context.Context, req *SupersedeTaxRegistrationRequest) (*taxregistrationpb.TaxRegistration, error) {
	// 1. Read the prior ACTIVE row.
	priorResp, err := uc.repositories.TaxRegistration.ReadTaxRegistration(ctx, &taxregistrationpb.ReadTaxRegistrationRequest{
		Data: &taxregistrationpb.TaxRegistration{Id: req.PriorID},
	})
	if err != nil {
		return nil, fmt.Errorf("read prior tax_registration: %w", err)
	}
	if priorResp == nil || len(priorResp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"tax_registration.validation.not_found", "Prior Tax Registration not found [DEFAULT]"))
	}
	prior := priorResp.GetData()[0]
	if prior.GetStatus() != taxregistrationpb.TaxRegistrationStatus_TAX_REGISTRATION_STATUS_ACTIVE {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"tax_registration.validation.not_active", "Only ACTIVE registrations can be superseded [DEFAULT]"))
	}

	// 2. Determine the effective_to for the prior row (= new row's effective_from).
	newEffectiveFrom := req.NewData.GetEffectiveFrom()
	if newEffectiveFrom == "" {
		newEffectiveFrom = time.Now().UTC().Format("2006-01-02")
	}

	// 3. Stamp the prior row as SUPERSEDED.
	priorCopy := proto_clone(prior)
	priorCopy.Status = taxregistrationpb.TaxRegistrationStatus_TAX_REGISTRATION_STATUS_SUPERSEDED
	priorCopy.EffectiveTo = &newEffectiveFrom
	if _, err := uc.repositories.TaxRegistration.UpdateTaxRegistration(ctx, &taxregistrationpb.UpdateTaxRegistrationRequest{
		Data: priorCopy,
	}); err != nil {
		return nil, fmt.Errorf("stamp prior tax_registration as SUPERSEDED: %w", err)
	}

	// 4. Read the kind to copy snapshots.
	kindResp, err := uc.repositories.TaxRegistrationKind.ReadTaxRegistrationKind(ctx,
		&taxregistrationkindpb.ReadTaxRegistrationKindRequest{
			Data: &taxregistrationkindpb.TaxRegistrationKind{Id: req.NewData.TaxRegistrationKindId},
		})
	if err != nil {
		return nil, fmt.Errorf("read tax_registration_kind for snapshot: %w", err)
	}

	// 5. Build the new ACTIVE row.
	now := time.Now()
	newRow := req.NewData
	if newRow.Id == "" {
		newRow.Id = uc.services.IDService.GenerateID()
	}
	newRow.Status = taxregistrationpb.TaxRegistrationStatus_TAX_REGISTRATION_STATUS_ACTIVE
	newRow.Active = true
	newRow.DateCreated = &[]int64{now.UnixMilli()}[0]
	newRow.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	newRow.DateModified = &[]int64{now.UnixMilli()}[0]
	newRow.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	priorID := req.PriorID
	newRow.SupersedesId = &priorID

	// Denorm compute_path + party_role from the kind (same as Create).
	if kindResp != nil && len(kindResp.GetData()) > 0 {
		kind := kindResp.GetData()[0]
		newRow.ComputePathSnapshot = taxregistrationpb.TaxRegistrationComputePathSnapshot(kind.GetComputePath().Number())
		newRow.PartyRoleSnapshot = taxregistrationpb.TaxRegistrationPartyRoleSnapshot(kind.GetPartyRole().Number())
		// Carry party_type and party_id from the prior row when the caller doesn't set them.
		if newRow.PartyType == taxregistrationpb.TaxRegistrationPartyType_TAX_REGISTRATION_PARTY_TYPE_UNSPECIFIED {
			newRow.PartyType = prior.GetPartyType()
		}
		if newRow.PartyId == "" {
			newRow.PartyId = prior.GetPartyId()
		}
		if newRow.TaxAuthorityId == "" {
			newRow.TaxAuthorityId = kind.GetTaxAuthorityId()
		}
		if newRow.WorkspaceId == "" {
			newRow.WorkspaceId = prior.GetWorkspaceId()
		}
	}

	// 6. Insert the new row.
	createResp, err := uc.repositories.TaxRegistration.CreateTaxRegistration(ctx, &taxregistrationpb.CreateTaxRegistrationRequest{
		Data: newRow,
	})
	if err != nil {
		return nil, fmt.Errorf("create new tax_registration (supersession): %w", err)
	}
	if createResp == nil || len(createResp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"tax_registration.errors.create_failed", "Failed to create superseding tax_registration [DEFAULT]"))
	}
	return createResp.GetData()[0], nil
}

// proto_clone is a minimal helper to clone a TaxRegistration proto for mutation.
// We avoid importing google.golang.org/protobuf/proto here to keep the dependency
// chain clear — instead we copy only the fields we need to stamp.
func proto_clone(src *taxregistrationpb.TaxRegistration) *taxregistrationpb.TaxRegistration {
	if src == nil {
		return nil
	}
	// Shallow-copy the pointer; status + effective_to are the only mutated fields.
	c := *src
	return &c
}

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
)

// RevokeTaxRegistrationRepositories groups repository dependencies.
type RevokeTaxRegistrationRepositories struct {
	TaxRegistration taxregistrationpb.TaxRegistrationDomainServiceServer
}

// RevokeTaxRegistrationServices groups service dependencies.
type RevokeTaxRegistrationServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// RevokeTaxRegistrationRequest is the input for revoking a tax_registration.
type RevokeTaxRegistrationRequest struct {
	// ID of the ACTIVE registration to revoke.
	ID string
	// EffectiveTo is the operator-supplied revocation date (status=CANCELLED, effective_to=this date).
	// Defaults to today if empty.
	EffectiveTo string
}

// RevokeTaxRegistrationUseCase stamps the ACTIVE row as CANCELLED with an
// operator-supplied effective_to date. Does NOT insert a new row.
// This use case maps to the "delete" route name for CRUD permission compatibility.
type RevokeTaxRegistrationUseCase struct {
	repositories RevokeTaxRegistrationRepositories
	services     RevokeTaxRegistrationServices
}

// NewRevokeTaxRegistrationUseCase creates a new RevokeTaxRegistrationUseCase.
func NewRevokeTaxRegistrationUseCase(repositories RevokeTaxRegistrationRepositories, services RevokeTaxRegistrationServices) *RevokeTaxRegistrationUseCase {
	return &RevokeTaxRegistrationUseCase{repositories: repositories, services: services}
}

// Execute performs the revoke operation.
func (uc *RevokeTaxRegistrationUseCase) Execute(ctx context.Context, req *RevokeTaxRegistrationRequest) (*taxregistrationpb.TaxRegistration, error) {
	// Revoke is a "delete" action in CRUD permission terms.
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityTaxRegistration, ports.ActionDelete); err != nil {
		return nil, err
	}

	if req == nil || req.ID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"tax_registration.validation.id_required", "Tax Registration ID is required [DEFAULT]"))
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *taxregistrationpb.TaxRegistration
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("tax_registration revoke failed: %w", err)
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

func (uc *RevokeTaxRegistrationUseCase) executeCore(ctx context.Context, req *RevokeTaxRegistrationRequest) (*taxregistrationpb.TaxRegistration, error) {
	// 1. Read the registration.
	readResp, err := uc.repositories.TaxRegistration.ReadTaxRegistration(ctx, &taxregistrationpb.ReadTaxRegistrationRequest{
		Data: &taxregistrationpb.TaxRegistration{Id: req.ID},
	})
	if err != nil {
		return nil, fmt.Errorf("read tax_registration: %w", err)
	}
	if readResp == nil || len(readResp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"tax_registration.validation.not_found", "Tax Registration not found [DEFAULT]"))
	}
	reg := readResp.GetData()[0]
	if reg.GetStatus() != taxregistrationpb.TaxRegistrationStatus_TAX_REGISTRATION_STATUS_ACTIVE {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"tax_registration.validation.not_active", "Only ACTIVE registrations can be revoked [DEFAULT]"))
	}

	// 2. Stamp as CANCELLED.
	effectiveTo := req.EffectiveTo
	if effectiveTo == "" {
		effectiveTo = time.Now().UTC().Format("2006-01-02")
	}
	stamped := proto_clone(reg)
	stamped.Status = taxregistrationpb.TaxRegistrationStatus_TAX_REGISTRATION_STATUS_CANCELLED
	stamped.EffectiveTo = &effectiveTo

	updateResp, err := uc.repositories.TaxRegistration.UpdateTaxRegistration(ctx, &taxregistrationpb.UpdateTaxRegistrationRequest{
		Data: stamped,
	})
	if err != nil {
		return nil, fmt.Errorf("stamp tax_registration as CANCELLED: %w", err)
	}
	if updateResp == nil || len(updateResp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"tax_registration.errors.update_failed", "Failed to revoke tax_registration [DEFAULT]"))
	}
	return updateResp.GetData()[0], nil
}

package suppliercontract

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
)

// RegisterBillingRequest holds the parameters for registering an Expenditure billing event.
type RegisterBillingRequest struct {
	ContractID     string
	BilledCentavos int64
}

// RegisterBillingRepositories groups repository dependencies.
type RegisterBillingRepositories struct {
	SupplierContract any
}

// RegisterBillingServices groups service dependencies.
type RegisterBillingServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// RegisterBillingUseCase increments billed_amount and recomputes remaining_amount
// when an Expenditure is posted against a supplier contract.
// Single write boundary: only this use case writes billed_amount.
type RegisterBillingUseCase struct {
	repositories RegisterBillingRepositories
	services     RegisterBillingServices
}

// NewRegisterBillingUseCase creates a use case with grouped dependencies.
func NewRegisterBillingUseCase(
	repositories RegisterBillingRepositories,
	services RegisterBillingServices,
) *RegisterBillingUseCase {
	return &RegisterBillingUseCase{repositories: repositories, services: services}
}

// Execute performs the register billing operation.
func (uc *RegisterBillingUseCase) Execute(ctx context.Context, req RegisterBillingRequest) error {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entitySupplierContract, ports.ActionUpdate); err != nil {
		return err
	}
	if req.ContractID == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract.validation.id_required", "Supplier contract ID is required [DEFAULT]"))
	}
	if req.BilledCentavos <= 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract.validation.amount_required", "Billed amount must be positive [DEFAULT]"))
	}
	balanceRepo, ok := uc.repositories.SupplierContract.(BalanceRepository)
	if !ok {
		return fmt.Errorf("supplier contract repository does not support balance updates")
	}
	return balanceRepo.RegisterBilling(ctx, req.ContractID, req.BilledCentavos)
}

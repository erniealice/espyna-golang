package suppliercontract

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
)

// RegisterCreditRequest holds the parameters for registering a credit/rebate against a contract.
type RegisterCreditRequest struct {
	ContractID     string
	CreditCentavos int64
}

// RegisterCreditRepositories groups repository dependencies.
type RegisterCreditRepositories struct {
	SupplierContract any
}

// RegisterCreditServices groups service dependencies.
type RegisterCreditServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// RegisterCreditUseCase handles negative-Expenditure flows (rebates, supplier credits).
// It decrements billed_amount and recomputes remaining_amount via a locked balance update.
// Single write boundary: only this use case writes the credit adjustment.
type RegisterCreditUseCase struct {
	repositories RegisterCreditRepositories
	services     RegisterCreditServices
}

// NewRegisterCreditUseCase creates a use case with grouped dependencies.
func NewRegisterCreditUseCase(
	repositories RegisterCreditRepositories,
	services RegisterCreditServices,
) *RegisterCreditUseCase {
	return &RegisterCreditUseCase{repositories: repositories, services: services}
}

// Execute performs the register credit operation.
func (uc *RegisterCreditUseCase) Execute(ctx context.Context, req RegisterCreditRequest) error {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entitySupplierContract,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return err
	}
	if req.ContractID == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"supplier_contract.validation.id_required", "Supplier contract ID is required [DEFAULT]"))
	}
	if req.CreditCentavos <= 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"supplier_contract.validation.amount_required", "Credit amount must be positive [DEFAULT]"))
	}
	balanceRepo, ok := uc.repositories.SupplierContract.(BalanceRepository)
	if !ok {
		return fmt.Errorf("supplier contract repository does not support balance updates")
	}
	return balanceRepo.RegisterCredit(ctx, req.ContractID, req.CreditCentavos)
}

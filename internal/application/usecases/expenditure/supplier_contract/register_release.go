package suppliercontract

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
)

// BalanceRepository is the narrow interface exposing balance-update closures.
// Only the PostgresSupplierContractRepository implements these methods; the proto
// server interface does not carry them so we use a separate interface here.
// This is the single write boundary per plan §11.7 risk #5.
type BalanceRepository interface {
	RegisterRelease(ctx context.Context, contractID string, releasedCentavos int64) error
	RegisterBilling(ctx context.Context, contractID string, billedCentavos int64) error
	RegisterCredit(ctx context.Context, contractID string, creditCentavos int64) error
}

// RegisterReleaseRequest holds the parameters for registering a PO release.
type RegisterReleaseRequest struct {
	ContractID       string
	ReleasedCentavos int64
}

// RegisterReleaseRepositories groups repository dependencies.
type RegisterReleaseRepositories struct {
	// SupplierContract must also implement BalanceRepository.
	// Stored as any here so injection can type-assert at use-case setup time.
	SupplierContract any
}

// RegisterReleaseServices groups service dependencies.
type RegisterReleaseServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// RegisterReleaseUseCase increments released_amount when a PO is created against a contract.
// Single write boundary: only this use case writes released_amount.
type RegisterReleaseUseCase struct {
	repositories RegisterReleaseRepositories
	services     RegisterReleaseServices
}

// NewRegisterReleaseUseCase creates a use case with grouped dependencies.
func NewRegisterReleaseUseCase(
	repositories RegisterReleaseRepositories,
	services RegisterReleaseServices,
) *RegisterReleaseUseCase {
	return &RegisterReleaseUseCase{repositories: repositories, services: services}
}

// Execute performs the register release operation.
func (uc *RegisterReleaseUseCase) Execute(ctx context.Context, req RegisterReleaseRequest) error {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entitySupplierContract, ports.ActionUpdate); err != nil {
		return err
	}
	if req.ContractID == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract.validation.id_required", "Supplier contract ID is required [DEFAULT]"))
	}
	if req.ReleasedCentavos <= 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract.validation.amount_required", "Released amount must be positive [DEFAULT]"))
	}
	balanceRepo, ok := uc.repositories.SupplierContract.(BalanceRepository)
	if !ok {
		return fmt.Errorf("supplier contract repository does not support balance updates")
	}
	return balanceRepo.RegisterRelease(ctx, req.ContractID, req.ReleasedCentavos)
}

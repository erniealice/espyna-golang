package tax_class

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	taxclasspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_class"
)

// TaxClassFindByCodeQueries is the narrow interface from the adapter.
type TaxClassFindByCodeQueries interface {
	FindByCode(ctx context.Context, code, direction string) (*taxclasspb.TaxClass, error)
}

// FindByCodeTaxClassRepositories groups repository dependencies.
type FindByCodeTaxClassRepositories struct {
	TaxClass taxclasspb.TaxClassDomainServiceServer
}

// FindByCodeTaxClassServices groups service dependencies.
type FindByCodeTaxClassServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// FindByCodeTaxClassUseCase wraps the adapter's FindByCode method.
// Used by ComputeTaxesForRevenue to resolve the TaxClass for a withholding_class_snapshot code.
type FindByCodeTaxClassUseCase struct {
	repositories FindByCodeTaxClassRepositories
	services     FindByCodeTaxClassServices
}

// NewFindByCodeTaxClassUseCase creates the use case.
func NewFindByCodeTaxClassUseCase(
	repositories FindByCodeTaxClassRepositories,
	services FindByCodeTaxClassServices,
) *FindByCodeTaxClassUseCase {
	return &FindByCodeTaxClassUseCase{repositories: repositories, services: services}
}

// Execute returns the TaxClass matching (code, direction).
func (uc *FindByCodeTaxClassUseCase) Execute(ctx context.Context, code, direction string) (*taxclasspb.TaxClass, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityTaxClass, ports.ActionRead); err != nil {
		return nil, err
	}
	if code == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"tax_class.validation.code_required", "Tax class code is required [DEFAULT]"))
	}

	q, ok := uc.repositories.TaxClass.(TaxClassFindByCodeQueries)
	if !ok {
		return nil, fmt.Errorf("tax_class repository does not support FindByCode")
	}
	return q.FindByCode(ctx, code, direction)
}

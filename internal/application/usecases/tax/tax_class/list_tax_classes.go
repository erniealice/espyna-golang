package tax_class

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	taxclasspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_class"
)

// ListTaxClassesRepositories groups repository dependencies.
type ListTaxClassesRepositories struct {
	TaxClass taxclasspb.TaxClassDomainServiceServer
}

// ListTaxClassesServices groups service dependencies.
type ListTaxClassesServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// ListTaxClassesUseCase handles listing tax classes.
type ListTaxClassesUseCase struct {
	repositories ListTaxClassesRepositories
	services     ListTaxClassesServices
}

// NewListTaxClassesUseCase creates a new ListTaxClassesUseCase.
func NewListTaxClassesUseCase(repositories ListTaxClassesRepositories, services ListTaxClassesServices) *ListTaxClassesUseCase {
	return &ListTaxClassesUseCase{repositories: repositories, services: services}
}

// Execute performs the list tax_classes operation.
func (uc *ListTaxClassesUseCase) Execute(ctx context.Context, req *taxclasspb.ListTaxClassesRequest) (*taxclasspb.ListTaxClassesResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityTaxClass, ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"tax_class.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.TaxClass.ListTaxClasses(ctx, req)
}

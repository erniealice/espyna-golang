package tax_class

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	taxclasspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_class"
)

// ListTaxClassesRepositories groups repository dependencies.
type ListTaxClassesRepositories struct {
	TaxClass taxclasspb.TaxClassDomainServiceServer
}

// ListTaxClassesServices groups service dependencies.
type ListTaxClassesServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
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
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityTaxClass, entityid.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"tax_class.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.TaxClass.ListTaxClasses(ctx, req)
}

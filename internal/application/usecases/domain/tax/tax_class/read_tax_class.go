package tax_class

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	taxclasspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_class"
)

// ReadTaxClassRepositories groups repository dependencies.
type ReadTaxClassRepositories struct {
	TaxClass taxclasspb.TaxClassDomainServiceServer
}

// ReadTaxClassServices groups service dependencies.
type ReadTaxClassServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ReadTaxClassUseCase handles reading a tax_class.
type ReadTaxClassUseCase struct {
	repositories ReadTaxClassRepositories
	services     ReadTaxClassServices
}

// NewReadTaxClassUseCase creates a new ReadTaxClassUseCase.
func NewReadTaxClassUseCase(repositories ReadTaxClassRepositories, services ReadTaxClassServices) *ReadTaxClassUseCase {
	return &ReadTaxClassUseCase{repositories: repositories, services: services}
}

// Execute performs the read tax_class operation.
func (uc *ReadTaxClassUseCase) Execute(ctx context.Context, req *taxclasspb.ReadTaxClassRequest) (*taxclasspb.ReadTaxClassResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityTaxClass,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"tax_class.validation.id_required", "Tax Class ID is required [DEFAULT]"))
	}
	return uc.repositories.TaxClass.ReadTaxClass(ctx, req)
}

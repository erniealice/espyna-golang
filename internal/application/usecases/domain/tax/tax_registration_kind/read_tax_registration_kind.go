package tax_registration_kind

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	taxregistrationkindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_registration_kind"
)

// ReadTaxRegistrationKindRepositories groups repository dependencies.
type ReadTaxRegistrationKindRepositories struct {
	TaxRegistrationKind taxregistrationkindpb.TaxRegistrationKindDomainServiceServer
}

// ReadTaxRegistrationKindServices groups service dependencies.
type ReadTaxRegistrationKindServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ReadTaxRegistrationKindUseCase handles reading a tax_registration_kind.
type ReadTaxRegistrationKindUseCase struct {
	repositories ReadTaxRegistrationKindRepositories
	services     ReadTaxRegistrationKindServices
}

// NewReadTaxRegistrationKindUseCase creates a new ReadTaxRegistrationKindUseCase.
func NewReadTaxRegistrationKindUseCase(repositories ReadTaxRegistrationKindRepositories, services ReadTaxRegistrationKindServices) *ReadTaxRegistrationKindUseCase {
	return &ReadTaxRegistrationKindUseCase{repositories: repositories, services: services}
}

// Execute performs the read tax_registration_kind operation.
func (uc *ReadTaxRegistrationKindUseCase) Execute(ctx context.Context, req *taxregistrationkindpb.ReadTaxRegistrationKindRequest) (*taxregistrationkindpb.ReadTaxRegistrationKindResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityTaxRegistrationKind,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"tax_registration_kind.validation.id_required", "Tax Registration Kind ID is required [DEFAULT]"))
	}
	return uc.repositories.TaxRegistrationKind.ReadTaxRegistrationKind(ctx, req)
}

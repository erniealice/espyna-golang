package tax_authority

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	taxauthoritypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_authority"
)

// ReadTaxAuthorityRepositories groups repository dependencies.
type ReadTaxAuthorityRepositories struct {
	TaxAuthority taxauthoritypb.TaxAuthorityDomainServiceServer
}

// ReadTaxAuthorityServices groups service dependencies.
type ReadTaxAuthorityServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ReadTaxAuthorityUseCase handles reading a tax_authority.
type ReadTaxAuthorityUseCase struct {
	repositories ReadTaxAuthorityRepositories
	services     ReadTaxAuthorityServices
}

// NewReadTaxAuthorityUseCase creates a new ReadTaxAuthorityUseCase.
func NewReadTaxAuthorityUseCase(repositories ReadTaxAuthorityRepositories, services ReadTaxAuthorityServices) *ReadTaxAuthorityUseCase {
	return &ReadTaxAuthorityUseCase{repositories: repositories, services: services}
}

// Execute performs the read tax_authority operation.
func (uc *ReadTaxAuthorityUseCase) Execute(ctx context.Context, req *taxauthoritypb.ReadTaxAuthorityRequest) (*taxauthoritypb.ReadTaxAuthorityResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityTaxAuthority,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"tax_authority.validation.id_required", "Tax Authority ID is required [DEFAULT]"))
	}
	return uc.repositories.TaxAuthority.ReadTaxAuthority(ctx, req)
}

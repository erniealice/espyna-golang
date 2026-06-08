package revenue_tax_line

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	revenuetaxlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_tax_line"
)

// ReadRevenueTaxLineRepositories groups repository dependencies.
type ReadRevenueTaxLineRepositories struct {
	RevenueTaxLine revenuetaxlinepb.RevenueTaxLineDomainServiceServer
}

// ReadRevenueTaxLineServices groups service dependencies.
type ReadRevenueTaxLineServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
}

// ReadRevenueTaxLineUseCase handles reading a revenue_tax_line.
type ReadRevenueTaxLineUseCase struct {
	repositories ReadRevenueTaxLineRepositories
	services     ReadRevenueTaxLineServices
}

// NewReadRevenueTaxLineUseCase creates a new ReadRevenueTaxLineUseCase.
func NewReadRevenueTaxLineUseCase(repositories ReadRevenueTaxLineRepositories, services ReadRevenueTaxLineServices) *ReadRevenueTaxLineUseCase {
	return &ReadRevenueTaxLineUseCase{repositories: repositories, services: services}
}

// Execute performs the read revenue_tax_line operation.
func (uc *ReadRevenueTaxLineUseCase) Execute(ctx context.Context, req *revenuetaxlinepb.ReadRevenueTaxLineRequest) (*revenuetaxlinepb.ReadRevenueTaxLineResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityRevenueTaxLine, entityid.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"revenue_tax_line.validation.id_required", "Revenue Tax Line ID is required [DEFAULT]"))
	}
	return uc.repositories.RevenueTaxLine.ReadRevenueTaxLine(ctx, req)
}

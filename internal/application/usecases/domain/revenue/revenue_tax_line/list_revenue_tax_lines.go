package revenue_tax_line

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	revenuetaxlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_tax_line"
)

// ListRevenueTaxLinesRepositories groups repository dependencies.
type ListRevenueTaxLinesRepositories struct {
	RevenueTaxLine revenuetaxlinepb.RevenueTaxLineDomainServiceServer
}

// ListRevenueTaxLinesServices groups service dependencies.
type ListRevenueTaxLinesServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// ListRevenueTaxLinesUseCase handles listing revenue tax lines.
type ListRevenueTaxLinesUseCase struct {
	repositories ListRevenueTaxLinesRepositories
	services     ListRevenueTaxLinesServices
}

// NewListRevenueTaxLinesUseCase creates a new ListRevenueTaxLinesUseCase.
func NewListRevenueTaxLinesUseCase(repositories ListRevenueTaxLinesRepositories, services ListRevenueTaxLinesServices) *ListRevenueTaxLinesUseCase {
	return &ListRevenueTaxLinesUseCase{repositories: repositories, services: services}
}

// Execute performs the list revenue_tax_lines operation.
func (uc *ListRevenueTaxLinesUseCase) Execute(ctx context.Context, req *revenuetaxlinepb.ListRevenueTaxLinesRequest) (*revenuetaxlinepb.ListRevenueTaxLinesResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityRevenueTaxLine, ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"revenue_tax_line.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.RevenueTaxLine.ListRevenueTaxLines(ctx, req)
}

package revenue_tax_line

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	revenuetaxlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_tax_line"
)

// RevenueTaxLineSystemWriteQueries is the narrow interface for system-write operations.
// Implemented by the postgres adapter via RevenueTaxLineQueries.
type RevenueTaxLineSystemWriteQueries interface {
	InsertForRevenue(ctx context.Context, lines []*revenuetaxlinepb.RevenueTaxLine) error
	DeleteByRevenueID(ctx context.Context, revenueID string) error
}

// CreateRevenueTaxLineRepositories groups repository dependencies.
type CreateRevenueTaxLineRepositories struct {
	RevenueTaxLine revenuetaxlinepb.RevenueTaxLineDomainServiceServer
}

// CreateRevenueTaxLineServices groups service dependencies.
type CreateRevenueTaxLineServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateRevenueTaxLineUseCase handles system-writes of revenue_tax_line rows.
// This is NOT an operator-facing CRUD use case — it is called internally by
// ComputeTaxesForRevenue after the compute algorithm produces its result.
// Authorization is checked against revenue_tax_line:create which is a system
// permission (not exposed in the operator seed).
type CreateRevenueTaxLineUseCase struct {
	repositories CreateRevenueTaxLineRepositories
	services     CreateRevenueTaxLineServices
}

// NewCreateRevenueTaxLineUseCase creates the use case.
func NewCreateRevenueTaxLineUseCase(
	repositories CreateRevenueTaxLineRepositories,
	services CreateRevenueTaxLineServices,
) *CreateRevenueTaxLineUseCase {
	return &CreateRevenueTaxLineUseCase{repositories: repositories, services: services}
}

// Execute inserts a single revenue_tax_line row.
func (uc *CreateRevenueTaxLineUseCase) Execute(ctx context.Context, line *revenuetaxlinepb.RevenueTaxLine) (*revenuetaxlinepb.RevenueTaxLine, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityRevenueTaxLine, ports.ActionCreate); err != nil {
		return nil, err
	}
	if line == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"revenue_tax_line.validation.data_required", "Revenue tax line data is required [DEFAULT]"))
	}
	if line.RevenueId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"revenue_tax_line.validation.revenue_id_required", "Revenue ID is required [DEFAULT]"))
	}

	now := time.Now()
	if line.Id == "" && uc.services.IDService != nil {
		line.Id = uc.services.IDService.GenerateID()
	}
	line.Active = true
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	line.DateCreated = &ms
	line.DateCreatedString = &s
	line.DateModified = &ms
	line.DateModifiedString = &s

	// RevenueTaxLineDomainServiceServer has no CreateRevenueTaxLine RPC in the proto
	// (the proto only exposes read operations). All writes go through the narrow
	// RevenueTaxLineSystemWriteQueries interface (InsertForRevenue / DeleteByRevenueID)
	// that the postgres adapter implements directly.
	q, ok := uc.repositories.RevenueTaxLine.(RevenueTaxLineSystemWriteQueries)
	if !ok {
		return nil, fmt.Errorf("revenue_tax_line repository does not support InsertForRevenue")
	}

	if err := q.InsertForRevenue(ctx, []*revenuetaxlinepb.RevenueTaxLine{line}); err != nil {
		return nil, fmt.Errorf("insert revenue_tax_line: %w", err)
	}
	return line, nil
}

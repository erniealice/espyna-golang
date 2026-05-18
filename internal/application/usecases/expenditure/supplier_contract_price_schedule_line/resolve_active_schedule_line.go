package suppliercontractpriceschedulesline

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	scpslpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_price_schedule_line"
)

// ResolveActiveScheduleLineRequest carries the inputs for the resolver.
type ResolveActiveScheduleLineRequest struct {
	SupplierContractLineID string
	AsOf                   time.Time
}

// ResolveActiveScheduleLineResponse carries the resolver output. ScheduleLine
// is nil when no schedule-line override exists for the contract line at AsOf —
// in that case callers fall back to SupplierContractLine.unit_price per the
// Model D precedence rule.
type ResolveActiveScheduleLineResponse struct {
	ScheduleLine *scpslpb.SupplierContractPriceScheduleLine
}

// scheduleLineResolver is the adapter-side capability satisfied by the
// PostgresSupplierContractPriceScheduleLineRepository. We assert against this
// interface here so that the use case stays decoupled from the postgres adapter
// import.
type scheduleLineResolver interface {
	ResolveActiveScheduleLine(ctx context.Context, supplierContractLineID string, asOf time.Time) (*scpslpb.SupplierContractPriceScheduleLine, error)
}

// ResolveActiveScheduleLineRepositories groups repository dependencies.
type ResolveActiveScheduleLineRepositories struct {
	SupplierContractPriceScheduleLine scpslpb.SupplierContractPriceScheduleLineDomainServiceServer
}

// ResolveActiveScheduleLineServices groups service dependencies.
type ResolveActiveScheduleLineServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// ResolveActiveScheduleLineUseCase resolves the active schedule line for a contract
// line at a given moment in time. The cross-callable resolver from F12 — used
// by the recurrence engine, procurement spawn helpers, and expense recognition.
type ResolveActiveScheduleLineUseCase struct {
	repositories ResolveActiveScheduleLineRepositories
	services     ResolveActiveScheduleLineServices
}

// NewResolveActiveScheduleLineUseCase creates a use case with grouped dependencies.
func NewResolveActiveScheduleLineUseCase(
	repositories ResolveActiveScheduleLineRepositories,
	services ResolveActiveScheduleLineServices,
) *ResolveActiveScheduleLineUseCase {
	return &ResolveActiveScheduleLineUseCase{repositories: repositories, services: services}
}

// Execute performs the resolve operation.
func (uc *ResolveActiveScheduleLineUseCase) Execute(ctx context.Context, req ResolveActiveScheduleLineRequest) (*ResolveActiveScheduleLineResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entitySupplierContractPriceScheduleLine, ports.ActionRead); err != nil {
		return nil, err
	}
	if req.SupplierContractLineID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract_price_schedule_line.validation.contract_line_id_required", "Supplier contract line ID is required [DEFAULT]"))
	}
	if req.AsOf.IsZero() {
		req.AsOf = time.Now().UTC()
	}

	resolver, ok := uc.repositories.SupplierContractPriceScheduleLine.(scheduleLineResolver)
	if !ok {
		return nil, fmt.Errorf("schedule line repository does not support active-line resolution")
	}
	line, err := resolver.ResolveActiveScheduleLine(ctx, req.SupplierContractLineID, req.AsOf)
	if err != nil {
		return nil, fmt.Errorf("resolve active schedule line: %w", err)
	}
	return &ResolveActiveScheduleLineResponse{ScheduleLine: line}, nil
}

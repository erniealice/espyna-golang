package domain_specific

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
)

// ListRevenueUseCase is the **Go-only** CSV/PDF feeder for raw revenue
// rows.
//
// **Q-SDM-MAP-SHAPES LOCKED 2026-05-20:** the legacy method returned
// `[]map[string]any` because the columns vary per call site (different
// export templates produce different column sets). Q-SDM-MAP-SHAPES
// rejects `google.protobuf.Struct` as a primary contract and keeps this
// method Go-only until a real column schema is chosen. There is no
// proto-shaped `Request`/`Response`; downstream views call
// `Execute(ctx, start, end)` directly.
//
// The same shape applies to [ListExpensesUseCase].
type ListRevenueUseCase struct {
	reporter             reporter
	authorizationService ports.Authorizer
	translationService   ports.Translator
	actionGatekeeper  *actiongate.ActionGatekeeper
}

// NewListRevenueUseCase wires the use case with nil-safe deps.
func NewListRevenueUseCase(
	r reporter,
	authSvc ports.Authorizer,
	i18nSvc ports.Translator,
	actionGate *actiongate.ActionGatekeeper,
) *ListRevenueUseCase {
	if i18nSvc == nil {
		i18nSvc = ports.NewNoOpTranslator()
	}
	return &ListRevenueUseCase{
		reporter:             r,
		authorizationService: authSvc,
		translationService:   i18nSvc,
		actionGatekeeper:     actionGate,
	}
}

// Execute walks revenue rows in [start, end] and returns them as
// `[]map[string]any` for CSV/PDF feeders.
//
// **Signature note:** this is a Go-only method (no proto Request). The
// columns in each map are determined by the underlying adapter query and
// are deliberately not validated here; downstream views shape them into
// their own typed structs at the call boundary.
func (uc *ListRevenueUseCase) Execute(
	ctx context.Context,
	start, end *time.Time,
) ([]map[string]any, error) {
	if err := uc.actionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: "reports",
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}
	if uc.reporter == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.translationService,
			"reports.errors.reporter_unavailable", "Revenue listing is unavailable [DEFAULT]"))
	}
	return uc.reporter.ListRevenue(ctx, start, end)
}

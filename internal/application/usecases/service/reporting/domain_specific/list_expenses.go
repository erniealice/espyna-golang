package domain_specific

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
)

// ListExpensesUseCase is the **Go-only** CSV/PDF feeder for raw expense
// rows. See the package-level comment + [ListRevenueUseCase]'s comment
// for Q-SDM-MAP-SHAPES rationale.
type ListExpensesUseCase struct {
	reporter             reporter
	authorizationService ports.AuthorizationService
	translationService   ports.TranslationService
}

// NewListExpensesUseCase wires the use case with nil-safe deps.
func NewListExpensesUseCase(
	r reporter,
	authSvc ports.AuthorizationService,
	i18nSvc ports.TranslationService,
) *ListExpensesUseCase {
	if i18nSvc == nil {
		i18nSvc = ports.NewNoOpTranslationService()
	}
	return &ListExpensesUseCase{
		reporter:             r,
		authorizationService: authSvc,
		translationService:   i18nSvc,
	}
}

// Execute walks expense rows in [start, end] and returns them as
// `[]map[string]any` for CSV/PDF feeders.
func (uc *ListExpensesUseCase) Execute(
	ctx context.Context,
	start, end *time.Time,
) ([]map[string]any, error) {
	if err := authcheck.Check(
		ctx,
		uc.authorizationService,
		uc.translationService,
		"reports",
		ports.ActionList,
	); err != nil {
		return nil, err
	}
	if uc.reporter == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.translationService,
			"reports.errors.reporter_unavailable", "Expense listing is unavailable [DEFAULT]"))
	}
	return uc.reporter.ListExpenses(ctx, start, end)
}

package tax_rate

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	taxratepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_rate"
)

// TaxRateFindApplicableQueries is the narrow interface from the adapter.
type TaxRateFindApplicableQueries interface {
	FindApplicable(ctx context.Context, workspaceID, jurisdiction, authorityCode, kind, treatment, direction string, asOf time.Time) (*taxratepb.TaxRate, error)
}

// FindApplicableTaxRateRepositories groups repository dependencies.
type FindApplicableTaxRateRepositories struct {
	TaxRate taxratepb.TaxRateDomainServiceServer
}

// FindApplicableTaxRateServices groups service dependencies.
type FindApplicableTaxRateServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// FindApplicableTaxRateRequest is the input for finding an applicable tax rate.
type FindApplicableTaxRateRequest struct {
	WorkspaceID   string
	Jurisdiction  string
	AuthorityCode string
	Kind          string
	Treatment     string
	Direction     string
	AsOf          time.Time
}

// FindApplicableTaxRateUseCase wraps the adapter's FindApplicable method.
// Used by ComputeTaxesForRevenue to look up the asOf-pinned rate.
type FindApplicableTaxRateUseCase struct {
	repositories FindApplicableTaxRateRepositories
	services     FindApplicableTaxRateServices
}

// NewFindApplicableTaxRateUseCase creates the use case.
func NewFindApplicableTaxRateUseCase(
	repositories FindApplicableTaxRateRepositories,
	services FindApplicableTaxRateServices,
) *FindApplicableTaxRateUseCase {
	return &FindApplicableTaxRateUseCase{repositories: repositories, services: services}
}

// Execute returns the most applicable TaxRate for the given parameters.
func (uc *FindApplicableTaxRateUseCase) Execute(ctx context.Context, req *FindApplicableTaxRateRequest) (*taxratepb.TaxRate, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityTaxRate,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"tax_rate.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Kind == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"tax_rate.validation.kind_required", "Rate kind is required [DEFAULT]"))
	}

	q, ok := uc.repositories.TaxRate.(TaxRateFindApplicableQueries)
	if !ok {
		return nil, fmt.Errorf("tax_rate repository does not support FindApplicable")
	}
	asOf := req.AsOf
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	return q.FindApplicable(ctx, req.WorkspaceID, req.Jurisdiction, req.AuthorityCode, req.Kind, req.Treatment, req.Direction, asOf)
}

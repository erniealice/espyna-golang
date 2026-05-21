package tax_registration

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	taxregistrationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_registration"
)

// FindActiveTaxRegistrationQueries is the narrow interface from the adapter.
type FindActiveTaxRegistrationQueries interface {
	FindActive(ctx context.Context, partyType, partyID string, asOf time.Time) ([]*taxregistrationpb.TaxRegistration, error)
}

// FindActiveTaxRegistrationRepositories groups repository dependencies.
type FindActiveTaxRegistrationRepositories struct {
	TaxRegistration taxregistrationpb.TaxRegistrationDomainServiceServer
}

// FindActiveTaxRegistrationServices groups service dependencies.
type FindActiveTaxRegistrationServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// FindActiveTaxRegistrationRequest is the input for finding active registrations.
type FindActiveTaxRegistrationRequest struct {
	PartyType string
	PartyID   string
	AsOf      time.Time
}

// FindActiveTaxRegistrationUseCase wraps the adapter's FindActive method.
// Returns all ACTIVE-at-asOf registrations for a given party.
type FindActiveTaxRegistrationUseCase struct {
	repositories FindActiveTaxRegistrationRepositories
	services     FindActiveTaxRegistrationServices
}

// NewFindActiveTaxRegistrationUseCase creates a new FindActiveTaxRegistrationUseCase.
func NewFindActiveTaxRegistrationUseCase(repositories FindActiveTaxRegistrationRepositories, services FindActiveTaxRegistrationServices) *FindActiveTaxRegistrationUseCase {
	return &FindActiveTaxRegistrationUseCase{repositories: repositories, services: services}
}

// Execute returns all ACTIVE tax_registrations for (partyType, partyID) valid at asOf.
func (uc *FindActiveTaxRegistrationUseCase) Execute(ctx context.Context, req *FindActiveTaxRegistrationRequest) ([]*taxregistrationpb.TaxRegistration, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityTaxRegistration, ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil || req.PartyID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"tax_registration.validation.party_id_required", "Party ID is required [DEFAULT]"))
	}

	q, ok := uc.repositories.TaxRegistration.(FindActiveTaxRegistrationQueries)
	if !ok {
		// Fall back to a full list and filter in memory when the adapter
		// does not implement the narrow interface (e.g. in-memory test stubs).
		return uc.fallbackFindActive(ctx, req)
	}

	asOf := req.AsOf
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}

	regs, err := q.FindActive(ctx, req.PartyType, req.PartyID, asOf)
	if err != nil {
		return nil, fmt.Errorf("find active tax_registrations: %w", err)
	}
	return regs, nil
}

// fallbackFindActive performs a list + in-memory filter for adapters that don't
// implement FindActive (test stubs, etc.).
func (uc *FindActiveTaxRegistrationUseCase) fallbackFindActive(ctx context.Context, req *FindActiveTaxRegistrationRequest) ([]*taxregistrationpb.TaxRegistration, error) {
	resp, err := uc.repositories.TaxRegistration.ListTaxRegistrations(ctx, &taxregistrationpb.ListTaxRegistrationsRequest{})
	if err != nil {
		return nil, fmt.Errorf("list tax_registrations (fallback): %w", err)
	}
	if resp == nil {
		return nil, nil
	}
	asOf := req.AsOf
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	asOfStr := asOf.Format("2006-01-02")
	var result []*taxregistrationpb.TaxRegistration
	for _, r := range resp.GetData() {
		// PartyType filter: compare the string name to the enum's String() output
		// (e.g. req.PartyType "workspace" vs enum String "TAX_REGISTRATION_PARTY_TYPE_WORKSPACE").
		// Skip filter if req.PartyType is empty or no match by suffix is attempted.
		if req.PartyType != "" {
			enumName := r.GetPartyType().String() // e.g. "TAX_REGISTRATION_PARTY_TYPE_WORKSPACE"
			if !partyTypeStringMatches(req.PartyType, enumName) {
				continue
			}
		}
		if r.GetPartyId() != req.PartyID {
			continue
		}
		if r.GetStatus() != taxregistrationpb.TaxRegistrationStatus_TAX_REGISTRATION_STATUS_ACTIVE {
			continue
		}
		if r.GetEffectiveFrom() > asOfStr {
			continue
		}
		// GetEffectiveTo() returns string (empty string when unset — open-ended row).
		if et := r.GetEffectiveTo(); et != "" && et <= asOfStr {
			continue
		}
		result = append(result, r)
	}
	return result, nil
}

// partyTypeStringMatches checks whether the short name (e.g. "workspace") matches the
// tail of the full enum name (e.g. "TAX_REGISTRATION_PARTY_TYPE_WORKSPACE").
func partyTypeStringMatches(short, enumName string) bool {
	if short == "" {
		return true
	}
	// Normalize: uppercase short for comparison with the enum suffix.
	shortUpper := ""
	for _, c := range short {
		if c >= 'a' && c <= 'z' {
			shortUpper += string(rune(c - 32))
		} else {
			shortUpper += string(c)
		}
	}
	// e.g. "WORKSPACE" is a suffix of "TAX_REGISTRATION_PARTY_TYPE_WORKSPACE"
	suffix := "_" + shortUpper
	return len(enumName) >= len(suffix) && enumName[len(enumName)-len(suffix):] == suffix
}

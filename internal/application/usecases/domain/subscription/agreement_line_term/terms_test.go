package agreement_line_term

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	agreementlinetermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/agreement_line_term"
)

func term(id, client, from, to string) *agreementlinetermpb.AgreementLineTerm {
	t := &agreementlinetermpb.AgreementLineTerm{Id: id, SubscriptionId: "sub-1", ClientId: client, EffectiveFrom: from, Active: true}
	if to != "" {
		t.EffectiveTo = &to
	}
	return t
}

func TestCoversHalfOpenIntervals(t *testing.T) {
	open := term("a", "c", "2026-01-01", "")
	closed := term("b", "c", "2026-01-01", "2026-02-01")
	for name, c := range map[string]struct {
		tm       *agreementlinetermpb.AgreementLineTerm
		from, to string
		want     bool
	}{
		"open-ended covers":              {open, "2026-01-01", "2026-03-01", true},
		"starts before the term":         {open, "2025-12-31", "2026-01-31", false},
		"ends exactly at effective_to":   {closed, "2026-01-01", "2026-02-01", true},
		"runs past effective_to":         {closed, "2026-01-15", "2026-02-02", false},
		"empty or inverted interval":     {open, "2026-02-01", "2026-02-01", false},
		"missing bound is never covered": {open, "", "2026-02-01", false},
	} {
		if got := Covers(c.tm, c.from, c.to); got != c.want {
			t.Errorf("%s: got %v want %v", name, got, c.want)
		}
	}
	if Overlaps(closed, term("n", "c", "2026-02-01", "")) {
		t.Error("[a,b) and [b,...) must not overlap")
	}
	if !Overlaps(closed, term("n", "c", "2026-01-31", "")) {
		t.Error("intersecting terms must overlap")
	}
}

func TestFindCoveringTerm(t *testing.T) {
	a := term("a", "c1", "2026-01-01", "2026-02-01")
	b := term("b", "c1", "2026-02-01", "")
	got, err := FindCoveringTerm([]*agreementlinetermpb.AgreementLineTerm{b, a}, "c1", "2026-01-05", "2026-01-25")
	if err != nil || got.Id != "a" {
		t.Fatalf("got %v %v", got, err)
	}
	if _, err := FindCoveringTerm([]*agreementlinetermpb.AgreementLineTerm{a, b}, "c1", "2026-01-15", "2026-02-15"); !usecaseerr.IsCode(err, "term_boundary_crossed") {
		t.Fatalf("want boundary crossed, got %v", err)
	}
	if _, err := FindCoveringTerm([]*agreementlinetermpb.AgreementLineTerm{a}, "other", "2026-01-05", "2026-01-25"); !usecaseerr.IsCode(err, "agreement_term_missing") {
		t.Fatalf("want missing, got %v", err)
	}
	inactive := term("x", "c1", "2026-01-01", "")
	inactive.Active = false
	if _, err := FindCoveringTerm([]*agreementlinetermpb.AgreementLineTerm{inactive}, "c1", "2026-01-05", "2026-01-25"); !usecaseerr.IsCode(err, "agreement_term_missing") {
		t.Fatalf("inactive terms are ignored, got %v", err)
	}
}

type listRepo struct {
	agreementlinetermpb.UnimplementedAgreementLineTermDomainServiceServer
	rows []*agreementlinetermpb.AgreementLineTerm
	last *agreementlinetermpb.ListAgreementLineTermsRequest
}

func (l *listRepo) ListAgreementLineTerms(_ context.Context, r *agreementlinetermpb.ListAgreementLineTermsRequest) (*agreementlinetermpb.ListAgreementLineTermsResponse, error) {
	// an adapter that ignored the filter: the helper must not trust it
	l.last = r
	return &agreementlinetermpb.ListAgreementLineTermsResponse{Data: l.rows, Success: true}, nil
}

func TestListBySubscriptionOrdersAndFilters(t *testing.T) {
	other := term("z", "c", "2026-01-01", "")
	other.SubscriptionId = "sub-2"
	repo := &listRepo{rows: []*agreementlinetermpb.AgreementLineTerm{term("b", "c", "2026-02-01", ""), other, term("a", "c", "2026-01-01", "2026-02-01")}}
	uc := NewUseCases(Repositories{AgreementLineTerm: repo}, Services{
		Authorizer: ports.NewNoOpAuthorizer(), Translator: ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
	})
	ctx := contextutil.WithUserID(context.Background(), "u1")
	sub := "sub-1"
	got, err := uc.ListAgreementLineTerms.Execute(ctx, &agreementlinetermpb.ListAgreementLineTermsRequest{SubscriptionId: &sub})
	if err != nil || len(got.Data) != 2 || got.Data[0].Id != "a" || got.Data[1].Id != "b" {
		t.Fatalf("got %v %v", got, err)
	}
	// the request forwarded to the repository carries the subscription filter and the default order
	if repo.last == nil || len(repo.last.GetFilters().GetFilters()) != 1 || repo.last.GetFilters().GetFilters()[0].GetField() != "subscription_id" ||
		len(repo.last.GetSort().GetFields()) != 2 {
		t.Fatalf("filter/sort not forwarded: %+v", repo.last)
	}
	blank := ""
	if _, err := uc.ListAgreementLineTerms.Execute(ctx, &agreementlinetermpb.ListAgreementLineTermsRequest{SubscriptionId: &blank}); err == nil {
		t.Fatal("a blank subscription id must be refused")
	}
}

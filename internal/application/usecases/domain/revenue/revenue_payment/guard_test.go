package revenuepayment

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_payment"
	agreementlinetermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/agreement_line_term"
	collectionapplicationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_application"
)

type fakePayments struct {
	pb.UnimplementedRevenuePaymentDomainServiceServer
	rows                      map[string]*pb.RevenuePayment
	creates, updates, deletes int
}

func (f *fakePayments) CreateRevenuePayment(_ context.Context, r *pb.CreateRevenuePaymentRequest) (*pb.CreateRevenuePaymentResponse, error) {
	f.creates++
	f.rows[r.Data.Id] = r.Data
	return &pb.CreateRevenuePaymentResponse{Success: true, Data: []*pb.RevenuePayment{r.Data}}, nil
}
func (f *fakePayments) ReadRevenuePayment(_ context.Context, r *pb.ReadRevenuePaymentRequest) (*pb.ReadRevenuePaymentResponse, error) {
	p, ok := f.rows[r.Data.Id]
	if !ok {
		return nil, errors.New("not found")
	}
	return &pb.ReadRevenuePaymentResponse{Data: []*pb.RevenuePayment{p}}, nil
}
func (f *fakePayments) UpdateRevenuePayment(_ context.Context, r *pb.UpdateRevenuePaymentRequest) (*pb.UpdateRevenuePaymentResponse, error) {
	f.updates++
	return &pb.UpdateRevenuePaymentResponse{Success: true}, nil
}
func (f *fakePayments) DeleteRevenuePayment(_ context.Context, r *pb.DeleteRevenuePaymentRequest) (*pb.DeleteRevenuePaymentResponse, error) {
	f.deletes++
	delete(f.rows, r.Data.Id)
	return &pb.DeleteRevenuePaymentResponse{Success: true}, nil
}

type fakeRevenues struct {
	revenuepb.UnimplementedRevenueDomainServiceServer
	rows map[string]*revenuepb.Revenue
}

func (f *fakeRevenues) ReadRevenue(_ context.Context, r *revenuepb.ReadRevenueRequest) (*revenuepb.ReadRevenueResponse, error) {
	v, ok := f.rows[r.Data.Id]
	if !ok {
		return nil, errors.New("not found")
	}
	return &revenuepb.ReadRevenueResponse{Data: []*revenuepb.Revenue{v}}, nil
}

type fakeApps struct {
	collectionapplicationpb.UnimplementedCollectionApplicationDomainServiceServer
	byRevenue map[string]int
}

func (f *fakeApps) ListCollectionApplications(_ context.Context, r *collectionapplicationpb.ListCollectionApplicationsRequest) (*collectionapplicationpb.ListCollectionApplicationsResponse, error) {
	out := &collectionapplicationpb.ListCollectionApplicationsResponse{Success: true}
	for i := 0; i < f.byRevenue[r.GetFilters().GetFilters()[0].GetStringFilter().GetValue()]; i++ {
		out.Data = append(out.Data, &collectionapplicationpb.CollectionApplication{Id: fmt.Sprint(i)})
	}
	return out, nil
}

type fakeTerms struct {
	agreementlinetermpb.UnimplementedAgreementLineTermDomainServiceServer
	bySub map[string]int
}

func (f *fakeTerms) ListAgreementLineTerms(_ context.Context, r *agreementlinetermpb.ListAgreementLineTermsRequest) (*agreementlinetermpb.ListAgreementLineTermsResponse, error) {
	out := &agreementlinetermpb.ListAgreementLineTermsResponse{Success: true}
	for i := 0; i < f.bySub[r.GetFilters().GetFilters()[0].GetStringFilter().GetValue()]; i++ {
		out.Data = append(out.Data, &agreementlinetermpb.AgreementLineTerm{Id: fmt.Sprint(i)})
	}
	return out, nil
}

type allowAll struct{}

func (allowAll) IsEnabled() bool { return true }
func (allowAll) HasPermission(context.Context, string, string) (bool, error) {
	return true, nil
}

type ids struct{}

func (ids) GenerateID() string                        { return "gen-id" }
func (ids) IsEnabled() bool                           { return true }
func (ids) GetProviderInfo() string                   { return "t" }
func (ids) GenerateIDWithPrefix(prefix string) string { return prefix + "gen-id" }

func ptr(s string) *string { return &s }

func newGuardHarness(withGuard bool) (*UseCases, *fakePayments, *fakeApps, *fakeTerms) {
	pays := &fakePayments{rows: map[string]*pb.RevenuePayment{
		"pay-rent":     {Id: "pay-rent", RevenueId: "rent-plain", Amount: 10},
		"pay-recovery": {Id: "pay-recovery", RevenueId: "rent-with-terms", Amount: 10},
		"pay-applied":  {Id: "pay-applied", RevenueId: "rent-with-app", Amount: 10},
	}}
	revs := &fakeRevenues{rows: map[string]*revenuepb.Revenue{
		"rent-plain":      {Id: "rent-plain", SubscriptionId: ptr("sub-plain")},
		"rent-with-terms": {Id: "rent-with-terms", SubscriptionId: ptr("sub-recovery")},
		"rent-with-app":   {Id: "rent-with-app", SubscriptionId: ptr("sub-plain")},
		"walk-in":         {Id: "walk-in"},
	}}
	apps := &fakeApps{byRevenue: map[string]int{"rent-with-app": 1}}
	terms := &fakeTerms{bySub: map[string]int{"sub-recovery": 2}}
	repos := RevenuePaymentRepositories{RevenuePayment: pays}
	if withGuard {
		repos.Revenue, repos.CollectionApplication, repos.AgreementLineTerm = revs, apps, terms
	}
	uc := NewUseCases(repos, RevenuePaymentServices{
		Authorizer: ports.NewNoOpAuthorizer(), Translator: ports.NewNoOpTranslator(), IDGenerator: ids{},
		ActionGatekeeper: actiongate.NewActionGatekeeper(allowAll{}, ports.NewNoOpTranslator()),
	})
	return uc, pays, apps, terms
}

func gctx() context.Context { return contextutil.WithUserID(context.Background(), "u1") }

// AC-UC-33: create/update/delete on a participating revenue (opted-in subscription OR an existing
// application) are refused with the named code and never reach the repository.
func TestLegacyPaymentRefusedForParticipatingRevenue(t *testing.T) {
	uc, pays, _, _ := newGuardHarness(true)
	for _, revID := range []string{"rent-with-terms", "rent-with-app"} {
		if _, err := uc.CreateRevenuePayment.Execute(gctx(), &pb.CreateRevenuePaymentRequest{Data: &pb.RevenuePayment{RevenueId: revID, Amount: 5}}); !usecaseerr.IsCode(err, codeLegacyPaymentRefusedParticipating) {
			t.Fatalf("create on %s: err = %v", revID, err)
		}
	}
	if _, err := uc.UpdateRevenuePayment.Execute(gctx(), &pb.UpdateRevenuePaymentRequest{Data: &pb.RevenuePayment{Id: "pay-recovery", Amount: 1}}); !usecaseerr.IsCode(err, codeLegacyPaymentRefusedParticipating) {
		t.Fatalf("update err = %v", err)
	}
	if _, err := uc.UpdateRevenuePayment.Execute(gctx(), &pb.UpdateRevenuePaymentRequest{Data: &pb.RevenuePayment{Id: "pay-rent", RevenueId: "rent-with-terms"}}); !usecaseerr.IsCode(err, codeLegacyPaymentRefusedParticipating) {
		t.Fatalf("moving a payment onto a participating revenue: err = %v", err)
	}
	if _, err := uc.DeleteRevenuePayment.Execute(gctx(), &pb.DeleteRevenuePaymentRequest{Data: &pb.RevenuePayment{Id: "pay-applied"}}); !usecaseerr.IsCode(err, codeLegacyPaymentRefusedParticipating) {
		t.Fatalf("delete err = %v", err)
	}
	if pays.creates+pays.updates+pays.deletes != 0 {
		t.Fatalf("a refused write reached the repository (%d/%d/%d)", pays.creates, pays.updates, pays.deletes)
	}
	_, err := uc.CreateRevenuePayment.Execute(gctx(), &pb.CreateRevenuePaymentRequest{Data: &pb.RevenuePayment{RevenueId: "rent-with-terms", Amount: 5}})
	var ce interface{ ErrorCode() string }
	if !errors.As(err, &ce) || ce.ErrorCode() != "legacy_payment_refused_participating" {
		t.Fatalf("error code must be legacy_payment_refused_participating, got %v", err)
	}
}

// AC-UC-33 / AC-UC-PRES: non-participating revenues, and an instance with no guard wired at all,
// behave exactly as before.
func TestLegacyPaymentUnchangedForNonParticipating(t *testing.T) {
	for _, withGuard := range []bool{true, false} {
		uc, pays, _, _ := newGuardHarness(withGuard)
		if _, err := uc.CreateRevenuePayment.Execute(gctx(), &pb.CreateRevenuePaymentRequest{Data: &pb.RevenuePayment{RevenueId: "rent-plain", Amount: 5}}); err != nil {
			t.Fatalf("guard=%v create: %v", withGuard, err)
		}
		if _, err := uc.CreateRevenuePayment.Execute(gctx(), &pb.CreateRevenuePaymentRequest{Data: &pb.RevenuePayment{RevenueId: "walk-in", Amount: 5}}); err != nil {
			t.Fatalf("guard=%v create (no subscription): %v", withGuard, err)
		}
		if _, err := uc.UpdateRevenuePayment.Execute(gctx(), &pb.UpdateRevenuePaymentRequest{Data: &pb.RevenuePayment{Id: "pay-rent", Amount: 1}}); err != nil {
			t.Fatalf("guard=%v update: %v", withGuard, err)
		}
		if _, err := uc.DeleteRevenuePayment.Execute(gctx(), &pb.DeleteRevenuePaymentRequest{Data: &pb.RevenuePayment{Id: "pay-rent"}}); err != nil {
			t.Fatalf("guard=%v delete: %v", withGuard, err)
		}
		if pays.creates != 2 || pays.updates != 1 || pays.deletes != 1 {
			t.Fatalf("guard=%v repo calls = %d/%d/%d", withGuard, pays.creates, pays.updates, pays.deletes)
		}
	}
	// without the guard wired even a participating revenue keeps legacy behaviour (nothing to consult)
	uc, _, _, _ := newGuardHarness(false)
	if _, err := uc.CreateRevenuePayment.Execute(gctx(), &pb.CreateRevenuePaymentRequest{Data: &pb.RevenuePayment{RevenueId: "rent-with-terms", Amount: 5}}); err != nil {
		t.Fatalf("unwired guard must not change behaviour: %v", err)
	}
}

// An unreadable revenue fails closed for create (the guard cannot prove non-participation).
func TestLegacyPaymentGuardFailsClosedWhenRevenueUnreadable(t *testing.T) {
	uc, pays, _, _ := newGuardHarness(true)
	if _, err := uc.CreateRevenuePayment.Execute(gctx(), &pb.CreateRevenuePaymentRequest{Data: &pb.RevenuePayment{RevenueId: "ghost", Amount: 5}}); err == nil {
		t.Fatal("create against an unreadable revenue must be refused")
	}
	if pays.creates != 0 {
		t.Fatal("nothing may be written")
	}
}

// brokenPayments fails every read with an infrastructure error.
type brokenPayments struct{ *fakePayments }

func (brokenPayments) ReadRevenuePayment(context.Context, *pb.ReadRevenuePaymentRequest) (*pb.ReadRevenuePaymentResponse, error) {
	return nil, errors.New("connection reset by peer")
}

// C12: when the stored payment cannot be read (an infrastructure failure, not "not found") update and
// delete are refused instead of skipping the guard.
func TestLegacyPaymentGuardFailsClosedWhenPaymentUnreadable(t *testing.T) {
	_, pays, apps, terms := newGuardHarness(true)
	revs := &fakeRevenues{rows: map[string]*revenuepb.Revenue{"rent-with-terms": {Id: "rent-with-terms", SubscriptionId: ptr("sub-recovery")}}}
	uc := NewUseCases(RevenuePaymentRepositories{RevenuePayment: brokenPayments{pays}, Revenue: revs, CollectionApplication: apps, AgreementLineTerm: terms},
		RevenuePaymentServices{Authorizer: ports.NewNoOpAuthorizer(), Translator: ports.NewNoOpTranslator(), IDGenerator: ids{},
			ActionGatekeeper: actiongate.NewActionGatekeeper(allowAll{}, ports.NewNoOpTranslator())})
	if _, err := uc.UpdateRevenuePayment.Execute(gctx(), &pb.UpdateRevenuePaymentRequest{Data: &pb.RevenuePayment{Id: "pay-recovery", Amount: 1}}); err == nil {
		t.Fatal("update with an unreadable stored payment must be refused")
	}
	if _, err := uc.DeleteRevenuePayment.Execute(gctx(), &pb.DeleteRevenuePaymentRequest{Data: &pb.RevenuePayment{Id: "pay-recovery"}}); err == nil {
		t.Fatal("delete with an unreadable stored payment must be refused")
	}
	if pays.updates != 0 || pays.deletes != 0 {
		t.Fatal("nothing may be written")
	}
}

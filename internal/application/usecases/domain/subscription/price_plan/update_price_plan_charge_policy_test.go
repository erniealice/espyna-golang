package price_plan

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	productpriceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/product_price_plan"
)

type cpAllowAll struct{}

func (cpAllowAll) IsEnabled() bool                                             { return true }
func (cpAllowAll) HasPermission(context.Context, string, string) (bool, error) { return true, nil }
func (cpAllowAll) HasGlobalPermission(context.Context, string, string) (bool, error) {
	return true, nil
}

type cpPlanRepo struct {
	planpb.UnimplementedPlanDomainServiceServer
}

func (cpPlanRepo) ReadPlan(_ context.Context, req *planpb.ReadPlanRequest) (*planpb.ReadPlanResponse, error) {
	return &planpb.ReadPlanResponse{Data: []*planpb.Plan{{Id: req.GetData().Id, Active: true}}, Success: true}, nil
}

type cpPricePlanRepo struct {
	priceplanpb.UnimplementedPricePlanDomainServiceServer
	stored  *priceplanpb.PricePlan
	readErr error
	updated []*priceplanpb.PricePlan
	locks   []string
	lockErr error
}

// LockPricePlanForUpdate satisfies domainports.PricePlanLocker.
func (r *cpPricePlanRepo) LockPricePlanForUpdate(_ context.Context, id string) error {
	if r.lockErr != nil {
		return r.lockErr
	}
	r.locks = append(r.locks, id)
	return nil
}

type cpTx struct{ calls int }

func (t *cpTx) SupportsTransactions() bool               { return true }
func (t *cpTx) IsTransactionActive(context.Context) bool { return false }
func (t *cpTx) ExecuteInTransaction(ctx context.Context, fn func(context.Context) error) error {
	t.calls++
	return fn(ctx)
}

func (r *cpPricePlanRepo) ReadPricePlan(context.Context, *priceplanpb.ReadPricePlanRequest) (*priceplanpb.ReadPricePlanResponse, error) {
	if r.readErr != nil {
		return nil, r.readErr
	}
	return &priceplanpb.ReadPricePlanResponse{Data: []*priceplanpb.PricePlan{r.stored}, Success: true}, nil
}
func (r *cpPricePlanRepo) UpdatePricePlan(_ context.Context, req *priceplanpb.UpdatePricePlanRequest) (*priceplanpb.UpdatePricePlanResponse, error) {
	r.updated = append(r.updated, req.Data)
	return &priceplanpb.UpdatePricePlanResponse{Data: []*priceplanpb.PricePlan{req.Data}, Success: true}, nil
}

type cpLineRepo struct {
	productpriceplanpb.UnimplementedProductPricePlanDomainServiceServer
	lines   []*productpriceplanpb.ProductPricePlan
	listErr error
}

func (r *cpLineRepo) ListProductPricePlans(context.Context, *productpriceplanpb.ListProductPricePlansRequest) (*productpriceplanpb.ListProductPricePlansResponse, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	return &productpriceplanpb.ListProductPricePlansResponse{Data: r.lines, Success: true}, nil
}

func cpUpdate(t *testing.T, pp *cpPricePlanRepo, lines *cpLineRepo, patch *priceplanpb.PricePlan) error {
	t.Helper()
	return cpUpdateWith(t, pp, lines, &cpTx{}, patch)
}

func cpUpdateWith(t *testing.T, pp *cpPricePlanRepo, lines productpriceplanpb.ProductPricePlanDomainServiceServer, tx ports.Transactor, patch *priceplanpb.PricePlan) error {
	t.Helper()
	uc := NewUpdatePricePlanUseCase(
		UpdatePricePlanRepositories{PricePlan: pp, Plan: cpPlanRepo{}, ProductPricePlan: lines},
		UpdatePricePlanServices{Transactor: tx, Authorizer: ports.NewNoOpAuthorizer(), Translator: ports.NewNoOpTranslator(), ActionGatekeeper: actiongate.NewActionGatekeeper(cpAllowAll{}, ports.NewNoOpTranslator())},
	)
	patch.Id, patch.PlanId, patch.BillingCurrency = "pp-1", "plan-1", "PHP"
	_, err := uc.Execute(contextutil.WithUserID(context.Background(), "u1"), &priceplanpb.UpdatePricePlanRequest{Data: patch})
	return err
}

// C12: changing the parent price plan's billing_kind / amount_basis re-checks the opted-in lines.
func TestUpdatePricePlanRechecksOptedInLines(t *testing.T) {
	polID := "pol-1"
	optedIn := []*productpriceplanpb.ProductPricePlan{{Id: "l1", PricePlanId: "pp-1"}, {Id: "l2", PricePlanId: "pp-1", ChargePolicyId: &polID}}
	legacy := []*productpriceplanpb.ProductPricePlan{{Id: "l1", PricePlanId: "pp-1"}}
	recurring := func() *cpPricePlanRepo {
		return &cpPricePlanRepo{stored: &priceplanpb.PricePlan{Id: "pp-1", BillingKind: priceplanpb.BillingKind_BILLING_KIND_RECURRING}}
	}
	code := func(err error) string {
		var ce interface{ ErrorCode() string }
		if errors.As(err, &ce) {
			return ce.ErrorCode()
		}
		return ""
	}

	t.Run("RECURRING -> ONE_TIME with an opted-in line is refused", func(t *testing.T) {
		pp := recurring()
		err := cpUpdate(t, pp, &cpLineRepo{lines: optedIn}, &priceplanpb.PricePlan{BillingKind: priceplanpb.BillingKind_BILLING_KIND_ONE_TIME})
		if code(err) != "charge_policy_billing_kind" {
			t.Fatalf("got %v", err)
		}
		if len(pp.updated) != 0 {
			t.Fatal("nothing may be written")
		}
	})
	t.Run("-> TOTAL_PACKAGE amount basis with an opted-in line is refused", func(t *testing.T) {
		err := cpUpdate(t, recurring(), &cpLineRepo{lines: optedIn}, &priceplanpb.PricePlan{AmountBasis: priceplanpb.AmountBasis_AMOUNT_BASIS_TOTAL_PACKAGE})
		if code(err) != "charge_policy_billing_kind" {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("the same change with no opted-in line is allowed", func(t *testing.T) {
		pp := recurring()
		if err := cpUpdate(t, pp, &cpLineRepo{lines: legacy}, &priceplanpb.PricePlan{BillingKind: priceplanpb.BillingKind_BILLING_KIND_ONE_TIME}); err != nil {
			t.Fatalf("got %v", err)
		}
		if len(pp.updated) != 1 {
			t.Fatal("update must be written")
		}
	})
	t.Run("a change to another allowed kind keeps opted-in lines", func(t *testing.T) {
		if err := cpUpdate(t, recurring(), &cpLineRepo{lines: optedIn}, &priceplanpb.PricePlan{BillingKind: priceplanpb.BillingKind_BILLING_KIND_CONTRACT}); err != nil {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("a request without the guarded fields skips the check (partial update keeps the stored kind)", func(t *testing.T) {
		if err := cpUpdate(t, recurring(), &cpLineRepo{lines: optedIn}, &priceplanpb.PricePlan{}); err != nil {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("line list error fails closed", func(t *testing.T) {
		pp := recurring()
		err := cpUpdate(t, pp, &cpLineRepo{listErr: fmt.Errorf("connection reset")}, &priceplanpb.PricePlan{BillingKind: priceplanpb.BillingKind_BILLING_KIND_ONE_TIME})
		if err == nil || !strings.Contains(err.Error(), "connection reset") || len(pp.updated) != 0 {
			t.Fatalf("got %v (updated=%d)", err, len(pp.updated))
		}
	})
	t.Run("stored price plan read error fails closed", func(t *testing.T) {
		pp := recurring()
		pp.readErr = fmt.Errorf("deadline exceeded")
		err := cpUpdate(t, pp, &cpLineRepo{lines: legacy}, &priceplanpb.PricePlan{BillingKind: priceplanpb.BillingKind_BILLING_KIND_ONE_TIME})
		if err == nil || !strings.Contains(err.Error(), "deadline exceeded") || len(pp.updated) != 0 {
			t.Fatalf("got %v", err)
		}
	})
}

// R4 m3: the re-check runs under the price plan lock inside one transaction and fails closed
// when the line repository, the locker or the transaction is missing.
func TestUpdatePricePlanGuardLocksAndFailsClosed(t *testing.T) {
	recurring := func() *cpPricePlanRepo {
		return &cpPricePlanRepo{stored: &priceplanpb.PricePlan{Id: "pp-1", BillingKind: priceplanpb.BillingKind_BILLING_KIND_RECURRING}}
	}
	oneTime := func() *priceplanpb.PricePlan {
		return &priceplanpb.PricePlan{BillingKind: priceplanpb.BillingKind_BILLING_KIND_ONE_TIME}
	}
	unverifiable := func(err error) bool {
		var ce interface{ ErrorCode() string }
		return errors.As(err, &ce) && ce.ErrorCode() == "charge_policy_unverifiable"
	}

	t.Run("guarded change locks the price plan inside a transaction", func(t *testing.T) {
		pp, tx := recurring(), &cpTx{}
		if err := cpUpdateWith(t, pp, &cpLineRepo{}, tx, oneTime()); err != nil {
			t.Fatal(err)
		}
		if len(pp.locks) != 1 || pp.locks[0] != "pp-1" || tx.calls != 1 {
			t.Fatalf("locks=%v tx=%d", pp.locks, tx.calls)
		}
	})
	t.Run("unguarded change takes no lock", func(t *testing.T) {
		pp, tx := recurring(), &cpTx{}
		if err := cpUpdateWith(t, pp, &cpLineRepo{}, tx, &priceplanpb.PricePlan{}); err != nil {
			t.Fatal(err)
		}
		if len(pp.locks) != 0 || tx.calls != 0 {
			t.Fatalf("locks=%v tx=%d", pp.locks, tx.calls)
		}
	})
	t.Run("unwired line repository refuses", func(t *testing.T) {
		pp := recurring()
		var none productpriceplanpb.ProductPricePlanDomainServiceServer
		if err := cpUpdateWith(t, pp, none, &cpTx{}, oneTime()); !unverifiable(err) || len(pp.updated) != 0 {
			t.Fatalf("got %v updated=%d", err, len(pp.updated))
		}
	})
	t.Run("no transaction refuses", func(t *testing.T) {
		pp := recurring()
		if err := cpUpdateWith(t, pp, &cpLineRepo{}, nil, oneTime()); !unverifiable(err) || len(pp.updated) != 0 {
			t.Fatalf("got %v updated=%d", err, len(pp.updated))
		}
	})
	t.Run("lock infrastructure error fails closed", func(t *testing.T) {
		pp := recurring()
		pp.lockErr = fmt.Errorf("deadlock detected")
		if err := cpUpdateWith(t, pp, &cpLineRepo{}, &cpTx{}, oneTime()); err == nil || !strings.Contains(err.Error(), "deadlock") || len(pp.updated) != 0 {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("only the exact record-not-found message counts as not found", func(t *testing.T) {
		pp := recurring()
		pp.readErr = fmt.Errorf("connection to server not found in pool")
		if err := cpUpdateWith(t, pp, &cpLineRepo{}, &cpTx{}, oneTime()); err == nil || len(pp.updated) != 0 {
			t.Fatalf("a loose 'not found' must fail closed, got %v", err)
		}
	})
}

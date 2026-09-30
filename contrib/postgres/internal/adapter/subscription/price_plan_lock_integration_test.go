//go:build postgresql

package subscription

import (
	"context"
	"errors"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core/scopetest"
	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	agreementlinetermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/agreement_line_term"
)

// R4 m3: LockPricePlanForUpdate locks the workspace-bearing parent plan row (price_plan has no
// workspace_id): the owner workspace locks, a foreign workspace / no workspace / a missing id see
// not found (domainports.ErrLockedRowNotFound), and outside a transaction it fails closed.
func TestPricePlanLockScopedThroughParentPlan(t *testing.T) {
	h := scopetest.New(t, "plan", "price_plan")
	repo := NewPostgresPricePlanRepository(h.Ops, entityid.PricePlan).(*PostgresPricePlanRepository)
	h.Run(t, func(base, ctxA, ctxB, ctxNone context.Context) {
		exec := postgresCore.TxExecutor(base, h.Ops)
		if _, err := exec.ExecContext(base, `INSERT INTO plan (id, workspace_id) VALUES ('s1lock-plan', $1)`, scopetest.WsA); err != nil {
			t.Fatalf("seed plan: %v", err)
		}
		if _, err := exec.ExecContext(base, `INSERT INTO price_plan (id, plan_id) VALUES ('s1lock-pp', 's1lock-plan')`); err != nil {
			t.Fatalf("seed price_plan: %v", err)
		}
		if err := repo.LockPricePlanForUpdate(ctxA, "s1lock-pp"); err != nil {
			t.Fatalf("owner workspace must lock: %v", err)
		}
		for name, ctx := range map[string]context.Context{"foreign workspace": ctxB, "no workspace": ctxNone} {
			if err := repo.LockPricePlanForUpdate(ctx, "s1lock-pp"); err == nil {
				t.Errorf("%s: lock must fail closed", name)
			}
		}
		if err := repo.LockPricePlanForUpdate(ctxB, "s1lock-pp"); !errors.Is(err, domainports.ErrLockedRowNotFound) {
			t.Errorf("foreign workspace must read as not found, got %v", err)
		}
		if err := repo.LockPricePlanForUpdate(ctxA, "s1lock-missing"); !errors.Is(err, domainports.ErrLockedRowNotFound) {
			t.Errorf("missing price plan must be not found, got %v", err)
		}
	})
	if err := repo.LockPricePlanForUpdate(scopetest.Ctx(context.Background(), scopetest.WsA), "s1lock-pp"); err == nil {
		t.Error("a lock outside a transaction must fail closed")
	}
}

// R4 m8: agreement_line_term rows are frozen snapshots; the adapter Delete refuses even for the
// owning workspace (terms end through effective_to).
func TestAgreementLineTermDeleteRefused(t *testing.T) {
	h := scopetest.New(t, "agreement_line_term")
	repo := NewPostgresAgreementLineTermRepository(h.Ops, entityid.AgreementLineTerm).(*PostgresAgreementLineTermRepository)
	h.Run(t, func(base, ctxA, _, _ context.Context) {
		_, err := repo.DeleteAgreementLineTerm(ctxA, &agreementlinetermpb.DeleteAgreementLineTermRequest{Data: &agreementlinetermpb.AgreementLineTerm{Id: "any"}})
		if !postgresCore.IsImmutableRecord(err) || !errors.Is(err, domainports.ErrImmutableRow) {
			t.Fatalf("delete must be refused as immutable, got %v", err)
		}
	})
}

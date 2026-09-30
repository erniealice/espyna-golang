//go:build postgresql

package treasury

import (
	"context"
	"errors"
	"testing"

	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core/scopetest"
	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
)

// C25: the collection locker the legacy update/delete take before refusing an applied receipt is
// workspace-scoped: the owning workspace locks the row, a foreign workspace sees the not-found
// sentinel, and a context without a workspace fails closed.
func TestCollectionLockerIsWorkspaceScoped(t *testing.T) {
	h := scopetest.New(t, "treasury_collection")
	repo := NewPostgresCollectionRepository(h.Ops, entityid.TreasuryCollection).(*PostgresCollectionRepository)
	const id = "c25-lock-treasury_collection"
	h.Run(t, func(base, ctxA, ctxB, ctxNone context.Context) {
		if _, err := repo.CreateCollection(ctxA, &collectionpb.CreateCollectionRequest{Data: &collectionpb.Collection{
			Id: id, Name: "Receipt", Amount: 100, Currency: "PHP", Status: "completed", CollectionType: "receipt", Active: true}}); err != nil {
			t.Fatalf("create in own workspace: %v", err)
		}
		got, err := repo.LockCollectionForUpdate(ctxA, id)
		if err != nil || got.GetId() != id {
			t.Fatalf("own lock must succeed: %v %+v", err, got)
		}
		if _, err := repo.LockCollectionForUpdate(ctxB, id); !errors.Is(err, domainports.ErrLockedRowNotFound) {
			t.Errorf("foreign workspace: want ErrLockedRowNotFound, got %v", err)
		}
		if _, err := repo.LockCollectionForUpdate(ctxNone, id); err == nil {
			t.Error("no workspace: the lock must fail closed")
		}
	})
}

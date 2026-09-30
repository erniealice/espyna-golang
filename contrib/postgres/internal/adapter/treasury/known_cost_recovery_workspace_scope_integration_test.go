//go:build postgresql

package treasury

import (
	"context"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core/scopetest"
	"testing"

	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	collectionapplicationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_application"
)

// CollectionApplication: every by-id and list operation carries a trusted, non-empty workspace predicate. A
// foreign workspace id, or a missing workspace, sees "not found"; a caller-supplied workspace_id
// never wins over the trusted one. Needs migration 20260930110000 on the leasing_usage1 clone.
func TestCollectionApplicationWorkspaceScope(t *testing.T) {
	h := scopetest.New(t, "collection_application")
	repo := NewPostgresCollectionApplicationRepository(h.Ops, entityid.CollectionApplication).(*PostgresCollectionApplicationRepository)
	const id = "s1scope-collection_application"
	h.Run(t, func(base, ctxA, ctxB, ctxNone context.Context) {
		if _, err := repo.CreateCollectionApplication(ctxA, &collectionapplicationpb.CreateCollectionApplicationRequest{Data: &collectionapplicationpb.CollectionApplication{
			Id: id, WorkspaceId: scopetest.WsB, // must be ignored in favour of the trusted workspace
			Active: true, TreasuryCollectionId: "tc1", ClientId: "cl1", TargetKind: collectionapplicationpb.ApplicationTargetKind_APPLICATION_TARGET_KIND_REVENUE, RevenueId: scopetest.Str("r1"), ApplicationKind: collectionapplicationpb.ApplicationKind_APPLICATION_KIND_CASH, Amount: 1, Currency: "PHP", Status: collectionapplicationpb.ApplicationStatus_APPLICATION_STATUS_APPLIED, AppliedBy: scopetest.Str("orig")}}); err != nil {
			t.Fatalf("create in own workspace: %v", err)
		}
		got, err := repo.ReadCollectionApplication(ctxA, &collectionapplicationpb.ReadCollectionApplicationRequest{Data: &collectionapplicationpb.CollectionApplication{Id: id}})
		if err != nil || len(got.Data) != 1 || got.Data[0].GetWorkspaceId() != scopetest.WsA {
			t.Fatalf("own read must succeed with the trusted workspace stamped: %v %+v", err, got)
		}
		for name, ctx := range map[string]context.Context{"foreign workspace": ctxB, "no workspace": ctxNone} {
			if _, err := repo.ReadCollectionApplication(ctx, &collectionapplicationpb.ReadCollectionApplicationRequest{Data: &collectionapplicationpb.CollectionApplication{Id: id}}); err == nil {
				t.Errorf("%s: read must fail (not found / fail closed)", name)
			}
			if _, err := repo.GetCollectionApplicationItemPageData(ctx, &collectionapplicationpb.GetCollectionApplicationItemPageDataRequest{CollectionApplicationId: id}); err == nil {
				t.Errorf("%s: item page data must fail", name)
			}
			if _, err := repo.UpdateCollectionApplication(ctx, &collectionapplicationpb.UpdateCollectionApplicationRequest{Data: &collectionapplicationpb.CollectionApplication{Id: id, AppliedBy: scopetest.Str("hijacked")}}); err == nil {
				t.Errorf("%s: update must fail", name)
			}
			if _, err := repo.DeleteCollectionApplication(ctx, &collectionapplicationpb.DeleteCollectionApplicationRequest{Data: &collectionapplicationpb.CollectionApplication{Id: id}}); err == nil {
				t.Errorf("%s: delete must fail", name)
			}
			if l, err := repo.ListCollectionApplications(ctx, &collectionapplicationpb.ListCollectionApplicationsRequest{}); err == nil {
				for _, r := range l.Data {
					if r.GetId() == id {
						t.Errorf("%s: list leaked a foreign row", name)
					}
				}
			}
			if l, err := repo.GetCollectionApplicationListPageData(ctx, &collectionapplicationpb.GetCollectionApplicationListPageDataRequest{}); err == nil {
				for _, r := range l.CollectionApplicationList {
					if r.GetId() == id {
						t.Errorf("%s: list page data leaked a foreign row", name)
					}
				}
			}
		}
		// C6: the row is an immutable financial record - even the owning workspace cannot delete it.
		if _, err := repo.DeleteCollectionApplication(ctxA, &collectionapplicationpb.DeleteCollectionApplicationRequest{Data: &collectionapplicationpb.CollectionApplication{Id: id}}); err == nil {
			t.Error("own-workspace delete must be refused (immutable)")
		} else if !postgresCore.IsImmutableRecord(err) {
			t.Errorf("own-workspace delete must be an immutable_record refusal, got %v", err)
		}
		// The stored row is untouched by the refused writes.
		still, err := repo.ReadCollectionApplication(ctxA, &collectionapplicationpb.ReadCollectionApplicationRequest{Data: &collectionapplicationpb.CollectionApplication{Id: id}})
		if err != nil || len(still.Data) != 1 || still.Data[0].GetAppliedBy() != "orig" {
			t.Errorf("foreign write changed the row: %v %+v", err, still)
		}
		// An OR filter would let the tenant predicate be OR-ed away: refused outright.
		if _, err := repo.ListCollectionApplications(ctxA, &collectionapplicationpb.ListCollectionApplicationsRequest{Filters: &commonpb.FilterRequest{Logic: commonpb.FilterLogic_OR, Filters: []*commonpb.TypedFilter{{Field: "id"}}}}); err == nil {
			t.Error("OR filter logic must be refused")
		}
	})
}

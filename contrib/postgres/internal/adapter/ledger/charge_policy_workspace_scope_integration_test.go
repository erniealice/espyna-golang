//go:build postgresql

package ledger

import (
	"context"
	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core/scopetest"
	"testing"

	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	policypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy"
	versionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version"
	editorpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version_editor"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/enums"
)

// AC-CP-08: every by-id and list operation of the charge policy adapters (incl. the C12 editor
// table) carries a trusted, non-empty workspace predicate. A foreign workspace id, or a missing
// workspace, sees "not found". Runs on the shared scope harness (core/scopetest): one
// rolled-back transaction on the leasing_usage1 clone, nothing persists.
func TestChargePolicyWorkspaceScope(t *testing.T) {
	h := scopetest.New(t, "charge_policy_version_editor")
	ops := h.Ops
	policies := NewPostgresChargePolicyRepository(ops, entityid.ChargePolicy).(*PostgresChargePolicyRepository)
	versions := NewPostgresChargePolicyVersionRepository(ops, entityid.ChargePolicyVersion).(*PostgresChargePolicyVersionRepository)
	editors := NewPostgresChargePolicyVersionEditorRepository(ops, entityid.ChargePolicyVersionEditor).(*PostgresChargePolicyVersionEditorRepository)

	const policyID, versionID, editorID = "w1a-scope-policy", "w1a-scope-version", "w1a-scope-editor"
	wsA, wsB := scopetest.WsA, scopetest.WsB

	h.Run(t, func(txBase, ctxA, ctxB, ctxNone context.Context) {
		if _, err := policies.CreateChargePolicy(ctxA, &policypb.CreateChargePolicyRequest{Data: &policypb.ChargePolicy{
			Id: policyID, Code: "W1A_SCOPE_TEST", Name: "scope test", Active: true,
			Status: enumspb.ChargePolicyStatus_CHARGE_POLICY_STATUS_ACTIVE,
			// a caller-supplied workspace must never win over the trusted one
			WorkspaceId: wsB,
		}}); err != nil {
			t.Fatalf("create in own workspace: %v", err)
		}
		if _, err := versions.CreateChargePolicyVersion(ctxA, &versionpb.CreateChargePolicyVersionRequest{Data: &versionpb.ChargePolicyVersion{
			Id: versionID, ChargePolicyId: policyID, VersionNumber: 1, Active: true,
			Status: enumspb.ChargePolicyVersionStatus_CHARGE_POLICY_VERSION_STATUS_DRAFT,
		}}); err != nil {
			t.Fatalf("create version: %v", err)
		}

		got, err := policies.ReadChargePolicy(ctxA, &policypb.ReadChargePolicyRequest{Data: &policypb.ChargePolicy{Id: policyID}})
		if err != nil || len(got.Data) != 1 || got.Data[0].GetWorkspaceId() != wsA {
			t.Fatalf("own read must succeed with the trusted workspace stamped: %v %+v", err, got)
		}

		for name, ctx := range map[string]context.Context{"foreign workspace": ctxB, "no workspace": ctxNone} {
			if _, err := policies.ReadChargePolicy(ctx, &policypb.ReadChargePolicyRequest{Data: &policypb.ChargePolicy{Id: policyID}}); err == nil {
				t.Errorf("%s: read must fail (not found / fail closed)", name)
			}
			if _, err := policies.GetChargePolicyItemPageData(ctx, &policypb.GetChargePolicyItemPageDataRequest{ChargePolicyId: policyID}); err == nil {
				t.Errorf("%s: item page data must fail", name)
			}
			if _, err := policies.UpdateChargePolicy(ctx, &policypb.UpdateChargePolicyRequest{Data: &policypb.ChargePolicy{Id: policyID, Name: "hijacked"}}); err == nil {
				t.Errorf("%s: update must fail", name)
			}
			if _, err := policies.DeleteChargePolicy(ctx, &policypb.DeleteChargePolicyRequest{Data: &policypb.ChargePolicy{Id: policyID}}); err == nil {
				t.Errorf("%s: delete must fail", name)
			}
			if _, err := versions.ReadChargePolicyVersion(ctx, &versionpb.ReadChargePolicyVersionRequest{Data: &versionpb.ChargePolicyVersion{Id: versionID}}); err == nil {
				t.Errorf("%s: version read must fail", name)
			}
			if l, err := policies.ListChargePolicies(ctx, &policypb.ListChargePoliciesRequest{}); err == nil {
				for _, p := range l.Data {
					if p.GetId() == policyID {
						t.Errorf("%s: list leaked a foreign row", name)
					}
				}
			}
			if l, err := versions.ListChargePolicyVersions(ctx, &versionpb.ListChargePolicyVersionsRequest{
				Filters: &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{{Field: "charge_policy_id", FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{Value: policyID, Operator: commonpb.StringOperator_STRING_EQUALS}}}}},
			}); err == nil && len(l.Data) != 0 {
				t.Errorf("%s: filtered list leaked a foreign row", name)
			}
			if _, err := policies.LockChargePolicyForUpdate(ctx, policyID); err == nil {
				t.Errorf("%s: policy lock must report not found", name)
			}
			if _, err := versions.LockChargePolicyVersionForUpdate(ctx, versionID); err == nil {
				t.Errorf("%s: version lock must report not found", name)
			}
		}
		// The stored row is untouched by the refused writes.
		still, _ := policies.ReadChargePolicy(ctxA, &policypb.ReadChargePolicyRequest{Data: &policypb.ChargePolicy{Id: policyID}})
		if still == nil || len(still.Data) != 1 || still.Data[0].GetName() != "scope test" {
			t.Errorf("foreign write changed the row: %+v", still)
		}

		// An OR filter would let the tenant predicate be OR-ed away: refused outright.
		if _, err := policies.ListChargePolicies(ctxA, &policypb.ListChargePoliciesRequest{Filters: &commonpb.FilterRequest{Logic: commonpb.FilterLogic_OR, Filters: []*commonpb.TypedFilter{{Field: "code"}}}}); err == nil {
			t.Error("OR filter logic must be refused")
		}

		// Own-workspace locks work inside the transaction; reference reader is scoped.
		if p, err := policies.LockChargePolicyForUpdate(ctxA, policyID); err != nil || p.GetId() != policyID {
			t.Errorf("own lock: %v", err)
		}
		if v, err := versions.LockChargePolicyVersionForUpdate(ctxA, versionID); err != nil || v.GetId() != versionID {
			t.Errorf("own version lock: %v", err)
		}
		if _, err := policies.ChargePolicyIDsReferencedByPricePlans(ctxNone, []string{policyID}); err == nil {
			t.Error("reference reader without a workspace must fail closed")
		}
		if m, err := policies.ChargePolicyIDsReferencedByPricePlans(ctxB, []string{policyID}); err != nil || m[policyID] {
			t.Errorf("foreign workspace must see no references: %v %v", m, err)
		}

		// C12 editor rows are workspace-scoped like every other charge policy table.
		if _, err := editors.CreateChargePolicyVersionEditor(ctxA, &editorpb.CreateChargePolicyVersionEditorRequest{Data: &editorpb.ChargePolicyVersionEditor{
			Id: editorID, ChargePolicyVersionId: versionID, UserId: "u-editor", Active: true, WorkspaceId: wsB,
		}}); err != nil {
			t.Fatalf("create editor: %v", err)
		}
		if got, err := editors.ReadChargePolicyVersionEditor(ctxA, &editorpb.ReadChargePolicyVersionEditorRequest{Data: &editorpb.ChargePolicyVersionEditor{Id: editorID}}); err != nil || len(got.Data) != 1 || got.Data[0].GetWorkspaceId() != wsA {
			t.Fatalf("own editor read must stamp the trusted workspace: %v %+v", err, got)
		}
		for name, ctx := range map[string]context.Context{"foreign workspace": ctxB, "no workspace": ctxNone} {
			if _, err := editors.ReadChargePolicyVersionEditor(ctx, &editorpb.ReadChargePolicyVersionEditorRequest{Data: &editorpb.ChargePolicyVersionEditor{Id: editorID}}); err == nil {
				t.Errorf("%s: editor read must fail", name)
			}
			if _, err := editors.DeleteChargePolicyVersionEditor(ctx, &editorpb.DeleteChargePolicyVersionEditorRequest{Data: &editorpb.ChargePolicyVersionEditor{Id: editorID}}); err == nil {
				t.Errorf("%s: editor delete must fail", name)
			}
			if l, err := editors.ListChargePolicyVersionEditors(ctx, &editorpb.ListChargePolicyVersionEditorsRequest{ChargePolicyVersionId: scopetest.Str(versionID)}); err == nil && len(l.Data) != 0 {
				t.Errorf("%s: editor list leaked a foreign row", name)
			}
		}
		// The scope field alone narrows the list (never silently ignored).
		if l, err := editors.ListChargePolicyVersionEditors(ctxA, &editorpb.ListChargePolicyVersionEditorsRequest{ChargePolicyVersionId: scopetest.Str("other-version")}); err != nil || len(l.Data) != 0 {
			t.Errorf("scope field must filter: %v %+v", err, l)
		}
		if l, err := editors.ListChargePolicyVersionEditors(ctxA, &editorpb.ListChargePolicyVersionEditorsRequest{ChargePolicyVersionId: scopetest.Str(versionID)}); err != nil || len(l.Data) != 1 || l.Pagination == nil {
			t.Errorf("own scoped list (with pagination): %v %+v", err, l)
		}
	})

	// Outside a transaction the lock fails closed (SELECT ... FOR UPDATE needs a tx).
	ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: wsA})
	if _, err := policies.LockChargePolicyForUpdate(ctx, policyID); err == nil {
		t.Error("lock outside a transaction must fail closed")
	}
}

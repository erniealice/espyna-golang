package charge_component

import (
	"context"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	chargecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/charge_component"
)

type recordingAuthz struct {
	deny  bool
	asked []string
}

func (a *recordingAuthz) IsEnabled() bool { return true }
func (a *recordingAuthz) HasPermission(_ context.Context, _ string, perm string) (bool, error) {
	a.asked = append(a.asked, perm)
	return !a.deny, nil
}

func newUC(a *recordingAuthz) *UseCases {
	return NewUseCases(Repositories{}, Services{
		Authorizer: ports.NewNoOpAuthorizer(), Translator: ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(a, ports.NewNoOpTranslator()),
	})
}

// The read use cases ask for the billable_charge permissions of build-spec §6.4 and fail closed:
// denied -> error before any repository access; allowed but no repository -> "unavailable".
func TestReadUseCasesGateAndFailClosed(t *testing.T) {
	ctx := contextutil.WithUserID(context.Background(), "u1")
	denied := &recordingAuthz{deny: true}
	uc := newUC(denied)
	if _, err := uc.ListChargeComponents.Execute(ctx, &chargecomponentpb.ListChargeComponentsRequest{}); err == nil {
		t.Error("list must be denied")
	}
	if _, err := uc.GetChargeComponentListPageData.Execute(ctx, &chargecomponentpb.GetChargeComponentListPageDataRequest{}); err == nil {
		t.Error("list page data must be denied")
	}
	if _, err := uc.ReadChargeComponent.Execute(ctx, &chargecomponentpb.ReadChargeComponentRequest{Data: &chargecomponentpb.ChargeComponent{Id: "x"}}); err == nil {
		t.Error("read must be denied")
	}
	want := map[string]bool{"billable_charge:list": false, "billable_charge:read": false}
	for _, p := range denied.asked {
		if _, ok := want[p]; !ok {
			t.Errorf("unexpected permission asked: %s", p)
		}
		want[p] = true
	}
	for p, seen := range want {
		if !seen {
			t.Errorf("permission %s was never checked", p)
		}
	}

	allowed := newUC(&recordingAuthz{})
	if _, err := allowed.ListChargeComponents.Execute(ctx, nil); err == nil || !strings.Contains(err.Error(), "repository unavailable") {
		t.Errorf("list without repository must fail closed, got %v", err)
	}
	if _, err := allowed.ReadChargeComponent.Execute(ctx, &chargecomponentpb.ReadChargeComponentRequest{}); err == nil {
		t.Error("read without id must fail")
	}
}

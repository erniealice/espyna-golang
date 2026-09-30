package recovery_document_line

import (
	"context"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	recoverydocumentlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document_line"
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

// The read use cases ask for the recovery_document permissions of build-spec §6.4 and fail closed:
// denied -> error before any repository access; allowed but no repository -> "unavailable".
func TestReadUseCasesGateAndFailClosed(t *testing.T) {
	ctx := contextutil.WithUserID(context.Background(), "u1")
	denied := &recordingAuthz{deny: true}
	uc := newUC(denied)
	if _, err := uc.ListRecoveryDocumentLines.Execute(ctx, &recoverydocumentlinepb.ListRecoveryDocumentLinesRequest{}); err == nil {
		t.Error("list must be denied")
	}
	if _, err := uc.GetRecoveryDocumentLineListPageData.Execute(ctx, &recoverydocumentlinepb.GetRecoveryDocumentLineListPageDataRequest{}); err == nil {
		t.Error("list page data must be denied")
	}
	if _, err := uc.ReadRecoveryDocumentLine.Execute(ctx, &recoverydocumentlinepb.ReadRecoveryDocumentLineRequest{Data: &recoverydocumentlinepb.RecoveryDocumentLine{Id: "x"}}); err == nil {
		t.Error("read must be denied")
	}
	want := map[string]bool{"recovery_document:list": false, "recovery_document:read": false}
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
	if _, err := allowed.ListRecoveryDocumentLines.Execute(ctx, nil); err == nil || !strings.Contains(err.Error(), "repository unavailable") {
		t.Errorf("list without repository must fail closed, got %v", err)
	}
	if _, err := allowed.ReadRecoveryDocumentLine.Execute(ctx, &recoverydocumentlinepb.ReadRecoveryDocumentLineRequest{}); err == nil {
		t.Error("read without id must fail")
	}
}

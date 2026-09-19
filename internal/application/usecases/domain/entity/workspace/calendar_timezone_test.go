package workspace

import (
	"context"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
	"testing"
)

type calendarRepo struct {
	workspacepb.UnimplementedWorkspaceDomainServiceServer
	requested string
	returned  string
	zone      string
}

func (r *calendarRepo) ReadWorkspace(_ context.Context, req *workspacepb.ReadWorkspaceRequest) (*workspacepb.ReadWorkspaceResponse, error) {
	r.requested = req.GetData().GetId()
	return &workspacepb.ReadWorkspaceResponse{Data: []*workspacepb.Workspace{{Id: r.returned, Timezone: &r.zone}}}, nil
}
func TestCalendarTimezoneUsesOnlyContextWorkspace(t *testing.T) {
	repo := &calendarRepo{returned: "ws-1", zone: "Asia/Manila"}
	uc := NewReadWorkspaceUseCase(ReadWorkspaceRepositories{Workspace: repo}, ReadWorkspaceServices{})
	ctx := contextutil.WithWorkspaceID(context.Background(), "ws-1")
	zone, err := uc.CalendarTimezone(ctx)
	if err != nil || zone != "Asia/Manila" || repo.requested != "ws-1" {
		t.Fatalf("zone=%s err=%v requested=%s", zone, err, repo.requested)
	}
	repo.returned = "other"
	if _, err = uc.CalendarTimezone(ctx); err == nil {
		t.Fatal("wrong workspace accepted")
	}
	repo.returned = "ws-1"
	repo.zone = "Invalid/Zone"
	if _, err = uc.CalendarTimezone(ctx); err == nil {
		t.Fatal("invalid timezone accepted")
	}
	repo.requested = ""
	if zone, err = uc.CalendarTimezone(context.Background()); err != nil || zone != "" || repo.requested != "" {
		t.Fatal("unscoped request read workspace")
	}
}

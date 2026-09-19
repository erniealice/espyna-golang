package plan

import (
	"context"
	"errors"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
	"google.golang.org/protobuf/types/known/timestamppb"
	"testing"
	"time"
)

type scheduleWorkspaceRepo struct {
	workspacepb.UnimplementedWorkspaceDomainServiceServer
	zone string
	err  error
}

func (r scheduleWorkspaceRepo) ReadWorkspace(_ context.Context, req *workspacepb.ReadWorkspaceRequest) (*workspacepb.ReadWorkspaceResponse, error) {
	if r.err != nil {
		return nil, r.err
	}
	return &workspacepb.ReadWorkspaceResponse{Data: []*workspacepb.Workspace{{Id: req.GetData().GetId(), Timezone: &r.zone}}}, nil
}
func TestClientScheduleDayStart(t *testing.T) {
	for _, tc := range []struct{ zone, now, want string }{
		{"Asia/Manila", "2026-09-16T14:56:17Z", "2026-09-15T16:00:00Z"},
		{"America/New_York", "2026-03-08T18:00:00Z", "2026-03-08T05:00:00Z"},
		{"America/New_York", "2026-11-01T18:00:00Z", "2026-11-01T04:00:00Z"},
	} {
		t.Run(tc.zone+tc.now, func(t *testing.T) {
			loc, err := time.LoadLocation(tc.zone)
			if err != nil {
				t.Fatal(err)
			}
			now, err := time.Parse(time.RFC3339, tc.now)
			if err != nil {
				t.Fatal(err)
			}
			if got := clientScheduleDayStart(now, loc).Format(time.RFC3339); got != tc.want {
				t.Fatalf("got %s want %s", got, tc.want)
			}
		})
	}
}
func TestClientScheduleDefaultStartMidnight(t *testing.T) {
	repo := newCalendarScheduleRepo()
	deps := &ResolveOrCreateClientScheduleRepos{PriceSchedule: repo, Workspace: scheduleWorkspaceRepo{zone: "Asia/Manila"}}
	got, reused, err := ResolveOrCreateClientPriceSchedule(context.Background(), deps, nil, "workspace-1", "", "client-1", "Card", nil)
	if err != nil {
		t.Fatal(err)
	}
	if reused {
		t.Fatal("unexpected reuse")
	}
	loc, _ := time.LoadLocation("Asia/Manila")
	created := time.UnixMilli(got.GetDateCreated())
	if want := clientScheduleDayStart(created, loc); !got.GetDateTimeStart().AsTime().Equal(want) {
		t.Fatalf("start %v want %v", got.GetDateTimeStart(), want)
	}
	if time.Since(created) > time.Minute {
		t.Fatal("creation timestamp was changed")
	}
	start := timestamppb.New(time.Date(2026, 9, 16, 14, 56, 17, 0, time.UTC))
	template := &priceschedulepb.PriceSchedule{DateTimeStart: start}
	got, _, err = ResolveOrCreateClientPriceSchedule(context.Background(), deps, nil, "workspace-1", "", "client-1", "Card", template)
	if err != nil || !got.GetDateTimeStart().AsTime().Equal(start.AsTime()) {
		t.Fatalf("explicit start changed: %v %v", got, err)
	}
	repo.listResult = []*priceschedulepb.PriceSchedule{got}
	count := len(repo.createCalls)
	reusedRow, reused, err := ResolveOrCreateClientPriceSchedule(context.Background(), deps, nil, "workspace-1", "", "client-1", "Card", nil)
	if err != nil || !reused || reusedRow != got || len(repo.createCalls) != count {
		t.Fatal("existing card changed")
	}
}
func TestClientScheduleTimezoneFailureDoesNotCreate(t *testing.T) {
	for _, zone := range []string{"Invalid/Zone", "read-error"} {
		t.Run(zone, func(t *testing.T) {
			repo := newCalendarScheduleRepo()
			ws := scheduleWorkspaceRepo{zone: zone}
			if zone == "read-error" {
				ws.err = errors.New("unavailable")
			}
			_, _, err := ResolveOrCreateClientPriceSchedule(context.Background(), &ResolveOrCreateClientScheduleRepos{PriceSchedule: repo, Workspace: ws}, nil, "workspace-1", "", "client-1", "Card", nil)
			if err == nil || len(repo.createCalls) != 0 {
				t.Fatal("created with unresolved workspace timezone")
			}
		})
	}
}

type calendarScheduleRepo struct {
	priceschedulepb.UnimplementedPriceScheduleDomainServiceServer
	listResult  []*priceschedulepb.PriceSchedule
	createCalls []*priceschedulepb.PriceSchedule
}

func newCalendarScheduleRepo() *calendarScheduleRepo { return &calendarScheduleRepo{} }
func (r *calendarScheduleRepo) ListPriceSchedules(context.Context, *priceschedulepb.ListPriceSchedulesRequest) (*priceschedulepb.ListPriceSchedulesResponse, error) {
	return &priceschedulepb.ListPriceSchedulesResponse{Data: r.listResult}, nil
}
func (r *calendarScheduleRepo) CreatePriceSchedule(_ context.Context, req *priceschedulepb.CreatePriceScheduleRequest) (*priceschedulepb.CreatePriceScheduleResponse, error) {
	r.createCalls = append(r.createCalls, req.Data)
	return &priceschedulepb.CreatePriceScheduleResponse{Data: []*priceschedulepb.PriceSchedule{req.Data}}, nil
}

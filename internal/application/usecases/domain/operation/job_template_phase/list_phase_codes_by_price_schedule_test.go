package job_template_phase

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/shared/identity"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
)

type phaseCodeReadProbe struct {
	pb.UnimplementedJobTemplatePhaseDomainServiceServer
	called bool
}

func (p *phaseCodeReadProbe) ListPhaseCodesByPriceSchedule(context.Context, *pb.ListPhaseCodesByPriceScheduleRequest) (*pb.ListPhaseCodesByPriceScheduleResponse, error) {
	p.called = true
	return &pb.ListPhaseCodesByPriceScheduleResponse{Success: true}, nil
}

func TestListPhaseCodesByPriceScheduleRejectsMissingScopeAndID(t *testing.T) {
	probe := &phaseCodeReadProbe{}
	uc := NewListPhaseCodesByPriceScheduleUseCase(probe, nil)
	if _, err := uc.Execute(context.Background(), &pb.ListPhaseCodesByPriceScheduleRequest{PriceScheduleId: "schedule"}); err == nil {
		t.Fatal("missing workspace must fail closed")
	}
	ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: "workspace"})
	if _, err := uc.Execute(ctx, &pb.ListPhaseCodesByPriceScheduleRequest{}); err == nil {
		t.Fatal("empty schedule ID must fail closed")
	}
	if _, err := uc.Execute(ctx, &pb.ListPhaseCodesByPriceScheduleRequest{PriceScheduleId: "schedule"}); err == nil {
		t.Fatal("missing action gate must fail closed")
	}
	if probe.called {
		t.Fatal("repository called without complete authorization and scope")
	}
}

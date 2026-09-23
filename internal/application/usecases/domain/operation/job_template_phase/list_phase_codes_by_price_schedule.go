package job_template_phase

import (
	"context"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
)

// ListPhaseCodesByPriceScheduleUseCase lists the phase vocabulary actually used
// by subscription jobs in one academic year. The adapter receives the trusted
// workspace identity through ctx; the request cannot select a workspace.
type ListPhaseCodesByPriceScheduleUseCase struct {
	repository pb.JobTemplatePhaseDomainServiceServer
	gate       *actiongate.ActionGatekeeper
}

func NewListPhaseCodesByPriceScheduleUseCase(repository pb.JobTemplatePhaseDomainServiceServer, gate *actiongate.ActionGatekeeper) *ListPhaseCodesByPriceScheduleUseCase {
	return &ListPhaseCodesByPriceScheduleUseCase{repository: repository, gate: gate}
}

func (uc *ListPhaseCodesByPriceScheduleUseCase) Execute(ctx context.Context, req *pb.ListPhaseCodesByPriceScheduleRequest) (*pb.ListPhaseCodesByPriceScheduleResponse, error) {
	if req == nil || strings.TrimSpace(req.GetPriceScheduleId()) == "" {
		return nil, fmt.Errorf("price schedule ID is required")
	}
	if _, err := identity.RequireWorkspace(ctx); err != nil {
		return nil, err
	}
	if uc == nil || uc.gate == nil || uc.repository == nil {
		return nil, fmt.Errorf("phase code reader is unavailable")
	}
	if err := uc.gate.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.JobTemplatePhase, Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}
	return uc.repository.ListPhaseCodesByPriceSchedule(ctx, req)
}

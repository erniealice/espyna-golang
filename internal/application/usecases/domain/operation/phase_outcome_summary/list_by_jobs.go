package phase_outcome_summary

import (
	"context"
	"fmt"

	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/phase_outcome_summary"
)

type ListByJobsUseCase struct {
	repository domainports.ReportCardPhaseSummaries
	gate       *actiongate.ActionGatekeeper
}

func NewListByJobsUseCase(repository pb.PhaseOutcomeSummaryDomainServiceServer, gate *actiongate.ActionGatekeeper) *ListByJobsUseCase {
	bulk, ok := repository.(domainports.ReportCardPhaseSummaries)
	if !ok {
		return nil
	}
	return &ListByJobsUseCase{repository: bulk, gate: gate}
}

func (uc *ListByJobsUseCase) Execute(ctx context.Context, jobIDs []string) ([]*pb.PhaseOutcomeSummary, error) {
	if uc == nil || uc.gate == nil || uc.repository == nil {
		return nil, fmt.Errorf("bulk phase summary reader is unavailable")
	}
	if _, err := identity.RequireWorkspace(ctx); err != nil {
		return nil, err
	}
	if err := uc.gate.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.PhaseOutcomeSummary, Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	return uc.repository.ListByJobs(ctx, jobIDs)
}

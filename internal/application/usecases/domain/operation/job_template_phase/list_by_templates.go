package job_template_phase

import (
	"context"
	"fmt"

	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
)

type ListByTemplatesUseCase struct {
	repository domainports.ReportCardTemplatePhases
	gate       *actiongate.ActionGatekeeper
}

func NewListByTemplatesUseCase(repository pb.JobTemplatePhaseDomainServiceServer, gate *actiongate.ActionGatekeeper) *ListByTemplatesUseCase {
	bulk, ok := repository.(domainports.ReportCardTemplatePhases)
	if !ok {
		return nil
	}
	return &ListByTemplatesUseCase{repository: bulk, gate: gate}
}

func (uc *ListByTemplatesUseCase) Execute(ctx context.Context, templateIDs []string) ([]*pb.JobTemplatePhase, error) {
	if uc == nil || uc.gate == nil || uc.repository == nil {
		return nil, fmt.Errorf("bulk template phase reader is unavailable")
	}
	if _, err := identity.RequireWorkspace(ctx); err != nil {
		return nil, err
	}
	if err := uc.gate.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.JobTemplatePhase, Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	return uc.repository.ListByTemplates(ctx, templateIDs)
}

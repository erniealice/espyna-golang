package job_outcome_summary_document_template

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary_document_template"
)

type PublishUseCase struct {
	repo pb.JobOutcomeSummaryDocumentTemplateDomainServiceServer
	svc  Services
}

func (uc *PublishUseCase) Execute(ctx context.Context, req *pb.PublishJobOutcomeSummaryDocumentTemplateRequest) (*pb.PublishJobOutcomeSummaryDocumentTemplateResponse, error) {
	// Publish is a controlled state transition, gated on the binding's update
	// permission (there is no separate publish verb in the catalog).
	if err := uc.svc.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.JobOutcomeSummaryDocumentTemplate, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if err := uc.svc.requireRequest(ctx, req != nil && req.Id != ""); err != nil {
		return nil, err
	}
	return uc.repo.PublishJobOutcomeSummaryDocumentTemplate(ctx, req)
}

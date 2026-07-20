package job_template_document_template

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_document_template"
)

type FindApplicableUseCase struct {
	repo pb.JobTemplateDocumentTemplateDomainServiceServer
	svc  Services
}

func (uc *FindApplicableUseCase) Execute(ctx context.Context, req *pb.FindApplicableJobTemplateDocumentTemplateRequest) (*pb.FindApplicableJobTemplateDocumentTemplateResponse, error) {
	if err := uc.svc.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.JobTemplateDocumentTemplate, Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	if err := uc.svc.requireRequest(ctx, req != nil); err != nil {
		return nil, err
	}
	return uc.repo.FindApplicableJobTemplateDocumentTemplate(ctx, req)
}

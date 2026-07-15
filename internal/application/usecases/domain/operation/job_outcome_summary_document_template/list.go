package job_outcome_summary_document_template

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary_document_template"
)

type ListUseCase struct {
	repo pb.JobOutcomeSummaryDocumentTemplateDomainServiceServer
	svc  Services
}

func (uc *ListUseCase) Execute(ctx context.Context, req *pb.ListJobOutcomeSummaryDocumentTemplatesRequest) (*pb.ListJobOutcomeSummaryDocumentTemplatesResponse, error) {
	if err := uc.svc.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.JobOutcomeSummaryDocumentTemplate, Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	if err := uc.svc.requireRequest(ctx, req != nil); err != nil {
		return nil, err
	}
	return uc.repo.ListJobOutcomeSummaryDocumentTemplates(ctx, req)
}

package job_outcome_summary_document_template

import (
	"context"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	enums "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary_document_template"
)

type UpdateUseCase struct {
	repo pb.JobOutcomeSummaryDocumentTemplateDomainServiceServer
	svc  Services
}

func (uc *UpdateUseCase) Execute(ctx context.Context, req *pb.UpdateJobOutcomeSummaryDocumentTemplateRequest) (*pb.UpdateJobOutcomeSummaryDocumentTemplateResponse, error) {
	if err := uc.svc.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.JobOutcomeSummaryDocumentTemplate, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if err := uc.svc.requireRequest(ctx, req != nil && req.Data != nil); err != nil {
		return nil, err
	}
	// workspace_id is the immutable tenant anchor; never honor a client-supplied
	// workspace on update (gate H1).
	req.Data.WorkspaceId = ""
	// version_status is server-owned: the Publish transaction is the ONLY path
	// that changes it. Clear it so a plain Update can never promote/demote a
	// binding (RA2 P1); the adapter also strips it and freezes published rows.
	req.Data.VersionStatus = enums.VersionStatus_VERSION_STATUS_UNSPECIFIED
	ms := time.Now().UnixMilli()
	req.Data.DateModified = &ms
	return uc.repo.UpdateJobOutcomeSummaryDocumentTemplate(ctx, req)
}

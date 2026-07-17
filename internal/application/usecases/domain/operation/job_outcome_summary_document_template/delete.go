package job_outcome_summary_document_template

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	enums "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary_document_template"
)

type DeleteUseCase struct {
	repo pb.JobOutcomeSummaryDocumentTemplateDomainServiceServer
	svc  Services
}

func (uc *DeleteUseCase) Execute(ctx context.Context, req *pb.DeleteJobOutcomeSummaryDocumentTemplateRequest) (*pb.DeleteJobOutcomeSummaryDocumentTemplateResponse, error) {
	if err := uc.svc.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.JobOutcomeSummaryDocumentTemplate, Action: entityid.ActionDelete}); err != nil {
		return nil, err
	}
	if err := uc.svc.requireRequest(ctx, req != nil && req.Data != nil && req.Data.Id != ""); err != nil {
		return nil, err
	}
	// Server-owned lifecycle: only a DRAFT binding may be deleted. A published or
	// deprecated binding is part of the immutable version history and must remain
	// resolvable for historical as_of renders; deletion is refused fail-closed,
	// mirroring the publish transaction's draft-only gate. The read is
	// workspace-scoped by the adapter (a cross-tenant id resolves nothing), and the
	// persistence layer re-checks this predicate with an affected-row guard.
	existing, err := uc.repo.ReadJobOutcomeSummaryDocumentTemplate(ctx, &pb.ReadJobOutcomeSummaryDocumentTemplateRequest{
		Data: &pb.JobOutcomeSummaryDocumentTemplate{Id: req.Data.Id},
	})
	if err != nil {
		return nil, err
	}
	rows := existing.GetData()
	if len(rows) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.svc.Translator,
			"job_outcome_summary_document_template.validation.not_found", "Binding not found [DEFAULT]"))
	}
	if rows[0].GetVersionStatus() != enums.VersionStatus_VERSION_STATUS_DRAFT {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.svc.Translator,
			"job_outcome_summary_document_template.validation.delete_not_draft", "Only a draft binding can be deleted [DEFAULT]"))
	}
	return uc.repo.DeleteJobOutcomeSummaryDocumentTemplate(ctx, req)
}

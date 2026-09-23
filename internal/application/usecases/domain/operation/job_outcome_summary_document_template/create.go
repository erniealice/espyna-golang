package job_outcome_summary_document_template

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	enums "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary_document_template"
	phasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var bindingPhaseCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func validBindingPhaseCode(code *string) bool {
	return code == nil || bindingPhaseCodePattern.MatchString(*code)
}

type CreateUseCase struct {
	repo       pb.JobOutcomeSummaryDocumentTemplateDomainServiceServer
	phaseCodes PhaseCodeReader
	svc        Services
}

func (uc *CreateUseCase) validSchedulePhaseCode(ctx context.Context, scheduleID, code string) error {
	if uc.phaseCodes == nil {
		return status.Error(codes.Internal, "period scope validation unavailable")
	}
	response, err := uc.phaseCodes.ListPhaseCodesByPriceSchedule(ctx, &phasepb.ListPhaseCodesByPriceScheduleRequest{PriceScheduleId: scheduleID})
	if err != nil {
		return status.Error(codes.Internal, "period scope validation unavailable")
	}
	if response == nil || !response.GetSuccess() {
		return status.Error(codes.Internal, "period scope validation unavailable")
	}
	for _, option := range response.GetOptions() {
		if option.GetCode() == code {
			return nil
		}
	}
	return status.Error(codes.InvalidArgument, "invalid period scope")
}

func (uc *CreateUseCase) Execute(ctx context.Context, req *pb.CreateJobOutcomeSummaryDocumentTemplateRequest) (*pb.CreateJobOutcomeSummaryDocumentTemplateResponse, error) {
	if err := uc.svc.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.JobOutcomeSummaryDocumentTemplate, Action: entityid.ActionCreate}); err != nil {
		return nil, err
	}
	if err := uc.svc.requireRequest(ctx, req != nil && req.Data != nil); err != nil {
		return nil, err
	}
	// S-6 — minimal required-field validation: a binding must reference the
	// document_template artifact it selects.
	if req.Data.DocumentTemplateId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.svc.Translator,
			"job_outcome_summary_document_template.validation.document_template_required",
			"A document template is required [DEFAULT]"))
	}
	if !validBindingPhaseCode(req.Data.JobTemplatePhaseCode) ||
		(req.Data.JobTemplatePhaseCode != nil && req.Data.GetPriceScheduleId() == "") {
		return nil, status.Error(codes.InvalidArgument, "invalid period scope")
	}
	if req.Data.JobTemplatePhaseCode != nil {
		if err := uc.validSchedulePhaseCode(ctx, req.Data.GetPriceScheduleId(), req.Data.GetJobTemplatePhaseCode()); err != nil {
			return nil, err
		}
	}
	// Tenancy is assigned from the trusted request context by the persistence
	// layer; never honor a client-supplied workspace on write (gate H1).
	req.Data.WorkspaceId = ""
	// RA2 P1 — server-owned lifecycle. A binding is ALWAYS born DRAFT and
	// unversioned/provisional (version=0); the publish audit is stamped only by
	// the Publish transaction. Never honor a client-supplied version_status,
	// version, or publish audit.
	req.Data.VersionStatus = enums.VersionStatus_VERSION_STATUS_DRAFT
	req.Data.Version = 0
	req.Data.PublishedAt = nil
	req.Data.PublishedBy = nil
	now := time.Now()
	if req.Data.Id == "" && uc.svc.IDGenerator != nil {
		req.Data.Id = uc.svc.IDGenerator.GenerateID()
	}
	req.Data.Active = true
	ms := now.UnixMilli()
	req.Data.DateCreated = &ms
	req.Data.DateModified = &ms
	return uc.repo.CreateJobOutcomeSummaryDocumentTemplate(ctx, req)
}

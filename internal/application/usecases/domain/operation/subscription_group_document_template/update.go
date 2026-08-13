package subscription_group_document_template

import (
	"context"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	enums "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/subscription_group_document_template"
)

type UpdateUseCase struct {
	repo pb.SubscriptionGroupDocumentTemplateDomainServiceServer
	svc  Services
}

func (uc *UpdateUseCase) Execute(ctx context.Context, req *pb.UpdateSubscriptionGroupDocumentTemplateRequest) (*pb.UpdateSubscriptionGroupDocumentTemplateResponse, error) {
	if err := uc.svc.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.SubscriptionGroupDocumentTemplate, Action: entityid.ActionUpdate}); err != nil {
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
	// version is server-owned too — only Publish allocates it (MAX(published)+1).
	// Clear it so a plain Update can never clobber a published lineage's version,
	// including via a publish/Update TOCTOU race (the frozen-field filter reads an
	// unlocked snapshot, so version must not depend on that read). The adapter
	// also strips it. (B4 codex finding #3 — lifecycle/version immutable to Update.)
	req.Data.Version = 0
	// Q3 (attendance-v2 follow-up): scope fields are immutable on the generic
	// Update route — the adapter unconditionally filters the write payload down
	// to {active, date_modified}, closing the scope-field TOCTOU where a Publish
	// interleaving with an Update could mutate a just-published row. Clear them
	// here too (belt + suspenders) so no scope mutation ever leaves the use case.
	// Scope changes go through delete-draft + re-create.
	req.Data.DocumentTemplateId = ""
	req.Data.PriceScheduleId = nil
	req.Data.PlanId = nil
	req.Data.JobCategoryId = nil
	req.Data.ValidityStart = nil
	req.Data.ValidityEnd = nil
	req.Data.SupersedesBindingId = nil
	// Publish audit is server-owned (only the Publish transaction stamps it).
	req.Data.PublishedAt = nil
	req.Data.PublishedBy = nil
	ms := time.Now().UnixMilli()
	req.Data.DateModified = &ms
	return uc.repo.UpdateSubscriptionGroupDocumentTemplate(ctx, req)
}

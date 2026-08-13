package subscription_group_document_template

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	documenttemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/document/template"
	enums "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/subscription_group_document_template"
)

type CreateUploadPairRepositories struct {
	DocumentTemplate                  documenttemplatepb.DocumentTemplateDomainServiceServer
	SubscriptionGroupDocumentTemplate pb.SubscriptionGroupDocumentTemplateDomainServiceServer
}

type CreateUploadPairRequest struct {
	Artifact *documenttemplatepb.DocumentTemplate
	Binding  *pb.SubscriptionGroupDocumentTemplate
}

type CreateUploadPairResponse struct {
	Artifact *documenttemplatepb.DocumentTemplate
	Binding  *pb.SubscriptionGroupDocumentTemplate
}

type CreateUploadPairUseCase struct {
	repositories CreateUploadPairRepositories
	services     Services
}

func NewCreateUploadPairUseCase(
	repositories CreateUploadPairRepositories,
	services Services,
) *CreateUploadPairUseCase {
	return &CreateUploadPairUseCase{
		repositories: repositories,
		services:     services,
	}
}

func (uc *CreateUploadPairUseCase) Execute(ctx context.Context, req *CreateUploadPairRequest) (*CreateUploadPairResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.DocumentTemplate,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.SubscriptionGroupDocumentTemplate,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}
	if err := uc.services.requireRequest(ctx, req != nil && req.Artifact != nil && req.Binding != nil); err != nil {
		return nil, err
	}

	artifact := req.Artifact
	binding := req.Binding
	if err := uc.validateAndNormalize(ctx, artifact, binding); err != nil {
		return nil, err
	}

	if uc.services.Transactor == nil || !uc.services.Transactor.SupportsTransactions() {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_document_template.errors.transaction_unavailable", "Transaction support is required to upload an artifact-binding pair [DEFAULT]"))
	}

	var (
		artifactOut *documenttemplatepb.DocumentTemplate
		bindingOut  *pb.SubscriptionGroupDocumentTemplate
	)

	if err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		artifactResp, err := uc.repositories.DocumentTemplate.CreateDocumentTemplate(txCtx, &documenttemplatepb.CreateDocumentTemplateRequest{
			Data: artifact,
		})
		if err != nil {
			return err
		}
		if artifactResp == nil || !artifactResp.GetSuccess() {
			return errors.New(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "subscription_group_document_template.errors.create_upload_pair_artifact_not_created", "Document template was not created [DEFAULT]"))
		}
		if len(artifactResp.GetData()) != 1 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "subscription_group_document_template.errors.create_upload_pair_artifact_count", "Unexpected document template create response [DEFAULT]"))
		}
		artifactOut = artifactResp.GetData()[0]
		if artifactOut.GetId() != artifact.GetId() {
			return errors.New(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "subscription_group_document_template.errors.create_upload_pair_artifact_id_mismatch", "Artifact ID mismatch [DEFAULT]"))
		}

		bindingResp, err := uc.repositories.SubscriptionGroupDocumentTemplate.CreateSubscriptionGroupDocumentTemplate(txCtx, &pb.CreateSubscriptionGroupDocumentTemplateRequest{
			Data: binding,
		})
		if err != nil {
			return err
		}
		if bindingResp == nil || !bindingResp.GetSuccess() {
			return errors.New(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "subscription_group_document_template.errors.create_upload_pair_binding_not_created", "Subscription group document template binding was not created [DEFAULT]"))
		}
		if len(bindingResp.GetData()) != 1 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "subscription_group_document_template.errors.create_upload_pair_binding_count", "Unexpected binding create response [DEFAULT]"))
		}
		bindingOut = bindingResp.GetData()[0]
		if bindingOut.GetId() != binding.GetId() {
			return errors.New(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "subscription_group_document_template.errors.create_upload_pair_binding_id_mismatch", "Binding ID mismatch [DEFAULT]"))
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return &CreateUploadPairResponse{
		Artifact: artifactOut,
		Binding:  bindingOut,
	}, nil
}

// ExecutePair exposes the atomic upload-pair operation through public protobuf
// types only. Consumer packages can obtain this use case through the aggregate,
// but Go's internal-package boundary intentionally prevents them from naming
// CreateUploadPairRequest/Response. Keeping the transport-neutral pair method
// here preserves the same authorization, validation, and transaction boundary
// without leaking an internal application DTO into Fayna.
func (uc *CreateUploadPairUseCase) ExecutePair(
	ctx context.Context,
	artifact *documenttemplatepb.DocumentTemplate,
	binding *pb.SubscriptionGroupDocumentTemplate,
) (*documenttemplatepb.DocumentTemplate, *pb.SubscriptionGroupDocumentTemplate, error) {
	resp, err := uc.Execute(ctx, &CreateUploadPairRequest{Artifact: artifact, Binding: binding})
	if err != nil {
		return nil, nil, err
	}
	if resp == nil {
		return nil, nil, errors.New("subscription group document template upload pair returned no response")
	}
	return resp.Artifact, resp.Binding, nil
}

func (uc *CreateUploadPairUseCase) validateAndNormalize(ctx context.Context, artifact *documenttemplatepb.DocumentTemplate, binding *pb.SubscriptionGroupDocumentTemplate) error {
	artifactID := strings.TrimSpace(artifact.GetId())
	if artifactID == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_document_template.validation.artifact_id_required", "artifact id is required [DEFAULT]"))
	}
	bindingID := strings.TrimSpace(binding.GetId())
	if bindingID == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_document_template.validation.binding_id_required", "binding id is required [DEFAULT]"))
	}

	if binding.GetDocumentTemplateId() != artifactID {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_document_template.validation.document_template_mismatch", "binding document_template_id must match artifact id [DEFAULT]"))
	}
	if !isSupportedRenderProfile(binding.GetRenderProfile()) {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_document_template.validation.invalid_render_profile", "unsupported render profile [DEFAULT]"))
	}

	jobCategoryID := strings.TrimSpace(binding.GetJobCategoryId())
	if jobCategoryID == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_document_template.validation.job_category_required", "job_category_id is required for this render profile [DEFAULT]"))
	}

	templateType := strings.TrimSpace(artifact.GetTemplateType())
	if templateType != "docx" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_document_template.validation.template_type_invalid", "template_type must be docx [DEFAULT]"))
	}
	artifact.TemplateType = templateType

	documentPurpose := strings.TrimSpace(artifact.GetDocumentPurpose())
	if documentPurpose != "subscription_group_outcome_summary" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_document_template.validation.document_purpose_invalid", "document_purpose must be subscription_group_outcome_summary [DEFAULT]"))
	}
	artifact.DocumentPurpose = documentPurpose

	if strings.TrimSpace(artifact.GetStatus()) != "active" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_document_template.validation.status_invalid", "status must be active [DEFAULT]"))
	}
	if !artifact.GetActive() {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_document_template.validation.artifact_inactive", "artifact must be active [DEFAULT]"))
	}

	storageContainer := strings.TrimSpace(artifact.GetStorageContainer())
	if storageContainer == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_document_template.validation.storage_container_required", "storage_container is required [DEFAULT]"))
	}
	artifact.StorageContainer = &storageContainer

	storageKey := strings.TrimSpace(artifact.GetStorageKey())
	if storageKey == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_document_template.validation.storage_key_required", "storage_key is required [DEFAULT]"))
	}
	artifact.StorageKey = &storageKey

	artifact.Id = artifactID
	binding.Id = bindingID
	binding.DocumentTemplateId = artifactID

	binding.JobCategory = nil
	binding.DocumentTemplate = nil
	binding.PriceSchedule = nil
	binding.Plan = nil
	binding.JobCategoryId = &jobCategoryID
	binding.VersionStatus = enums.VersionStatus_VERSION_STATUS_DRAFT
	binding.Version = 0
	binding.SupersedesBindingId = nil
	binding.Active = true
	binding.CreatedBy = nil
	binding.PublishedAt = nil
	binding.PublishedAtString = nil
	binding.PublishedBy = nil

	artifact.WorkspaceId = nil
	artifact.CreatedBy = nil
	binding.WorkspaceId = ""

	now := time.Now()
	nowMs := now.UnixMilli()
	nowStr := now.Format(time.RFC3339)
	artifact.DateCreated = &nowMs
	artifact.DateModified = &nowMs
	artifact.DateCreatedString = &nowStr
	artifact.DateModifiedString = &nowStr

	binding.DateCreated = &nowMs
	binding.DateModified = &nowMs
	binding.DateCreatedString = &nowStr
	binding.DateModifiedString = &nowStr

	return nil
}

func isSupportedRenderProfile(profile pb.RenderProfile) bool {
	switch profile {
	case pb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1:
		return true
	default:
		return false
	}
}

package subscription_group_document_template

import (
	"context"
	"errors"
	"fmt"
	"strings"

	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	documenttemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/document/template"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/subscription_group_document_template"
)

// DeleteDraftPairRequest is the transport-neutral input for the atomic
// binding-plus-artifact delete operation.
type DeleteDraftPairRequest struct {
	BindingID string
}

// DeleteDraftPairResponse carries the artifact locator only after the database
// transaction has committed successfully.
type DeleteDraftPairResponse struct {
	Artifact *documenttemplatepb.DocumentTemplate
}

// DeleteDraftPairUseCase owns the dual-delete authorization and transaction
// boundary. Object-storage deletion deliberately remains a post-commit caller
// responsibility.
type DeleteDraftPairUseCase struct {
	repo            domainports.SubscriptionGroupDocumentTemplateDraftPairDeleter
	repositoryError error
	svc             Services
}

func NewDeleteDraftPairUseCase(
	repository pb.SubscriptionGroupDocumentTemplateDomainServiceServer,
	services Services,
) *DeleteDraftPairUseCase {
	deleter, ok := repository.(domainports.SubscriptionGroupDocumentTemplateDraftPairDeleter)
	uc := &DeleteDraftPairUseCase{repo: deleter, svc: services}
	if !ok {
		uc.repositoryError = fmt.Errorf(
			"subscription group document template repository %T does not support atomic draft-pair deletion",
			repository,
		)
	}
	return uc
}

func (uc *DeleteDraftPairUseCase) Execute(ctx context.Context, req *DeleteDraftPairRequest) (*DeleteDraftPairResponse, error) {
	if err := uc.svc.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.SubscriptionGroupDocumentTemplate,
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}
	if err := uc.svc.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.DocumentTemplate,
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}

	bindingID := ""
	if req != nil {
		bindingID = strings.TrimSpace(req.BindingID)
	}
	if bindingID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.svc.Translator,
			"subscription_group_document_template.validation.binding_id_required",
			"Binding ID is required [DEFAULT]",
		))
	}
	if uc.repositoryError != nil {
		return nil, uc.repositoryError
	}
	if uc.repo == nil {
		return nil, fmt.Errorf("subscription group document template atomic draft-pair delete repository is unavailable")
	}
	if uc.svc.Transactor == nil || !uc.svc.Transactor.SupportsTransactions() {
		return nil, fmt.Errorf("subscription group document template draft-pair delete requires transaction support")
	}

	var artifact *documenttemplatepb.DocumentTemplate
	if err := uc.svc.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		var deleteErr error
		artifact, deleteErr = uc.repo.DeleteDraftPair(txCtx, bindingID)
		if deleteErr != nil {
			return deleteErr
		}
		if artifact == nil || strings.TrimSpace(artifact.GetId()) == "" {
			return fmt.Errorf("draft-pair delete returned no document template")
		}
		if strings.TrimSpace(artifact.GetStorageContainer()) == "" || strings.TrimSpace(artifact.GetStorageKey()) == "" {
			return fmt.Errorf("draft-pair delete returned an incomplete storage locator")
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return &DeleteDraftPairResponse{Artifact: artifact}, nil
}

// ExecutePair exposes the atomic operation without leaking its internal request
// and response DTOs across the composition boundary.
func (uc *DeleteDraftPairUseCase) ExecutePair(ctx context.Context, bindingID string) (*documenttemplatepb.DocumentTemplate, error) {
	resp, err := uc.Execute(ctx, &DeleteDraftPairRequest{BindingID: bindingID})
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Artifact == nil {
		return nil, fmt.Errorf("subscription group document template draft-pair delete returned no response")
	}
	return resp.Artifact, nil
}

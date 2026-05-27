package collectionmethod

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	collectionmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method"
)

// Lifecycle transitions — canonical state machine for the CollectionMethod
// TEMPLATE (D-1.8 Q7 Way 3). Fixed states: DRAFT / ACTIVE / CLOSED / ARCHIVED.
//
//	PublishCollectionMethod  DRAFT   -> ACTIVE   (version_status DRAFT -> PUBLISHED)
//	CloseCollectionMethod    ACTIVE  -> CLOSED
//	ArchiveCollectionMethod  CLOSED  -> ARCHIVED
//	ReviseCollectionMethod   ACTIVE  -> new DRAFT revision (supersedes predecessor)
//
// Discipline (load-bearing per D-1.8): these are the CANONICAL transitions that
// fire all validators. The Stage-6 approval gate will WRAP these (mark request
// APPROVED, then call the same transition) — the outcome must be byte-identical
// with or without a gate. Therefore NO gate / approval-request logic lives here.
//
// Each transition reads the current template, validates the source state, sets
// the new lifecycle, and persists via the wrapping UpdateCollectionMethod use
// case (which is transaction-aware and runs the standard update path).

// TransitionCollectionMethodRepositories groups all repository dependencies.
type TransitionCollectionMethodRepositories struct {
	CollectionMethod collectionmethodpb.CollectionMethodDomainServiceServer
}

// TransitionCollectionMethodServices groups all business service dependencies.
type TransitionCollectionMethodServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// --- Publish ---------------------------------------------------------------

// PublishCollectionMethodUseCase transitions DRAFT -> ACTIVE.
type PublishCollectionMethodUseCase struct {
	repositories TransitionCollectionMethodRepositories
	services     TransitionCollectionMethodServices
	update       *UpdateCollectionMethodUseCase
}

// NewPublishCollectionMethodUseCase creates the publish transition use case.
func NewPublishCollectionMethodUseCase(
	repositories TransitionCollectionMethodRepositories,
	services TransitionCollectionMethodServices,
	update *UpdateCollectionMethodUseCase,
) *PublishCollectionMethodUseCase {
	return &PublishCollectionMethodUseCase{repositories: repositories, services: services, update: update}
}

// Execute promotes a DRAFT template to ACTIVE.
func (uc *PublishCollectionMethodUseCase) Execute(ctx context.Context, id string) (*collectionmethodpb.CollectionMethod, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityCollectionMethod, ports.ActionUpdate); err != nil {
		return nil, err
	}

	cm, err := loadCollectionMethod(ctx, uc.repositories.CollectionMethod, uc.services.Translator, id)
	if err != nil {
		return nil, err
	}

	if cm.GetLifecycle() != collectionmethodpb.CollectionMethodLifecycle_COLLECTION_METHOD_LIFECYCLE_DRAFT {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.publish_requires_draft", "[ERR-DEFAULT] Only a DRAFT collection method can be published"))
	}

	cm.Lifecycle = collectionmethodpb.CollectionMethodLifecycle_COLLECTION_METHOD_LIFECYCLE_ACTIVE
	cm.VersionStatus = collectionmethodpb.CollectionMethodVersionStatus_COLLECTION_METHOD_VERSION_STATUS_PUBLISHED

	return persistTransition(ctx, uc.update, cm)
}

// --- Close -----------------------------------------------------------------

// CloseCollectionMethodUseCase transitions ACTIVE -> CLOSED.
type CloseCollectionMethodUseCase struct {
	repositories TransitionCollectionMethodRepositories
	services     TransitionCollectionMethodServices
	update       *UpdateCollectionMethodUseCase
}

// NewCloseCollectionMethodUseCase creates the close transition use case.
func NewCloseCollectionMethodUseCase(
	repositories TransitionCollectionMethodRepositories,
	services TransitionCollectionMethodServices,
	update *UpdateCollectionMethodUseCase,
) *CloseCollectionMethodUseCase {
	return &CloseCollectionMethodUseCase{repositories: repositories, services: services, update: update}
}

// Execute transitions an ACTIVE template to CLOSED.
func (uc *CloseCollectionMethodUseCase) Execute(ctx context.Context, id string) (*collectionmethodpb.CollectionMethod, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityCollectionMethod, ports.ActionUpdate); err != nil {
		return nil, err
	}

	cm, err := loadCollectionMethod(ctx, uc.repositories.CollectionMethod, uc.services.Translator, id)
	if err != nil {
		return nil, err
	}

	if cm.GetLifecycle() != collectionmethodpb.CollectionMethodLifecycle_COLLECTION_METHOD_LIFECYCLE_ACTIVE {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.close_requires_active", "[ERR-DEFAULT] Only an ACTIVE collection method can be closed"))
	}

	cm.Lifecycle = collectionmethodpb.CollectionMethodLifecycle_COLLECTION_METHOD_LIFECYCLE_CLOSED
	return persistTransition(ctx, uc.update, cm)
}

// --- Archive ---------------------------------------------------------------

// ArchiveCollectionMethodUseCase transitions CLOSED -> ARCHIVED.
type ArchiveCollectionMethodUseCase struct {
	repositories TransitionCollectionMethodRepositories
	services     TransitionCollectionMethodServices
	update       *UpdateCollectionMethodUseCase
}

// NewArchiveCollectionMethodUseCase creates the archive transition use case.
func NewArchiveCollectionMethodUseCase(
	repositories TransitionCollectionMethodRepositories,
	services TransitionCollectionMethodServices,
	update *UpdateCollectionMethodUseCase,
) *ArchiveCollectionMethodUseCase {
	return &ArchiveCollectionMethodUseCase{repositories: repositories, services: services, update: update}
}

// Execute transitions a CLOSED template to ARCHIVED.
func (uc *ArchiveCollectionMethodUseCase) Execute(ctx context.Context, id string) (*collectionmethodpb.CollectionMethod, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityCollectionMethod, ports.ActionUpdate); err != nil {
		return nil, err
	}

	cm, err := loadCollectionMethod(ctx, uc.repositories.CollectionMethod, uc.services.Translator, id)
	if err != nil {
		return nil, err
	}

	if cm.GetLifecycle() != collectionmethodpb.CollectionMethodLifecycle_COLLECTION_METHOD_LIFECYCLE_CLOSED {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.archive_requires_closed", "[ERR-DEFAULT] Only a CLOSED collection method can be archived"))
	}

	cm.Lifecycle = collectionmethodpb.CollectionMethodLifecycle_COLLECTION_METHOD_LIFECYCLE_ARCHIVED
	return persistTransition(ctx, uc.update, cm)
}

// --- Revise ----------------------------------------------------------------

// ReviseCollectionMethodUseCase creates a new DRAFT revision of an ACTIVE
// template per the versioning model (D-1.14): the new row shares the
// predecessor's template_code, bumps revision, marks the predecessor SUPERSEDED,
// and links the new row back via supersedes_collection_method_id.
//
// NOTE (Stage 1 minimal): this creates the successor DRAFT row and marks the
// predecessor SUPERSEDED. It does NOT yet deep-copy the predecessor's
// template_details oneof / eligibility-rule wiring (those entities land in later
// stages). The versioning lineage fields (template_code / revision /
// version_status / supersedes_*) ARE set correctly so the chain is queryable.
type ReviseCollectionMethodUseCase struct {
	repositories TransitionCollectionMethodRepositories
	services     TransitionCollectionMethodServices
	create       *CreateCollectionMethodUseCase
	update       *UpdateCollectionMethodUseCase
}

// NewReviseCollectionMethodUseCase creates the revise transition use case.
func NewReviseCollectionMethodUseCase(
	repositories TransitionCollectionMethodRepositories,
	services TransitionCollectionMethodServices,
	create *CreateCollectionMethodUseCase,
	update *UpdateCollectionMethodUseCase,
) *ReviseCollectionMethodUseCase {
	return &ReviseCollectionMethodUseCase{repositories: repositories, services: services, create: create, update: update}
}

// Execute creates a new DRAFT revision superseding an ACTIVE template.
func (uc *ReviseCollectionMethodUseCase) Execute(ctx context.Context, id string) (*collectionmethodpb.CollectionMethod, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityCollectionMethod, ports.ActionCreate); err != nil {
		return nil, err
	}

	predecessor, err := loadCollectionMethod(ctx, uc.repositories.CollectionMethod, uc.services.Translator, id)
	if err != nil {
		return nil, err
	}

	if predecessor.GetLifecycle() != collectionmethodpb.CollectionMethodLifecycle_COLLECTION_METHOD_LIFECYCLE_ACTIVE {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.revise_requires_active", "[ERR-DEFAULT] Only an ACTIVE collection method can be revised"))
	}

	if uc.create == nil || uc.update == nil {
		return nil, errors.New("collection method revise dependencies are not available")
	}

	templateCode := predecessor.GetTemplateCode()
	if templateCode == "" {
		templateCode = predecessor.GetId()
	}
	predID := predecessor.GetId()

	// Build the successor DRAFT row. Carry forward the template-level config;
	// the create use case stamps id/dates/lifecycle DRAFT/version_status DRAFT.
	successor := &collectionmethodpb.CollectionMethod{
		Name:                         predecessor.GetName(),
		ProviderName:                 cloneStringPtr(predecessor.ProviderName),
		WorkspaceId:                  predecessor.GetWorkspaceId(),
		PostingKind:                  predecessor.GetPostingKind(),
		Category:                     predecessor.GetCategory(),
		AudienceMode:                 predecessor.GetAudienceMode(),
		TaxEffectKind:                predecessor.GetTaxEffectKind(),
		DefaultEligibilityRuleId:     cloneStringPtr(predecessor.DefaultEligibilityRuleId),
		BalanceAccountId:             cloneStringPtr(predecessor.BalanceAccountId),
		TargetAccountId:              cloneStringPtr(predecessor.TargetAccountId),
		Source:                       predecessor.GetSource(),
		TemplateCode:                 templateCode,
		Revision:                     predecessor.GetRevision() + 1,
		SupersedesCollectionMethodId: &predID,
	}

	createResp, err := uc.create.Execute(ctx, &collectionmethodpb.CreateCollectionMethodRequest{Data: successor})
	if err != nil {
		return nil, err
	}
	if createResp == nil || len(createResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.errors.revise_failed", "[ERR-DEFAULT] Failed to create the revised collection method"))
	}
	newRow := createResp.Data[0]

	// Mark the predecessor SUPERSEDED. The predecessor stays ACTIVE in lifecycle
	// terms (still usable for issuance until explicitly CLOSED); only its
	// version_status flips so the chain head is unambiguous.
	predecessor.VersionStatus = collectionmethodpb.CollectionMethodVersionStatus_COLLECTION_METHOD_VERSION_STATUS_SUPERSEDED
	if _, err := uc.update.Execute(ctx, &collectionmethodpb.UpdateCollectionMethodRequest{Data: predecessor}); err != nil {
		return nil, err
	}

	return newRow, nil
}

// --- helpers ---------------------------------------------------------------

func loadCollectionMethod(ctx context.Context, repo collectionmethodpb.CollectionMethodDomainServiceServer, tr ports.Translator, id string) (*collectionmethodpb.CollectionMethod, error) {
	if id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr, "collection_method.validation.id_required", "Collection method ID is required [DEFAULT]"))
	}
	if repo == nil {
		return nil, errors.New("collection method repository is not available")
	}
	resp, err := repo.ReadCollectionMethod(ctx, &collectionmethodpb.ReadCollectionMethodRequest{
		Data: &collectionmethodpb.CollectionMethod{Id: id},
	})
	if err != nil {
		return nil, err
	}
	if resp == nil || len(resp.Data) == 0 || resp.Data[0] == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr, "collection_method.errors.not_found", "[ERR-DEFAULT] Collection method not found"))
	}
	return resp.Data[0], nil
}

func persistTransition(ctx context.Context, update *UpdateCollectionMethodUseCase, cm *collectionmethodpb.CollectionMethod) (*collectionmethodpb.CollectionMethod, error) {
	if update == nil {
		return nil, fmt.Errorf("collection method update use case is not available")
	}
	resp, err := update.Execute(ctx, &collectionmethodpb.UpdateCollectionMethodRequest{Data: cm})
	if err != nil {
		return nil, err
	}
	if resp != nil && len(resp.Data) > 0 {
		return resp.Data[0], nil
	}
	return cm, nil
}

func cloneStringPtr(p *string) *string {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

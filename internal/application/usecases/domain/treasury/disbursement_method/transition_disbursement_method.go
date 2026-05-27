package disbursementmethod

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	disbursementmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement_method"
)

// Lifecycle transitions — canonical state machine for the DisbursementMethod
// TEMPLATE (D-1.8 Q7 Way 3), symmetric to the collection_method side.
//
//	PublishDisbursementMethod  DRAFT  -> ACTIVE
//	CloseDisbursementMethod    ACTIVE -> CLOSED
//	ArchiveDisbursementMethod  CLOSED -> ARCHIVED
//	ReviseDisbursementMethod   ACTIVE -> new DRAFT revision (supersedes predecessor)
//
// Discipline (D-1.8): canonical transitions fire all validators; the Stage-6
// approval gate WRAPS these and the outcome must be byte-identical. NO gate /
// approval-request logic here.

// TransitionDisbursementMethodRepositories groups all repository dependencies.
type TransitionDisbursementMethodRepositories struct {
	DisbursementMethod disbursementmethodpb.DisbursementMethodDomainServiceServer
}

// TransitionDisbursementMethodServices groups all business service dependencies.
type TransitionDisbursementMethodServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// --- Publish ---------------------------------------------------------------

// PublishDisbursementMethodUseCase transitions DRAFT -> ACTIVE.
type PublishDisbursementMethodUseCase struct {
	repositories TransitionDisbursementMethodRepositories
	services     TransitionDisbursementMethodServices
	update       *UpdateDisbursementMethodUseCase
}

// NewPublishDisbursementMethodUseCase creates the publish transition use case.
func NewPublishDisbursementMethodUseCase(
	repositories TransitionDisbursementMethodRepositories,
	services TransitionDisbursementMethodServices,
	update *UpdateDisbursementMethodUseCase,
) *PublishDisbursementMethodUseCase {
	return &PublishDisbursementMethodUseCase{repositories: repositories, services: services, update: update}
}

// Execute promotes a DRAFT template to ACTIVE.
func (uc *PublishDisbursementMethodUseCase) Execute(ctx context.Context, id string) (*disbursementmethodpb.DisbursementMethod, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityDisbursementMethod, ports.ActionUpdate); err != nil {
		return nil, err
	}

	dm, err := loadDisbursementMethod(ctx, uc.repositories.DisbursementMethod, uc.services.Translator, id)
	if err != nil {
		return nil, err
	}

	if dm.GetLifecycle() != disbursementmethodpb.DisbursementMethodLifecycle_DISBURSEMENT_METHOD_LIFECYCLE_DRAFT {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "disbursement_method.validation.publish_requires_draft", "[ERR-DEFAULT] Only a DRAFT disbursement method can be published"))
	}

	dm.Lifecycle = disbursementmethodpb.DisbursementMethodLifecycle_DISBURSEMENT_METHOD_LIFECYCLE_ACTIVE
	dm.VersionStatus = disbursementmethodpb.DisbursementMethodVersionStatus_DISBURSEMENT_METHOD_VERSION_STATUS_PUBLISHED

	return persistTransition(ctx, uc.update, dm)
}

// --- Close -----------------------------------------------------------------

// CloseDisbursementMethodUseCase transitions ACTIVE -> CLOSED.
type CloseDisbursementMethodUseCase struct {
	repositories TransitionDisbursementMethodRepositories
	services     TransitionDisbursementMethodServices
	update       *UpdateDisbursementMethodUseCase
}

// NewCloseDisbursementMethodUseCase creates the close transition use case.
func NewCloseDisbursementMethodUseCase(
	repositories TransitionDisbursementMethodRepositories,
	services TransitionDisbursementMethodServices,
	update *UpdateDisbursementMethodUseCase,
) *CloseDisbursementMethodUseCase {
	return &CloseDisbursementMethodUseCase{repositories: repositories, services: services, update: update}
}

// Execute transitions an ACTIVE template to CLOSED.
func (uc *CloseDisbursementMethodUseCase) Execute(ctx context.Context, id string) (*disbursementmethodpb.DisbursementMethod, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityDisbursementMethod, ports.ActionUpdate); err != nil {
		return nil, err
	}

	dm, err := loadDisbursementMethod(ctx, uc.repositories.DisbursementMethod, uc.services.Translator, id)
	if err != nil {
		return nil, err
	}

	if dm.GetLifecycle() != disbursementmethodpb.DisbursementMethodLifecycle_DISBURSEMENT_METHOD_LIFECYCLE_ACTIVE {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "disbursement_method.validation.close_requires_active", "[ERR-DEFAULT] Only an ACTIVE disbursement method can be closed"))
	}

	dm.Lifecycle = disbursementmethodpb.DisbursementMethodLifecycle_DISBURSEMENT_METHOD_LIFECYCLE_CLOSED
	return persistTransition(ctx, uc.update, dm)
}

// --- Archive ---------------------------------------------------------------

// ArchiveDisbursementMethodUseCase transitions CLOSED -> ARCHIVED.
type ArchiveDisbursementMethodUseCase struct {
	repositories TransitionDisbursementMethodRepositories
	services     TransitionDisbursementMethodServices
	update       *UpdateDisbursementMethodUseCase
}

// NewArchiveDisbursementMethodUseCase creates the archive transition use case.
func NewArchiveDisbursementMethodUseCase(
	repositories TransitionDisbursementMethodRepositories,
	services TransitionDisbursementMethodServices,
	update *UpdateDisbursementMethodUseCase,
) *ArchiveDisbursementMethodUseCase {
	return &ArchiveDisbursementMethodUseCase{repositories: repositories, services: services, update: update}
}

// Execute transitions a CLOSED template to ARCHIVED.
func (uc *ArchiveDisbursementMethodUseCase) Execute(ctx context.Context, id string) (*disbursementmethodpb.DisbursementMethod, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityDisbursementMethod, ports.ActionUpdate); err != nil {
		return nil, err
	}

	dm, err := loadDisbursementMethod(ctx, uc.repositories.DisbursementMethod, uc.services.Translator, id)
	if err != nil {
		return nil, err
	}

	if dm.GetLifecycle() != disbursementmethodpb.DisbursementMethodLifecycle_DISBURSEMENT_METHOD_LIFECYCLE_CLOSED {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "disbursement_method.validation.archive_requires_closed", "[ERR-DEFAULT] Only a CLOSED disbursement method can be archived"))
	}

	dm.Lifecycle = disbursementmethodpb.DisbursementMethodLifecycle_DISBURSEMENT_METHOD_LIFECYCLE_ARCHIVED
	return persistTransition(ctx, uc.update, dm)
}

// --- Revise ----------------------------------------------------------------

// ReviseDisbursementMethodUseCase creates a new DRAFT revision of an ACTIVE
// template (D-1.14). Same Stage-1-minimal note as the CM side: it sets the
// versioning lineage correctly (template_code / revision / version_status /
// supersedes_*) and marks the predecessor SUPERSEDED, but does not deep-copy
// the template_details oneof (later-stage entities).
type ReviseDisbursementMethodUseCase struct {
	repositories TransitionDisbursementMethodRepositories
	services     TransitionDisbursementMethodServices
	create       *CreateDisbursementMethodUseCase
	update       *UpdateDisbursementMethodUseCase
}

// NewReviseDisbursementMethodUseCase creates the revise transition use case.
func NewReviseDisbursementMethodUseCase(
	repositories TransitionDisbursementMethodRepositories,
	services TransitionDisbursementMethodServices,
	create *CreateDisbursementMethodUseCase,
	update *UpdateDisbursementMethodUseCase,
) *ReviseDisbursementMethodUseCase {
	return &ReviseDisbursementMethodUseCase{repositories: repositories, services: services, create: create, update: update}
}

// Execute creates a new DRAFT revision superseding an ACTIVE template.
func (uc *ReviseDisbursementMethodUseCase) Execute(ctx context.Context, id string) (*disbursementmethodpb.DisbursementMethod, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityDisbursementMethod, ports.ActionCreate); err != nil {
		return nil, err
	}

	predecessor, err := loadDisbursementMethod(ctx, uc.repositories.DisbursementMethod, uc.services.Translator, id)
	if err != nil {
		return nil, err
	}

	if predecessor.GetLifecycle() != disbursementmethodpb.DisbursementMethodLifecycle_DISBURSEMENT_METHOD_LIFECYCLE_ACTIVE {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "disbursement_method.validation.revise_requires_active", "[ERR-DEFAULT] Only an ACTIVE disbursement method can be revised"))
	}

	if uc.create == nil || uc.update == nil {
		return nil, errors.New("disbursement method revise dependencies are not available")
	}

	templateCode := predecessor.GetTemplateCode()
	if templateCode == "" {
		templateCode = predecessor.GetId()
	}
	predID := predecessor.GetId()

	// Buying-side asymmetry (D-4.9): no audience_mode field on DisbursementMethod.
	successor := &disbursementmethodpb.DisbursementMethod{
		Name:                           predecessor.GetName(),
		ProviderName:                   cloneStringPtr(predecessor.ProviderName),
		WorkspaceId:                    predecessor.GetWorkspaceId(),
		PostingKind:                    predecessor.GetPostingKind(),
		Category:                       predecessor.GetCategory(),
		TaxEffectKind:                  predecessor.GetTaxEffectKind(),
		DefaultEligibilityRuleId:       cloneStringPtr(predecessor.DefaultEligibilityRuleId),
		BalanceAccountId:               cloneStringPtr(predecessor.BalanceAccountId),
		TargetAccountId:                cloneStringPtr(predecessor.TargetAccountId),
		Source:                         predecessor.GetSource(),
		TemplateCode:                   templateCode,
		Revision:                       predecessor.GetRevision() + 1,
		SupersedesDisbursementMethodId: &predID,
	}

	createResp, err := uc.create.Execute(ctx, &disbursementmethodpb.CreateDisbursementMethodRequest{Data: successor})
	if err != nil {
		return nil, err
	}
	if createResp == nil || len(createResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "disbursement_method.errors.revise_failed", "[ERR-DEFAULT] Failed to create the revised disbursement method"))
	}
	newRow := createResp.Data[0]

	predecessor.VersionStatus = disbursementmethodpb.DisbursementMethodVersionStatus_DISBURSEMENT_METHOD_VERSION_STATUS_SUPERSEDED
	if _, err := uc.update.Execute(ctx, &disbursementmethodpb.UpdateDisbursementMethodRequest{Data: predecessor}); err != nil {
		return nil, err
	}

	return newRow, nil
}

// --- helpers ---------------------------------------------------------------

func loadDisbursementMethod(ctx context.Context, repo disbursementmethodpb.DisbursementMethodDomainServiceServer, tr ports.Translator, id string) (*disbursementmethodpb.DisbursementMethod, error) {
	if id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr, "disbursement_method.validation.id_required", "Disbursement method ID is required [DEFAULT]"))
	}
	if repo == nil {
		return nil, errors.New("disbursement method repository is not available")
	}
	resp, err := repo.ReadDisbursementMethod(ctx, &disbursementmethodpb.ReadDisbursementMethodRequest{
		Data: &disbursementmethodpb.DisbursementMethod{Id: id},
	})
	if err != nil {
		return nil, err
	}
	if resp == nil || len(resp.Data) == 0 || resp.Data[0] == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr, "disbursement_method.errors.not_found", "[ERR-DEFAULT] Disbursement method not found"))
	}
	return resp.Data[0], nil
}

func persistTransition(ctx context.Context, update *UpdateDisbursementMethodUseCase, dm *disbursementmethodpb.DisbursementMethod) (*disbursementmethodpb.DisbursementMethod, error) {
	if update == nil {
		return nil, fmt.Errorf("disbursement method update use case is not available")
	}
	resp, err := update.Execute(ctx, &disbursementmethodpb.UpdateDisbursementMethodRequest{Data: dm})
	if err != nil {
		return nil, err
	}
	if resp != nil && len(resp.Data) > 0 {
		return resp.Data[0], nil
	}
	return dm, nil
}

func cloneStringPtr(p *string) *string {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

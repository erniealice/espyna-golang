package collectionmethodgrant

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	collectionmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method"
	grantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method_grant"
)

// entityCollectionMethodGrant is the permission namespace + translation key root.
const entityCollectionMethodGrant = "collection_method_grant"

// actionRevoke / actionBulkGrant are the non-CRUD permission actions this entity
// uses (matching the planned seeds collection_method_grant:{revoke,bulk_grant}).
const (
	actionRevoke    = "revoke"
	actionBulkGrant = "bulk_grant"
)

// CreateCollectionMethodGrantRepositories groups all repository dependencies.
type CreateCollectionMethodGrantRepositories struct {
	CollectionMethodGrant grantpb.CollectionMethodGrantDomainServiceServer
	// CollectionMethod is the TEMPLATE repo the audience-mode guardrail reads.
	CollectionMethod collectionmethodpb.CollectionMethodDomainServiceServer
}

// CreateCollectionMethodGrantServices groups all business service dependencies.
type CreateCollectionMethodGrantServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// CreateCollectionMethodGrantUseCase handles the business logic for creating a grant.
type CreateCollectionMethodGrantUseCase struct {
	repositories CreateCollectionMethodGrantRepositories
	services     CreateCollectionMethodGrantServices
}

// NewCreateCollectionMethodGrantUseCase creates use case with grouped dependencies.
func NewCreateCollectionMethodGrantUseCase(
	repositories CreateCollectionMethodGrantRepositories,
	services CreateCollectionMethodGrantServices,
) *CreateCollectionMethodGrantUseCase {
	return &CreateCollectionMethodGrantUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create grant operation.
func (uc *CreateCollectionMethodGrantUseCase) Execute(ctx context.Context, req *grantpb.CreateCollectionMethodGrantRequest) (*grantpb.CreateCollectionMethodGrantResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityCollectionMethodGrant, ports.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *grantpb.CreateCollectionMethodGrantResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "collection_method_grant.errors.creation_failed", "Collection method grant creation failed [DEFAULT]")
				return fmt.Errorf("%s: %w", translatedError, err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	return uc.executeCore(ctx, req)
}

func (uc *CreateCollectionMethodGrantUseCase) executeCore(ctx context.Context, req *grantpb.CreateCollectionMethodGrantRequest) (*grantpb.CreateCollectionMethodGrantResponse, error) {
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	if err := uc.enrichData(req.Data); err != nil {
		return nil, err
	}

	// Audience-mode guardrail (§E-4): the prospective ACTIVE-grant count after this
	// create = current ACTIVE grants for the method + 1.
	currentActive, err := countActiveGrantsForMethod(ctx, uc.repositories.CollectionMethodGrant, req.Data.GetCollectionMethodId())
	if err != nil {
		return nil, err
	}
	if err := validateAudienceModeGuardrail(ctx, uc.services.Translator, uc.repositories.CollectionMethod, req.Data.GetCollectionMethodId(), currentActive+1); err != nil {
		return nil, err
	}

	if uc.repositories.CollectionMethodGrant == nil {
		return nil, errors.New("collection method grant repository is not available")
	}
	return uc.repositories.CollectionMethodGrant.CreateCollectionMethodGrant(ctx, req)
}

func (uc *CreateCollectionMethodGrantUseCase) validateInput(ctx context.Context, req *grantpb.CreateCollectionMethodGrantRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method_grant.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method_grant.validation.data_required", "[ERR-DEFAULT] Collection method grant data is required"))
	}

	req.Data.CollectionMethodId = strings.TrimSpace(req.Data.CollectionMethodId)
	if req.Data.CollectionMethodId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method_grant.validation.collection_method_id_required", "[ERR-DEFAULT] Collection method ID is required"))
	}

	// subject=CLIENT requires a client_id (§E-4).
	if req.Data.GetSubject() == grantpb.CollectionMethodGrantSubject_COLLECTION_METHOD_GRANT_SUBJECT_CLIENT {
		req.Data.ClientId = strings.TrimSpace(req.Data.ClientId)
		if req.Data.ClientId == "" {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method_grant.validation.client_id_required", "[ERR-DEFAULT] Client ID is required when subject is CLIENT"))
		}
	}

	return nil
}

func (uc *CreateCollectionMethodGrantUseCase) enrichData(grant *grantpb.CollectionMethodGrant) error {
	now := time.Now()
	if grant.Id == "" {
		grant.Id = uc.services.IDGenerator.GenerateID()
	}
	grant.DateCreated = &[]int64{now.UnixMilli()}[0]
	grant.DateModified = &[]int64{now.UnixMilli()}[0]
	grant.Active = true
	// Default subject to CLIENT when a client_id is supplied and subject is unset.
	if grant.GetSubject() == grantpb.CollectionMethodGrantSubject_COLLECTION_METHOD_GRANT_SUBJECT_UNSPECIFIED && grant.GetClientId() != "" {
		grant.Subject = grantpb.CollectionMethodGrantSubject_COLLECTION_METHOD_GRANT_SUBJECT_CLIENT
	}
	// A freshly created grant is always ACTIVE (grants do not mutate except via revoke).
	grant.Status = grantpb.CollectionMethodGrantStatus_COLLECTION_METHOD_GRANT_STATUS_ACTIVE
	return nil
}

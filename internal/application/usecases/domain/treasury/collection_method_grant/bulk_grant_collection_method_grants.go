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

// BulkGrantCollectionMethodGrantsRepositories groups all repository dependencies.
type BulkGrantCollectionMethodGrantsRepositories struct {
	CollectionMethodGrant grantpb.CollectionMethodGrantDomainServiceServer
	// CollectionMethod is the TEMPLATE repo the audience-mode guardrail reads.
	CollectionMethod collectionmethodpb.CollectionMethodDomainServiceServer
}

// BulkGrantCollectionMethodGrantsServices groups all business service dependencies.
type BulkGrantCollectionMethodGrantsServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// BulkGrantCollectionMethodGrantsUseCase grants many clients access to one CM
// template in a single call. The audience-mode guardrail is evaluated once against
// the final prospective ACTIVE-grant count for the method.
type BulkGrantCollectionMethodGrantsUseCase struct {
	repositories BulkGrantCollectionMethodGrantsRepositories
	services     BulkGrantCollectionMethodGrantsServices
}

// NewBulkGrantCollectionMethodGrantsUseCase creates use case with grouped dependencies.
func NewBulkGrantCollectionMethodGrantsUseCase(
	repositories BulkGrantCollectionMethodGrantsRepositories,
	services BulkGrantCollectionMethodGrantsServices,
) *BulkGrantCollectionMethodGrantsUseCase {
	return &BulkGrantCollectionMethodGrantsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the bulk-grant operation.
func (uc *BulkGrantCollectionMethodGrantsUseCase) Execute(ctx context.Context, req *grantpb.BulkGrantCollectionMethodGrantsRequest) (*grantpb.BulkGrantCollectionMethodGrantsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityCollectionMethodGrant, actionBulkGrant); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *grantpb.BulkGrantCollectionMethodGrantsResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "collection_method_grant.errors.bulk_grant_failed", "Collection method bulk grant failed [DEFAULT]")
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

func (uc *BulkGrantCollectionMethodGrantsUseCase) executeCore(ctx context.Context, req *grantpb.BulkGrantCollectionMethodGrantsRequest) (*grantpb.BulkGrantCollectionMethodGrantsResponse, error) {
	if req == nil || len(req.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method_grant.validation.bulk_data_required", "[ERR-DEFAULT] At least one grant is required"))
	}

	// All grants in a bulk call must target the SAME CM template (the guardrail is
	// a per-method count). Validate + enrich each row.
	methodID := strings.TrimSpace(req.Data[0].GetCollectionMethodId())
	if methodID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method_grant.validation.collection_method_id_required", "[ERR-DEFAULT] Collection method ID is required"))
	}

	for _, grant := range req.Data {
		if grant == nil {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method_grant.validation.data_required", "[ERR-DEFAULT] Collection method grant data is required"))
		}
		grant.CollectionMethodId = strings.TrimSpace(grant.CollectionMethodId)
		if grant.CollectionMethodId != methodID {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method_grant.validation.bulk_single_method", "[ERR-DEFAULT] All grants in a bulk call must target the same collection method"))
		}
		if grant.GetSubject() == grantpb.CollectionMethodGrantSubject_COLLECTION_METHOD_GRANT_SUBJECT_UNSPECIFIED && grant.GetClientId() != "" {
			grant.Subject = grantpb.CollectionMethodGrantSubject_COLLECTION_METHOD_GRANT_SUBJECT_CLIENT
		}
		if grant.GetSubject() == grantpb.CollectionMethodGrantSubject_COLLECTION_METHOD_GRANT_SUBJECT_CLIENT {
			grant.ClientId = strings.TrimSpace(grant.ClientId)
			if grant.ClientId == "" {
				return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method_grant.validation.client_id_required", "[ERR-DEFAULT] Client ID is required when subject is CLIENT"))
			}
		}
		uc.enrichData(grant)
	}

	// Audience-mode guardrail (§E-4): prospective ACTIVE-grant count = current
	// ACTIVE grants for the method + the number being granted now.
	currentActive, err := countActiveGrantsForMethod(ctx, uc.repositories.CollectionMethodGrant, methodID)
	if err != nil {
		return nil, err
	}
	if err := validateAudienceModeGuardrail(ctx, uc.services.Translator, uc.repositories.CollectionMethod, methodID, currentActive+len(req.Data)); err != nil {
		return nil, err
	}

	if uc.repositories.CollectionMethodGrant == nil {
		return nil, errors.New("collection method grant repository is not available")
	}
	return uc.repositories.CollectionMethodGrant.BulkGrantCollectionMethodGrants(ctx, req)
}

func (uc *BulkGrantCollectionMethodGrantsUseCase) enrichData(grant *grantpb.CollectionMethodGrant) {
	now := time.Now()
	if grant.Id == "" {
		grant.Id = uc.services.IDGenerator.GenerateID()
	}
	grant.DateCreated = &[]int64{now.UnixMilli()}[0]
	grant.DateModified = &[]int64{now.UnixMilli()}[0]
	grant.Active = true
	grant.Status = grantpb.CollectionMethodGrantStatus_COLLECTION_METHOD_GRANT_STATUS_ACTIVE
}

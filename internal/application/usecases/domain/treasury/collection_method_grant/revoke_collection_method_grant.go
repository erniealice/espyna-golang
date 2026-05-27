package collectionmethodgrant

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	grantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method_grant"
)

// RevokeCollectionMethodGrantRepositories groups all repository dependencies.
type RevokeCollectionMethodGrantRepositories struct {
	CollectionMethodGrant grantpb.CollectionMethodGrantDomainServiceServer
}

// RevokeCollectionMethodGrantServices groups all business service dependencies.
type RevokeCollectionMethodGrantServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// RevokeCollectionMethodGrantUseCase handles the only state change a grant
// undergoes: ACTIVE → REVOKED. There is deliberately no Update use case — grants
// are immutable apart from revocation (§E-4 / Q6).
type RevokeCollectionMethodGrantUseCase struct {
	repositories RevokeCollectionMethodGrantRepositories
	services     RevokeCollectionMethodGrantServices
}

// NewRevokeCollectionMethodGrantUseCase creates use case with grouped dependencies.
func NewRevokeCollectionMethodGrantUseCase(
	repositories RevokeCollectionMethodGrantRepositories,
	services RevokeCollectionMethodGrantServices,
) *RevokeCollectionMethodGrantUseCase {
	return &RevokeCollectionMethodGrantUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the revoke grant operation.
func (uc *RevokeCollectionMethodGrantUseCase) Execute(ctx context.Context, req *grantpb.RevokeCollectionMethodGrantRequest) (*grantpb.RevokeCollectionMethodGrantResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityCollectionMethodGrant, actionRevoke); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method_grant.validation.id_required", "Collection method grant ID is required [DEFAULT]"))
	}

	if uc.repositories.CollectionMethodGrant == nil {
		return nil, errors.New("collection method grant repository is not available")
	}
	return uc.repositories.CollectionMethodGrant.RevokeCollectionMethodGrant(ctx, req)
}

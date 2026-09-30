package collection_application

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	recoverydocumentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	revenuepaymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_payment"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
	collectionapplicationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_application"
	collectionmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method"
)

// PreviewCollectionApplicationRepositories groups repository dependencies (read only).
type PreviewCollectionApplicationRepositories struct {
	Collection            collectionpb.CollectionDomainServiceServer
	CollectionApplication collectionapplicationpb.CollectionApplicationDomainServiceServer
	Revenue               revenuepb.RevenueDomainServiceServer
	RevenuePayment        revenuepaymentpb.RevenuePaymentDomainServiceServer
	RecoveryDocument      recoverydocumentpb.RecoveryDocumentDomainServiceServer
	// Reference readers (C5/C17): the request's client (and collection method, when named) is read
	// through the same row-scoped reads receive uses before any list runs.
	Client           clientpb.ClientDomainServiceServer
	CollectionMethod collectionmethodpb.CollectionMethodDomainServiceServer
}

// PreviewCollectionApplicationServices groups service dependencies.
type PreviewCollectionApplicationServices struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// PreviewCollectionApplicationUseCase returns the allocation ReceiveAndApplyCollection would make,
// without writing anything. Permission: collection_application:create, strict gate like receive (C3);
// the request's client_id is verified through the row-scoped ReadClient before any list (C17).
type PreviewCollectionApplicationUseCase struct {
	repositories PreviewCollectionApplicationRepositories
	services     PreviewCollectionApplicationServices
}

// NewPreviewCollectionApplicationUseCase creates the use case with grouped dependencies.
func NewPreviewCollectionApplicationUseCase(r PreviewCollectionApplicationRepositories, s PreviewCollectionApplicationServices) *PreviewCollectionApplicationUseCase {
	return &PreviewCollectionApplicationUseCase{repositories: r, services: s}
}

// Execute returns the planned allocation.
func (uc *PreviewCollectionApplicationUseCase) Execute(ctx context.Context, req *collectionapplicationpb.PreviewCollectionApplicationRequest) (*collectionapplicationpb.PreviewCollectionApplicationResponse, error) {
	if err := gateStrict(ctx, uc.services.ActionGatekeeper, entityid.ActionCreate); err != nil {
		return nil, err
	}
	r := uc.repositories
	l := ledger{Collection: r.Collection, CollectionApplication: r.CollectionApplication, Revenue: r.Revenue,
		RevenuePayment: r.RevenuePayment, RecoveryDocument: r.RecoveryDocument, Client: r.Client, CollectionMethod: r.CollectionMethod}
	params := paramsFromPreview(req)
	if err := validate(params); err != nil {
		return nil, usecaseerr.Localize(ctx, uc.services.Translator, "collection_application", err)
	}
	// C17: the request-supplied client is read row-scoped before the workspace-scoped lists run.
	if err := l.verifyReferences(ctx, params); err != nil {
		return nil, usecaseerr.Localize(ctx, uc.services.Translator, "collection_application", err)
	}
	plan, err := l.buildPlan(ctx, params)
	if err != nil {
		return nil, usecaseerr.Localize(ctx, uc.services.Translator, "collection_application", err)
	}
	return &collectionapplicationpb.PreviewCollectionApplicationResponse{Plan: plan, Success: true}, nil
}

// Package collectionmethodgrant holds the treasury collection_method_grant use
// cases (treasury-domain-rebuild Stage 3, entity-layer-map.md Layer 7).
//
// Scope: audience-eligibility CONFIG. A grant binds a client to a CollectionMethod
// TEMPLATE (never an instance). This is CONFIG, never an EVENT (Q6 LOCKED): there
// is NO usage_count / last_used_at / redemption_count anywhere — usage lives on
// treasury_collection / revenue. Grants do not mutate; the only state change is
// ACTIVE → REVOKED, so there is deliberately NO Update use case.
//
// Use cases: create / read / revoke / list / bulk_grant + GetListPageData /
// GetItemPageData. The shared validate_audience_mode_guardrail.go helper is called
// from create + bulk_grant to enforce the §E-4 audience-mode rule (OPEN → zero
// grants, RESTRICTED → ≥1, SINGLE_CLIENT → exactly 1) against the method template's
// audience_mode.
package collectionmethodgrant

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	collectionmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method"
	grantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method_grant"
)

// CollectionMethodGrantRepositories groups all repository dependencies for grant use cases.
type CollectionMethodGrantRepositories struct {
	CollectionMethodGrant grantpb.CollectionMethodGrantDomainServiceServer
	// CollectionMethod is the TEMPLATE repo the audience-mode guardrail reads to
	// resolve a method's audience_mode at create / bulk_grant time.
	CollectionMethod collectionmethodpb.CollectionMethodDomainServiceServer
}

// CollectionMethodGrantServices groups all business service dependencies.
type CollectionMethodGrantServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases contains all collection method grant-related use cases.
type UseCases struct {
	CreateCollectionMethodGrant          *CreateCollectionMethodGrantUseCase
	ReadCollectionMethodGrant            *ReadCollectionMethodGrantUseCase
	RevokeCollectionMethodGrant          *RevokeCollectionMethodGrantUseCase
	ListCollectionMethodGrants           *ListCollectionMethodGrantsUseCase
	BulkGrantCollectionMethodGrants      *BulkGrantCollectionMethodGrantsUseCase
	GetCollectionMethodGrantListPageData *GetCollectionMethodGrantListPageDataUseCase
	GetCollectionMethodGrantItemPageData *GetCollectionMethodGrantItemPageDataUseCase
}

// NewUseCases creates a new collection of collection method grant use cases.
func NewUseCases(
	repositories CollectionMethodGrantRepositories,
	services CollectionMethodGrantServices,
) *UseCases {
	createUC := NewCreateCollectionMethodGrantUseCase(
		CreateCollectionMethodGrantRepositories{
			CollectionMethodGrant: repositories.CollectionMethodGrant,
			CollectionMethod:      repositories.CollectionMethod,
		},
		CreateCollectionMethodGrantServices{
			Authorizer:  services.Authorizer,
			Transactor:  services.Transactor,
			Translator:  services.Translator,
			IDGenerator: services.IDGenerator,
		},
	)

	readUC := NewReadCollectionMethodGrantUseCase(
		ReadCollectionMethodGrantRepositories{
			CollectionMethodGrant: repositories.CollectionMethodGrant,
		},
		ReadCollectionMethodGrantServices{
			Authorizer: services.Authorizer,
			Transactor: services.Transactor,
			Translator: services.Translator,
		},
	)

	revokeUC := NewRevokeCollectionMethodGrantUseCase(
		RevokeCollectionMethodGrantRepositories{
			CollectionMethodGrant: repositories.CollectionMethodGrant,
		},
		RevokeCollectionMethodGrantServices{
			Authorizer: services.Authorizer,
			Transactor: services.Transactor,
			Translator: services.Translator,
		},
	)

	listUC := NewListCollectionMethodGrantsUseCase(
		ListCollectionMethodGrantsRepositories{
			CollectionMethodGrant: repositories.CollectionMethodGrant,
		},
		ListCollectionMethodGrantsServices{
			Authorizer: services.Authorizer,
			Transactor: services.Transactor,
			Translator: services.Translator,
		},
	)

	bulkGrantUC := NewBulkGrantCollectionMethodGrantsUseCase(
		BulkGrantCollectionMethodGrantsRepositories{
			CollectionMethodGrant: repositories.CollectionMethodGrant,
			CollectionMethod:      repositories.CollectionMethod,
		},
		BulkGrantCollectionMethodGrantsServices{
			Authorizer:  services.Authorizer,
			Transactor:  services.Transactor,
			Translator:  services.Translator,
			IDGenerator: services.IDGenerator,
		},
	)

	listPageDataUC := NewGetCollectionMethodGrantListPageDataUseCase(
		GetCollectionMethodGrantListPageDataRepositories{
			CollectionMethodGrant: repositories.CollectionMethodGrant,
		},
		GetCollectionMethodGrantListPageDataServices{
			Authorizer: services.Authorizer,
			Transactor: services.Transactor,
			Translator: services.Translator,
		},
	)

	itemPageDataUC := NewGetCollectionMethodGrantItemPageDataUseCase(
		GetCollectionMethodGrantItemPageDataRepositories{
			CollectionMethodGrant: repositories.CollectionMethodGrant,
		},
		GetCollectionMethodGrantItemPageDataServices{
			Authorizer: services.Authorizer,
			Transactor: services.Transactor,
			Translator: services.Translator,
		},
	)

	return &UseCases{
		CreateCollectionMethodGrant:          createUC,
		ReadCollectionMethodGrant:            readUC,
		RevokeCollectionMethodGrant:          revokeUC,
		ListCollectionMethodGrants:           listUC,
		BulkGrantCollectionMethodGrants:      bulkGrantUC,
		GetCollectionMethodGrantListPageData: listPageDataUC,
		GetCollectionMethodGrantItemPageData: itemPageDataUC,
	}
}

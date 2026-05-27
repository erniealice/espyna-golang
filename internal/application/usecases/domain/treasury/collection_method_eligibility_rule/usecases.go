// Package collectionmethodeligibilityrule holds the treasury
// collection_method_eligibility_rule use cases (treasury-domain-rebuild Stage 2,
// entity-layer-map.md Layer 7).
//
// Scope: rule CRUD + page-data. This is the entity that
// CollectionMethod.default_eligibility_rule_id (field 16) points to; its fields
// are SNAPSHOTTED onto voucher/advance instances at issuance (entities.md §E-19,
// a later stage). No lifecycle transitions and no grant/instance entities here.
package collectionmethodeligibilityrule

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	eligibilityrulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method_eligibility_rule"
)

// CollectionMethodEligibilityRuleRepositories groups all repository dependencies for eligibility rule use cases.
type CollectionMethodEligibilityRuleRepositories struct {
	CollectionMethodEligibilityRule eligibilityrulepb.CollectionMethodEligibilityRuleDomainServiceServer
}

// CollectionMethodEligibilityRuleServices groups all business service dependencies.
type CollectionMethodEligibilityRuleServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases contains all collection method eligibility rule-related use cases.
type UseCases struct {
	CreateCollectionMethodEligibilityRule          *CreateCollectionMethodEligibilityRuleUseCase
	ReadCollectionMethodEligibilityRule            *ReadCollectionMethodEligibilityRuleUseCase
	UpdateCollectionMethodEligibilityRule          *UpdateCollectionMethodEligibilityRuleUseCase
	DeleteCollectionMethodEligibilityRule          *DeleteCollectionMethodEligibilityRuleUseCase
	ListCollectionMethodEligibilityRules           *ListCollectionMethodEligibilityRulesUseCase
	GetCollectionMethodEligibilityRuleListPageData *GetCollectionMethodEligibilityRuleListPageDataUseCase
	GetCollectionMethodEligibilityRuleItemPageData *GetCollectionMethodEligibilityRuleItemPageDataUseCase
}

// NewUseCases creates a new collection of collection method eligibility rule use cases.
func NewUseCases(
	repositories CollectionMethodEligibilityRuleRepositories,
	services CollectionMethodEligibilityRuleServices,
) *UseCases {
	createUC := NewCreateCollectionMethodEligibilityRuleUseCase(
		CreateCollectionMethodEligibilityRuleRepositories(repositories),
		CreateCollectionMethodEligibilityRuleServices{
			Authorizer:  services.Authorizer,
			Transactor:  services.Transactor,
			Translator:  services.Translator,
			IDGenerator: services.IDGenerator,
		},
	)

	readUC := NewReadCollectionMethodEligibilityRuleUseCase(
		ReadCollectionMethodEligibilityRuleRepositories(repositories),
		ReadCollectionMethodEligibilityRuleServices{
			Authorizer: services.Authorizer,
			Transactor: services.Transactor,
			Translator: services.Translator,
		},
	)

	updateUC := NewUpdateCollectionMethodEligibilityRuleUseCase(
		UpdateCollectionMethodEligibilityRuleRepositories(repositories),
		UpdateCollectionMethodEligibilityRuleServices{
			Authorizer: services.Authorizer,
			Transactor: services.Transactor,
			Translator: services.Translator,
		},
	)

	deleteUC := NewDeleteCollectionMethodEligibilityRuleUseCase(
		DeleteCollectionMethodEligibilityRuleRepositories(repositories),
		DeleteCollectionMethodEligibilityRuleServices{
			Authorizer: services.Authorizer,
			Transactor: services.Transactor,
			Translator: services.Translator,
		},
	)

	listUC := NewListCollectionMethodEligibilityRulesUseCase(
		ListCollectionMethodEligibilityRulesRepositories(repositories),
		ListCollectionMethodEligibilityRulesServices{
			Authorizer: services.Authorizer,
			Transactor: services.Transactor,
			Translator: services.Translator,
		},
	)

	listPageDataUC := NewGetCollectionMethodEligibilityRuleListPageDataUseCase(
		GetCollectionMethodEligibilityRuleListPageDataRepositories(repositories),
		GetCollectionMethodEligibilityRuleListPageDataServices{
			Authorizer: services.Authorizer,
			Transactor: services.Transactor,
			Translator: services.Translator,
		},
	)

	itemPageDataUC := NewGetCollectionMethodEligibilityRuleItemPageDataUseCase(
		GetCollectionMethodEligibilityRuleItemPageDataRepositories(repositories),
		GetCollectionMethodEligibilityRuleItemPageDataServices{
			Authorizer: services.Authorizer,
			Transactor: services.Transactor,
			Translator: services.Translator,
		},
	)

	return &UseCases{
		CreateCollectionMethodEligibilityRule:          createUC,
		ReadCollectionMethodEligibilityRule:            readUC,
		UpdateCollectionMethodEligibilityRule:          updateUC,
		DeleteCollectionMethodEligibilityRule:          deleteUC,
		ListCollectionMethodEligibilityRules:           listUC,
		GetCollectionMethodEligibilityRuleListPageData: listPageDataUC,
		GetCollectionMethodEligibilityRuleItemPageData: itemPageDataUC,
	}
}

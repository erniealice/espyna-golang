package collectionmethodeligibilityrule

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	eligibilityrulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method_eligibility_rule"
)

// ListCollectionMethodEligibilityRulesRepositories groups all repository dependencies.
type ListCollectionMethodEligibilityRulesRepositories struct {
	CollectionMethodEligibilityRule eligibilityrulepb.CollectionMethodEligibilityRuleDomainServiceServer
}

// ListCollectionMethodEligibilityRulesServices groups all business service dependencies.
type ListCollectionMethodEligibilityRulesServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ListCollectionMethodEligibilityRulesUseCase handles the business logic for listing eligibility rules.
type ListCollectionMethodEligibilityRulesUseCase struct {
	repositories ListCollectionMethodEligibilityRulesRepositories
	services     ListCollectionMethodEligibilityRulesServices
}

// NewListCollectionMethodEligibilityRulesUseCase creates a new ListCollectionMethodEligibilityRulesUseCase.
func NewListCollectionMethodEligibilityRulesUseCase(
	repositories ListCollectionMethodEligibilityRulesRepositories,
	services ListCollectionMethodEligibilityRulesServices,
) *ListCollectionMethodEligibilityRulesUseCase {
	return &ListCollectionMethodEligibilityRulesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list eligibility rules operation.
func (uc *ListCollectionMethodEligibilityRulesUseCase) Execute(ctx context.Context, req *eligibilityrulepb.ListCollectionMethodEligibilityRulesRequest) (*eligibilityrulepb.ListCollectionMethodEligibilityRulesResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityCollectionMethodEligibilityRule, ports.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method_eligibility_rule.validation.request_required", "Request is required [DEFAULT]"))
	}

	if uc.repositories.CollectionMethodEligibilityRule == nil {
		return nil, errors.New("collection method eligibility rule repository is not available")
	}
	return uc.repositories.CollectionMethodEligibilityRule.ListCollectionMethodEligibilityRules(ctx, req)
}

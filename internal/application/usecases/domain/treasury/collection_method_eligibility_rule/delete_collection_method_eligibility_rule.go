package collectionmethodeligibilityrule

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	eligibilityrulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method_eligibility_rule"
)

// DeleteCollectionMethodEligibilityRuleRepositories groups all repository dependencies.
type DeleteCollectionMethodEligibilityRuleRepositories struct {
	CollectionMethodEligibilityRule eligibilityrulepb.CollectionMethodEligibilityRuleDomainServiceServer
}

// DeleteCollectionMethodEligibilityRuleServices groups all business service dependencies.
type DeleteCollectionMethodEligibilityRuleServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// DeleteCollectionMethodEligibilityRuleUseCase handles the business logic for deleting eligibility rules.
type DeleteCollectionMethodEligibilityRuleUseCase struct {
	repositories DeleteCollectionMethodEligibilityRuleRepositories
	services     DeleteCollectionMethodEligibilityRuleServices
}

// NewDeleteCollectionMethodEligibilityRuleUseCase creates a new DeleteCollectionMethodEligibilityRuleUseCase.
func NewDeleteCollectionMethodEligibilityRuleUseCase(
	repositories DeleteCollectionMethodEligibilityRuleRepositories,
	services DeleteCollectionMethodEligibilityRuleServices,
) *DeleteCollectionMethodEligibilityRuleUseCase {
	return &DeleteCollectionMethodEligibilityRuleUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete eligibility rule operation.
func (uc *DeleteCollectionMethodEligibilityRuleUseCase) Execute(ctx context.Context, req *eligibilityrulepb.DeleteCollectionMethodEligibilityRuleRequest) (*eligibilityrulepb.DeleteCollectionMethodEligibilityRuleResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityCollectionMethodEligibilityRule, ports.ActionDelete); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method_eligibility_rule.validation.id_required", "Collection method eligibility rule ID is required [DEFAULT]"))
	}

	if uc.repositories.CollectionMethodEligibilityRule == nil {
		return nil, errors.New("collection method eligibility rule repository is not available")
	}
	return uc.repositories.CollectionMethodEligibilityRule.DeleteCollectionMethodEligibilityRule(ctx, req)
}

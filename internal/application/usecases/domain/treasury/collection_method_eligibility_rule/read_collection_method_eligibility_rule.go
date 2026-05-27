package collectionmethodeligibilityrule

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	eligibilityrulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method_eligibility_rule"
)

// ReadCollectionMethodEligibilityRuleRepositories groups all repository dependencies.
type ReadCollectionMethodEligibilityRuleRepositories struct {
	CollectionMethodEligibilityRule eligibilityrulepb.CollectionMethodEligibilityRuleDomainServiceServer
}

// ReadCollectionMethodEligibilityRuleServices groups all business service dependencies.
type ReadCollectionMethodEligibilityRuleServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ReadCollectionMethodEligibilityRuleUseCase handles the business logic for reading an eligibility rule.
type ReadCollectionMethodEligibilityRuleUseCase struct {
	repositories ReadCollectionMethodEligibilityRuleRepositories
	services     ReadCollectionMethodEligibilityRuleServices
}

// NewReadCollectionMethodEligibilityRuleUseCase creates use case with grouped dependencies.
func NewReadCollectionMethodEligibilityRuleUseCase(
	repositories ReadCollectionMethodEligibilityRuleRepositories,
	services ReadCollectionMethodEligibilityRuleServices,
) *ReadCollectionMethodEligibilityRuleUseCase {
	return &ReadCollectionMethodEligibilityRuleUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read eligibility rule operation.
func (uc *ReadCollectionMethodEligibilityRuleUseCase) Execute(ctx context.Context, req *eligibilityrulepb.ReadCollectionMethodEligibilityRuleRequest) (*eligibilityrulepb.ReadCollectionMethodEligibilityRuleResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityCollectionMethodEligibilityRule, ports.ActionRead); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method_eligibility_rule.validation.id_required", "Collection method eligibility rule ID is required [DEFAULT]"))
	}

	if uc.repositories.CollectionMethodEligibilityRule == nil {
		return nil, errors.New("collection method eligibility rule repository is not available")
	}
	return uc.repositories.CollectionMethodEligibilityRule.ReadCollectionMethodEligibilityRule(ctx, req)
}

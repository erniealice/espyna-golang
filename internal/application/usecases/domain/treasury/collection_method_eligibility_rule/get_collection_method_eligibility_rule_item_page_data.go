package collectionmethodeligibilityrule

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	eligibilityrulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method_eligibility_rule"
)

// GetCollectionMethodEligibilityRuleItemPageDataRepositories groups all repository dependencies.
type GetCollectionMethodEligibilityRuleItemPageDataRepositories struct {
	CollectionMethodEligibilityRule eligibilityrulepb.CollectionMethodEligibilityRuleDomainServiceServer
}

// GetCollectionMethodEligibilityRuleItemPageDataServices groups all business service dependencies.
type GetCollectionMethodEligibilityRuleItemPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetCollectionMethodEligibilityRuleItemPageDataUseCase handles fetching a single enriched item.
type GetCollectionMethodEligibilityRuleItemPageDataUseCase struct {
	repositories GetCollectionMethodEligibilityRuleItemPageDataRepositories
	services     GetCollectionMethodEligibilityRuleItemPageDataServices
}

// NewGetCollectionMethodEligibilityRuleItemPageDataUseCase creates use case with grouped dependencies.
func NewGetCollectionMethodEligibilityRuleItemPageDataUseCase(
	repositories GetCollectionMethodEligibilityRuleItemPageDataRepositories,
	services GetCollectionMethodEligibilityRuleItemPageDataServices,
) *GetCollectionMethodEligibilityRuleItemPageDataUseCase {
	return &GetCollectionMethodEligibilityRuleItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get eligibility rule item page data operation.
func (uc *GetCollectionMethodEligibilityRuleItemPageDataUseCase) Execute(ctx context.Context, req *eligibilityrulepb.GetCollectionMethodEligibilityRuleItemPageDataRequest) (*eligibilityrulepb.GetCollectionMethodEligibilityRuleItemPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityCollectionMethodEligibilityRule, ports.ActionRead); err != nil {
		return nil, err
	}

	if req == nil || req.CollectionMethodEligibilityRuleId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method_eligibility_rule.validation.id_required", "Collection method eligibility rule ID is required [DEFAULT]"))
	}

	if uc.repositories.CollectionMethodEligibilityRule == nil {
		return nil, errors.New("collection method eligibility rule repository is not available")
	}
	resp, err := uc.repositories.CollectionMethodEligibilityRule.GetCollectionMethodEligibilityRuleItemPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method_eligibility_rule.errors.get_item_page_data_failed", "[ERR-DEFAULT] Failed to load collection method eligibility rule")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

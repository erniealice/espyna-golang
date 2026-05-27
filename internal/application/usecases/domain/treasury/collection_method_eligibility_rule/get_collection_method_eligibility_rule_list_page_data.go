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

// GetCollectionMethodEligibilityRuleListPageDataRepositories groups all repository dependencies.
type GetCollectionMethodEligibilityRuleListPageDataRepositories struct {
	CollectionMethodEligibilityRule eligibilityrulepb.CollectionMethodEligibilityRuleDomainServiceServer
}

// GetCollectionMethodEligibilityRuleListPageDataServices groups all business service dependencies.
type GetCollectionMethodEligibilityRuleListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetCollectionMethodEligibilityRuleListPageDataUseCase handles fetching paginated, searchable list data.
type GetCollectionMethodEligibilityRuleListPageDataUseCase struct {
	repositories GetCollectionMethodEligibilityRuleListPageDataRepositories
	services     GetCollectionMethodEligibilityRuleListPageDataServices
}

// NewGetCollectionMethodEligibilityRuleListPageDataUseCase creates use case with grouped dependencies.
func NewGetCollectionMethodEligibilityRuleListPageDataUseCase(
	repositories GetCollectionMethodEligibilityRuleListPageDataRepositories,
	services GetCollectionMethodEligibilityRuleListPageDataServices,
) *GetCollectionMethodEligibilityRuleListPageDataUseCase {
	return &GetCollectionMethodEligibilityRuleListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get eligibility rule list page data operation.
func (uc *GetCollectionMethodEligibilityRuleListPageDataUseCase) Execute(ctx context.Context, req *eligibilityrulepb.GetCollectionMethodEligibilityRuleListPageDataRequest) (*eligibilityrulepb.GetCollectionMethodEligibilityRuleListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityCollectionMethodEligibilityRule, ports.ActionList); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method_eligibility_rule.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	if uc.repositories.CollectionMethodEligibilityRule == nil {
		return nil, errors.New("collection method eligibility rule repository is not available")
	}
	resp, err := uc.repositories.CollectionMethodEligibilityRule.GetCollectionMethodEligibilityRuleListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method_eligibility_rule.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load collection method eligibility rule list")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

func (uc *GetCollectionMethodEligibilityRuleListPageDataUseCase) validateInput(ctx context.Context, req *eligibilityrulepb.GetCollectionMethodEligibilityRuleListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method_eligibility_rule.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Pagination != nil && req.Pagination.Limit > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method_eligibility_rule.validation.invalid_pagination_limit", "[ERR-DEFAULT] Invalid pagination limit"))
	}
	if req.Search != nil && len(req.Search.Query) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method_eligibility_rule.validation.search_query_too_long", "[ERR-DEFAULT] Search query is too long"))
	}
	return nil
}

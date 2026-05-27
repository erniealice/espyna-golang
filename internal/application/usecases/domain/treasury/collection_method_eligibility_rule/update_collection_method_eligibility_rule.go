package collectionmethodeligibilityrule

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	eligibilityrulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method_eligibility_rule"
)

// UpdateCollectionMethodEligibilityRuleRepositories groups all repository dependencies.
type UpdateCollectionMethodEligibilityRuleRepositories struct {
	CollectionMethodEligibilityRule eligibilityrulepb.CollectionMethodEligibilityRuleDomainServiceServer
}

// UpdateCollectionMethodEligibilityRuleServices groups all business service dependencies.
type UpdateCollectionMethodEligibilityRuleServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// UpdateCollectionMethodEligibilityRuleUseCase handles the business logic for updating eligibility rules.
type UpdateCollectionMethodEligibilityRuleUseCase struct {
	repositories UpdateCollectionMethodEligibilityRuleRepositories
	services     UpdateCollectionMethodEligibilityRuleServices
}

// NewUpdateCollectionMethodEligibilityRuleUseCase creates use case with grouped dependencies.
func NewUpdateCollectionMethodEligibilityRuleUseCase(
	repositories UpdateCollectionMethodEligibilityRuleRepositories,
	services UpdateCollectionMethodEligibilityRuleServices,
) *UpdateCollectionMethodEligibilityRuleUseCase {
	return &UpdateCollectionMethodEligibilityRuleUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update eligibility rule operation.
//
// Transaction-aware: when called from inside an outer ExecuteInTransaction it
// short-circuits to executeCore so no nested independent tx is started.
func (uc *UpdateCollectionMethodEligibilityRuleUseCase) Execute(ctx context.Context, req *eligibilityrulepb.UpdateCollectionMethodEligibilityRuleRequest) (*eligibilityrulepb.UpdateCollectionMethodEligibilityRuleResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityCollectionMethodEligibilityRule, ports.ActionUpdate); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		if uc.services.Transactor.IsTransactionActive(ctx) {
			return uc.executeCore(ctx, req)
		}
		var result *eligibilityrulepb.UpdateCollectionMethodEligibilityRuleResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("collection method eligibility rule update failed: %w", err)
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

func (uc *UpdateCollectionMethodEligibilityRuleUseCase) executeCore(ctx context.Context, req *eligibilityrulepb.UpdateCollectionMethodEligibilityRuleRequest) (*eligibilityrulepb.UpdateCollectionMethodEligibilityRuleResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method_eligibility_rule.validation.id_required", "Collection method eligibility rule ID is required [DEFAULT]"))
	}

	req.Data.Name = strings.TrimSpace(req.Data.Name)
	if req.Data.Name == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method_eligibility_rule.validation.name_required", "[ERR-DEFAULT] Name is required"))
	}

	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	if uc.repositories.CollectionMethodEligibilityRule == nil {
		return nil, errors.New("collection method eligibility rule repository is not available")
	}
	return uc.repositories.CollectionMethodEligibilityRule.UpdateCollectionMethodEligibilityRule(ctx, req)
}

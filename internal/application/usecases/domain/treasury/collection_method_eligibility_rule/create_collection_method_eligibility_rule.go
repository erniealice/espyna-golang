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

// entityCollectionMethodEligibilityRule is the permission namespace + translation key root.
const entityCollectionMethodEligibilityRule = "collection_method_eligibility_rule"

// CreateCollectionMethodEligibilityRuleRepositories groups all repository dependencies.
type CreateCollectionMethodEligibilityRuleRepositories struct {
	CollectionMethodEligibilityRule eligibilityrulepb.CollectionMethodEligibilityRuleDomainServiceServer
}

// CreateCollectionMethodEligibilityRuleServices groups all business service dependencies.
type CreateCollectionMethodEligibilityRuleServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// CreateCollectionMethodEligibilityRuleUseCase handles the business logic for creating eligibility rules.
type CreateCollectionMethodEligibilityRuleUseCase struct {
	repositories CreateCollectionMethodEligibilityRuleRepositories
	services     CreateCollectionMethodEligibilityRuleServices
}

// NewCreateCollectionMethodEligibilityRuleUseCase creates use case with grouped dependencies.
func NewCreateCollectionMethodEligibilityRuleUseCase(
	repositories CreateCollectionMethodEligibilityRuleRepositories,
	services CreateCollectionMethodEligibilityRuleServices,
) *CreateCollectionMethodEligibilityRuleUseCase {
	return &CreateCollectionMethodEligibilityRuleUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create eligibility rule operation.
func (uc *CreateCollectionMethodEligibilityRuleUseCase) Execute(ctx context.Context, req *eligibilityrulepb.CreateCollectionMethodEligibilityRuleRequest) (*eligibilityrulepb.CreateCollectionMethodEligibilityRuleResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityCollectionMethodEligibilityRule, ports.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *eligibilityrulepb.CreateCollectionMethodEligibilityRuleResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "collection_method_eligibility_rule.errors.creation_failed", "Collection method eligibility rule creation failed [DEFAULT]")
				return fmt.Errorf("%s: %w", translatedError, err)
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

func (uc *CreateCollectionMethodEligibilityRuleUseCase) executeCore(ctx context.Context, req *eligibilityrulepb.CreateCollectionMethodEligibilityRuleRequest) (*eligibilityrulepb.CreateCollectionMethodEligibilityRuleResponse, error) {
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	if err := uc.enrichData(req.Data); err != nil {
		return nil, err
	}

	if uc.repositories.CollectionMethodEligibilityRule == nil {
		return nil, errors.New("collection method eligibility rule repository is not available")
	}
	return uc.repositories.CollectionMethodEligibilityRule.CreateCollectionMethodEligibilityRule(ctx, req)
}

func (uc *CreateCollectionMethodEligibilityRuleUseCase) validateInput(ctx context.Context, req *eligibilityrulepb.CreateCollectionMethodEligibilityRuleRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method_eligibility_rule.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method_eligibility_rule.validation.data_required", "[ERR-DEFAULT] Collection method eligibility rule data is required"))
	}

	req.Data.Name = strings.TrimSpace(req.Data.Name)
	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method_eligibility_rule.validation.name_required", "[ERR-DEFAULT] Name is required"))
	}

	return nil
}

func (uc *CreateCollectionMethodEligibilityRuleUseCase) enrichData(rule *eligibilityrulepb.CollectionMethodEligibilityRule) error {
	now := time.Now()
	if rule.Id == "" {
		rule.Id = uc.services.IDGenerator.GenerateID()
	}
	rule.DateCreated = &[]int64{now.UnixMilli()}[0]
	rule.DateModified = &[]int64{now.UnixMilli()}[0]
	rule.Active = true
	return nil
}

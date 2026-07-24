package subscription_group_product_plan

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	subscriptiongrouppb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan"
)

type CreateSubscriptionGroupProductPlanRepositories struct {
	SubscriptionGroupProductPlan pb.SubscriptionGroupProductPlanDomainServiceServer
	// Class-invariant guard anchors (plan.md §1.2 #1-3).
	ProductPlan       productplanpb.ProductPlanDomainServiceServer
	SubscriptionGroup subscriptiongrouppb.SubscriptionGroupDomainServiceServer
	// JobTemplate is best-effort (composition may leave it nil); see validation.go.
	JobTemplate jobtemplatepb.JobTemplateDomainServiceServer
}

type CreateSubscriptionGroupProductPlanServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type CreateSubscriptionGroupProductPlanUseCase struct {
	repositories CreateSubscriptionGroupProductPlanRepositories
	services     CreateSubscriptionGroupProductPlanServices
}

func NewCreateSubscriptionGroupProductPlanUseCase(r CreateSubscriptionGroupProductPlanRepositories, s CreateSubscriptionGroupProductPlanServices) *CreateSubscriptionGroupProductPlanUseCase {
	return &CreateSubscriptionGroupProductPlanUseCase{repositories: r, services: s}
}

func (uc *CreateSubscriptionGroupProductPlanUseCase) Execute(ctx context.Context, req *pb.CreateSubscriptionGroupProductPlanRequest) (*pb.CreateSubscriptionGroupProductPlanResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.SubscriptionGroupProductPlan, Action: entityid.ActionCreate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_product_plan.validation.data_required", "Data is required [DEFAULT]"))
	}

	// Fail-closed invariant guard + friendly duplicate check, run in the SAME
	// transaction as the write so a concurrent add-offering race cannot slip an
	// invalid or duplicate class row in between the check and the insert
	// (mirrors the sgpps eligibility-guard atomicity rationale).
	wsID := contextutil.ExtractWorkspaceIDFromContext(ctx)
	uc.enrich(req.Data)

	writeFn := func(txCtx context.Context) (*pb.CreateSubscriptionGroupProductPlanResponse, error) {
		if err := validateClassInvariants(txCtx, uc.invariantRepos(), uc.services.Translator, wsID, req.Data); err != nil {
			return nil, err
		}
		if err := checkDuplicateClass(txCtx, uc.repositories.SubscriptionGroupProductPlan, uc.services.Translator, wsID, req.Data.GetSubscriptionGroupId(), req.Data.GetProductPlanId(), ""); err != nil {
			return nil, err
		}
		return uc.repositories.SubscriptionGroupProductPlan.CreateSubscriptionGroupProductPlan(txCtx, req)
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *pb.CreateSubscriptionGroupProductPlanResponse
		if err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := writeFn(txCtx)
			if err != nil {
				return err
			}
			result = res
			return nil
		}); err != nil {
			return nil, err
		}
		return result, nil
	}
	return writeFn(ctx)
}

func (uc *CreateSubscriptionGroupProductPlanUseCase) invariantRepos() classInvariantRepos {
	return classInvariantRepos{
		ProductPlan:       uc.repositories.ProductPlan,
		SubscriptionGroup: uc.repositories.SubscriptionGroup,
		JobTemplate:       uc.repositories.JobTemplate,
	}
}

// enrich stamps audit fields and defaults status=ACTIVE (plan.md §2: "checked
// lines → sgpp rows created, status ACTIVE, no staff") — the entity-status
// convention (active bool + status enum must stay in sync; a NULL/UNSPECIFIED
// status would silently drop the row from any status-filtered list view).
func (uc *CreateSubscriptionGroupProductPlanUseCase) enrich(data *pb.SubscriptionGroupProductPlan) {
	now := time.Now()
	if data.Id == "" && uc.services.IDGenerator != nil {
		data.Id = uc.services.IDGenerator.GenerateID()
	}
	data.Active = true
	if data.Status == pb.SubscriptionGroupProductPlanStatus_SUBSCRIPTION_GROUP_PRODUCT_PLAN_STATUS_UNSPECIFIED {
		data.Status = pb.SubscriptionGroupProductPlanStatus_SUBSCRIPTION_GROUP_PRODUCT_PLAN_STATUS_ACTIVE
	}
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	data.DateCreated = &ms
	data.DateCreatedString = &s
	data.DateModified = &ms
	data.DateModifiedString = &s
}

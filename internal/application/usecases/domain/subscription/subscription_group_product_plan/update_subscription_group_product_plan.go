package subscription_group_product_plan

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	espynaports "github.com/erniealice/espyna-golang/ports"
	"github.com/erniealice/espyna-golang/registry/entityid"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	subscriptiongrouppb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan"
)

type UpdateSubscriptionGroupProductPlanRepositories struct {
	SubscriptionGroupProductPlan pb.SubscriptionGroupProductPlanDomainServiceServer
	ProductPlan                  productplanpb.ProductPlanDomainServiceServer
	SubscriptionGroup            subscriptiongrouppb.SubscriptionGroupDomainServiceServer
	JobTemplate                  jobtemplatepb.JobTemplateDomainServiceServer
}

type UpdateSubscriptionGroupProductPlanServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
	// ReferenceChecker guards the ACTIVE->EXCLUDED transition (plan.md §2
	// "Delete/Exclude guards: refuse when active assignments exist"). Optional:
	// a nil checker degrades to un-guarded here (the Delete use case's own
	// dependent-count guard is the hard backstop; mirrors plan.UpdatePlanServices).
	ReferenceChecker espynaports.Checker
}

type UpdateSubscriptionGroupProductPlanUseCase struct {
	repositories UpdateSubscriptionGroupProductPlanRepositories
	services     UpdateSubscriptionGroupProductPlanServices
}

func NewUpdateSubscriptionGroupProductPlanUseCase(r UpdateSubscriptionGroupProductPlanRepositories, s UpdateSubscriptionGroupProductPlanServices) *UpdateSubscriptionGroupProductPlanUseCase {
	return &UpdateSubscriptionGroupProductPlanUseCase{repositories: r, services: s}
}

func (uc *UpdateSubscriptionGroupProductPlanUseCase) Execute(ctx context.Context, req *pb.UpdateSubscriptionGroupProductPlanRequest) (*pb.UpdateSubscriptionGroupProductPlanResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.SubscriptionGroupProductPlan, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.GetId() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_product_plan.validation.data_required", "Data is required [DEFAULT]"))
	}

	wsID := contextutil.ExtractWorkspaceIDFromContext(ctx)
	now := time.Now()
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	req.Data.DateModified = &ms
	req.Data.DateModifiedString = &s

	writeFn := func(txCtx context.Context) (*pb.UpdateSubscriptionGroupProductPlanResponse, error) {
		existing, effective, err := uc.effectiveClass(txCtx, req.Data)
		if err != nil {
			return nil, err
		}
		if err := validateClassInvariants(txCtx, uc.invariantRepos(), uc.services.Translator, wsID, effective); err != nil {
			return nil, err
		}
		if err := checkDuplicateClass(txCtx, uc.repositories.SubscriptionGroupProductPlan, uc.services.Translator, wsID, effective.GetSubscriptionGroupId(), effective.GetProductPlanId(), effective.GetId()); err != nil {
			return nil, err
		}
		// Exclude guard (plan.md §2): refuse the ACTIVE->EXCLUDED transition while
		// active assignments (or, once wired, live realized courses) reference the
		// class — mirrors the Delete guard so both write paths share one rule.
		if isExcludeTransition(existing, effective) {
			if err := guardNotInUse(txCtx, uc.services.ReferenceChecker, uc.services.Translator, effective.GetId()); err != nil {
				return nil, err
			}
		}
		return uc.repositories.SubscriptionGroupProductPlan.UpdateSubscriptionGroupProductPlan(txCtx, req)
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *pb.UpdateSubscriptionGroupProductPlanResponse
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

func (uc *UpdateSubscriptionGroupProductPlanUseCase) invariantRepos() classInvariantRepos {
	return classInvariantRepos{
		ProductPlan:       uc.repositories.ProductPlan,
		SubscriptionGroup: uc.repositories.SubscriptionGroup,
		JobTemplate:       uc.repositories.JobTemplate,
	}
}

// effectiveClass reads the persisted row and merges the update body over it, so
// a partial update (FK/status omitted) still validates against — and guards —
// the resulting class. Returns (existing, effective).
func (uc *UpdateSubscriptionGroupProductPlanUseCase) effectiveClass(ctx context.Context, in *pb.SubscriptionGroupProductPlan) (*pb.SubscriptionGroupProductPlan, *pb.SubscriptionGroupProductPlan, error) {
	existingResp, err := uc.repositories.SubscriptionGroupProductPlan.ReadSubscriptionGroupProductPlan(ctx, &pb.ReadSubscriptionGroupProductPlanRequest{
		Data: &pb.SubscriptionGroupProductPlan{Id: in.GetId()},
	})
	if err != nil || existingResp == nil || len(existingResp.GetData()) == 0 {
		return nil, nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_product_plan.errors.not_found", "class not found [DEFAULT]"))
	}
	existing := existingResp.GetData()[0]
	effective := &pb.SubscriptionGroupProductPlan{
		Id:                  in.GetId(),
		SubscriptionGroupId: firstNonEmpty(in.GetSubscriptionGroupId(), existing.GetSubscriptionGroupId()),
		ProductPlanId:       firstNonEmpty(in.GetProductPlanId(), existing.GetProductPlanId()),
		JobTemplateId:       firstNonEmpty(in.GetJobTemplateId(), existing.GetJobTemplateId()),
		Status:              in.GetStatus(),
	}
	if in.Status == pb.SubscriptionGroupProductPlanStatus_SUBSCRIPTION_GROUP_PRODUCT_PLAN_STATUS_UNSPECIFIED {
		effective.Status = existing.GetStatus()
	}
	return existing, effective, nil
}

// isExcludeTransition reports whether this update moves the class from a
// non-EXCLUDED status to EXCLUDED (plan.md §2 "Offering exclude (old I-4) ->
// sgpp.status = EXCLUDED ... Guards carry").
func isExcludeTransition(existing, effective *pb.SubscriptionGroupProductPlan) bool {
	if existing == nil || effective == nil {
		return false
	}
	excluded := pb.SubscriptionGroupProductPlanStatus_SUBSCRIPTION_GROUP_PRODUCT_PLAN_STATUS_EXCLUDED
	return existing.GetStatus() != excluded && effective.GetStatus() == excluded
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

package subscription_group_product_plan_staff

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	productplanstaffpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan_staff"
	subscriptiongrouppb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan_staff"
)

type UpdateSubscriptionGroupProductPlanStaffRepositories struct {
	SubscriptionGroupProductPlanStaff pb.SubscriptionGroupProductPlanStaffDomainServiceServer
	ProductPlanStaff                  productplanstaffpb.ProductPlanStaffDomainServiceServer
	ProductPlan                       productplanpb.ProductPlanDomainServiceServer
	SubscriptionGroup                 subscriptiongrouppb.SubscriptionGroupDomainServiceServer
}

type UpdateSubscriptionGroupProductPlanStaffServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type UpdateSubscriptionGroupProductPlanStaffUseCase struct {
	repositories UpdateSubscriptionGroupProductPlanStaffRepositories
	services     UpdateSubscriptionGroupProductPlanStaffServices
}

func NewUpdateSubscriptionGroupProductPlanStaffUseCase(r UpdateSubscriptionGroupProductPlanStaffRepositories, s UpdateSubscriptionGroupProductPlanStaffServices) *UpdateSubscriptionGroupProductPlanStaffUseCase {
	return &UpdateSubscriptionGroupProductPlanStaffUseCase{repositories: r, services: s}
}

func (uc *UpdateSubscriptionGroupProductPlanStaffUseCase) Execute(ctx context.Context, req *pb.UpdateSubscriptionGroupProductPlanStaffRequest) (*pb.UpdateSubscriptionGroupProductPlanStaffResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.SubscriptionGroupProductPlanStaff, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.GetId() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_product_plan_staff.validation.data_required", "Data is required [DEFAULT]"))
	}

	// Fail-closed eligibility guard on the effective (post-update) edge, run in
	// the SAME transaction as the write so the merge-read, the eligibility check
	// and the update commit atomically — a concurrent deactivation of the
	// matching product_plan_staff row cannot slip an ineligible edge through the
	// TOCTOU window (red-team MED). FKs are mutable: merge any FK absent from the
	// update body with the existing row, then re-prove eligibility + plan match +
	// single-workspace tenancy.
	wsID := contextutil.ExtractWorkspaceIDFromContext(ctx)
	now := time.Now()
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	req.Data.DateModified = &ms
	req.Data.DateModifiedString = &s

	writeFn := func(txCtx context.Context) (*pb.UpdateSubscriptionGroupProductPlanStaffResponse, error) {
		effective, err := uc.effectiveEdge(txCtx, req.Data)
		if err != nil {
			return nil, err
		}
		if err := validateEligibility(txCtx, uc.eligibilityRepos(), uc.services.Translator, wsID, effective); err != nil {
			return nil, err
		}
		return uc.repositories.SubscriptionGroupProductPlanStaff.UpdateSubscriptionGroupProductPlanStaff(txCtx, req)
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *pb.UpdateSubscriptionGroupProductPlanStaffResponse
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

func (uc *UpdateSubscriptionGroupProductPlanStaffUseCase) eligibilityRepos() eligibilityRepos {
	return eligibilityRepos{
		ProductPlanStaff:  uc.repositories.ProductPlanStaff,
		ProductPlan:       uc.repositories.ProductPlan,
		SubscriptionGroup: uc.repositories.SubscriptionGroup,
	}
}

// effectiveEdge merges the update body over the persisted row so a partial
// update (FK omitted) still validates against the resulting edge.
func (uc *UpdateSubscriptionGroupProductPlanStaffUseCase) effectiveEdge(ctx context.Context, in *pb.SubscriptionGroupProductPlanStaff) (*pb.SubscriptionGroupProductPlanStaff, error) {
	existingResp, err := uc.repositories.SubscriptionGroupProductPlanStaff.ReadSubscriptionGroupProductPlanStaff(ctx, &pb.ReadSubscriptionGroupProductPlanStaffRequest{
		Data: &pb.SubscriptionGroupProductPlanStaff{Id: in.GetId()},
	})
	if err != nil || existingResp == nil || len(existingResp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_product_plan_staff.errors.not_found", "class edge not found [DEFAULT]"))
	}
	existing := existingResp.GetData()[0]
	merged := &pb.SubscriptionGroupProductPlanStaff{
		Id:                  in.GetId(),
		SubscriptionGroupId: firstNonEmpty(in.GetSubscriptionGroupId(), existing.GetSubscriptionGroupId()),
		ProductPlanId:       firstNonEmpty(in.GetProductPlanId(), existing.GetProductPlanId()),
		StaffId:             firstNonEmpty(in.GetStaffId(), existing.GetStaffId()),
	}
	return merged, nil
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

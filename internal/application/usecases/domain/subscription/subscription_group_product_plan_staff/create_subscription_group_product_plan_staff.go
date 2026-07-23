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

type CreateSubscriptionGroupProductPlanStaffRepositories struct {
	SubscriptionGroupProductPlanStaff pb.SubscriptionGroupProductPlanStaffDomainServiceServer
	// Eligibility guard anchors (red-team HIGH #5).
	ProductPlanStaff  productplanstaffpb.ProductPlanStaffDomainServiceServer
	ProductPlan       productplanpb.ProductPlanDomainServiceServer
	SubscriptionGroup subscriptiongrouppb.SubscriptionGroupDomainServiceServer
}

type CreateSubscriptionGroupProductPlanStaffServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type CreateSubscriptionGroupProductPlanStaffUseCase struct {
	repositories CreateSubscriptionGroupProductPlanStaffRepositories
	services     CreateSubscriptionGroupProductPlanStaffServices
}

func NewCreateSubscriptionGroupProductPlanStaffUseCase(r CreateSubscriptionGroupProductPlanStaffRepositories, s CreateSubscriptionGroupProductPlanStaffServices) *CreateSubscriptionGroupProductPlanStaffUseCase {
	return &CreateSubscriptionGroupProductPlanStaffUseCase{repositories: r, services: s}
}

func (uc *CreateSubscriptionGroupProductPlanStaffUseCase) Execute(ctx context.Context, req *pb.CreateSubscriptionGroupProductPlanStaffRequest) (*pb.CreateSubscriptionGroupProductPlanStaffResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.SubscriptionGroupProductPlanStaff, Action: entityid.ActionCreate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_product_plan_staff.validation.data_required", "Data is required [DEFAULT]"))
	}

	// Fail-closed eligibility guard, run in the SAME transaction as the write so a
	// concurrent deactivation/deletion of the matching product_plan_staff row
	// cannot slip an ineligible edge in between the check and the insert
	// (red-team MED — eligibility snapshot not atomic). enrich() is a pure
	// in-memory step, so it runs before the transaction.
	wsID := contextutil.ExtractWorkspaceIDFromContext(ctx)
	uc.enrich(req.Data)

	writeFn := func(txCtx context.Context) (*pb.CreateSubscriptionGroupProductPlanStaffResponse, error) {
		if err := validateEligibility(txCtx, uc.eligibilityRepos(), uc.services.Translator, wsID, req.Data); err != nil {
			return nil, err
		}
		return uc.repositories.SubscriptionGroupProductPlanStaff.CreateSubscriptionGroupProductPlanStaff(txCtx, req)
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *pb.CreateSubscriptionGroupProductPlanStaffResponse
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

func (uc *CreateSubscriptionGroupProductPlanStaffUseCase) eligibilityRepos() eligibilityRepos {
	return eligibilityRepos{
		ProductPlanStaff:  uc.repositories.ProductPlanStaff,
		ProductPlan:       uc.repositories.ProductPlan,
		SubscriptionGroup: uc.repositories.SubscriptionGroup,
	}
}

func (uc *CreateSubscriptionGroupProductPlanStaffUseCase) enrich(data *pb.SubscriptionGroupProductPlanStaff) {
	now := time.Now()
	if data.Id == "" && uc.services.IDGenerator != nil {
		data.Id = uc.services.IDGenerator.GenerateID()
	}
	data.Active = true
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	data.DateCreated = &ms
	data.DateCreatedString = &s
	data.DateModified = &ms
	data.DateModifiedString = &s
}

package subscription_group_product_plan_staff

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan_staff"
)

type DeleteSubscriptionGroupProductPlanStaffRepositories struct {
	SubscriptionGroupProductPlanStaff pb.SubscriptionGroupProductPlanStaffDomainServiceServer
}

type DeleteSubscriptionGroupProductPlanStaffServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type DeleteSubscriptionGroupProductPlanStaffUseCase struct {
	repositories DeleteSubscriptionGroupProductPlanStaffRepositories
	services     DeleteSubscriptionGroupProductPlanStaffServices
}

func NewDeleteSubscriptionGroupProductPlanStaffUseCase(r DeleteSubscriptionGroupProductPlanStaffRepositories, s DeleteSubscriptionGroupProductPlanStaffServices) *DeleteSubscriptionGroupProductPlanStaffUseCase {
	return &DeleteSubscriptionGroupProductPlanStaffUseCase{repositories: r, services: s}
}

func (uc *DeleteSubscriptionGroupProductPlanStaffUseCase) Execute(ctx context.Context, req *pb.DeleteSubscriptionGroupProductPlanStaffRequest) (*pb.DeleteSubscriptionGroupProductPlanStaffResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.SubscriptionGroupProductPlanStaff, Action: entityid.ActionDelete}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_product_plan_staff.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.SubscriptionGroupProductPlanStaff.DeleteSubscriptionGroupProductPlanStaff(ctx, req)
}

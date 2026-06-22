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

type GetSubscriptionGroupProductPlanStaffItemPageDataRepositories struct {
	SubscriptionGroupProductPlanStaff pb.SubscriptionGroupProductPlanStaffDomainServiceServer
}

type GetSubscriptionGroupProductPlanStaffItemPageDataServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type GetSubscriptionGroupProductPlanStaffItemPageDataUseCase struct {
	repositories GetSubscriptionGroupProductPlanStaffItemPageDataRepositories
	services     GetSubscriptionGroupProductPlanStaffItemPageDataServices
}

func NewGetSubscriptionGroupProductPlanStaffItemPageDataUseCase(r GetSubscriptionGroupProductPlanStaffItemPageDataRepositories, s GetSubscriptionGroupProductPlanStaffItemPageDataServices) *GetSubscriptionGroupProductPlanStaffItemPageDataUseCase {
	return &GetSubscriptionGroupProductPlanStaffItemPageDataUseCase{repositories: r, services: s}
}

func (uc *GetSubscriptionGroupProductPlanStaffItemPageDataUseCase) Execute(ctx context.Context, req *pb.GetSubscriptionGroupProductPlanStaffItemPageDataRequest) (*pb.GetSubscriptionGroupProductPlanStaffItemPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.SubscriptionGroupProductPlanStaff, Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_product_plan_staff.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.SubscriptionGroupProductPlanStaff.GetSubscriptionGroupProductPlanStaffItemPageData(ctx, req)
}

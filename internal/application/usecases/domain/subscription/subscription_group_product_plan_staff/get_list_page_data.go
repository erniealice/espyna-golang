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

type GetSubscriptionGroupProductPlanStaffListPageDataRepositories struct {
	SubscriptionGroupProductPlanStaff pb.SubscriptionGroupProductPlanStaffDomainServiceServer
}

type GetSubscriptionGroupProductPlanStaffListPageDataServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type GetSubscriptionGroupProductPlanStaffListPageDataUseCase struct {
	repositories GetSubscriptionGroupProductPlanStaffListPageDataRepositories
	services     GetSubscriptionGroupProductPlanStaffListPageDataServices
}

func NewGetSubscriptionGroupProductPlanStaffListPageDataUseCase(r GetSubscriptionGroupProductPlanStaffListPageDataRepositories, s GetSubscriptionGroupProductPlanStaffListPageDataServices) *GetSubscriptionGroupProductPlanStaffListPageDataUseCase {
	return &GetSubscriptionGroupProductPlanStaffListPageDataUseCase{repositories: r, services: s}
}

func (uc *GetSubscriptionGroupProductPlanStaffListPageDataUseCase) Execute(ctx context.Context, req *pb.GetSubscriptionGroupProductPlanStaffListPageDataRequest) (*pb.GetSubscriptionGroupProductPlanStaffListPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.SubscriptionGroupProductPlanStaff, Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_product_plan_staff.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.SubscriptionGroupProductPlanStaff.GetSubscriptionGroupProductPlanStaffListPageData(ctx, req)
}

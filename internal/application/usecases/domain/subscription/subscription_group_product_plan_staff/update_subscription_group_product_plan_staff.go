package subscription_group_product_plan_staff

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan_staff"
)

type UpdateSubscriptionGroupProductPlanStaffRepositories struct {
	SubscriptionGroupProductPlanStaff pb.SubscriptionGroupProductPlanStaffDomainServiceServer
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
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_product_plan_staff.validation.data_required", "Data is required [DEFAULT]"))
	}
	now := time.Now()
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	req.Data.DateModified = &ms
	req.Data.DateModifiedString = &s
	return uc.repositories.SubscriptionGroupProductPlanStaff.UpdateSubscriptionGroupProductPlanStaff(ctx, req)
}

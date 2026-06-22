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

type CreateSubscriptionGroupProductPlanStaffRepositories struct {
	SubscriptionGroupProductPlanStaff pb.SubscriptionGroupProductPlanStaffDomainServiceServer
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
	uc.enrich(req.Data)
	return uc.repositories.SubscriptionGroupProductPlanStaff.CreateSubscriptionGroupProductPlanStaff(ctx, req)
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

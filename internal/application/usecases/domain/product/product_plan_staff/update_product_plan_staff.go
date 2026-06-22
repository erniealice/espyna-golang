package product_plan_staff

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan_staff"
)

type UpdateProductPlanStaffRepositories struct {
	ProductPlanStaff pb.ProductPlanStaffDomainServiceServer
}

type UpdateProductPlanStaffServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type UpdateProductPlanStaffUseCase struct {
	repositories UpdateProductPlanStaffRepositories
	services     UpdateProductPlanStaffServices
}

func NewUpdateProductPlanStaffUseCase(r UpdateProductPlanStaffRepositories, s UpdateProductPlanStaffServices) *UpdateProductPlanStaffUseCase {
	return &UpdateProductPlanStaffUseCase{repositories: r, services: s}
}

func (uc *UpdateProductPlanStaffUseCase) Execute(ctx context.Context, req *pb.UpdateProductPlanStaffRequest) (*pb.UpdateProductPlanStaffResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.ProductPlanStaff, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_plan_staff.validation.data_required", "Data is required [DEFAULT]"))
	}
	now := time.Now()
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	req.Data.DateModified = &ms
	req.Data.DateModifiedString = &s
	return uc.repositories.ProductPlanStaff.UpdateProductPlanStaff(ctx, req)
}

package product_plan_staff

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan_staff"
)

type ReadProductPlanStaffRepositories struct {
	ProductPlanStaff pb.ProductPlanStaffDomainServiceServer
}

type ReadProductPlanStaffServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type ReadProductPlanStaffUseCase struct {
	repositories ReadProductPlanStaffRepositories
	services     ReadProductPlanStaffServices
}

func NewReadProductPlanStaffUseCase(r ReadProductPlanStaffRepositories, s ReadProductPlanStaffServices) *ReadProductPlanStaffUseCase {
	return &ReadProductPlanStaffUseCase{repositories: r, services: s}
}

func (uc *ReadProductPlanStaffUseCase) Execute(ctx context.Context, req *pb.ReadProductPlanStaffRequest) (*pb.ReadProductPlanStaffResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.ProductPlanStaff, Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_plan_staff.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.ProductPlanStaff.ReadProductPlanStaff(ctx, req)
}

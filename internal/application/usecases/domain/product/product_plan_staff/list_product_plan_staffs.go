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

type ListProductPlanStaffsRepositories struct {
	ProductPlanStaff pb.ProductPlanStaffDomainServiceServer
}

type ListProductPlanStaffsServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type ListProductPlanStaffsUseCase struct {
	repositories ListProductPlanStaffsRepositories
	services     ListProductPlanStaffsServices
}

func NewListProductPlanStaffsUseCase(r ListProductPlanStaffsRepositories, s ListProductPlanStaffsServices) *ListProductPlanStaffsUseCase {
	return &ListProductPlanStaffsUseCase{repositories: r, services: s}
}

func (uc *ListProductPlanStaffsUseCase) Execute(ctx context.Context, req *pb.ListProductPlanStaffsRequest) (*pb.ListProductPlanStaffsResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.ProductPlanStaff, Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_plan_staff.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.ProductPlanStaff.ListProductPlanStaffs(ctx, req)
}

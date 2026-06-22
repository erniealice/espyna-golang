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

type CreateProductPlanStaffRepositories struct {
	ProductPlanStaff pb.ProductPlanStaffDomainServiceServer
}

type CreateProductPlanStaffServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type CreateProductPlanStaffUseCase struct {
	repositories CreateProductPlanStaffRepositories
	services     CreateProductPlanStaffServices
}

func NewCreateProductPlanStaffUseCase(r CreateProductPlanStaffRepositories, s CreateProductPlanStaffServices) *CreateProductPlanStaffUseCase {
	return &CreateProductPlanStaffUseCase{repositories: r, services: s}
}

func (uc *CreateProductPlanStaffUseCase) Execute(ctx context.Context, req *pb.CreateProductPlanStaffRequest) (*pb.CreateProductPlanStaffResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.ProductPlanStaff, Action: entityid.ActionCreate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_plan_staff.validation.data_required", "Data is required [DEFAULT]"))
	}
	uc.enrich(req.Data)
	return uc.repositories.ProductPlanStaff.CreateProductPlanStaff(ctx, req)
}

func (uc *CreateProductPlanStaffUseCase) enrich(data *pb.ProductPlanStaff) {
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

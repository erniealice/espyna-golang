package product_plan_staff

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan_staff"
)

type UseCases struct {
	CreateProductPlanStaff          *CreateProductPlanStaffUseCase
	ReadProductPlanStaff            *ReadProductPlanStaffUseCase
	UpdateProductPlanStaff          *UpdateProductPlanStaffUseCase
	DeleteProductPlanStaff          *DeleteProductPlanStaffUseCase
	ListProductPlanStaffs           *ListProductPlanStaffsUseCase
	GetProductPlanStaffListPageData *GetProductPlanStaffListPageDataUseCase
	GetProductPlanStaffItemPageData *GetProductPlanStaffItemPageDataUseCase
}

type Repositories struct {
	ProductPlanStaff pb.ProductPlanStaffDomainServiceServer
}

type Services struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

func NewUseCases(r Repositories, s Services) *UseCases {
	repo := r.ProductPlanStaff
	return &UseCases{
		CreateProductPlanStaff:          NewCreateProductPlanStaffUseCase(CreateProductPlanStaffRepositories{ProductPlanStaff: repo}, CreateProductPlanStaffServices(s)),
		ReadProductPlanStaff:            NewReadProductPlanStaffUseCase(ReadProductPlanStaffRepositories{ProductPlanStaff: repo}, ReadProductPlanStaffServices(s)),
		UpdateProductPlanStaff:          NewUpdateProductPlanStaffUseCase(UpdateProductPlanStaffRepositories{ProductPlanStaff: repo}, UpdateProductPlanStaffServices(s)),
		DeleteProductPlanStaff:          NewDeleteProductPlanStaffUseCase(DeleteProductPlanStaffRepositories{ProductPlanStaff: repo}, DeleteProductPlanStaffServices(s)),
		ListProductPlanStaffs:           NewListProductPlanStaffsUseCase(ListProductPlanStaffsRepositories{ProductPlanStaff: repo}, ListProductPlanStaffsServices(s)),
		GetProductPlanStaffListPageData: NewGetProductPlanStaffListPageDataUseCase(GetProductPlanStaffListPageDataRepositories{ProductPlanStaff: repo}, GetProductPlanStaffListPageDataServices(s)),
		GetProductPlanStaffItemPageData: NewGetProductPlanStaffItemPageDataUseCase(GetProductPlanStaffItemPageDataRepositories{ProductPlanStaff: repo}, GetProductPlanStaffItemPageDataServices(s)),
	}
}

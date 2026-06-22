package subscription_group_product_plan_staff

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan_staff"
)

type UseCases struct {
	CreateSubscriptionGroupProductPlanStaff          *CreateSubscriptionGroupProductPlanStaffUseCase
	ReadSubscriptionGroupProductPlanStaff            *ReadSubscriptionGroupProductPlanStaffUseCase
	UpdateSubscriptionGroupProductPlanStaff          *UpdateSubscriptionGroupProductPlanStaffUseCase
	DeleteSubscriptionGroupProductPlanStaff          *DeleteSubscriptionGroupProductPlanStaffUseCase
	ListSubscriptionGroupProductPlanStaffs           *ListSubscriptionGroupProductPlanStaffsUseCase
	GetSubscriptionGroupProductPlanStaffListPageData *GetSubscriptionGroupProductPlanStaffListPageDataUseCase
	GetSubscriptionGroupProductPlanStaffItemPageData *GetSubscriptionGroupProductPlanStaffItemPageDataUseCase
}

type Repositories struct {
	SubscriptionGroupProductPlanStaff pb.SubscriptionGroupProductPlanStaffDomainServiceServer
}

type Services struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

func NewUseCases(r Repositories, s Services) *UseCases {
	repo := r.SubscriptionGroupProductPlanStaff
	return &UseCases{
		CreateSubscriptionGroupProductPlanStaff:          NewCreateSubscriptionGroupProductPlanStaffUseCase(CreateSubscriptionGroupProductPlanStaffRepositories{SubscriptionGroupProductPlanStaff: repo}, CreateSubscriptionGroupProductPlanStaffServices(s)),
		ReadSubscriptionGroupProductPlanStaff:            NewReadSubscriptionGroupProductPlanStaffUseCase(ReadSubscriptionGroupProductPlanStaffRepositories{SubscriptionGroupProductPlanStaff: repo}, ReadSubscriptionGroupProductPlanStaffServices(s)),
		UpdateSubscriptionGroupProductPlanStaff:          NewUpdateSubscriptionGroupProductPlanStaffUseCase(UpdateSubscriptionGroupProductPlanStaffRepositories{SubscriptionGroupProductPlanStaff: repo}, UpdateSubscriptionGroupProductPlanStaffServices(s)),
		DeleteSubscriptionGroupProductPlanStaff:          NewDeleteSubscriptionGroupProductPlanStaffUseCase(DeleteSubscriptionGroupProductPlanStaffRepositories{SubscriptionGroupProductPlanStaff: repo}, DeleteSubscriptionGroupProductPlanStaffServices(s)),
		ListSubscriptionGroupProductPlanStaffs:           NewListSubscriptionGroupProductPlanStaffsUseCase(ListSubscriptionGroupProductPlanStaffsRepositories{SubscriptionGroupProductPlanStaff: repo}, ListSubscriptionGroupProductPlanStaffsServices(s)),
		GetSubscriptionGroupProductPlanStaffListPageData: NewGetSubscriptionGroupProductPlanStaffListPageDataUseCase(GetSubscriptionGroupProductPlanStaffListPageDataRepositories{SubscriptionGroupProductPlanStaff: repo}, GetSubscriptionGroupProductPlanStaffListPageDataServices(s)),
		GetSubscriptionGroupProductPlanStaffItemPageData: NewGetSubscriptionGroupProductPlanStaffItemPageDataUseCase(GetSubscriptionGroupProductPlanStaffItemPageDataRepositories{SubscriptionGroupProductPlanStaff: repo}, GetSubscriptionGroupProductPlanStaffItemPageDataServices(s)),
	}
}

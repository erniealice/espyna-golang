package subscription_group_product_plan_staff

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	jobtemplatephasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	productplanstaffpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan_staff"
	subscriptiongrouppb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group"
	sgpppb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan_staff"
)

type UseCases struct {
	CreateSubscriptionGroupProductPlanStaff          *CreateSubscriptionGroupProductPlanStaffUseCase
	ReadSubscriptionGroupProductPlanStaff            *ReadSubscriptionGroupProductPlanStaffUseCase
	UpdateSubscriptionGroupProductPlanStaff          *UpdateSubscriptionGroupProductPlanStaffUseCase
	DeleteSubscriptionGroupProductPlanStaff          *DeleteSubscriptionGroupProductPlanStaffUseCase
	AssignSubscriptionGroupProductPlanStaff          *AssignSubscriptionGroupProductPlanStaffUseCase
	ListSubscriptionGroupProductPlanStaffs           *ListSubscriptionGroupProductPlanStaffsUseCase
	GetSubscriptionGroupProductPlanStaffListPageData *GetSubscriptionGroupProductPlanStaffListPageDataUseCase
	GetSubscriptionGroupProductPlanStaffItemPageData *GetSubscriptionGroupProductPlanStaffItemPageDataUseCase
}

type Repositories struct {
	SubscriptionGroupProductPlanStaff pb.SubscriptionGroupProductPlanStaffDomainServiceServer
	// Eligibility guard anchors (red-team HIGH #5). Optional: when nil the
	// Create/Update guards fail-closed rather than silently skipping.
	ProductPlanStaff  productplanstaffpb.ProductPlanStaffDomainServiceServer
	ProductPlan       productplanpb.ProductPlanDomainServiceServer
	SubscriptionGroup subscriptiongrouppb.SubscriptionGroupDomainServiceServer
	// v2 class-edge anchors (docs/plan/20260724-section-assignment-merged/espyna.md
	// §2). Optional/best-effort — see class_edge_v2.go.
	SubscriptionGroupProductPlan sgpppb.SubscriptionGroupProductPlanDomainServiceServer
	JobTemplatePhase             jobtemplatephasepb.JobTemplatePhaseDomainServiceServer
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
		CreateSubscriptionGroupProductPlanStaff: NewCreateSubscriptionGroupProductPlanStaffUseCase(CreateSubscriptionGroupProductPlanStaffRepositories{
			SubscriptionGroupProductPlanStaff: repo,
			ProductPlanStaff:                  r.ProductPlanStaff,
			ProductPlan:                       r.ProductPlan,
			SubscriptionGroup:                 r.SubscriptionGroup,
			SubscriptionGroupProductPlan:      r.SubscriptionGroupProductPlan,
			JobTemplatePhase:                  r.JobTemplatePhase,
		}, CreateSubscriptionGroupProductPlanStaffServices(s)),
		ReadSubscriptionGroupProductPlanStaff: NewReadSubscriptionGroupProductPlanStaffUseCase(ReadSubscriptionGroupProductPlanStaffRepositories{SubscriptionGroupProductPlanStaff: repo}, ReadSubscriptionGroupProductPlanStaffServices(s)),
		UpdateSubscriptionGroupProductPlanStaff: NewUpdateSubscriptionGroupProductPlanStaffUseCase(UpdateSubscriptionGroupProductPlanStaffRepositories{
			SubscriptionGroupProductPlanStaff: repo,
			ProductPlanStaff:                  r.ProductPlanStaff,
			ProductPlan:                       r.ProductPlan,
			SubscriptionGroup:                 r.SubscriptionGroup,
			SubscriptionGroupProductPlan:      r.SubscriptionGroupProductPlan,
			JobTemplatePhase:                  r.JobTemplatePhase,
		}, UpdateSubscriptionGroupProductPlanStaffServices(s)),
		DeleteSubscriptionGroupProductPlanStaff: NewDeleteSubscriptionGroupProductPlanStaffUseCase(DeleteSubscriptionGroupProductPlanStaffRepositories{SubscriptionGroupProductPlanStaff: repo}, DeleteSubscriptionGroupProductPlanStaffServices(s)),
		AssignSubscriptionGroupProductPlanStaff: NewAssignSubscriptionGroupProductPlanStaffUseCase(AssignSubscriptionGroupProductPlanStaffRepositories{
			SubscriptionGroupProductPlanStaff: repo,
			ProductPlanStaff:                  r.ProductPlanStaff,
			ProductPlan:                       r.ProductPlan,
			SubscriptionGroup:                 r.SubscriptionGroup,
		}, AssignSubscriptionGroupProductPlanStaffServices(s)),
		ListSubscriptionGroupProductPlanStaffs:           NewListSubscriptionGroupProductPlanStaffsUseCase(ListSubscriptionGroupProductPlanStaffsRepositories{SubscriptionGroupProductPlanStaff: repo}, ListSubscriptionGroupProductPlanStaffsServices(s)),
		GetSubscriptionGroupProductPlanStaffListPageData: NewGetSubscriptionGroupProductPlanStaffListPageDataUseCase(GetSubscriptionGroupProductPlanStaffListPageDataRepositories{SubscriptionGroupProductPlanStaff: repo}, GetSubscriptionGroupProductPlanStaffListPageDataServices(s)),
		GetSubscriptionGroupProductPlanStaffItemPageData: NewGetSubscriptionGroupProductPlanStaffItemPageDataUseCase(GetSubscriptionGroupProductPlanStaffItemPageDataRepositories{SubscriptionGroupProductPlanStaff: repo}, GetSubscriptionGroupProductPlanStaffItemPageDataServices(s)),
	}
}

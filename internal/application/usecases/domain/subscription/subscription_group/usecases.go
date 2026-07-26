package subscription_group

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group"
	memberpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_member"
	sgpppb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan"
	sgppspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan_staff"
	sgwupb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_workspace_user"
)

type UseCases struct {
	CreateSubscriptionGroup          *CreateSubscriptionGroupUseCase
	ReadSubscriptionGroup            *ReadSubscriptionGroupUseCase
	UpdateSubscriptionGroup          *UpdateSubscriptionGroupUseCase
	DeleteSubscriptionGroup          *DeleteSubscriptionGroupUseCase
	ListSubscriptionGroups           *ListSubscriptionGroupsUseCase
	GetSubscriptionGroupListPageData *GetSubscriptionGroupListPageDataUseCase
	GetSubscriptionGroupItemPageData *GetSubscriptionGroupItemPageDataUseCase
}

type Repositories struct {
	SubscriptionGroup pb.SubscriptionGroupDomainServiceServer

	// Dependent repos for the referential delete guard (parent-not-deletable-
	// while-active-dependents-exist). Every table carrying a subscription_group_id
	// FK. Wired by the subscription-domain initializer; nil-tolerant in the guard.
	SubscriptionGroupMember           memberpb.SubscriptionGroupMemberDomainServiceServer
	SubscriptionGroupProductPlan      sgpppb.SubscriptionGroupProductPlanDomainServiceServer
	SubscriptionGroupProductPlanStaff sgppspb.SubscriptionGroupProductPlanStaffDomainServiceServer
	SubscriptionGroupWorkspaceUser    sgwupb.SubscriptionGroupWorkspaceUserDomainServiceServer
}

type Services struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

func NewUseCases(r Repositories, s Services) *UseCases {
	repo := r.SubscriptionGroup
	return &UseCases{
		CreateSubscriptionGroup: NewCreateSubscriptionGroupUseCase(CreateSubscriptionGroupRepositories{SubscriptionGroup: repo}, CreateSubscriptionGroupServices(s)),
		ReadSubscriptionGroup:   NewReadSubscriptionGroupUseCase(ReadSubscriptionGroupRepositories{SubscriptionGroup: repo}, ReadSubscriptionGroupServices(s)),
		UpdateSubscriptionGroup: NewUpdateSubscriptionGroupUseCase(UpdateSubscriptionGroupRepositories{SubscriptionGroup: repo}, UpdateSubscriptionGroupServices(s)),
		DeleteSubscriptionGroup: NewDeleteSubscriptionGroupUseCase(DeleteSubscriptionGroupRepositories{
			SubscriptionGroup: repo,
			Member:            r.SubscriptionGroupMember,
			Offering:          r.SubscriptionGroupProductPlan,
			TeachingStaff:     r.SubscriptionGroupProductPlanStaff,
			AccessGrant:       r.SubscriptionGroupWorkspaceUser,
		}, DeleteSubscriptionGroupServices(s)),
		ListSubscriptionGroups:           NewListSubscriptionGroupsUseCase(ListSubscriptionGroupsRepositories{SubscriptionGroup: repo}, ListSubscriptionGroupsServices(s)),
		GetSubscriptionGroupListPageData: NewGetSubscriptionGroupListPageDataUseCase(GetSubscriptionGroupListPageDataRepositories{SubscriptionGroup: repo}, GetSubscriptionGroupListPageDataServices(s)),
		GetSubscriptionGroupItemPageData: NewGetSubscriptionGroupItemPageDataUseCase(GetSubscriptionGroupItemPageDataRepositories{SubscriptionGroup: repo}, GetSubscriptionGroupItemPageDataServices(s)),
	}
}

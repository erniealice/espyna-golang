package subscription_group_member

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_member"
)

type UseCases struct {
	CreateSubscriptionGroupMember          *CreateSubscriptionGroupMemberUseCase
	ReadSubscriptionGroupMember            *ReadSubscriptionGroupMemberUseCase
	UpdateSubscriptionGroupMember          *UpdateSubscriptionGroupMemberUseCase
	DeleteSubscriptionGroupMember          *DeleteSubscriptionGroupMemberUseCase
	ListSubscriptionGroupMembers           *ListSubscriptionGroupMembersUseCase
	GetSubscriptionGroupMemberListPageData *GetSubscriptionGroupMemberListPageDataUseCase
	GetSubscriptionGroupMemberItemPageData *GetSubscriptionGroupMemberItemPageDataUseCase
}

type Repositories struct {
	SubscriptionGroupMember pb.SubscriptionGroupMemberDomainServiceServer
}

type Services struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

func NewUseCases(r Repositories, s Services) *UseCases {
	repo := r.SubscriptionGroupMember
	return &UseCases{
		CreateSubscriptionGroupMember:          NewCreateSubscriptionGroupMemberUseCase(CreateSubscriptionGroupMemberRepositories{SubscriptionGroupMember: repo}, CreateSubscriptionGroupMemberServices(s)),
		ReadSubscriptionGroupMember:            NewReadSubscriptionGroupMemberUseCase(ReadSubscriptionGroupMemberRepositories{SubscriptionGroupMember: repo}, ReadSubscriptionGroupMemberServices(s)),
		UpdateSubscriptionGroupMember:          NewUpdateSubscriptionGroupMemberUseCase(UpdateSubscriptionGroupMemberRepositories{SubscriptionGroupMember: repo}, UpdateSubscriptionGroupMemberServices(s)),
		DeleteSubscriptionGroupMember:          NewDeleteSubscriptionGroupMemberUseCase(DeleteSubscriptionGroupMemberRepositories{SubscriptionGroupMember: repo}, DeleteSubscriptionGroupMemberServices(s)),
		ListSubscriptionGroupMembers:           NewListSubscriptionGroupMembersUseCase(ListSubscriptionGroupMembersRepositories{SubscriptionGroupMember: repo}, ListSubscriptionGroupMembersServices(s)),
		GetSubscriptionGroupMemberListPageData: NewGetSubscriptionGroupMemberListPageDataUseCase(GetSubscriptionGroupMemberListPageDataRepositories{SubscriptionGroupMember: repo}, GetSubscriptionGroupMemberListPageDataServices(s)),
		GetSubscriptionGroupMemberItemPageData: NewGetSubscriptionGroupMemberItemPageDataUseCase(GetSubscriptionGroupMemberItemPageDataRepositories{SubscriptionGroupMember: repo}, GetSubscriptionGroupMemberItemPageDataServices(s)),
	}
}

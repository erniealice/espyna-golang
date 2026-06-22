package subscription_group

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group"
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
		CreateSubscriptionGroup:          NewCreateSubscriptionGroupUseCase(CreateSubscriptionGroupRepositories{SubscriptionGroup: repo}, CreateSubscriptionGroupServices(s)),
		ReadSubscriptionGroup:            NewReadSubscriptionGroupUseCase(ReadSubscriptionGroupRepositories{SubscriptionGroup: repo}, ReadSubscriptionGroupServices(s)),
		UpdateSubscriptionGroup:          NewUpdateSubscriptionGroupUseCase(UpdateSubscriptionGroupRepositories{SubscriptionGroup: repo}, UpdateSubscriptionGroupServices(s)),
		DeleteSubscriptionGroup:          NewDeleteSubscriptionGroupUseCase(DeleteSubscriptionGroupRepositories{SubscriptionGroup: repo}, DeleteSubscriptionGroupServices(s)),
		ListSubscriptionGroups:           NewListSubscriptionGroupsUseCase(ListSubscriptionGroupsRepositories{SubscriptionGroup: repo}, ListSubscriptionGroupsServices(s)),
		GetSubscriptionGroupListPageData: NewGetSubscriptionGroupListPageDataUseCase(GetSubscriptionGroupListPageDataRepositories{SubscriptionGroup: repo}, GetSubscriptionGroupListPageDataServices(s)),
		GetSubscriptionGroupItemPageData: NewGetSubscriptionGroupItemPageDataUseCase(GetSubscriptionGroupItemPageDataRepositories{SubscriptionGroup: repo}, GetSubscriptionGroupItemPageDataServices(s)),
	}
}

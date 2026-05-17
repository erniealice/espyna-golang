package tenant_subscription

import (
	"context"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	tenantsubscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tenancy/tenant_subscription"
)

const entityTenantSubscription = "tenant_subscription"

// TenantSubscriptionRepositories groups repository dependencies.
type TenantSubscriptionRepositories struct {
	TenantSubscription tenantsubscriptionpb.TenantSubscriptionDomainServiceServer
}

// TenantSubscriptionServices groups service dependencies.
type TenantSubscriptionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all tenant_subscription use cases.
type UseCases struct {
	Create *CreateTenantSubscriptionUseCase
	Read   *ReadTenantSubscriptionUseCase
	Update *UpdateTenantSubscriptionUseCase
	Delete *DeleteTenantSubscriptionUseCase
	List   *ListTenantSubscriptionsUseCase
}

// NewUseCases creates a new collection of tenant_subscription use cases.
func NewUseCases(repos TenantSubscriptionRepositories, services TenantSubscriptionServices) *UseCases {
	return &UseCases{
		Create: &CreateTenantSubscriptionUseCase{repo: repos.TenantSubscription, services: services},
		Read:   &ReadTenantSubscriptionUseCase{repo: repos.TenantSubscription, services: services},
		Update: &UpdateTenantSubscriptionUseCase{repo: repos.TenantSubscription, services: services},
		Delete: &DeleteTenantSubscriptionUseCase{repo: repos.TenantSubscription, services: services},
		List:   &ListTenantSubscriptionsUseCase{repo: repos.TenantSubscription, services: services},
	}
}

// CreateTenantSubscriptionUseCase handles creating a tenant subscription.
type CreateTenantSubscriptionUseCase struct {
	repo     tenantsubscriptionpb.TenantSubscriptionDomainServiceServer
	services TenantSubscriptionServices
}

func (uc *CreateTenantSubscriptionUseCase) Execute(ctx context.Context, req *tenantsubscriptionpb.CreateTenantSubscriptionRequest) (*tenantsubscriptionpb.CreateTenantSubscriptionResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityTenantSubscription, ports.ActionCreate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("tenant_subscription data is required")
	}
	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDService.GenerateID()
	}
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.Active = true
	return uc.repo.CreateTenantSubscription(ctx, req)
}

// ReadTenantSubscriptionUseCase handles reading a tenant subscription.
type ReadTenantSubscriptionUseCase struct {
	repo     tenantsubscriptionpb.TenantSubscriptionDomainServiceServer
	services TenantSubscriptionServices
}

func (uc *ReadTenantSubscriptionUseCase) Execute(ctx context.Context, req *tenantsubscriptionpb.ReadTenantSubscriptionRequest) (*tenantsubscriptionpb.ReadTenantSubscriptionResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityTenantSubscription, ports.ActionRead); err != nil {
		return nil, err
	}
	return uc.repo.ReadTenantSubscription(ctx, req)
}

// UpdateTenantSubscriptionUseCase handles updating a tenant subscription.
type UpdateTenantSubscriptionUseCase struct {
	repo     tenantsubscriptionpb.TenantSubscriptionDomainServiceServer
	services TenantSubscriptionServices
}

func (uc *UpdateTenantSubscriptionUseCase) Execute(ctx context.Context, req *tenantsubscriptionpb.UpdateTenantSubscriptionRequest) (*tenantsubscriptionpb.UpdateTenantSubscriptionResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityTenantSubscription, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("tenant_subscription ID is required")
	}
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	return uc.repo.UpdateTenantSubscription(ctx, req)
}

// DeleteTenantSubscriptionUseCase handles deleting a tenant subscription.
type DeleteTenantSubscriptionUseCase struct {
	repo     tenantsubscriptionpb.TenantSubscriptionDomainServiceServer
	services TenantSubscriptionServices
}

func (uc *DeleteTenantSubscriptionUseCase) Execute(ctx context.Context, req *tenantsubscriptionpb.DeleteTenantSubscriptionRequest) (*tenantsubscriptionpb.DeleteTenantSubscriptionResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityTenantSubscription, ports.ActionDelete); err != nil {
		return nil, err
	}
	return uc.repo.DeleteTenantSubscription(ctx, req)
}

// ListTenantSubscriptionsUseCase handles listing tenant subscriptions.
type ListTenantSubscriptionsUseCase struct {
	repo     tenantsubscriptionpb.TenantSubscriptionDomainServiceServer
	services TenantSubscriptionServices
}

func (uc *ListTenantSubscriptionsUseCase) Execute(ctx context.Context, req *tenantsubscriptionpb.ListTenantSubscriptionsRequest) (*tenantsubscriptionpb.ListTenantSubscriptionsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityTenantSubscription, ports.ActionRead); err != nil {
		return nil, err
	}
	return uc.repo.ListTenantSubscriptions(ctx, req)
}

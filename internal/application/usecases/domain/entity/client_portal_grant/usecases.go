package client_portal_grant

import (
	"context"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	clientportalgrantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_portal_grant"
)

const entityClientPortalGrant = "client_portal_grant"

// ClientPortalGrantRepositories groups repository dependencies.
type ClientPortalGrantRepositories struct {
	ClientPortalGrant clientportalgrantpb.ClientPortalGrantDomainServiceServer
}

// ClientPortalGrantServices groups service dependencies.
type ClientPortalGrantServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all client_portal_grant use cases.
type UseCases struct {
	Create *CreateClientPortalGrantUseCase
	Read   *ReadClientPortalGrantUseCase
	Update *UpdateClientPortalGrantUseCase
	Delete *DeleteClientPortalGrantUseCase
	List   *ListClientPortalGrantsUseCase
}

// NewUseCases creates a new collection of client_portal_grant use cases.
func NewUseCases(repos ClientPortalGrantRepositories, services ClientPortalGrantServices) *UseCases {
	return &UseCases{
		Create: &CreateClientPortalGrantUseCase{repo: repos.ClientPortalGrant, services: services},
		Read:   &ReadClientPortalGrantUseCase{repo: repos.ClientPortalGrant, services: services},
		Update: &UpdateClientPortalGrantUseCase{repo: repos.ClientPortalGrant, services: services},
		Delete: &DeleteClientPortalGrantUseCase{repo: repos.ClientPortalGrant, services: services},
		List:   &ListClientPortalGrantsUseCase{repo: repos.ClientPortalGrant, services: services},
	}
}

// CreateClientPortalGrantUseCase handles creating a client portal grant.
type CreateClientPortalGrantUseCase struct {
	repo     clientportalgrantpb.ClientPortalGrantDomainServiceServer
	services ClientPortalGrantServices
}

func (uc *CreateClientPortalGrantUseCase) Execute(ctx context.Context, req *clientportalgrantpb.CreateClientPortalGrantRequest) (*clientportalgrantpb.CreateClientPortalGrantResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityClientPortalGrant, ports.ActionCreate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("client_portal_grant data is required")
	}
	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDService.GenerateID()
	}
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.Active = true
	return uc.repo.CreateClientPortalGrant(ctx, req)
}

// ReadClientPortalGrantUseCase handles reading a client portal grant.
type ReadClientPortalGrantUseCase struct {
	repo     clientportalgrantpb.ClientPortalGrantDomainServiceServer
	services ClientPortalGrantServices
}

func (uc *ReadClientPortalGrantUseCase) Execute(ctx context.Context, req *clientportalgrantpb.ReadClientPortalGrantRequest) (*clientportalgrantpb.ReadClientPortalGrantResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityClientPortalGrant, ports.ActionRead); err != nil {
		return nil, err
	}
	return uc.repo.ReadClientPortalGrant(ctx, req)
}

// UpdateClientPortalGrantUseCase handles updating a client portal grant.
type UpdateClientPortalGrantUseCase struct {
	repo     clientportalgrantpb.ClientPortalGrantDomainServiceServer
	services ClientPortalGrantServices
}

func (uc *UpdateClientPortalGrantUseCase) Execute(ctx context.Context, req *clientportalgrantpb.UpdateClientPortalGrantRequest) (*clientportalgrantpb.UpdateClientPortalGrantResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityClientPortalGrant, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client_portal_grant ID is required")
	}
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	return uc.repo.UpdateClientPortalGrant(ctx, req)
}

// DeleteClientPortalGrantUseCase handles deleting a client portal grant.
type DeleteClientPortalGrantUseCase struct {
	repo     clientportalgrantpb.ClientPortalGrantDomainServiceServer
	services ClientPortalGrantServices
}

func (uc *DeleteClientPortalGrantUseCase) Execute(ctx context.Context, req *clientportalgrantpb.DeleteClientPortalGrantRequest) (*clientportalgrantpb.DeleteClientPortalGrantResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityClientPortalGrant, ports.ActionDelete); err != nil {
		return nil, err
	}
	return uc.repo.DeleteClientPortalGrant(ctx, req)
}

// ListClientPortalGrantsUseCase handles listing client portal grants.
type ListClientPortalGrantsUseCase struct {
	repo     clientportalgrantpb.ClientPortalGrantDomainServiceServer
	services ClientPortalGrantServices
}

func (uc *ListClientPortalGrantsUseCase) Execute(ctx context.Context, req *clientportalgrantpb.ListClientPortalGrantsRequest) (*clientportalgrantpb.ListClientPortalGrantsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityClientPortalGrant, ports.ActionRead); err != nil {
		return nil, err
	}
	return uc.repo.ListClientPortalGrants(ctx, req)
}

package user_preference

import (
	"context"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	userpreferencepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user_preference"
)

const entityUserPreference = "user_preference"

// UserPreferenceRepositories groups repository dependencies.
type UserPreferenceRepositories struct {
	UserPreference userpreferencepb.UserPreferenceDomainServiceServer
}

// UserPreferenceServices groups service dependencies.
type UserPreferenceServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all user_preference use cases.
type UseCases struct {
	Create *CreateUserPreferenceUseCase
	Read   *ReadUserPreferenceUseCase
	Update *UpdateUserPreferenceUseCase
	Delete *DeleteUserPreferenceUseCase
	List   *ListUserPreferencesUseCase
}

// NewUseCases creates a new collection of user_preference use cases.
func NewUseCases(repos UserPreferenceRepositories, services UserPreferenceServices) *UseCases {
	return &UseCases{
		Create: &CreateUserPreferenceUseCase{repo: repos.UserPreference, services: services},
		Read:   &ReadUserPreferenceUseCase{repo: repos.UserPreference, services: services},
		Update: &UpdateUserPreferenceUseCase{repo: repos.UserPreference, services: services},
		Delete: &DeleteUserPreferenceUseCase{repo: repos.UserPreference, services: services},
		List:   &ListUserPreferencesUseCase{repo: repos.UserPreference, services: services},
	}
}

// CreateUserPreferenceUseCase handles creating a user preference.
type CreateUserPreferenceUseCase struct {
	repo     userpreferencepb.UserPreferenceDomainServiceServer
	services UserPreferenceServices
}

func (uc *CreateUserPreferenceUseCase) Execute(ctx context.Context, req *userpreferencepb.CreateUserPreferenceRequest) (*userpreferencepb.CreateUserPreferenceResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityUserPreference, ports.ActionCreate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("user_preference data is required")
	}
	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDService.GenerateID()
	}
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.Active = true
	return uc.repo.CreateUserPreference(ctx, req)
}

// ReadUserPreferenceUseCase handles reading a user preference.
type ReadUserPreferenceUseCase struct {
	repo     userpreferencepb.UserPreferenceDomainServiceServer
	services UserPreferenceServices
}

func (uc *ReadUserPreferenceUseCase) Execute(ctx context.Context, req *userpreferencepb.ReadUserPreferenceRequest) (*userpreferencepb.ReadUserPreferenceResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityUserPreference, ports.ActionRead); err != nil {
		return nil, err
	}
	return uc.repo.ReadUserPreference(ctx, req)
}

// UpdateUserPreferenceUseCase handles updating a user preference.
type UpdateUserPreferenceUseCase struct {
	repo     userpreferencepb.UserPreferenceDomainServiceServer
	services UserPreferenceServices
}

func (uc *UpdateUserPreferenceUseCase) Execute(ctx context.Context, req *userpreferencepb.UpdateUserPreferenceRequest) (*userpreferencepb.UpdateUserPreferenceResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityUserPreference, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("user_preference ID is required")
	}
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	return uc.repo.UpdateUserPreference(ctx, req)
}

// DeleteUserPreferenceUseCase handles deleting a user preference.
type DeleteUserPreferenceUseCase struct {
	repo     userpreferencepb.UserPreferenceDomainServiceServer
	services UserPreferenceServices
}

func (uc *DeleteUserPreferenceUseCase) Execute(ctx context.Context, req *userpreferencepb.DeleteUserPreferenceRequest) (*userpreferencepb.DeleteUserPreferenceResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityUserPreference, ports.ActionDelete); err != nil {
		return nil, err
	}
	return uc.repo.DeleteUserPreference(ctx, req)
}

// ListUserPreferencesUseCase handles listing user preferences.
type ListUserPreferencesUseCase struct {
	repo     userpreferencepb.UserPreferenceDomainServiceServer
	services UserPreferenceServices
}

func (uc *ListUserPreferencesUseCase) Execute(ctx context.Context, req *userpreferencepb.ListUserPreferencesRequest) (*userpreferencepb.ListUserPreferencesResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityUserPreference, ports.ActionRead); err != nil {
		return nil, err
	}
	return uc.repo.ListUserPreferences(ctx, req)
}

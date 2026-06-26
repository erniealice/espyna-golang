package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	infraports "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// AdminResetPasswordServices groups all business service dependencies.
// AuthService is the inward port that performs the credential effect at the
// active provider (firebase: UpdateUser{Password} or PasswordResetLink;
// password: bcrypt write to user.password_hash). It is required — a nil
// AuthService means no auth provider is configured.
type AdminResetPasswordServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	AuthService      infraports.AuthService
}

// AdminResetPasswordUseCase performs an admin-initiated password reset for a
// user — either by setting a new password directly or by generating a
// provider-issued reset link (the request oneof). See design §5.
type AdminResetPasswordUseCase struct {
	services AdminResetPasswordServices
}

// NewAdminResetPasswordUseCase creates the use case with grouped dependencies.
func NewAdminResetPasswordUseCase(
	services AdminResetPasswordServices,
) *AdminResetPasswordUseCase {
	return &AdminResetPasswordUseCase{
		services: services,
	}
}

// Execute resets the user's password via the configured auth provider.
func (uc *AdminResetPasswordUseCase) Execute(ctx context.Context, req *userpb.AdminResetPasswordRequest) (*userpb.AdminResetPasswordResponse, error) {
	// Authorization check — user:reset-password.
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.User,
		Action: entityid.ActionResetPassword,
	}); err != nil {
		return nil, err
	}

	// Input validation.
	if req == nil || req.GetUserId() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "user.validation.id_required", "User ID is required [DEFAULT]"))
	}

	if uc.services.AuthService == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "user.errors.auth_unavailable", "Auth provider is not available [DEFAULT]"))
	}

	// The request oneof selects the reset method.
	switch method := req.GetMethod().(type) {
	case *userpb.AdminResetPasswordRequest_NewPassword:
		newPassword := method.NewPassword
		if newPassword == "" {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "user.validation.password_required", "New password is required [DEFAULT]"))
		}
		if err := uc.services.AuthService.AdminSetPassword(ctx, req.GetUserId(), newPassword); err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "user.errors.reset_password_failed", "Password reset failed [DEFAULT]")
			return nil, fmt.Errorf("%s: %w", translatedError, err)
		}
		return &userpb.AdminResetPasswordResponse{Reset_: true, Success: true}, nil

	case *userpb.AdminResetPasswordRequest_GenerateLink:
		link, err := uc.services.AuthService.GeneratePasswordResetLink(ctx, req.GetUserId())
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "user.errors.reset_password_failed", "Password reset failed [DEFAULT]")
			return nil, fmt.Errorf("%s: %w", translatedError, err)
		}
		resp := &userpb.AdminResetPasswordResponse{Reset_: true, Success: true}
		if link != "" {
			resp.ResetLink = &link
		}
		return resp, nil

	default:
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "user.validation.reset_method_required", "A reset method (new password or generate link) is required [DEFAULT]"))
	}
}

package admin

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	adminpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/admin"
)

// UpdateAdminRepositories groups all repository dependencies
type UpdateAdminRepositories struct {
	Admin adminpb.AdminDomainServiceServer // Primary entity repository
}

// UpdateAdminServices groups all business service dependencies
type UpdateAdminServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UpdateAdminUseCase handles the business logic for updating an admin
type UpdateAdminUseCase struct {
	repositories UpdateAdminRepositories
	services     UpdateAdminServices
}

// NewUpdateAdminUseCase creates use case with grouped dependencies
func NewUpdateAdminUseCase(
	repositories UpdateAdminRepositories,
	services UpdateAdminServices,
) *UpdateAdminUseCase {
	return &UpdateAdminUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update admin operation
func (uc *UpdateAdminUseCase) Execute(ctx context.Context, req *adminpb.UpdateAdminRequest) (*adminpb.UpdateAdminResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Admin,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "admin.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	if req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "admin.validation.data_required", "[ERR-DEFAULT] Admin data is required"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "admin.validation.id_required", "[ERR-DEFAULT] Admin ID is required"))
	}

	// Validate email format if email is provided
	if req.Data.User != nil && req.Data.User.EmailAddress != "" {
		if err := uc.validateEmail(req.Data.User.EmailAddress); err != nil {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "admin.validation.email_invalid", "[ERR-DEFAULT] Invalid email format"))
		}
	}

	// Call repository
	resp, err := uc.repositories.Admin.UpdateAdmin(ctx, req)
	if err != nil {
		if err.Error() == contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "admin.errors.not_found", "[ERR-DEFAULT] Admin not found") {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "admin.errors.not_found", "[ERR-DEFAULT] Admin not found"))
		}
		return nil, fmt.Errorf("%s: %w", contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "admin.errors.update_failed", "[ERR-DEFAULT] Admin update failed"), err)
	}

	return resp, nil
}

// validateEmail validates email format
func (uc *UpdateAdminUseCase) validateEmail(email string) error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return errors.New("invalid email format")
	}
	return nil
}

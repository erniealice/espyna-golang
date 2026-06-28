package staff

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	principaltypepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/principal_type"
	staffpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/staff"
)

// UpdateStaffRepositories groups all repository dependencies
type UpdateStaffRepositories struct {
	Staff staffpb.StaffDomainServiceServer // Primary entity repository
}

// UpdateStaffServices groups all business service dependencies
type UpdateStaffServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UpdateStaffUseCase handles the business logic for updating a staff
type UpdateStaffUseCase struct {
	repositories UpdateStaffRepositories
	services     UpdateStaffServices
}

// NewUpdateStaffUseCase creates use case with grouped dependencies
func NewUpdateStaffUseCase(
	repositories UpdateStaffRepositories,
	services UpdateStaffServices,
) *UpdateStaffUseCase {
	return &UpdateStaffUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdateStaffUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewUpdateStaffUseCase with grouped parameters instead
func NewUpdateStaffUseCaseUngrouped(staffRepo staffpb.StaffDomainServiceServer) *UpdateStaffUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := UpdateStaffRepositories{
		Staff: staffRepo,
	}

	services := UpdateStaffServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
	}

	return NewUpdateStaffUseCase(repositories, services)
}

func (uc *UpdateStaffUseCase) Execute(ctx context.Context, req *staffpb.UpdateStaffRequest) (*staffpb.UpdateStaffResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Staff,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "staff.validation.request_required", "Request is required for staff [DEFAULT]"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "staff.validation.id_required", "Staff ID is required [DEFAULT]"))
	}

	// Self-escalation guard (defense-in-depth).
	//
	// A STAFF principal (PRINCIPAL_TYPE_STAFF = 7) must NOT be able to mutate
	// their own staff.role_id. role_id is the sole permission source for the
	// staff principal kind (userRolesStaffCTE); self-mutation would allow any
	// staff user holding the staff:update permission to grant themselves an
	// arbitrarily powerful role without administrator approval.
	//
	// Three-condition conjunction — all three must be true to reject:
	//   (1) actor is a STAFF principal       → other principal kinds do not have a
	//       corresponding staff row as their anchor
	//   (2) target staff row == actor's own  → an admin updating a DIFFERENT staff
	//       member's role is permitted
	//   (3) request carries a role_id change → non-role self-updates (status,
	//       seniority, employment_type, …) are NOT blocked
	//
	// Uses identity.FromContext (not Must) so that non-HTTP callers (seeder,
	// internal migration tooling) that have no RequestIdentity on the context
	// bypass the guard safely — they are never STAFF principals.
	if actorID, ok := identity.FromContext(ctx); ok &&
		actorID.PrincipalType == int32(principaltypepb.PrincipalType_PRINCIPAL_TYPE_STAFF) &&
		actorID.PrincipalID == req.Data.Id &&
		req.Data.RoleId != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"staff.validation.self_role_elevation_denied",
			"Staff cannot change their own role [DEFAULT]",
		))
	}

	// Business logic validation
	if req.Data.User != nil && req.Data.User.EmailAddress == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "staff.validation.email_required", "Staff email is required [DEFAULT]"))
	}

	// Call repository
	resp, err := uc.repositories.Staff.UpdateStaff(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "staff.errors.update_failed", "Staff update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

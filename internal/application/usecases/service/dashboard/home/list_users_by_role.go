package home

import (
	"context"
)

// ---------- Repository interface ----------

// UsersByRoleRepository queries users assigned to a role.
// Satisfied by PostgresWorkspaceUserRoleRepository.
type UsersByRoleRepository interface {
	// ListUsersByRoleID returns workspace users assigned to the given role.
	ListUsersByRoleID(ctx context.Context, workspaceID, roleID string) ([]UserByRole, error)
}

// ---------- Return type (Go-only) ----------

// UserByRole is one user assigned to a role. Field names mirror the entydad
// roleusers.UserByRole type — the composition layer maps 1:1.
type UserByRole struct {
	WorkspaceUserRoleID string
	WorkspaceUserID     string
	UserID              string
	UserName            string
	Email               string
	DateAssigned        string
}

// ---------- Use case ----------

// ListUsersByRoleUseCase looks up users assigned to a given role within the
// workspace. Used by the role detail page and role-user assign action.
type ListUsersByRoleUseCase struct {
	repository UsersByRoleRepository
}

// NewListUsersByRoleUseCase wires the use case.
func NewListUsersByRoleUseCase(repo UsersByRoleRepository) *ListUsersByRoleUseCase {
	return &ListUsersByRoleUseCase{repository: repo}
}

// Execute returns users assigned to the given role in the workspace.
// Returns nil (not error) when the repository is not wired.
func (uc *ListUsersByRoleUseCase) Execute(
	ctx context.Context,
	workspaceID, roleID string,
) ([]UserByRole, error) {
	if uc.repository == nil {
		return nil, nil
	}
	return uc.repository.ListUsersByRoleID(ctx, workspaceID, roleID)
}

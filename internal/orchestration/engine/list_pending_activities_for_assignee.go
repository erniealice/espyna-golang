package engine

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports/domain"
)

// AssigneeQueryRepository is the adapter interface that the use case depends
// on. The postgres adapter (PostgresAssigneeQueryRepository) satisfies this;
// other providers can supply their own implementation.
type AssigneeQueryRepository interface {
	ListPendingActivitiesForAssignee(ctx context.Context, req *domain.ListPendingActivitiesForAssigneeRequest) (*domain.ListPendingActivitiesForAssigneeResponse, error)
}

// ListPendingActivitiesForAssigneeUseCase resolves pending engine activities
// assigned to a given workspace user through the identity bridge:
//
//	activity.assigned_to (global user.id)
//	    ↕ workspace_user.user_id (proto f5, indexed)
//	workspace_user.id = session principal_id
//
// This is a read-only query use case — it never writes Activity.assigned_to.
// Validation (fail-closed): empty workspace_user_id or workspace_id returns
// an empty result with no SQL executed.
type ListPendingActivitiesForAssigneeUseCase struct {
	repo AssigneeQueryRepository
}

// NewListPendingActivitiesForAssigneeUseCase creates the use case.
func NewListPendingActivitiesForAssigneeUseCase(repo AssigneeQueryRepository) *ListPendingActivitiesForAssigneeUseCase {
	return &ListPendingActivitiesForAssigneeUseCase{repo: repo}
}

// Execute validates the identity inputs and delegates to the adapter.
//
// Fail-closed invariant: if either identity input is empty, return an empty
// result without executing any SQL. This covers the STAFF_SELF fail-close
// (Q-EIB-HAT) and the general fail-closed-empty-identity test case.
func (uc *ListPendingActivitiesForAssigneeUseCase) Execute(
	ctx context.Context,
	req *domain.ListPendingActivitiesForAssigneeRequest,
) (*domain.ListPendingActivitiesForAssigneeResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}

	// ── Fail-closed: empty identity ⇒ deny, empty result, no SQL ──
	if req.WorkspaceUserID == "" || req.WorkspaceID == "" {
		return &domain.ListPendingActivitiesForAssigneeResponse{
			Activities: nil,
			Total:      0,
		}, nil
	}

	return uc.repo.ListPendingActivitiesForAssignee(ctx, req)
}

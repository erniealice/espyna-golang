package work_request

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	workspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user"
	work_requestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/work_request"
)

// AssignWorkRequestRequest is the Go-shaped input for assignment.
type AssignWorkRequestRequest struct {
	WorkRequestID             string
	AssignedToWorkspaceUserID string
}

// AssignWorkRequestRepositories groups all repository dependencies.
type AssignWorkRequestRepositories struct {
	WorkRequest   work_requestpb.WorkRequestDomainServiceServer
	WorkspaceUser workspaceuserpb.WorkspaceUserDomainServiceServer // FK validation
}

// AssignWorkRequestServices groups all business service dependencies.
type AssignWorkRequestServices struct {
	ActionGatekeeper *actiongate.ActionGatekeeper
	Transactor       ports.Transactor
	Translator       ports.Translator
}

// AssignWorkRequestUseCase sets the assigned_to_workspace_user_id on a work request.
//
// This is the single triage/case owner on the record (Q-ASSIGN-IDENTITY).
// Per-task / per-stage dynamic owners are the engine's Activity.assigned_to,
// NOT a WorkTask table.
//
// Validates that the workspace_user exists before assignment.
type AssignWorkRequestUseCase struct {
	repositories AssignWorkRequestRepositories
	services     AssignWorkRequestServices
}

func NewAssignWorkRequestUseCase(repositories AssignWorkRequestRepositories, services AssignWorkRequestServices) *AssignWorkRequestUseCase {
	return &AssignWorkRequestUseCase{repositories: repositories, services: services}
}

func (uc *AssignWorkRequestUseCase) Execute(ctx context.Context, req *AssignWorkRequestRequest) (*work_requestpb.UpdateWorkRequestResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.WorkRequest,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.WorkRequestID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workRequest.validation.id_required", "Work request ID is required [DEFAULT]"))
	}
	if req.AssignedToWorkspaceUserID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workRequest.validation.assignee_required", "Assignee workspace user ID is required [DEFAULT]"))
	}

	// FK validation: workspace_user must exist.
	if uc.repositories.WorkspaceUser != nil {
		wuResp, err := uc.repositories.WorkspaceUser.ReadWorkspaceUser(ctx, &workspaceuserpb.ReadWorkspaceUserRequest{
			Data: &workspaceuserpb.WorkspaceUser{Id: req.AssignedToWorkspaceUserID},
		})
		if err != nil {
			return nil, err
		}
		if wuResp == nil || len(wuResp.Data) == 0 {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workRequest.errors.assignee_not_found", "Assignee workspace user not found [DEFAULT]"))
		}
	}

	// Load the current work request.
	readResp, err := uc.repositories.WorkRequest.ReadWorkRequest(ctx, &work_requestpb.ReadWorkRequestRequest{
		Data: &work_requestpb.WorkRequest{Id: req.WorkRequestID},
	})
	if err != nil {
		return nil, err
	}
	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workRequest.errors.not_found", "Work request not found [DEFAULT]"))
	}
	wr := readResp.Data[0]

	// Cannot assign to a terminal request.
	if isTerminalStatus(wr.Status) {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workRequest.errors.cannot_assign_terminal", "Cannot assign a completed, declined, or cancelled request [DEFAULT]"))
	}

	now := time.Now()
	wr.AssignedToWorkspaceUserId = &req.AssignedToWorkspaceUserID
	wr.DateModified = &[]int64{now.UnixMilli()}[0]
	wr.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	resp, err := uc.repositories.WorkRequest.UpdateWorkRequest(ctx, &work_requestpb.UpdateWorkRequestRequest{Data: wr})
	if err != nil {
		translated := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workRequest.errors.assign_failed", "Work request assignment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translated, err)
	}
	return resp, nil
}

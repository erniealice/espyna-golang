package executor

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
)

// PermissionAwareExecutor wraps an ActivityExecutor with an authcheck.Check call
// before delegating to the inner executor. This ensures that any use case
// invoked through the workflow engine has permission enforcement, even if the
// underlying use case was not individually instrumented with authcheck.
//
// Usage:
//
//	inner := executor.New[*pb.CreateClientRequest, *pb.CreateClientResponse](uc.Execute)
//	wrapped := executor.NewPermissionAware(inner, authSvc, translationSvc, "client", "create")
type PermissionAwareExecutor struct {
	inner              ports.ActivityExecutor
	authService        ports.AuthorizationService
	translationService ports.TranslationService
	entity             string
	action             string
}

// NewPermissionAware creates a PermissionAwareExecutor that wraps the given executor
// with an authorization check for the specified entity and action.
//
// Parameters:
//   - inner: the underlying executor to delegate to after the permission check
//   - authService: the authorization service used by authcheck.Check
//   - translationService: the translation service for i18n error messages
//   - entity: the entity type (e.g., "client", "product", "admin")
//   - action: the action (e.g., "create", "read", "update", "delete", "list")
func NewPermissionAware(
	inner ports.ActivityExecutor,
	authService ports.AuthorizationService,
	translationService ports.TranslationService,
	entity string,
	action string,
) ports.ActivityExecutor {
	return &PermissionAwareExecutor{
		inner:              inner,
		authService:        authService,
		translationService: translationService,
		entity:             entity,
		action:             action,
	}
}

// Execute checks authorization before delegating to the inner executor.
// Returns an error immediately if the user lacks the required permission.
func (e *PermissionAwareExecutor) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	if e.inner == nil {
		return nil, fmt.Errorf("PermissionAwareExecutor: inner executor is nil")
	}

	// Enforce permission check via authcheck.Check
	if err := authcheck.Check(ctx, e.authService, e.translationService, e.entity, e.action); err != nil {
		return nil, err
	}

	// Delegate to the inner executor
	return e.inner.Execute(ctx, input)
}

// Package actiongate provides the ActionGatekeeper — the Gate 1 service that
// answers "can this principal perform this action on this entity type?" It is
// the AWS Action dimension, implemented as RBAC capability checks (ALLOW/DENY
// deny-wins). It complements the ResourceGatekeeper (Gate 2: the Resource
// dimension).
//
// The two gates form a sequential authorization chain:
//
//	Gate 1 (Action):   "can you DO evaluation:list?"         → error (hard stop)
//	Gate 2 (Resource): "can you SEE this client's data?"     → bool  (row filter)
//
// Charter — this package MUST NOT import:
//   - proto entity types (esqyma/...)
//   - DB drivers or adapter packages
//   - anything under internal/application/usecases/...
//
// Depends on: Go stdlib + shared/context (principalID extraction) +
// registry/entityid (EntityPermission helper).
//
// Consumers (keep in sync): ~957 use case Execute methods across usecases/.
package actiongate

import (
	"context"
	"errors"
	"log"

	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
)

// Authorizer is the minimal RBAC capability port. Matches ports.Authorizer
// but declared here so this package stays a leaf without importing ports.
type Authorizer interface {
	HasPermission(ctx context.Context, userID, permission string) (bool, error)
	IsEnabled() bool
}

// Translator is the minimal translation port for error messages.
type Translator interface {
	GetWithDefault(ctx context.Context, businessType, key, defaultMessage string, params ...any) string
}

// CheckActionRequest is the structured input for the action gate.
type CheckActionRequest struct {
	Entity string // entityid.Evaluation
	Action string // entityid.ActionList
}

// ActionGatekeeper is the Gate 1 authorization service. Constructed at DI time;
// consumers carry one struct instead of threading Authorizer + Translator per call.
type ActionGatekeeper struct {
	authorizer Authorizer
	translator Translator
}

// NewActionGatekeeper constructs the gatekeeper.
func NewActionGatekeeper(authorizer Authorizer, translator Translator) *ActionGatekeeper {
	return &ActionGatekeeper{authorizer: authorizer, translator: translator}
}

// Check verifies that the acting principal holds the given permission.
// Returns nil if authorized, or an error if denied/missing.
// Fail-closed: nil gatekeeper, nil authorizer, nil request, or empty
// Entity/Action all deny.
func (g *ActionGatekeeper) Check(ctx context.Context, req *CheckActionRequest) error {
	if g == nil {
		log.Println("WARNING: ActionGatekeeper is nil — denying by default")
		return errors.New("authorization denied: action gatekeeper not configured")
	}
	if req == nil {
		log.Println("WARNING: CheckActionRequest is nil — denying by default")
		return errors.New("authorization denied: nil action request")
	}
	if req.Entity == "" || req.Action == "" {
		log.Printf("WARNING: CheckActionRequest has empty Entity=%q or Action=%q — denying", req.Entity, req.Action)
		return errors.New("authorization denied: entity and action are required")
	}
	if g.authorizer == nil {
		log.Println("WARNING: Authorizer is nil — denying by default")
		return errors.New(g.translate(ctx, "common.errors.authorization_failed", "Authorization not configured"))
	}

	if !g.authorizer.IsEnabled() {
		return nil
	}

	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		return errors.New(g.translate(ctx, "common.errors.authorization_failed", "Authorization failed"))
	}

	permission := entityid.EntityPermission(req.Entity, req.Action)
	hasPerm, err := g.authorizer.HasPermission(ctx, userID, permission)
	if err != nil {
		log.Printf("AUTHZ_ERROR | user=%s | permission=%s | error=%v", userID, permission, err)
		return errors.New(g.translate(ctx, "common.errors.authorization_failed", "Authorization failed"))
	}

	if !hasPerm {
		log.Printf("AUTHZ_DENIED | user=%s | permission=%s", userID, permission)
		return errors.New(g.translate(ctx, "common.errors.permission_denied", "Permission denied"))
	}

	return nil
}

func (g *ActionGatekeeper) translate(ctx context.Context, key, defaultMsg string) string {
	if g.translator == nil {
		return defaultMsg
	}
	bt := contextutil.ExtractBusinessTypeFromContext(ctx)
	return g.translator.GetWithDefault(ctx, bt, key, defaultMsg)
}

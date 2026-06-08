// Package resourcegate provides the ResourceGatekeeper — the Gate 2 service
// that answers "can this principal access resources scoped to this
// client/subscription?" It is the AWS Resource dimension, implemented as
// junction-based membership checks (ReBAC). It complements authcheck / the
// future ActionGatekeeper (Gate 1: the Action dimension).
//
// The two gates form a sequential authorization chain:
//
//	Gate 1 (Action):   "can you DO evaluation:list?"         → error (hard stop)
//	Gate 2 (Resource): "can you SEE this client's data?"     → bool  (row filter)
//
// The triage_all bypass is the Resource: "*" wildcard — if the principal holds
// "<entity>:triage_all", they can access any resource regardless of scope.
//
// Charter — this package MUST NOT import:
//   - proto entity types (esqyma/...)
//   - DB drivers or adapter packages
//   - anything under internal/application/usecases/...
//   - internal/application/ports (declares its own minimal interfaces)
//
// Depends only on the Go standard library plus
// internal/application/shared/context (principalID extraction from ctx).
//
// Consumers (keep in sync):
//   - usecases/service/performance/get_performance_panel_data.go
//   - (future) conversation read/write scoping
//   - (future) work_request servicing-scoped access
package resourcegate

import (
	"context"

	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
)

// CheckAccessRequest is the structured input for the resource-scope gate.
// The Entity field drives the triage_all bypass: "<entity>:triage_all".
type CheckAccessRequest struct {
	Entity         string
	ClientID       string
	SubscriptionID *string
}

// ResourceGatekeeper is the Gate 2 authorization service. Constructed at DI
// time with its dependencies; consumers carry one struct, not four.
type ResourceGatekeeper struct {
	authorizer Authorizer
	client     ClientScopeChecker
	sub        SubscriptionScopeChecker
}

// NewResourceGatekeeper constructs the gatekeeper. Nil dependencies are safe —
// CanAccess fail-closes (deny) when any dependency is nil.
func NewResourceGatekeeper(
	authorizer Authorizer,
	client ClientScopeChecker,
	sub SubscriptionScopeChecker,
) *ResourceGatekeeper {
	return &ResourceGatekeeper{
		authorizer: authorizer,
		client:     client,
		sub:        sub,
	}
}

// CanAccess answers "can the acting principal access resources scoped to this
// client (and optionally this subscription)?" Called AFTER the action gate
// (authcheck / ActionGatekeeper) has already verified the action capability.
//
// PrincipalID is extracted from context — the same source as the session
// middleware and authcheck. This keeps the two gates deterministic: both read
// the same identity from the same source.
//
// Fail-closed: nil gatekeeper, nil dependencies, empty principal, missing
// context user, or any lookup error → false (deny).
func (g *ResourceGatekeeper) CanAccess(ctx context.Context, req *CheckAccessRequest) bool {
	if g == nil || req == nil {
		return false
	}

	principalID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil || principalID == "" {
		return false
	}

	if g.triageAllBypass(ctx, principalID, req.Entity) {
		return true
	}

	if req.SubscriptionID == nil {
		return g.checkClientScope(ctx, principalID, req.ClientID)
	}
	return g.checkSubscriptionScope(ctx, principalID, *req.SubscriptionID)
}

// triageAllBypass checks if the principal holds "<entity>:triage_all" —
// the Resource: "*" wildcard that bypasses membership checks.
func (g *ResourceGatekeeper) triageAllBypass(ctx context.Context, principalID, entity string) bool {
	if g.authorizer == nil || entity == "" {
		return false
	}
	ok, err := g.authorizer.HasPermission(ctx, principalID, entity+":triage_all")
	return err == nil && ok
}

// checkClientScope checks ACCOUNT-level membership via client_workspace_user.
func (g *ResourceGatekeeper) checkClientScope(ctx context.Context, principalID, clientID string) bool {
	if g.client == nil || clientID == "" {
		return false
	}
	ok, err := g.client.CanAccessClient(ctx, principalID, clientID)
	return err == nil && ok
}

// checkSubscriptionScope checks PROJECT-level membership via subscription_workspace_user.
func (g *ResourceGatekeeper) checkSubscriptionScope(ctx context.Context, principalID, subscriptionID string) bool {
	if g.sub == nil || subscriptionID == "" {
		return false
	}
	ok, err := g.sub.CanAccessSubscription(ctx, principalID, subscriptionID)
	return err == nil && ok
}

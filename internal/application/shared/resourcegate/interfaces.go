package resourcegate

import "context"

// ClientScopeChecker answers "can this principal access resources scoped to
// this client?" by checking the client_workspace_user junction. The postgres
// adapter satisfies this structurally via IsActiveAccountTeamMember (renamed
// to CanAccessClient at this interface boundary).
type ClientScopeChecker interface {
	CanAccessClient(ctx context.Context, principalID string, clientID string) (bool, error)
}

// SubscriptionScopeChecker answers "can this principal access resources scoped
// to this subscription?" by checking the subscription_workspace_user junction.
// The postgres adapter satisfies this structurally via IsActiveServicer
// (renamed to CanAccessSubscription at this interface boundary).
type SubscriptionScopeChecker interface {
	CanAccessSubscription(ctx context.Context, principalID string, subscriptionID string) (bool, error)
}

// Authorizer is the minimal permission-check dependency: it answers whether
// the principal holds a given permission code. Used for the triage_all bypass
// (Resource: "*"). Implementations adapt the wider RBAC surface; this package
// only needs HasPermission.
type Authorizer interface {
	HasPermission(ctx context.Context, userID string, permission string) (bool, error)
}

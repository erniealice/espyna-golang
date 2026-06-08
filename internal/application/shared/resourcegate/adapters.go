package resourcegate

import "context"

// ClientScopeAdapter bridges an existing IsActiveAccountTeamMember method to
// the ClientScopeChecker interface. Use at DI time when wiring the postgres
// adapter (which already implements the membership query but under the old
// method name).
type ClientScopeAdapter struct {
	Reader interface {
		IsActiveAccountTeamMember(ctx context.Context, principalID string, clientID string) (bool, error)
	}
}

func (a ClientScopeAdapter) CanAccessClient(ctx context.Context, principalID, clientID string) (bool, error) {
	if a.Reader == nil {
		return false, nil
	}
	return a.Reader.IsActiveAccountTeamMember(ctx, principalID, clientID)
}

// SubscriptionScopeAdapter bridges an existing IsActiveServicer method to
// the SubscriptionScopeChecker interface.
type SubscriptionScopeAdapter struct {
	Reader interface {
		IsActiveServicer(ctx context.Context, principalID string, subscriptionID string) (bool, error)
	}
}

func (a SubscriptionScopeAdapter) CanAccessSubscription(ctx context.Context, principalID, subscriptionID string) (bool, error) {
	if a.Reader == nil {
		return false, nil
	}
	return a.Reader.IsActiveServicer(ctx, principalID, subscriptionID)
}

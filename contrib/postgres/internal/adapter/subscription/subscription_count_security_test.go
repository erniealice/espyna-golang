//go:build postgresql

package subscription

import (
	"context"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/shared/identity"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

func TestCountActiveByClientIdsValidatesRequestIdentityAndBudgetBeforeQuery(t *testing.T) {
	repo := NewPostgresSubscriptionRepository(newListCaptureOps(t, baseListResult()), "subscription")

	if _, err := repo.CountActiveByClientIds(context.Background(), nil); err == nil || !strings.Contains(err.Error(), "request is required") {
		t.Fatalf("nil request error = %v, want request validation", err)
	}
	if _, err := repo.CountActiveByClientIds(context.Background(), &subscriptionpb.CountActiveByClientIdsRequest{}); err == nil {
		t.Fatal("missing workspace identity was accepted")
	}

	ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: "ws-1"})
	tooMany := make([]string, 101)
	for i := range tooMany {
		tooMany[i] = "client-id"
	}
	if _, err := repo.CountActiveByClientIds(ctx, &subscriptionpb.CountActiveByClientIdsRequest{ClientIds: tooMany}); err == nil || !strings.Contains(err.Error(), "too many query IDs") {
		t.Fatalf("oversized client ID set error = %v, want finite-work rejection", err)
	}
}

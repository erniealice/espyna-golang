package firebase

import (
	"context"
	"os"
	"testing"
)

// TestFirebaseIntegration tests Firebase Auth client creation against the
// AUTH_FIREBASE_ credential set. It is gated by testing.Short().
func TestFirebaseIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	manager, err := NewFirebaseClientManager(ctx, os.Getenv("AUTH_FIREBASE_PROJECT_ID"))
	if err != nil {
		t.Fatalf("Failed to create Firebase client manager: %v", err)
	}
	defer manager.Close()

	// Test Auth client
	authClient, err := manager.GetAuthClient(ctx)
	if err != nil {
		t.Fatalf("Failed to get auth client: %v", err)
	}
	if authClient == nil {
		t.Error("Expected non-nil auth client")
	}

	projectID := manager.GetProjectID()
	if projectID == "" {
		t.Error("Expected non-empty project ID")
	}
}

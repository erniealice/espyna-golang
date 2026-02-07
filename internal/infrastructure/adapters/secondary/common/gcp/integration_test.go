//go:build google && integration

package gcp_test

import (
	"context"
	"testing"

	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/common/firebase"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/common/google"
)

// TestGoogleStorageIntegration tests Google Cloud Storage client creation
func TestGoogleStorageIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	manager, err := google.NewGoogleClientManager(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to create Google client manager: %v", err)
	}
	defer manager.Close()

	client := manager.GetStorageClient()
	if client == nil {
		t.Error("Expected non-nil storage client")
	}

	projectID := manager.GetProjectID()
	if projectID == "" {
		t.Error("Expected non-empty project ID")
	}
}

// TestFirebaseIntegration tests Firebase client creation
func TestFirebaseIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	manager, err := firebase.NewFirebaseClientManager(ctx, "")
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

	// Test Firestore client
	firestoreClient, err := manager.GetFirestoreClient(ctx)
	if err != nil {
		t.Fatalf("Failed to get firestore client: %v", err)
	}
	if firestoreClient == nil {
		t.Error("Expected non-nil firestore client")
	}

	projectID := manager.GetProjectID()
	if projectID == "" {
		t.Error("Expected non-empty project ID")
	}
}

// TestSharedCredentials verifies both Google and Firebase use same credential logic
func TestSharedCredentials(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Create both managers
	googleManager, err := google.NewGoogleClientManager(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to create Google manager: %v", err)
	}
	defer googleManager.Close()

	firebaseManager, err := firebase.NewFirebaseClientManager(ctx, "")
	if err != nil {
		t.Fatalf("Failed to create Firebase manager: %v", err)
	}
	defer firebaseManager.Close()

	// Both should work if credentials are configured correctly
	if googleManager.GetProjectID() == "" {
		t.Error("Google manager has no project ID")
	}

	if firebaseManager.GetProjectID() == "" {
		t.Error("Firebase manager has no project ID")
	}
}
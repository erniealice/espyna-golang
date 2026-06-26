package gcs

import (
	"context"
	"testing"
)

// TestGCSStorageIntegration tests Google Cloud Storage client creation against
// the STORAGE_GCS_ credential set. It is gated by testing.Short().
func TestGCSStorageIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	manager, err := NewGCSClientManager(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to create GCS client manager: %v", err)
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
